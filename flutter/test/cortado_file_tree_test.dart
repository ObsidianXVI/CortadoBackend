import 'dart:async';
import 'dart:collection';
import 'dart:typed_data';

import 'package:cortado/cortado.dart';
import 'package:cortado/src/gen/agent/v1/agent.pbenum.dart' as agentpbenum;
import 'package:cortado/src/gen/agent/v1/agent.pb.dart' as agentpb;
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('CortadoFileTree', () {
    testWidgets('loads the root, opens the file watch channel, and expands directories lazily',
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

    testWidgets('applies file watch events from the mux channel', (tester) async {
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
