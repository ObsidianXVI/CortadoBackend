import 'dart:collection';

import 'package:cortado/cortado.dart';
import 'package:cortado/src/gen/agent/v1/agent.pbenum.dart' as agentpbenum;
import 'package:cortado/src/gen/agent/v1/agent.pb.dart' as agentpb;
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('VfsNotifier', () {
    test('starts with an unloaded root directory', () {
      final container = ProviderContainer(
        overrides: <Override>[
          cortadoWorkspaceManagerProvider
              .overrideWith((ref) => _FakeWorkspaceManager()),
          cortadoWorkspaceIdProvider.overrideWith((ref) => 'ws-123'),
        ],
      );
      addTearDown(container.dispose);

      final state = container.read(cortadoVfsProvider).requireValue;
      expect(state.keys, contains(vfsRootPath));
      expect(
          state[vfsRootPath],
          const VfsNode.directory(
            path: vfsRootPath,
            name: '',
            childPaths: <String>[],
          ));
    });

    test('loads directories lazily and preserves later expands', () async {
      final manager = _FakeWorkspaceManager(
        responses: <String, List<List<WorkspaceDirectoryEntry>>>{
          vfsRootPath: <List<WorkspaceDirectoryEntry>>[
            <WorkspaceDirectoryEntry>[
              WorkspaceDirectoryEntry(
                name: 'lib',
                size: 0,
                isDir: true,
                modTime: DateTime.utc(2026, 5, 23, 21),
                permissions: 493,
              ),
              WorkspaceDirectoryEntry(
                name: 'README.md',
                size: 10,
                isDir: false,
                modTime: DateTime.utc(2026, 5, 23, 21),
                permissions: 420,
              ),
            ],
          ],
          '/lib': <List<WorkspaceDirectoryEntry>>[
            <WorkspaceDirectoryEntry>[
              WorkspaceDirectoryEntry(
                name: 'main.dart',
                size: 42,
                isDir: false,
                modTime: DateTime.utc(2026, 5, 23, 21, 5),
                permissions: 420,
              ),
            ],
          ],
        },
      );
      final container = ProviderContainer(
        overrides: <Override>[
          cortadoWorkspaceManagerProvider.overrideWith((ref) => manager),
          cortadoWorkspaceIdProvider.overrideWith((ref) => 'ws-123'),
        ],
      );
      addTearDown(container.dispose);

      final notifier = container.read(cortadoVfsProvider.notifier);

      await notifier.loadDirectory(vfsRootPath);
      var state = container.read(cortadoVfsProvider).requireValue;
      expect(
        (state[vfsRootPath] as VfsDir).childPaths,
        const <String>['/README.md', '/lib'],
      );
      expect(state['/README.md'], isA<VfsFile>());
      expect(state['/lib'], isA<VfsDir>());

      await notifier.setDirectoryExpanded('/lib', true);
      state = container.read(cortadoVfsProvider).requireValue;
      expect((state['/lib'] as VfsDir).loaded, isTrue);
      expect((state['/lib'] as VfsDir).expanded, isTrue);
      expect((state['/lib'] as VfsDir).childPaths,
          const <String>['/lib/main.dart']);

      await notifier.setDirectoryExpanded('/lib', true);
      expect(manager.requests, const <String>['ws-123:/', 'ws-123:/lib']);
    });

    test('applyEvent removes deleted nodes and refreshes modified parents',
        () async {
      final manager = _FakeWorkspaceManager(
        responses: <String, List<List<WorkspaceDirectoryEntry>>>{
          vfsRootPath: <List<WorkspaceDirectoryEntry>>[
            <WorkspaceDirectoryEntry>[
              WorkspaceDirectoryEntry(
                name: 'lib',
                size: 0,
                isDir: true,
                modTime: DateTime.utc(2026, 5, 23, 21),
                permissions: 493,
              ),
              WorkspaceDirectoryEntry(
                name: 'README.md',
                size: 10,
                isDir: false,
                modTime: DateTime.utc(2026, 5, 23, 21),
                permissions: 420,
              ),
            ],
          ],
          '/lib': <List<WorkspaceDirectoryEntry>>[
            <WorkspaceDirectoryEntry>[
              WorkspaceDirectoryEntry(
                name: 'main.dart',
                size: 1,
                isDir: false,
                modTime: DateTime.utc(2026, 5, 23, 21, 5),
                permissions: 420,
              ),
            ],
            <WorkspaceDirectoryEntry>[
              WorkspaceDirectoryEntry(
                name: 'main.dart',
                size: 2,
                isDir: false,
                modTime: DateTime.utc(2026, 5, 23, 21, 6),
                permissions: 420,
              ),
            ],
          ],
        },
      );
      final container = ProviderContainer(
        overrides: <Override>[
          cortadoWorkspaceManagerProvider.overrideWith((ref) => manager),
          cortadoWorkspaceIdProvider.overrideWith((ref) => 'ws-123'),
        ],
      );
      addTearDown(container.dispose);

      final notifier = container.read(cortadoVfsProvider.notifier);
      await notifier.loadDirectory(vfsRootPath);
      await notifier.loadDirectory('/lib');
      final libBefore = container.read(cortadoVfsProvider).requireValue['/lib'];

      await notifier.applyEvent(
        agentpb.FileEvent(
          path: 'README.md',
          type: agentpbenum.FileEventType.FILE_EVENT_TYPE_DELETED,
        ),
      );

      var state = container.read(cortadoVfsProvider).requireValue;
      expect(state.containsKey('/README.md'), isFalse);
      expect((state[vfsRootPath] as VfsDir).childPaths, const <String>['/lib']);

      await notifier.applyEvent(
        agentpb.FileEvent(
          path: 'lib/main.dart',
          type: agentpbenum.FileEventType.FILE_EVENT_TYPE_MODIFIED,
        ),
      );

      state = container.read(cortadoVfsProvider).requireValue;
      expect((state['/lib/main.dart'] as VfsFile).size, 2);
      expect(identical(state['/lib'], libBefore), isTrue);
      expect(manager.requests,
          const <String>['ws-123:/', 'ws-123:/lib', 'ws-123:/lib']);
    });

    test(
        'loadDirectory removes stale descendants when a directory becomes a file',
        () async {
      final manager = _FakeWorkspaceManager(
        responses: <String, List<List<WorkspaceDirectoryEntry>>>{
          vfsRootPath: <List<WorkspaceDirectoryEntry>>[
            <WorkspaceDirectoryEntry>[
              WorkspaceDirectoryEntry(
                name: 'docs',
                size: 0,
                isDir: true,
                modTime: DateTime.utc(2026, 5, 23, 21),
                permissions: 493,
              ),
            ],
            <WorkspaceDirectoryEntry>[
              WorkspaceDirectoryEntry(
                name: 'docs',
                size: 12,
                isDir: false,
                modTime: DateTime.utc(2026, 5, 23, 21, 10),
                permissions: 420,
              ),
            ],
          ],
          '/docs': <List<WorkspaceDirectoryEntry>>[
            <WorkspaceDirectoryEntry>[
              WorkspaceDirectoryEntry(
                name: 'guide.md',
                size: 4,
                isDir: false,
                modTime: DateTime.utc(2026, 5, 23, 21, 5),
                permissions: 420,
              ),
            ],
          ],
        },
      );
      final container = ProviderContainer(
        overrides: <Override>[
          cortadoWorkspaceManagerProvider.overrideWith((ref) => manager),
          cortadoWorkspaceIdProvider.overrideWith((ref) => 'ws-123'),
        ],
      );
      addTearDown(container.dispose);

      final notifier = container.read(cortadoVfsProvider.notifier);
      await notifier.loadDirectory(vfsRootPath);
      await notifier.loadDirectory('/docs');
      await notifier.loadDirectory(vfsRootPath, force: true);

      final state = container.read(cortadoVfsProvider).requireValue;
      expect(state['/docs'], isA<VfsFile>());
      expect(state.containsKey('/docs/guide.md'), isFalse);
    });

    test('normalizes VFS paths with duplicate separators and dot segments', () {
      expect(normalizeVfsPath(''), vfsRootPath);
      expect(normalizeVfsPath('./lib//src/../main.dart'), '/lib/main.dart');
      expect(parentVfsPath('/lib/main.dart'), '/lib');
      expect(childVfsPath(vfsRootPath, 'lib'), '/lib');
    });

    test('applies local daemon sync status to existing and later-loaded nodes',
        () async {
      final manager = _FakeWorkspaceManager(
        responses: <String, List<List<WorkspaceDirectoryEntry>>>{
          vfsRootPath: <List<WorkspaceDirectoryEntry>>[
            <WorkspaceDirectoryEntry>[
              WorkspaceDirectoryEntry(
                name: 'lib',
                size: 0,
                isDir: true,
                modTime: DateTime.utc(2026, 5, 24, 9),
                permissions: 493,
              ),
            ],
          ],
          '/lib': <List<WorkspaceDirectoryEntry>>[
            <WorkspaceDirectoryEntry>[
              WorkspaceDirectoryEntry(
                name: 'main.dart',
                size: 42,
                isDir: false,
                modTime: DateTime.utc(2026, 5, 24, 9, 5),
                permissions: 420,
              ),
            ],
          ],
        },
      );
      final container = ProviderContainer(
        overrides: <Override>[
          cortadoWorkspaceManagerProvider.overrideWith((ref) => manager),
          cortadoWorkspaceIdProvider.overrideWith((ref) => 'ws-123'),
        ],
      );
      addTearDown(container.dispose);

      final notifier = container.read(cortadoVfsProvider.notifier);
      await notifier.loadDirectory(vfsRootPath);

      notifier.applyLocalDaemonSyncStatus(
        const CortadoLocalDaemonSyncStatus(
          localPath: '/tmp/workspace',
          state: CortadoLocalDaemonSyncState.syncing,
          workspaceId: 'ws-123',
          workspacePath: '/lib/main.dart',
        ),
      );

      await notifier.loadDirectory('/lib');

      var state = container.read(cortadoVfsProvider).requireValue;
      expect(
        (state['/lib/main.dart'] as VfsFile).syncState,
        VfsNodeSyncState.syncing,
      );

      notifier.applyLocalDaemonSyncStatus(
        const CortadoLocalDaemonSyncStatus(
          localPath: '/tmp/workspace',
          message: 'manual merge required',
          state: CortadoLocalDaemonSyncState.conflicted,
          workspaceId: 'ws-123',
          workspacePath: '/lib/main.dart',
        ),
      );

      state = container.read(cortadoVfsProvider).requireValue;
      expect(
        (state['/lib/main.dart'] as VfsFile).syncState,
        VfsNodeSyncState.conflicted,
      );
      expect(
        (state['/lib/main.dart'] as VfsFile).syncMessage,
        'manual merge required',
      );

      notifier.applyLocalDaemonSyncStatus(
        const CortadoLocalDaemonSyncStatus(
          localPath: '/tmp/workspace',
          state: CortadoLocalDaemonSyncState.idle,
          workspaceId: 'ws-123',
          workspacePath: '/lib/main.dart',
        ),
      );

      state = container.read(cortadoVfsProvider).requireValue;
      expect(
        (state['/lib/main.dart'] as VfsFile).syncState,
        VfsNodeSyncState.idle,
      );
      expect((state['/lib/main.dart'] as VfsFile).syncMessage, isNull);
    });
  });
}

class _FakeWorkspaceManager extends WorkspaceManager {
  _FakeWorkspaceManager({
    Map<String, List<List<WorkspaceDirectoryEntry>>> responses =
        const <String, List<List<WorkspaceDirectoryEntry>>>{},
  })  : _responses = responses.map(
          (path, entries) => MapEntry(
            path,
            Queue<List<WorkspaceDirectoryEntry>>.of(entries),
          ),
        ),
        super(baseUrl: 'http://localhost:8080');

  final Map<String, Queue<List<WorkspaceDirectoryEntry>>> _responses;
  final List<String> requests = <String>[];

  @override
  Future<List<WorkspaceDirectoryEntry>> listDirectory(
    String workspaceId, {
    String path = '/',
  }) async {
    requests.add('$workspaceId:$path');
    final queue = _responses[path];
    if (queue == null || queue.isEmpty) {
      return const <WorkspaceDirectoryEntry>[];
    }
    if (queue.length == 1) {
      return queue.first;
    }
    return queue.removeFirst();
  }
}
