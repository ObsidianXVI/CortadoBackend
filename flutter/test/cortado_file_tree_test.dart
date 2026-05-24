import 'dart:async';
import 'dart:collection';
import 'dart:typed_data';

import 'package:cortado/cortado.dart';
import 'package:cortado/src/editor/editor_diagnostics.dart';
import 'package:cortado/src/gen/agent/v1/agent.pbenum.dart' as agentpbenum;
import 'package:cortado/src/gen/agent/v1/agent.pb.dart' as agentpb;
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('CortadoFileTree', () {
    testWidgets(
        'loads the root, opens the file watch channel, and expands directories lazily',
        (tester) async {
      final manager = _FakeWorkspaceManager(
        responses: <String, List<List<WorkspaceDirectoryEntry>>>{
          vfsRootPath: <List<WorkspaceDirectoryEntry>>[
            <WorkspaceDirectoryEntry>[
              WorkspaceDirectoryEntry(
                name: 'lib',
                size: 0,
                isDir: true,
                modTime: DateTime.utc(2026, 5, 23, 22),
                permissions: 493,
              ),
              WorkspaceDirectoryEntry(
                name: 'README.md',
                size: 12,
                isDir: false,
                modTime: DateTime.utc(2026, 5, 23, 22),
                permissions: 420,
              ),
            ],
          ],
          '/lib': <List<WorkspaceDirectoryEntry>>[
            <WorkspaceDirectoryEntry>[
              WorkspaceDirectoryEntry(
                name: 'main.dart',
                size: 32,
                isDir: false,
                modTime: DateTime.utc(2026, 5, 23, 22, 5),
                permissions: 420,
              ),
            ],
          ],
        },
      );
      final client = _FakeCortadoClient();
      String? selectedPath;

      await tester.pumpWidget(
        _wrapTree(
          client: client,
          manager: manager,
          child: CortadoFileTree(
            client: client,
            onFileSelected: (path) => selectedPath = path,
          ),
        ),
      );
      await tester.pumpAndSettle();

      expect(find.text('lib'), findsOneWidget);
      expect(find.text('README.md'), findsOneWidget);
      expect(client.sentFrames, hasLength(1));
      expect(client.sentFrames.single.channelId, muxFileSyncChannelId);
      expect(client.sentFrames.single.messageType, muxMessageTypeOpen);

      await tester.tap(find.text('lib'));
      await tester.pumpAndSettle();

      expect(find.text('main.dart'), findsOneWidget);
      expect(manager.requests, const <String>['ws-123:/', 'ws-123:/lib']);

      await tester.tap(find.text('README.md'));
      expect(selectedPath, '/README.md');

      await tester.pumpWidget(const SizedBox.shrink());
      await tester.pump();

      expect(client.sentFrames.last.messageType, muxMessageTypeClose);
    });

    testWidgets('applies file watch events from the mux channel',
        (tester) async {
      final manager = _FakeWorkspaceManager(
        responses: <String, List<List<WorkspaceDirectoryEntry>>>{
          vfsRootPath: <List<WorkspaceDirectoryEntry>>[
            <WorkspaceDirectoryEntry>[
              WorkspaceDirectoryEntry(
                name: 'README.md',
                size: 12,
                isDir: false,
                modTime: DateTime.utc(2026, 5, 23, 22),
                permissions: 420,
              ),
            ],
          ],
        },
      );
      final client = _FakeCortadoClient();
      String? closedMessage;

      await tester.pumpWidget(
        _wrapTree(
          client: client,
          manager: manager,
          child: CortadoFileTree(
            client: client,
            onClosed: (message) => closedMessage = message,
          ),
        ),
      );
      await tester.pumpAndSettle();

      expect(find.text('README.md'), findsOneWidget);

      client.emitFrame(
        MuxFrame(
          muxFileSyncChannelId,
          muxMessageTypeData,
          agentpb.FileEvent(
            path: 'README.md',
            type: agentpbenum.FileEventType.FILE_EVENT_TYPE_DELETED,
          ).writeToBuffer(),
        ),
      );
      await tester.pumpAndSettle();

      expect(find.text('README.md'), findsNothing);

      client.emitFrame(
        MuxFrame(
          muxFileSyncChannelId,
          muxMessageTypeClose,
          Uint8List.fromList('watch closed'.codeUnits),
        ),
      );
      await tester.pump();

      expect(closedMessage, 'watch closed');
    });

    testWidgets('context menu creates folders and files from directory rows',
        (tester) async {
      final manager = _FakeWorkspaceManager(
        responses: <String, List<List<WorkspaceDirectoryEntry>>>{
          vfsRootPath: <List<WorkspaceDirectoryEntry>>[
            <WorkspaceDirectoryEntry>[
              WorkspaceDirectoryEntry(
                name: 'lib',
                size: 0,
                isDir: true,
                modTime: DateTime.utc(2026, 5, 23, 22),
                permissions: 493,
              ),
            ],
          ],
          '/lib': <List<WorkspaceDirectoryEntry>>[
            <WorkspaceDirectoryEntry>[
              WorkspaceDirectoryEntry(
                name: 'main.dart',
                size: 32,
                isDir: false,
                modTime: DateTime.utc(2026, 5, 23, 22, 5),
                permissions: 420,
              ),
            ],
            <WorkspaceDirectoryEntry>[
              WorkspaceDirectoryEntry(
                name: 'docs',
                size: 0,
                isDir: true,
                modTime: DateTime.utc(2026, 5, 23, 22, 10),
                permissions: 493,
              ),
              WorkspaceDirectoryEntry(
                name: 'main.dart',
                size: 32,
                isDir: false,
                modTime: DateTime.utc(2026, 5, 23, 22, 5),
                permissions: 420,
              ),
            ],
            <WorkspaceDirectoryEntry>[
              WorkspaceDirectoryEntry(
                name: 'docs',
                size: 0,
                isDir: true,
                modTime: DateTime.utc(2026, 5, 23, 22, 10),
                permissions: 493,
              ),
              WorkspaceDirectoryEntry(
                name: 'main.dart',
                size: 32,
                isDir: false,
                modTime: DateTime.utc(2026, 5, 23, 22, 5),
                permissions: 420,
              ),
              WorkspaceDirectoryEntry(
                name: 'notes.txt',
                size: 0,
                isDir: false,
                modTime: DateTime.utc(2026, 5, 23, 22, 15),
                permissions: 420,
              ),
            ],
          ],
        },
      );
      final client = _FakeCortadoClient();
      final selectedFiles = <String>[];

      await tester.pumpWidget(
        _wrapTree(
          client: client,
          manager: manager,
          child: CortadoFileTree(
            client: client,
            onFileSelected: selectedFiles.add,
          ),
        ),
      );
      await tester.pumpAndSettle();

      await tester.tap(find.text('lib'));
      await tester.pumpAndSettle();

      await tester.longPress(find.text('lib'));
      await tester.pumpAndSettle();
      await tester.tap(find.text('New Folder'));
      await tester.pumpAndSettle();
      await tester.enterText(find.byType(TextField).last, 'docs');
      await tester.tap(find.text('Create Folder'));
      await tester.pumpAndSettle();

      expect(manager.makeDirRequests, const <String>['ws-123:/lib/docs']);
      expect(find.text('docs'), findsOneWidget);

      await tester.longPress(find.text('lib'));
      await tester.pumpAndSettle();
      await tester.tap(find.text('New File'));
      await tester.pumpAndSettle();
      await tester.enterText(find.byType(TextField).last, 'notes.txt');
      await tester.tap(find.text('Create File'));
      await tester.pumpAndSettle();

      expect(
        manager.writeRequests,
        hasLength(1),
      );
      expect(manager.writeRequests.single.path, 'ws-123:/lib/notes.txt');
      expect(manager.writeRequests.single.content, isEmpty);
      expect(selectedFiles, contains('/lib/notes.txt'));
      expect(find.text('notes.txt'), findsOneWidget);
    });

    testWidgets('rename action shows an inline editor and commits the rename',
        (tester) async {
      final manager = _FakeWorkspaceManager(
        responses: <String, List<List<WorkspaceDirectoryEntry>>>{
          vfsRootPath: <List<WorkspaceDirectoryEntry>>[
            <WorkspaceDirectoryEntry>[
              WorkspaceDirectoryEntry(
                name: 'README.md',
                size: 12,
                isDir: false,
                modTime: DateTime.utc(2026, 5, 23, 22),
                permissions: 420,
              ),
            ],
            <WorkspaceDirectoryEntry>[
              WorkspaceDirectoryEntry(
                name: 'CHANGELOG.md',
                size: 12,
                isDir: false,
                modTime: DateTime.utc(2026, 5, 23, 22, 10),
                permissions: 420,
              ),
            ],
          ],
        },
      );
      final client = _FakeCortadoClient();

      await tester.pumpWidget(
        _wrapTree(
          client: client,
          manager: manager,
          child: CortadoFileTree(client: client),
        ),
      );
      await tester.pumpAndSettle();

      await tester.longPress(find.text('README.md'));
      await tester.pumpAndSettle();
      await tester.tap(find.text('Rename'));
      await tester.pump();

      expect(find.byType(TextField), findsOneWidget);
      await tester.enterText(find.byType(TextField), 'CHANGELOG.md');
      await tester.testTextInput.receiveAction(TextInputAction.done);
      await tester.pumpAndSettle();

      expect(
        manager.renameRequests,
        const <_RenameRequest>[
          _RenameRequest('ws-123:/README.md', 'ws-123:/CHANGELOG.md'),
        ],
      );
      expect(find.text('CHANGELOG.md'), findsOneWidget);
    });

    testWidgets('delete action confirms and removes the node', (tester) async {
      final manager = _FakeWorkspaceManager(
        responses: <String, List<List<WorkspaceDirectoryEntry>>>{
          vfsRootPath: <List<WorkspaceDirectoryEntry>>[
            <WorkspaceDirectoryEntry>[
              WorkspaceDirectoryEntry(
                name: 'README.md',
                size: 12,
                isDir: false,
                modTime: DateTime.utc(2026, 5, 23, 22),
                permissions: 420,
              ),
              WorkspaceDirectoryEntry(
                name: 'lib',
                size: 0,
                isDir: true,
                modTime: DateTime.utc(2026, 5, 23, 22),
                permissions: 493,
              ),
            ],
            <WorkspaceDirectoryEntry>[
              WorkspaceDirectoryEntry(
                name: 'lib',
                size: 0,
                isDir: true,
                modTime: DateTime.utc(2026, 5, 23, 22),
                permissions: 493,
              ),
            ],
          ],
        },
      );
      final client = _FakeCortadoClient();

      await tester.pumpWidget(
        _wrapTree(
          client: client,
          manager: manager,
          child: CortadoFileTree(client: client),
        ),
      );
      await tester.pumpAndSettle();

      await tester.longPress(find.text('README.md'));
      await tester.pumpAndSettle();
      await tester.tap(find.text('Delete'));
      await tester.pumpAndSettle();
      await tester.tap(find.text('Delete').last);
      await tester.pumpAndSettle();

      expect(manager.deleteRequests, const <String>['ws-123:/README.md']);
      expect(find.text('README.md'), findsNothing);
    });

    testWidgets('renders file diagnostic dots from workspace diagnostics state',
        (tester) async {
      final manager = _FakeWorkspaceManager(
        responses: <String, List<List<WorkspaceDirectoryEntry>>>{
          vfsRootPath: <List<WorkspaceDirectoryEntry>>[
            <WorkspaceDirectoryEntry>[
              WorkspaceDirectoryEntry(
                name: 'lib',
                size: 0,
                isDir: true,
                modTime: DateTime.utc(2026, 5, 23, 22),
                permissions: 493,
              ),
            ],
          ],
          '/lib': <List<WorkspaceDirectoryEntry>>[
            <WorkspaceDirectoryEntry>[
              WorkspaceDirectoryEntry(
                name: 'main.dart',
                size: 32,
                isDir: false,
                modTime: DateTime.utc(2026, 5, 23, 22, 5),
                permissions: 420,
              ),
            ],
          ],
        },
      );
      final client = _FakeCortadoClient();

      await tester.pumpWidget(
        _wrapTree(
          client: client,
          manager: manager,
          child: CortadoFileTree(client: client),
        ),
      );
      await tester.pumpAndSettle();

      await tester.tap(find.text('lib'));
      await tester.pumpAndSettle();

      final container = ProviderScope.containerOf(
        tester.element(find.byType(CortadoFileTree)),
      );
      container.read(cortadoWorkspaceDiagnosticStatusProvider.notifier).state =
          <String, CortadoFileDiagnosticStatus>{
        '/lib/main.dart': CortadoFileDiagnosticStatus.warning,
      };
      await tester.pump();

      expect(
        find.byKey(const ValueKey('file-tree-diagnostic-dot:/lib/main.dart')),
        findsOneWidget,
      );

      container.read(cortadoWorkspaceDiagnosticStatusProvider.notifier).state =
          const <String, CortadoFileDiagnosticStatus>{};
      await tester.pump();

      expect(
        find.byKey(const ValueKey('file-tree-diagnostic-dot:/lib/main.dart')),
        findsNothing,
      );
    });

    testWidgets('renders sync status indicators from VFS state',
        (tester) async {
      final manager = _FakeWorkspaceManager(
        responses: <String, List<List<WorkspaceDirectoryEntry>>>{
          vfsRootPath: <List<WorkspaceDirectoryEntry>>[
            <WorkspaceDirectoryEntry>[
              WorkspaceDirectoryEntry(
                name: 'README.md',
                size: 12,
                isDir: false,
                modTime: DateTime.utc(2026, 5, 24, 10),
                permissions: 420,
              ),
              WorkspaceDirectoryEntry(
                name: 'lib',
                size: 0,
                isDir: true,
                modTime: DateTime.utc(2026, 5, 24, 10),
                permissions: 493,
              ),
            ],
          ],
          '/lib': <List<WorkspaceDirectoryEntry>>[
            <WorkspaceDirectoryEntry>[
              WorkspaceDirectoryEntry(
                name: 'main.dart',
                size: 32,
                isDir: false,
                modTime: DateTime.utc(2026, 5, 24, 10, 5),
                permissions: 420,
              ),
            ],
          ],
        },
      );
      final client = _FakeCortadoClient();

      await tester.pumpWidget(
        _wrapTree(
          client: client,
          manager: manager,
          child: CortadoFileTree(client: client),
        ),
      );
      await tester.pumpAndSettle();

      final container = ProviderScope.containerOf(
        tester.element(find.byType(CortadoFileTree)),
      );
      final notifier = container.read(cortadoVfsProvider.notifier);
      notifier.applyLocalDaemonSyncStatus(
        const CortadoLocalDaemonSyncStatus(
          localPath: '/tmp/workspace',
          state: CortadoLocalDaemonSyncState.syncing,
          workspaceId: 'ws-123',
          workspacePath: '/README.md',
        ),
      );
      await tester.pump();

      expect(
        find.byKey(const ValueKey('file-tree-sync-spinner:/README.md')),
        findsOneWidget,
      );

      await tester.tap(find.text('lib'));
      await tester.pump();
      await tester.pump();
      notifier.applyLocalDaemonSyncStatus(
        const CortadoLocalDaemonSyncStatus(
          localPath: '/tmp/workspace',
          message: 'manual merge required',
          state: CortadoLocalDaemonSyncState.conflicted,
          workspaceId: 'ws-123',
          workspacePath: '/lib/main.dart',
        ),
      );
      await tester.pump();

      expect(
        find.byKey(const ValueKey('file-tree-sync-conflict:/lib/main.dart')),
        findsOneWidget,
      );
    });
  });
}

