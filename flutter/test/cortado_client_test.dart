import 'dart:async';
import 'dart:convert';
import 'dart:typed_data';

import 'package:cortado/cortado.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:web_socket_channel/web_socket_channel.dart';

void main() {
  group('CortadoClient', () {
    test('uses a token query parameter for browser websocket connections',
        () async {
      final connector = FakeWebSocketConnector();
      final timers = <FakeTimer>[];
      final authSession = CortadoAuthSession(
        baseUrl: 'https://api.example.dev',
        now: () => DateTime.utc(2026, 5, 23, 14),
        timerFactory: (duration, callback) {
          final timer = FakeTimer(duration, callback);
          timers.add(timer);
          return timer;
        },
      )..setTokens(
          accessToken: _jwtExpiringAt(DateTime.utc(2026, 5, 23, 15)),
          refreshToken: 'refresh-token',
        );
      final client = CortadoClient(
        baseUrl: 'https://api.example.dev/base',
        authSession: authSession,
        connector: connector,
        useBrowserWebSocket: true,
      );

      await client.connect('ws-123');

      expect(
        connector.lastUri,
        Uri.parse(
            'wss://api.example.dev/base/v1/workspaces/ws-123/connect?token=${_jwtExpiringAt(DateTime.utc(2026, 5, 23, 15))}'),
      );
      expect(connector.lastHeaders, isNull);

      await client.dispose();
      await authSession.dispose();
      expect(timers, hasLength(1));
    });

    test('uses authorization headers for non-browser websocket connections',
        () async {
      final connector = FakeWebSocketConnector();
      final accessToken = _jwtExpiringAt(DateTime.utc(2026, 5, 23, 15));
      final timers = <FakeTimer>[];
      final authSession = CortadoAuthSession(
        baseUrl: 'http://localhost:8080',
        now: () => DateTime.utc(2026, 5, 23, 14),
        timerFactory: (duration, callback) {
          final timer = FakeTimer(duration, callback);
          timers.add(timer);
          return timer;
        },
      )..setTokens(
          accessToken: accessToken,
          refreshToken: 'refresh-token',
        );
      final client = CortadoClient(
        baseUrl: 'http://localhost:8080',
        authSession: authSession,
        connector: connector,
        useBrowserWebSocket: false,
      );

      await client.connect('ws-123');

      expect(
        connector.lastUri,
        Uri.parse('ws://localhost:8080/v1/workspaces/ws-123/connect'),
      );
      expect(
        connector.lastHeaders,
        <String, Object>{'Authorization': 'Bearer $accessToken'},
      );

      await client.dispose();
      await authSession.dispose();
      expect(timers, hasLength(1));
    });

    test('falls back to dev_token query auth without a session', () async {
      final connector = FakeWebSocketConnector();
      final client = CortadoClient(
        baseUrl: 'https://api.example.dev/base',
        connector: connector,
        useBrowserWebSocket: true,
      );

      await client.connect('ws-123');

      expect(
        connector.lastUri,
        Uri.parse(
            'wss://api.example.dev/base/v1/workspaces/ws-123/connect?dev_token=dev-bypass'),
      );
      expect(connector.lastHeaders, isNull);

      await client.dispose();
    });

    test('falls back to dev headers without a session', () async {
      final connector = FakeWebSocketConnector();
      final client = CortadoClient(
        baseUrl: 'http://localhost:8080',
        connector: connector,
        useBrowserWebSocket: false,
      );

      await client.connect('ws-123');

      expect(
        connector.lastHeaders,
        <String, Object>{'X-Cortado-Dev-Token': 'dev-bypass'},
      );

      await client.dispose();
    });

    test('decodes inbound mux frames and forwards outbound frames', () async {
      final connector = FakeWebSocketConnector();
      final client = CortadoClient(
        baseUrl: 'http://localhost:8080',
        connector: connector,
        useBrowserWebSocket: false,
      );

      await client.connect('ws-123');

      final frameFuture = client.framesForChannel(muxTerminalChannelId).first;
      connector.channel.addIncoming(
        MuxFrame(
          muxTerminalChannelId,
          muxMessageTypeData,
          Uint8List.fromList(<int>[0x41, 0x42]),
        ).encode(),
      );

      final decoded = await frameFuture;
      expect(decoded.messageType, muxMessageTypeData);
      expect(decoded.payload, orderedEquals(<int>[0x41, 0x42]));

      final outboundFuture = connector.channel.outboundFrames.first;
      client.sendFrame(
        muxTerminalChannelId,
        muxMessageTypeOpen,
        Uint8List.fromList(<int>[0x43]),
      );

      final outbound = await outboundFuture;
      expect(
        outbound,
        orderedEquals(
          MuxFrame(
            muxTerminalChannelId,
            muxMessageTypeOpen,
            Uint8List.fromList(<int>[0x43]),
          ).encode(),
        ),
      );

      await client.dispose();
    });

    test('rethrows ready failures and publishes them on the error stream',
        () async {
      final connector = FakeWebSocketConnector(
        ready: Future<void>.delayed(
          Duration.zero,
          () => throw StateError('connection failed'),
        ),
      );
      final client = CortadoClient(
        baseUrl: 'http://localhost:8080',
        connector: connector,
        useBrowserWebSocket: false,
      );

      final errorFuture = client.errors.first;
      final connectFuture = client.connect('ws-123');

      await expectLater(
        connectFuture,
        throwsA(isA<StateError>()),
      );
      expect(await errorFuture, isA<StateError>());

      await client.dispose();
    });

    test('publishes a close error when the websocket closes', () async {
      final connector = FakeWebSocketConnector();
      final client = CortadoClient(
        baseUrl: 'http://localhost:8080',
        connector: connector,
        useBrowserWebSocket: false,
      );

      await client.connect('ws-123');

      final errorFuture = client.errors.first;
      await connector.channel.closeIncoming();

      expect(await errorFuture, isA<StateError>());

      await client.dispose();
    });
  });
}

