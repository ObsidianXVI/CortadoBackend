import 'dart:async';
import 'dart:convert';
import 'dart:typed_data';

import 'package:cortado/cortado.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:web_socket_channel/web_socket_channel.dart';

void main() {
  test(
      'short-expiry refresh smoke rotates the token before expiry and both HTTP and WebSocket paths use the refreshed JWT',
      () async {
    final initialToken = _jwtExpiringAt(
      DateTime.now().toUtc().add(const Duration(minutes: 5, milliseconds: 200)),
      marker: 'initial',
    );
    final refreshedToken = _jwtExpiringAt(
      DateTime.now().toUtc().add(const Duration(hours: 1)),
      marker: 'refreshed',
    );

    final refreshClient = RecordingClient((request, body) async {
      expect(request.method, 'POST');
      expect(request.url,
          Uri.parse('https://api.example.dev/v1/sessions/refresh'));
      expect(
        jsonDecode(utf8.decode(body)),
        <String, Object?>{'refresh_token': 'refresh-token'},
      );
      return _jsonResponse(200, <String, Object?>{
        'access_token': refreshedToken,
      });
    });
    final authSession = CortadoAuthSession(
      baseUrl: 'https://api.example.dev',
      httpClient: refreshClient,
    )..setTokens(
        accessToken: initialToken,
        refreshToken: 'refresh-token',
      );

    await Future<void>.delayed(const Duration(milliseconds: 350));

    expect(authSession.accessToken, refreshedToken);

    final requests = <RecordedRequest>[];
    final workspaceHttpClient = RecordingClient((request, body) async {
      requests.add(RecordedRequest(request, body));
      return _jsonResponse(204, const <String, Object?>{});
    });
    final manager = WorkspaceManager(
      baseUrl: 'https://api.example.dev',
      httpClient: workspaceHttpClient,
      authSession: authSession,
      useBrowserAuth: true,
    );

    await manager.start('ws-123');

    expect(requests, hasLength(1));
    expect(
      requests.single.request.headers['Authorization'],
      'Bearer $refreshedToken',
    );

    final connector = FakeWebSocketConnector();
    final client = CortadoClient(
      baseUrl: 'https://api.example.dev',
      authSession: authSession,
      connector: connector,
      useBrowserWebSocket: true,
    );

    await client.connect('ws-123');

    expect(
      connector.lastUri,
      Uri.parse(
        'wss://api.example.dev/v1/workspaces/ws-123/connect?token=$refreshedToken',
      ),
    );

    await client.dispose();
    await manager.dispose();
    await authSession.dispose();
  });
}

http.StreamedResponse _jsonResponse(int status, Map<String, Object?> body) {
  final bytes = utf8.encode(jsonEncode(body));
  return http.StreamedResponse(
    Stream<List<int>>.fromIterable(<List<int>>[Uint8List.fromList(bytes)]),
    status,
    headers: const <String, String>{'Content-Type': 'application/json'},
  );
}

String _jwtExpiringAt(DateTime timestamp, {required String marker}) {
  final header = base64Url.encode(utf8.encode(jsonEncode(<String, String>{
    'alg': 'RS256',
    'typ': 'JWT',
  })));
  final payload = base64Url.encode(utf8.encode(jsonEncode(<String, Object>{
    'exp': timestamp.millisecondsSinceEpoch ~/ 1000,
    'marker': marker,
  })));
  return '$header.$payload.signature';
}

class RecordingClient extends http.BaseClient {
  RecordingClient(this._handler);

  final Future<http.StreamedResponse> Function(
    http.BaseRequest request,
    List<int> bodyBytes,
  ) _handler;

  @override
  Future<http.StreamedResponse> send(http.BaseRequest request) async {
    final bodyBytes = await http.ByteStream(request.finalize()).toBytes();
    return _handler(request, bodyBytes);
  }
}

class RecordedRequest {
  RecordedRequest(this.request, this.bodyBytes);

  final http.BaseRequest request;
  final List<int> bodyBytes;
}

class FakeWebSocketConnector implements WebSocketConnector {
  final FakeWebSocketChannel channel = FakeWebSocketChannel();
  Uri? lastUri;

  @override
  WebSocketChannel connect(
    Uri uri, {
    Iterable<String> protocols = const <String>[],
    Map<String, Object>? headers,
  }) {
    lastUri = uri;
    return channel;
  }
}

class FakeWebSocketChannel implements WebSocketChannel {
  final StreamController<dynamic> _incoming =
      StreamController<dynamic>.broadcast();
  final StreamController<dynamic> _outgoing =
      StreamController<dynamic>.broadcast();
  late final FakeWebSocketSink _sink = FakeWebSocketSink(_outgoing);

  @override
  int? get closeCode => null;

  @override
  String? get closeReason => null;

  @override
  String? get protocol => 'cortado-v1';

  @override
  Future<void> get ready => Future<void>.value();

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
