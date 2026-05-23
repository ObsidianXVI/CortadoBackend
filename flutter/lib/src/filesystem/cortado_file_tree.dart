import 'dart:async';
import 'dart:typed_data';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../cortado_client.dart';
import '../gen/agent/v1/agent.pb.dart' as agentpb;
import '../mux_frame.dart';
import 'vfs_node.dart';
import 'vfs_notifier.dart';

class CortadoFileTree extends ConsumerStatefulWidget {
  const CortadoFileTree({
    super.key,
    required this.client,
    this.channelId = muxFileSyncChannelId,
    this.rootPath = vfsRootPath,
    this.autoLoadRoot = true,
    this.autoWatch = true,
    this.indent = 16,
    this.selectedPath,
    this.onClosed,
    this.onError,
    this.onFileSelected,
  });

  final CortadoClient client;
  final int channelId;
  final String rootPath;
  final bool autoLoadRoot;
  final bool autoWatch;
  final double indent;
  final String? selectedPath;
  final ValueChanged<String>? onClosed;
  final ValueChanged<String>? onError;
  final ValueChanged<String>? onFileSelected;

  @override
  ConsumerState<CortadoFileTree> createState() => _CortadoFileTreeState();
}

class _CortadoFileTreeState extends ConsumerState<CortadoFileTree> {
  StreamSubscription<MuxFrame>? _frameSubscription;
  bool _watchOpen = false;

  @override
  void initState() {
    super.initState();
    _subscribeToFrames();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted) {
        return;
      }
      unawaited(_initializeTree());
    });
  }

  @override
  void didUpdateWidget(CortadoFileTree oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.client == widget.client &&
        oldWidget.channelId == widget.channelId &&
        oldWidget.rootPath == widget.rootPath &&
        oldWidget.autoLoadRoot == widget.autoLoadRoot &&
        oldWidget.autoWatch == widget.autoWatch) {
      return;
    }

    _frameSubscription?.cancel();
    unawaited(_closeWatchChannel());
    _watchOpen = false;
    _subscribeToFrames();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted) {
        return;
      }
      unawaited(_initializeTree(forceRootLoad: true));
    });
  }

  @override
  void dispose() {
    _frameSubscription?.cancel();
    unawaited(_closeWatchChannel());
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final nodes = ref.watch(cortadoVfsProvider).value ?? const <String, VfsNode>{};
    final visibleNodes = _visibleNodes(
      nodes,
      rootPath: normalizeVfsPath(widget.rootPath),
    );

    return ListView.builder(
      itemCount: visibleNodes.length,
      itemBuilder: (context, index) {
        final visibleNode = visibleNodes[index];
        final node = visibleNode.node;
        return FileTreeRow(
          depth: visibleNode.depth,
          expanded: node is VfsDir ? node.expanded : null,
          indent: widget.indent,
          isDirectory: node is VfsDir,
          label: node.name,
          selected: normalizeVfsPath(widget.selectedPath ?? '') == node.path,
          onTap: () => _handleNodeTap(node),
        );
      },
    );
  }

  Future<void> _initializeTree({bool forceRootLoad = false}) async {
    if (widget.autoLoadRoot) {
      try {
        await ref.read(cortadoVfsProvider.notifier).loadDirectory(
              widget.rootPath,
              force: forceRootLoad,
            );
      } catch (error) {
        widget.onError?.call(error.toString());
      }
    }

    if (widget.autoWatch) {
      _openWatchChannel();
    }
  }

  void _subscribeToFrames() {
    _frameSubscription =
        widget.client.framesForChannel(widget.channelId).listen(_handleFrame);
  }

  Future<void> _handleFrame(MuxFrame frame) async {
    switch (frame.messageType) {
      case muxMessageTypeData:
        try {
          final event = agentpb.FileEvent.fromBuffer(frame.payload);
          await ref.read(cortadoVfsProvider.notifier).applyEvent(event);
        } catch (error) {
          widget.onError?.call(error.toString());
        }
        return;
      case muxMessageTypeError:
        widget.onError?.call(String.fromCharCodes(frame.payload));
        return;
      case muxMessageTypeClose:
        _watchOpen = false;
        widget.onClosed?.call(String.fromCharCodes(frame.payload));
        return;
      default:
        return;
    }
  }

  Future<void> _handleNodeTap(VfsNode node) async {
    if (node case final VfsDir directory) {
      try {
        await ref.read(cortadoVfsProvider.notifier).setDirectoryExpanded(
              directory.path,
              !directory.expanded,
            );
      } catch (error) {
        widget.onError?.call(error.toString());
      }
      return;
    }

    widget.onFileSelected?.call(node.path);
  }

  void _openWatchChannel() {
    if (_watchOpen) {
      return;
    }

    try {
      widget.client.sendFrame(
        widget.channelId,
        muxMessageTypeOpen,
        Uint8List(0),
      );
      _watchOpen = true;
    } catch (error) {
      widget.onError?.call(error.toString());
    }
  }

  Future<void> _closeWatchChannel() async {
    if (!_watchOpen) {
      return;
    }

    try {
      widget.client.sendFrame(
        widget.channelId,
        muxMessageTypeClose,
        Uint8List(0),
      );
    } catch (_) {
      // Ignore close failures during disposal or disconnected sockets.
    } finally {
      _watchOpen = false;
    }
  }
}