String _jwtExpiringAt(DateTime timestamp) {
  final header = base64Url.encode(utf8.encode(jsonEncode(<String, String>{
    'alg': 'RS256',
    'typ': 'JWT',
  })));
  final payload = base64Url.encode(utf8.encode(jsonEncode(<String, Object>{
    'exp': timestamp.millisecondsSinceEpoch ~/ 1000,
  })));
  return '$header.$payload.signature';
}

class FakeWebSocketConnector implements WebSocketConnector {
  FakeWebSocketConnector({Future<void>? ready})
      : channel = FakeWebSocketChannel(ready: ready);

  final FakeWebSocketChannel channel;
  Map<String, Object>? lastHeaders;
  Uri? lastUri;

  @override
  WebSocketChannel connect(
    Uri uri, {
    Iterable<String> protocols = const <String>[],
    Map<String, Object>? headers,
  }) {
    lastUri = uri;
    lastHeaders = headers;
    channel.protocols = protocols.toList(growable: false);
    return channel;
  }
}

class FakeTimer implements Timer {
  FakeTimer(this.duration, this._callback);

  final Duration duration;
  final void Function() _callback;
  bool _isActive = true;

  void fire() {
    if (!_isActive) {
      return;
    }
    _callback();
  }

  @override
  void cancel() {
    _isActive = false;
  }

  @override
  bool get isActive => _isActive;

  @override
  int get tick => 0;
}

class FakeWebSocketChannel implements WebSocketChannel {
  FakeWebSocketChannel({Future<void>? ready})
      : _ready = ready ?? Future<void>.value(),
        _incoming = StreamController<dynamic>.broadcast(),
        _outgoing = StreamController<dynamic>.broadcast() {
    _sink = FakeWebSocketSink(_outgoing);
  }

  final StreamController<dynamic> _incoming;
  final StreamController<dynamic> _outgoing;
  final Future<void> _ready;
  int? _closeCode;
  String? _closeReason;
  late final FakeWebSocketSink _sink;

  List<String> protocols = const <String>[];

  Stream<List<int>> get outboundFrames => _outgoing.stream
      .cast<Uint8List>()
      .map((value) => value.toList(growable: false));

  void addIncoming(List<int> bytes) {
    _incoming.add(Uint8List.fromList(bytes));
  }

  Future<void> closeIncoming() async {
    await _incoming.close();
  }

  @override
  int? get closeCode => _closeCode;

  @override
  String? get closeReason => _closeReason;

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

class FakeWebSocketSink implements WebSocketSink {
  FakeWebSocketSink(this._controller);

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
