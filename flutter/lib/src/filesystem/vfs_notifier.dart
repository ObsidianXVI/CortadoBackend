import 'dart:async';

import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../cortado_workspace_provider.dart';
import '../gen/agent/v1/agent.pb.dart' as agentpb;
import '../workspace_manager.dart';
import 'vfs_node.dart';

const String vfsRootPath = '/';

final cortadoVfsProvider =
    AsyncNotifierProvider.autoDispose<VfsNotifier, Map<String, VfsNode>>(
  VfsNotifier.new,
);

class VfsNotifier extends AutoDisposeAsyncNotifier<Map<String, VfsNode>> {
  @override
  Map<String, VfsNode> build() => <String, VfsNode>{
        vfsRootPath: const VfsNode.directory(
          path: vfsRootPath,
          name: '',
          childPaths: <String>[],
        ),
      };

  Future<void> loadDirectory(String path, {bool force = false}) async {
    final normalizedPath = normalizeVfsPath(path);
    final current = state.value ?? build();
    final directory = current[normalizedPath];
    if (directory is! VfsDir) {
      throw ArgumentError.value(
        path,
        'path',
        'A directory path is required to load children.',
      );
    }
    if (directory.loaded && !force) {
      return;
    }

    final entries = await _manager.listDirectory(
      _workspaceId,
      path: normalizedPath,
    );
    state = AsyncData(_mergeDirectoryListing(current, directory, entries));
  }

  Future<void> setDirectoryExpanded(String path, bool expanded) async {
    final normalizedPath = normalizeVfsPath(path);
    final current = state.value ?? build();
    final directory = current[normalizedPath];
    if (directory is! VfsDir) {
      return;
    }

    if (expanded && !directory.loaded) {
      await loadDirectory(normalizedPath);
    }

    final refreshed = state.value ?? current;
    final currentDirectory = refreshed[normalizedPath];
    if (currentDirectory is! VfsDir || currentDirectory.expanded == expanded) {
      return;
    }

    final updated = Map<String, VfsNode>.of(refreshed);
    updated[normalizedPath] = currentDirectory.copyWith(expanded: expanded);
    state = AsyncData(updated);
  }

  Future<void> applyEvent(agentpb.FileEvent event) async {
    final path = normalizeVfsPath(event.path);
    switch (event.type) {
      case agentpb.FileEventType.FILE_EVENT_TYPE_CREATED:
      case agentpb.FileEventType.FILE_EVENT_TYPE_MODIFIED:
        return loadDirectory(parentVfsPath(path), force: true);
      case agentpb.FileEventType.FILE_EVENT_TYPE_DELETED:
      case agentpb.FileEventType.FILE_EVENT_TYPE_RENAMED:
        final current = state.value ?? build();
        final updated = Map<String, VfsNode>.of(current);
        _removeSubtree(updated, path);

        final parentPath = parentVfsPath(path);
        if (updated[parentPath] case final VfsDir parentDir) {
          updated[parentPath] = parentDir.copyWith(
            childPaths: parentDir.childPaths
                .where((childPath) => childPath != path)
                .toList(growable: false),
          );
        }

        state = AsyncData(updated);
        return;
      case agentpb.FileEventType.FILE_EVENT_TYPE_UNSPECIFIED:
        return;
    }
  }

  WorkspaceManager get _manager => ref.read(cortadoWorkspaceManagerProvider);
  String get _workspaceId => ref.read(cortadoWorkspaceIdProvider);

  Map<String, VfsNode> _mergeDirectoryListing(
    Map<String, VfsNode> current,
    VfsDir directory,
    List<WorkspaceDirectoryEntry> entries,
  ) {
    final updated = Map<String, VfsNode>.of(current);
    final childPaths = entries
        .map((entry) => childVfsPath(directory.path, entry.name))
        .toList(growable: false)
      ..sort();

    for (final existingChild in directory.childPaths) {
      if (!childPaths.contains(existingChild)) {
        _removeSubtree(updated, existingChild);
      }
    }

    for (final entry in entries) {
      final childPath = childVfsPath(directory.path, entry.name);
      final existingNode = updated[childPath];
      if (!entry.isDir && existingNode is VfsDir) {
        _removeSubtree(updated, childPath);
      }
      updated[childPath] = switch ((entry.isDir, existingNode)) {
        (true, VfsDir dir) => dir.copyWith(name: entry.name),
        (true, _) => VfsNode.directory(
            path: childPath,
            name: entry.name,
            childPaths: const <String>[],
          ),
        (false, _) => VfsNode.file(
            path: childPath,
            name: entry.name,
            size: entry.size,
            modTime: entry.modTime.toUtc(),
          ),
      };
    }

    if (listEquals(directory.childPaths, childPaths) && directory.loaded) {
      updated[directory.path] = directory;
    } else {
      updated[directory.path] = directory.copyWith(
        childPaths: childPaths,
        loaded: true,
      );
    }

    return updated;
  }
}

String childVfsPath(String parentPath, String childName) {
  final normalizedParent = normalizeVfsPath(parentPath);
  if (normalizedParent == vfsRootPath) {
    return normalizeVfsPath('/$childName');
  }
  return normalizeVfsPath('$normalizedParent/$childName');
}

String normalizeVfsPath(String path) {
  final trimmed = path.trim().replaceAll('\\', '/');
  if (trimmed.isEmpty || trimmed == '.' || trimmed == '/') {
    return vfsRootPath;
  }

  final normalizedSegments = <String>[];
  for (final segment in trimmed.split('/')) {
    if (segment.isEmpty || segment == '.') {
      continue;
    }
    if (segment == '..') {
      if (normalizedSegments.isNotEmpty) {
        normalizedSegments.removeLast();
      }
      continue;
    }
    normalizedSegments.add(segment);
  }

  if (normalizedSegments.isEmpty) {
    return vfsRootPath;
  }
  return '/${normalizedSegments.join('/')}';
}

String parentVfsPath(String path) {
  final normalizedPath = normalizeVfsPath(path);
  if (normalizedPath == vfsRootPath) {
    return vfsRootPath;
  }

  final slashIndex = normalizedPath.lastIndexOf('/');
  if (slashIndex <= 0) {
    return vfsRootPath;
  }
  return normalizedPath.substring(0, slashIndex);
}

void _removeSubtree(Map<String, VfsNode> nodes, String path) {
  final normalizedPath = normalizeVfsPath(path);
  final descendants = nodes.keys
      .where(
        (candidate) =>
            candidate == normalizedPath ||
            candidate.startsWith('$normalizedPath/'),
      )
      .toList(growable: false);
  for (final descendant in descendants) {
    nodes.remove(descendant);
  }
}