Widget _wrapTree({
  required CortadoClient client,
  required Widget child,
  required WorkspaceManager manager,
}) {
  return MaterialApp(
    home: Scaffold(
      body: CortadoWorkspaceProvider(
        workspaceId: 'ws-123',
        manager: manager,
        child: child,
      ),
    ),
  );
}

class _FakeCortadoClient extends CortadoClient {
  _FakeCortadoClient() : super(baseUrl: 'ws://localhost:8080');

  final List<MuxFrame> sentFrames = <MuxFrame>[];
  final Map<int, StreamController<MuxFrame>> _controllers =
      <int, StreamController<MuxFrame>>{};

  @override
  Stream<MuxFrame> framesForChannel(int channelId) {
    return _controller(channelId).stream;
  }

  @override
  void sendFrame(int channelId, int messageType, Uint8List payload) {
    sentFrames.add(MuxFrame(channelId, messageType, payload));
  }

  void emitFrame(MuxFrame frame) {
    _controller(frame.channelId).add(frame);
  }

  StreamController<MuxFrame> _controller(int channelId) {
    return _controllers.putIfAbsent(
      channelId,
      () => StreamController<MuxFrame>.broadcast(),
    );
  }
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
  final List<String> makeDirRequests = <String>[];
  final List<_RenameRequest> renameRequests = <_RenameRequest>[];
  final List<String> deleteRequests = <String>[];
  final List<_WriteRequest> writeRequests = <_WriteRequest>[];

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