class FileTreeRow extends StatelessWidget {
  const FileTreeRow({
    super.key,
    required this.depth,
    required this.indent,
    required this.isDirectory,
    required this.label,
    required this.onTap,
    this.expanded,
    this.selected = false,
    this.trailing,
  });

  final int depth;
  final double indent;
  final bool isDirectory;
  final String label;
  final VoidCallback onTap;
  final bool? expanded;
  final bool selected;
  final Widget? trailing;

  @override
  Widget build(BuildContext context) {
    final rowColor = selected ? const Color(0x142F6FEB) : Colors.transparent;
    final accentColor = selected ? const Color(0xFF2F6FEB) : const Color(0xFF4B5563);
    final icon = isDirectory
        ? expanded == true
            ? Icons.keyboard_arrow_down
            : Icons.keyboard_arrow_right
        : Icons.insert_drive_file_outlined;

    return DecoratedBox(
      decoration: BoxDecoration(color: rowColor),
      child: GestureDetector(
        behavior: HitTestBehavior.opaque,
        onTap: onTap,
        child: Padding(
          padding: EdgeInsetsDirectional.only(
            start: depth * indent,
            top: 6,
            end: 12,
            bottom: 6,
          ),
          child: Row(
            children: <Widget>[
              SizedBox(
                width: 20,
                child: Icon(
                  icon,
                  size: 18,
                  color: isDirectory ? const Color(0xFFB7791F) : accentColor,
                ),
              ),
              if (isDirectory)
                Padding(
                  padding: const EdgeInsetsDirectional.only(end: 6),
                  child: Icon(
                    expanded == true ? Icons.folder_open : Icons.folder_outlined,
                    size: 18,
                    color: const Color(0xFFB7791F),
                  ),
                )
              else
                const SizedBox(width: 6),
              Expanded(
                child: Text(
                  label,
                  overflow: TextOverflow.ellipsis,
                  style: TextStyle(
                    color: const Color(0xFF111827),
                    fontSize: 13,
                    fontWeight: selected ? FontWeight.w600 : FontWeight.w400,
                  ),
                ),
              ),
              if (trailing != null) trailing!,
            ],
          ),
        ),
      ),
    );
  }
}

class _VisibleNode {
  const _VisibleNode({
    required this.depth,
    required this.node,
  });

  final int depth;
  final VfsNode node;
}

List<_VisibleNode> _visibleNodes(
  Map<String, VfsNode> nodes, {
  required String rootPath,
}) {
  final rootNode = nodes[rootPath];
  if (rootNode is! VfsDir) {
    return const <_VisibleNode>[];
  }

  final visible = <_VisibleNode>[];

  void visitDirectory(VfsDir directory, int depth) {
    final childNodes = directory.childPaths
        .map((childPath) => nodes[childPath])
        .whereType<VfsNode>()
        .toList(growable: false)
      ..sort(_compareNodes);

    for (final child in childNodes) {
      visible.add(_VisibleNode(depth: depth, node: child));
      if (child case final VfsDir childDirectory when childDirectory.expanded) {
        visitDirectory(childDirectory, depth + 1);
      }
    }
  }

  visitDirectory(rootNode, 0);
  return visible;
}

int _compareNodes(VfsNode left, VfsNode right) {
  final leftIsDir = left is VfsDir;
  final rightIsDir = right is VfsDir;
  if (leftIsDir != rightIsDir) {
    return leftIsDir ? -1 : 1;
  }
  return left.name.toLowerCase().compareTo(right.name.toLowerCase());
}
