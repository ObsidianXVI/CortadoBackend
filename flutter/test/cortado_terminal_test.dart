import 'dart:async';
import 'dart:convert';
import 'dart:typed_data';

import 'package:cortado/cortado.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter_test/flutter_test.dart';

const String workspaceResumedBanner =
    '\r\n\x1b[33m--- Workspace resumed ---\x1b[0m\r\n';

void main() {
  testWidgets('renders a non-web fallback when HtmlElementView is unavailable',
      (WidgetTester tester) async {
    await tester.pumpWidget(
      Directionality(
        textDirection: TextDirection.ltr,
        child: CortadoTerminal(
          client: CortadoClient(baseUrl: 'http://localhost:8080'),
        ),
      ),
    );

    expect(
        find.text(
            'CortadoTerminal is currently supported on Flutter Web only.'),
        findsOneWidget);
  });

  testWidgets('reconnects after websocket loss and shows resumed banner',
      (WidgetTester tester) async {
    final client = TestCortadoClient();
    final manager = TestWorkspaceManager(
      statuses: <WorkspaceLifecycleState>[
        WorkspaceLifecycleState.starting,
        WorkspaceLifecycleState.running,
      ],
      startDelay: const Duration(milliseconds: 20),
    );
    final platform = TestTerminalPlatform();

    await tester.pumpWidget(
      Directionality(
        textDirection: TextDirection.ltr,
        child: CortadoTerminal(
          client: client,
          workspaceManager: manager,
          workspaceId: 'ws-123',
          platform: platform,
          reconnectPolicy: const CortadoTerminalReconnectPolicy(
            statusInitialBackoff: Duration(milliseconds: 1),
            statusMaxBackoff: Duration(milliseconds: 4),
            openRetryDelay: Duration(milliseconds: 2),
          ),
        ),
      ),
    );

    await tester.pump(const Duration(milliseconds: 3));
    expect(client.openFrameCount, 1);

    client.addSocketError(StateError('WebSocket connection closed.'));
    await tester.pump(const Duration(milliseconds: 1));
    expect(find.text('Reconnecting...'), findsOneWidget);

    await tester.pump(const Duration(milliseconds: 30));
    await tester.pump();

    expect(manager.startCalls, <String>['ws-123']);
    expect(manager.watchCalls, 2);
    expect(client.connectCalls, <String>['ws-123']);
    expect(client.openFrameCount, 2);
    expect(platform.writes, contains(workspaceResumedBanner));
    expect(find.text('Reconnecting...'), findsNothing);
  });

  testWidgets('retries terminal open after a retryable close during resume',
      (WidgetTester tester) async {
    final client = TestCortadoClient();
    final manager = TestWorkspaceManager(
      statuses: <WorkspaceLifecycleState>[WorkspaceLifecycleState.running],
    );
    final platform = TestTerminalPlatform();
    final errors = <String>[];

    await tester.pumpWidget(
      Directionality(
        textDirection: TextDirection.ltr,
        child: CortadoTerminal(
          client: client,
          workspaceManager: manager,
          workspaceId: 'ws-456',
          platform: platform,
          reconnectPolicy: const CortadoTerminalReconnectPolicy(
            statusInitialBackoff: Duration(milliseconds: 1),
            statusMaxBackoff: Duration(milliseconds: 1),
            openRetryDelay: Duration(milliseconds: 10),
            maxOpenAttempts: 3,
          ),
          onError: errors.add,
        ),
      ),
    );

    await tester.pump(const Duration(milliseconds: 12));
    expect(client.openFrameCount, 1);

    client.addSocketError(StateError('WebSocket connection closed.'));
    await tester.pump(const Duration(milliseconds: 1));

    await tester.pump(const Duration(milliseconds: 1));
    expect(client.openFrameCount, 2);

    client.addFrame(
      MuxFrame(
        muxTerminalChannelId,
        muxMessageTypeClose,
        Uint8List.fromList(
          utf8.encode(
            'open terminal: rpc error: code = Unavailable desc = workspace starting',
          ),
        ),
      ),
    );
    await tester.pump();

    await tester.pump(const Duration(milliseconds: 24));
    await tester.pump();

    expect(client.openFrameCount, 3);
    expect(platform.writes, contains(workspaceResumedBanner));
    expect(errors, isEmpty);
    expect(find.text('Reconnecting...'), findsNothing);
  });
}

class TestCortadoClient extends CortadoClient {
  TestCortadoClient()
      : _frames = StreamController<MuxFrame>.broadcast(),
        _errors = StreamController<Object>.broadcast(),
        super(baseUrl: 'http://localhost:8080', useBrowserWebSocket: false);

  final StreamController<MuxFrame> _frames;
  final StreamController<Object> _errors;
  final List<String> connectCalls = <String>[];
  final List<MuxFrame> sentFrames = <MuxFrame>[];

  int get openFrameCount => sentFrames
      .where((frame) => frame.messageType == muxMessageTypeOpen)
      .length;

  @override
  Stream<Object> get errors => _errors.stream;

  @override
  Stream<MuxFrame> get frames => _frames.stream;

  @override
  Future<void> connect(String workspaceId) async {
    connectCalls.add(workspaceId);
  }

  @override
  Stream<MuxFrame> framesForChannel(int channelId) {
    return _frames.stream.where((frame) => frame.channelId == channelId);
  }

  @override
  void sendFrame(int channelId, int messageType, Uint8List payload) {
    sentFrames.add(MuxFrame(channelId, messageType, payload));
  }

  void addFrame(MuxFrame frame) {
    _frames.add(frame);
  }

  void addSocketError(Object error) {
    _errors.add(error);
  }

  @override
  Future<void> dispose() async {
    await _frames.close();
    await _errors.close();
  }
}

class TestWorkspaceManager extends WorkspaceManager {
  TestWorkspaceManager({
    required List<WorkspaceLifecycleState> statuses,
    this.startDelay = Duration.zero,
  })  : _remainingStatuses = List<WorkspaceLifecycleState>.from(statuses),
        super(baseUrl: 'http://localhost:8080', useBrowserAuth: false);

  final List<WorkspaceLifecycleState> _remainingStatuses;
  final Duration startDelay;
  final List<String> startCalls = <String>[];
  int watchCalls = 0;

  @override
  Future<void> start(String id) async {
    startCalls.add(id);
    if (startDelay > Duration.zero) {
      await Future<void>.delayed(startDelay);
    }
  }

  @override
  Stream<WorkspaceStatus> watchStatus(String id) {
    watchCalls += 1;
    final state = _remainingStatuses.isNotEmpty
        ? _remainingStatuses.removeAt(0)
        : WorkspaceLifecycleState.running;
    return Stream<WorkspaceStatus>.value(
      WorkspaceStatus(
        workspaceId: id,
        status: state,
        updatedAt: DateTime.utc(2026, 1, 1),
      ),
    );
  }
}

class TestTerminalPlatform extends CortadoTerminalPlatformAdapter {
  final List<String> writes = <String>[];

  @override
  bool get supportsPlatformView => true;

  @override
  Widget buildView(String viewType) {
    return const SizedBox.expand();
  }

  @override
  void disposeView(String terminalId) {}

  @override
  void registerViewFactory({
    required String viewType,
    required String terminalId,
    required dynamic onData,
    required dynamic onResize,
  }) {}

  @override
  void writeOutput(String terminalId, String data) {
    writes.add(data);
  }
}
