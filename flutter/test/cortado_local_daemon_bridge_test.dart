import 'dart:async';
import 'dart:convert';
import 'dart:typed_data';

import 'package:cortado/cortado.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:web_socket_channel/web_socket_channel.dart';

void main() {
  group('CortadoLocalDaemonBridge', () {
    test('connects, sends sync commands, and tracks sync status', () async {
      final connector = _FakeWebSocketConnector(
        onConnect: (channel) {
          scheduleMicrotask(() {
            channel.addIncomingText(jsonEncode(<String, Object>{
              'type': 'hello',
            }));
          });
          unawaited(
            channel.outboundMessages.listen((dynamic message) {
              final decoded =
                  jsonDecode(message as String) as Map<String, dynamic>;
              channel.addIncomingText(jsonEncode(<String, Object?>{
                'type': 'sync_status',
                'requestId': decoded['requestId'],
                'localPath': decoded['localPath'],
                'workspaceId': decoded['workspaceId'],
                'workspacePath': '/',
                'state': decoded['type'] == 'stop_sync' ? 'IDLE' : 'SYNCING',
              }));
            }).asFuture<void>(),
          );
        },
      );
      final bridge = CortadoLocalDaemonBridge(
        connector: connector,
      );

      final connected = await bridge.connect();
      expect(connected, isTrue);
      expect(bridge.availability.state,
          CortadoLocalDaemonAvailabilityState.connected);

      final started = await bridge.startSync('/tmp/workspace', 'ws-123');
      expect(started.state, CortadoLocalDaemonSyncState.syncing);
      expect(started.workspaceId, 'ws-123');

      final status = await bridge.getSyncStatus('/tmp/workspace', 'ws-123');
      expect(status.state, CortadoLocalDaemonSyncState.syncing);
      expect(
        bridge.currentSyncStatuses['ws-123::/tmp/workspace']?.state,
        CortadoLocalDaemonSyncState.syncing,
      );

      final stopped = await bridge.stopSync('/tmp/workspace', 'ws-123');
      expect(stopped.state, CortadoLocalDaemonSyncState.idle);

      await bridge.dispose();
    });

    test('maps conflict frames onto tracked workspace paths', () async {
      final connector = _FakeWebSocketConnector(
        onConnect: (channel) {
          scheduleMicrotask(() {
            channel.addIncomingText(jsonEncode(<String, Object>{
              'type': 'hello',
            }));
          });
          unawaited(
            channel.outboundMessages.listen((dynamic message) {
              final decoded =
                  jsonDecode(message as String) as Map<String, dynamic>;
              channel.addIncomingText(jsonEncode(<String, Object?>{
                'type': 'sync_status',
                'requestId': decoded['requestId'],
                'localPath': decoded['localPath'],
                'workspaceId': decoded['workspaceId'],
                'workspacePath': '/',
                'state': 'SYNCING',
              }));
            }).asFuture<void>(),
          );
        },
      );
      final bridge = CortadoLocalDaemonBridge(
        connector: connector,
      );
      await bridge.startSync('/tmp/workspace', 'ws-123');

      final conflictFuture = bridge.conflicts.first;
      connector.channel.addIncomingBinary(
        MuxFrame(
          muxConflictNoticeChannelId,
          muxMessageTypeData,
          Uint8List.fromList(
            utf8.encode(jsonEncode(<String, Object>{
              'lastSyncedClock': 1,
              'localClock': 2,
              'path': '/tmp/workspace/lib/main.dart',
              'reason': 'manual merge required',
              'remoteClock': 3,
            })),
          ),
        ).encode(),
      );

      final conflict = await conflictFuture;
      expect(conflict.workspaceId, 'ws-123');
      expect(conflict.workspacePath, '/lib/main.dart');
      expect(
        bridge.currentSyncStatuses['ws-123::/tmp/workspace']?.state,
        CortadoLocalDaemonSyncState.conflicted,
      );

      await bridge.dispose();
    });

    test('surfaces unavailable state when the daemon cannot be reached',
        () async {
      final connector = _FakeWebSocketConnector(
        ready: Future<void>.delayed(
          Duration.zero,
          () => throw StateError('connection failed'),
        ),
      );
      final bridge = CortadoLocalDaemonBridge(connector: connector);

      await expectLater(bridge.connect(), throwsA(isA<StateError>()));
      expect(
        bridge.availability.state,
        CortadoLocalDaemonAvailabilityState.unavailable,
      );

      await bridge.dispose();
    });
  });
}

class _FakeWebSocketConnector implements WebSocketConnector {
  _FakeWebSocketConnector({
    this.onConnect,
    Future<void>? ready,
  }) : channel = _FakeWebSocketChannel(ready: ready);

  final _FakeWebSocketChannel channel;
  final void Function(_FakeWebSocketChannel channel)? onConnect;

  @override
  WebSocketChannel connect(
    Uri uri, {
    Iterable<String> protocols = const <String>[],
    Map<String, Object>? headers,
  }) {
    channel.protocols = protocols.toList(growable: false);
    onConnect?.call(channel);
    return channel;
  }
}

class _FakeWebSocketChannel implements WebSocketChannel {
  _FakeWebSocketChannel({Future<void>? ready})
      : _ready = ready ?? Future<void>.value(),
        _incoming = StreamController<dynamic>.broadcast(),
        _outgoing = StreamController<dynamic>.broadcast() {
    _sink = _FakeWebSocketSink(_outgoing);
  }

  final StreamController<dynamic> _incoming;
  final StreamController<dynamic> _outgoing;
  final Future<void> _ready;
  late final _FakeWebSocketSink _sink;

  List<String> protocols = const <String>[];

  void addIncomingBinary(List<int> bytes) {
    _incoming.add(Uint8List.fromList(bytes));
  }

  void addIncomingText(String message) {
    _incoming.add(message);
  }

  Stream<dynamic> get outboundMessages => _outgoing.stream;

  @override
  int? get closeCode => null;

  @override
  String? get closeReason => null;

  @override
  String? get protocol => protocols.isEmpty ? null : protocols.first;

  @override
  Future<void> get ready => _ready;

  @override
  WebSocketSink get sink => _sink;

  @override
  Stream<dynamic> get stream => _incoming.stream;

  @override
  dynamic noSuchMethod(Invocation invocation) => super.noSuchMethod(invocation);
}

class _FakeWebSocketSink implements WebSocketSink {
  _FakeWebSocketSink(this._controller);

  final StreamController<dynamic> _controller;

  @override
  void add(dynamic data) {
    _controller.add(data);
  }

  @override
  Future<void> addStream(Stream<dynamic> stream) {
    return _controller.addStream(stream);
  }

  @override
  Future<void> close([int? closeCode, String? closeReason]) async {
    await _controller.close();
  }

  @override
  Future<void> get done => _controller.done;

  @override
  void addError(Object error, [StackTrace? stackTrace]) {
    _controller.addError(error, stackTrace);
  }

  @override
  dynamic noSuchMethod(Invocation invocation) => super.noSuchMethod(invocation);
}