  @override
  Future<void> makeDir(
    String workspaceId, {
    required String path,
  }) async {
    makeDirRequests.add('$workspaceId:$path');
  }

  @override
  Future<void> renamePath(
    String workspaceId, {
    required String oldPath,
    required String newPath,
  }) async {
    renameRequests
        .add(_RenameRequest('$workspaceId:$oldPath', '$workspaceId:$newPath'));
  }

  @override
  Future<void> deletePath(
    String workspaceId, {
    required String path,
  }) async {
    deleteRequests.add('$workspaceId:$path');
  }

  @override
  Future<WorkspaceWriteFileResult> writeFile(
    String workspaceId, {
    required String path,
    List<int> content = const <int>[],
    bool createMissingDirs = true,
  }) async {
    writeRequests.add(
      _WriteRequest(
        content: Uint8List.fromList(content),
        path: '$workspaceId:$path',
      ),
    );
    return WorkspaceWriteFileResult(
      bytesWritten: 0,
      checksum: Uint8List(0),
    );
  }
}

class _RenameRequest {
  const _RenameRequest(this.oldPath, this.newPath);

  final String oldPath;
  final String newPath;

  @override
  bool operator ==(Object other) {
    return other is _RenameRequest &&
        other.oldPath == oldPath &&
        other.newPath == newPath;
  }

  @override
  int get hashCode => Object.hash(oldPath, newPath);
}

class _WriteRequest {
  const _WriteRequest({
    required this.content,
    required this.path,
  });

  final Uint8List content;
  final String path;
}
