import 'dart:async';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../cortado_client.dart';
import '../cortado_workspace_provider.dart';
import '../editor/editor_diagnostics.dart';
import '../gen/agent/v1/agent.pb.dart' as agentpb;
import '../mux_frame.dart';
import '../workspace_manager.dart';
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
  String? _activePath;
  FocusNode? _renameFocusNode;
  TextEditingController? _renameController;
  _RenameSession? _renameSession;
  bool _renameSubmitting = false;
  final FocusNode _treeFocusNode = FocusNode(debugLabel: 'cortado-file-tree');

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
    _disposeRenameEditor();
    _treeFocusNode.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final nodes =
        ref.watch(cortadoVfsProvider).value ?? const <String, VfsNode>{};
    final diagnosticStatuses =
        ref.watch(cortadoWorkspaceDiagnosticStatusProvider);
    final visibleNodes = _visibleNodes(
      nodes,
      rootPath: normalizeVfsPath(widget.rootPath),
    );
    final selectedPath = normalizeVfsPath(
      widget.selectedPath ?? _activePath ?? '',
    );

    return Shortcuts(
      shortcuts: const <ShortcutActivator, Intent>{
        SingleActivator(LogicalKeyboardKey.f2): _RenameNodeIntent(),
      },
      child: Actions(
        actions: <Type, Action<Intent>>{
          _RenameNodeIntent: CallbackAction<_RenameNodeIntent>(
            onInvoke: (intent) {
              _handleRenameShortcut(nodes);
              return null;
            },
          ),
        },
        child: Focus(
          focusNode: _treeFocusNode,
          child: ListView.builder(
            itemCount: visibleNodes.length,
            itemBuilder: (context, index) {
              final visibleNode = visibleNodes[index];
              final node = visibleNode.node;
              return FileTreeRow(
                depth: visibleNode.depth,
                expanded: node is VfsDir ? node.expanded : null,
                indent: widget.indent,
                isDirectory: node is VfsDir,
                label: _labelForNode(node),
                labelWidget: _buildLabelWidget(node),
                trailing: node is VfsDir
                    ? null
                    : switch (diagnosticStatuses[node.path] ??
                        CortadoFileDiagnosticStatus.none) {
                        CortadoFileDiagnosticStatus.error => _FileDiagnosticDot(
                            key: ValueKey(
                              'file-tree-diagnostic-dot:${node.path}',
                            ),
                            color: Color(0xFFEF4444),
                          ),
                        CortadoFileDiagnosticStatus.warning =>
                          _FileDiagnosticDot(
                            key: ValueKey(
                              'file-tree-diagnostic-dot:${node.path}',
                            ),
                            color: Color(0xFFF59E0B),
                          ),
                        CortadoFileDiagnosticStatus.none => null,
                      },
                selected: selectedPath == node.path,
                onLongPressStart: (details) =>
                    _showContextMenu(node, details.globalPosition),
                onSecondaryTapDown: (details) =>
                    _showContextMenu(node, details.globalPosition),
                onTap: () => _handleNodeTap(node),
              );
            },
          ),
        ),
      ),
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
    _setActivePath(node.path);
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

  void _handleRenameShortcut(Map<String, VfsNode> nodes) {
    final selectedPath =
        normalizeVfsPath(widget.selectedPath ?? _activePath ?? '');
    final node = nodes[selectedPath];
    if (node == null || node.path == vfsRootPath) {
      return;
    }
    _beginRename(node);
  }

  void _setActivePath(String path) {
    final normalizedPath = normalizeVfsPath(path);
    if (_activePath != normalizedPath && mounted) {
      setState(() {
        _activePath = normalizedPath;
      });
    }
    if (!_treeFocusNode.hasFocus) {
      _treeFocusNode.requestFocus();
    }
  }

  String _labelForNode(VfsNode node) {
    if (node.path == vfsRootPath) {
      return '/';
    }
    return node.name;
  }

  Widget? _buildLabelWidget(VfsNode node) {
    if (_renameSession?.path != node.path) {
      return null;
    }

    final controller = _renameController;
    final focusNode = _renameFocusNode;
    if (controller == null || focusNode == null) {
      return null;
    }

    return TextField(
      autofocus: false,
      controller: controller,
      focusNode: focusNode,
      onSubmitted: (_) => unawaited(_submitRename()),
      decoration: const InputDecoration(
        border: InputBorder.none,
        isDense: true,
        contentPadding: EdgeInsets.zero,
      ),
      style: const TextStyle(
        color: Color(0xFF111827),
        fontSize: 13,
        fontWeight: FontWeight.w600,
      ),
    );
  }

  Future<void> _showContextMenu(VfsNode node, Offset globalPosition) async {
    _setActivePath(node.path);

    final overlay = Overlay.of(context).context.findRenderObject() as RenderBox;
    final selection = await showMenu<_FileTreeAction>(
      context: context,
      position: RelativeRect.fromRect(
        Rect.fromPoints(globalPosition, globalPosition),
        Offset.zero & overlay.size,
      ),
      items: <PopupMenuEntry<_FileTreeAction>>[
        if (node is VfsDir)
          const PopupMenuItem<_FileTreeAction>(
            value: _FileTreeAction.newFile,
            child: Text('New File'),
          ),
        if (node is VfsDir)
          const PopupMenuItem<_FileTreeAction>(
            value: _FileTreeAction.newFolder,
            child: Text('New Folder'),
          ),
        const PopupMenuItem<_FileTreeAction>(
          value: _FileTreeAction.rename,
          child: Text('Rename'),
        ),
        const PopupMenuItem<_FileTreeAction>(
          value: _FileTreeAction.delete,
          child: Text('Delete'),
        ),
      ],
    );

    switch (selection) {
      case _FileTreeAction.newFile:
        if (node case final VfsDir directory) {
          await _createChild(directory, isDirectory: false);
        }
        return;
      case _FileTreeAction.newFolder:
        if (node case final VfsDir directory) {
          await _createChild(directory, isDirectory: true);
        }
        return;
      case _FileTreeAction.rename:
        _beginRename(node);
        return;
      case _FileTreeAction.delete:
        await _confirmDelete(node);
        return;
      case null:
        return;
    }
  }

  Future<void> _createChild(
    VfsDir directory, {
    required bool isDirectory,
  }) async {
    final name = await _promptForName(
      title: isDirectory ? 'New Folder' : 'New File',
      actionLabel: isDirectory ? 'Create Folder' : 'Create File',
      initialValue: '',
    );
    if (name == null) {
      return;
    }

    final childPath = childVfsPath(directory.path, name);
    try {
      if (isDirectory) {
        await _manager.makeDir(_workspaceId, path: childPath);
      } else {
        await _manager.writeFile(_workspaceId, path: childPath);
        widget.onFileSelected?.call(childPath);
      }

      await ref.read(cortadoVfsProvider.notifier).setDirectoryExpanded(
            directory.path,
            true,
          );
      await ref.read(cortadoVfsProvider.notifier).loadDirectory(
            directory.path,
            force: true,
          );
      _setActivePath(childPath);
    } catch (error) {
      widget.onError?.call(error.toString());
    }
  }

  void _beginRename(VfsNode node) {
    _setActivePath(node.path);
    _disposeRenameEditor();

    final focusNode = FocusNode(debugLabel: 'file-tree-rename');
    focusNode.addListener(_handleRenameFocusChange);
    final controller = TextEditingController(text: _labelForNode(node));

    setState(() {
      _renameSession = _RenameSession(
        isDirectory: node is VfsDir,
        parentPath: parentVfsPath(node.path),
        path: node.path,
      );
      _renameController = controller;
      _renameFocusNode = focusNode;
    });

    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted || _renameSession?.path != node.path) {
        return;
      }
      focusNode.requestFocus();
      controller.selection = TextSelection(
        baseOffset: 0,
        extentOffset: controller.text.length,
      );
    });
  }

  void _handleRenameFocusChange() {
    if (_renameFocusNode case final FocusNode focusNode
        when !focusNode.hasFocus) {
      unawaited(_submitRename());
    }
  }

  Future<void> _submitRename() async {
    final session = _renameSession;
    final controller = _renameController;
    if (session == null || controller == null || _renameSubmitting) {
      return;
    }

    final nextName = controller.text.trim();
    if (nextName.isEmpty) {
      _cancelRename();
      return;
    }

    final nextPath = childVfsPath(session.parentPath, nextName);
    if (nextPath == session.path) {
      _cancelRename();
      return;
    }

    _renameSubmitting = true;
    try {
      await _manager.renamePath(
        _workspaceId,
        oldPath: session.path,
        newPath: nextPath,
      );
      await _refreshAfterRename(session.path, nextPath);
      _setActivePath(nextPath);
      if (!session.isDirectory &&
          normalizeVfsPath(widget.selectedPath ?? _activePath ?? '') ==
              session.path) {
        widget.onFileSelected?.call(nextPath);
      }
    } catch (error) {
      widget.onError?.call(error.toString());
    } finally {
      _renameSubmitting = false;
      _cancelRename();
    }
  }

  Future<void> _refreshAfterRename(String oldPath, String newPath) async {
    final notifier = ref.read(cortadoVfsProvider.notifier);
    final oldParent = parentVfsPath(oldPath);
    final newParent = parentVfsPath(newPath);
    await notifier.loadDirectory(oldParent, force: true);
    if (newParent != oldParent) {
      await notifier.loadDirectory(newParent, force: true);
    }
  }

  Future<void> _confirmDelete(VfsNode node) async {
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (context) {
        return AlertDialog(
          title: const Text('Delete'),
          content: Text('Delete ${_labelForNode(node)}?'),
          actions: <Widget>[
            TextButton(
              onPressed: () => Navigator.of(context).pop(false),
              child: const Text('Cancel'),
            ),
            TextButton(
              onPressed: () => Navigator.of(context).pop(true),
              child: const Text('Delete'),
            ),
          ],
        );
      },
    );
    if (confirmed != true) {
      return;
    }

    try {
      await _manager.deletePath(_workspaceId, path: node.path);
      await ref.read(cortadoVfsProvider.notifier).loadDirectory(
            parentVfsPath(node.path),
            force: true,
          );
      if (_activePath == node.path) {
        setState(() {
          _activePath = null;
        });
      }
    } catch (error) {
      widget.onError?.call(error.toString());
    }
  }

  Future<String?> _promptForName({
    required String title,
    required String actionLabel,
    required String initialValue,
  }) async {
    final controller = TextEditingController(text: initialValue);
    final focusNode = FocusNode(debugLabel: 'file-tree-prompt');

    final result = await showDialog<String>(
      context: context,
      builder: (context) {
        WidgetsBinding.instance.addPostFrameCallback((_) {
          if (!focusNode.hasFocus) {
            focusNode.requestFocus();
            controller.selection = TextSelection(
              baseOffset: 0,
              extentOffset: controller.text.length,
            );
          }
        });

        return AlertDialog(
          title: Text(title),
          content: TextField(
            controller: controller,
            focusNode: focusNode,
            onSubmitted: (value) => Navigator.of(context).pop(value.trim()),
          ),
          actions: <Widget>[
            TextButton(
              onPressed: () => Navigator.of(context).pop(),
              child: const Text('Cancel'),
            ),
            TextButton(
              onPressed: () =>
                  Navigator.of(context).pop(controller.text.trim()),
              child: Text(actionLabel),
            ),
          ],
        );
      },
    );

    controller.dispose();
    focusNode.dispose();

    if (result == null || result.trim().isEmpty) {
      return null;
    }
    return result.trim();
  }

  void _cancelRename() {
    if (!mounted) {
      _disposeRenameEditor();
      return;
    }
    setState(_disposeRenameEditor);
  }

  void _disposeRenameEditor() {
    _renameFocusNode?.removeListener(_handleRenameFocusChange);
    _renameFocusNode?.dispose();
    _renameFocusNode = null;
    _renameController?.dispose();
    _renameController = null;
    _renameSession = null;
  }

  WorkspaceManager get _manager => ref.read(cortadoWorkspaceManagerProvider);
  String get _workspaceId => ref.read(cortadoWorkspaceIdProvider);

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
    this.labelWidget,
    this.onLongPressStart,
    this.onSecondaryTapDown,
    this.selected = false,
    this.trailing,
  });

  final int depth;
  final double indent;
  final bool isDirectory;
  final String label;
  final VoidCallback onTap;
  final bool? expanded;
  final Widget? labelWidget;
  final GestureLongPressStartCallback? onLongPressStart;
  final GestureTapDownCallback? onSecondaryTapDown;
  final bool selected;
  final Widget? trailing;

  @override
  Widget build(BuildContext context) {
    final rowColor = selected ? const Color(0x142F6FEB) : Colors.transparent;
    final accentColor =
        selected ? const Color(0xFF2F6FEB) : const Color(0xFF4B5563);
    final icon = isDirectory
        ? expanded == true
            ? Icons.keyboard_arrow_down
            : Icons.keyboard_arrow_right
        : Icons.insert_drive_file_outlined;

    return DecoratedBox(
      decoration: BoxDecoration(color: rowColor),
      child: GestureDetector(
        behavior: HitTestBehavior.opaque,
        onLongPressStart: onLongPressStart,
        onSecondaryTapDown: onSecondaryTapDown,
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
                    expanded == true
                        ? Icons.folder_open
                        : Icons.folder_outlined,
                    size: 18,
                    color: const Color(0xFFB7791F),
                  ),
                )
              else
                const SizedBox(width: 6),
              Expanded(
                child: labelWidget ??
                    Text(
                      label,
                      overflow: TextOverflow.ellipsis,
                      style: TextStyle(
                        color: const Color(0xFF111827),
                        fontSize: 13,
                        fontWeight:
                            selected ? FontWeight.w600 : FontWeight.w400,
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

class _FileDiagnosticDot extends StatelessWidget {
  const _FileDiagnosticDot({
    required this.color,
    super.key,
  });

  final Color color;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsetsDirectional.only(start: 8),
      child: DecoratedBox(
        decoration: BoxDecoration(
          color: color,
          shape: BoxShape.circle,
        ),
        child: const SizedBox(width: 8, height: 8),
      ),
    );
  }
}

enum _FileTreeAction {
  newFile,
  newFolder,
  rename,
  delete,
}

class _RenameNodeIntent extends Intent {
  const _RenameNodeIntent();
}

class _RenameSession {
  const _RenameSession({
    required this.isDirectory,
    required this.parentPath,
    required this.path,
  });

  final bool isDirectory;
  final String parentPath;
  final String path;
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
