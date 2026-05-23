import 'dart:async';

import 'package:flutter/foundation.dart';
import 'package:web_socket_channel/web_socket_channel.dart';

import 'cortado_auth_session.dart';
import 'mux_frame.dart';
import 'platform_web_socket_connector.dart';

const String cortadoProtocol = 'cortado-v1';
const String defaultDevToken = 'dev-bypass';

abstract class WebSocketConnector {
  const WebSocketConnector();

  WebSocketChannel connect(
    Uri uri, {
    Iterable<String> protocols = const <String>[],
    Map<String, Object>? headers,
  });
}

class DefaultWebSocketConnector implements WebSocketConnector {
  const DefaultWebSocketConnector();

  @override
  WebSocketChannel connect(
    Uri uri, {
    Iterable<String> protocols = const <String>[],
    Map<String, Object>? headers,
  }) {
    return connectPlatformWebSocket(
      uri,
      protocols: protocols,
      headers: headers,
    );
  }
}

class CortadoClient {
  CortadoClient({
    required this.baseUrl,
    this.authSession,
    String devToken = defaultDevToken,
    WebSocketConnector connector = const DefaultWebSocketConnector(),
    bool? useBrowserWebSocket,
  })  : _connector = connector,
        _devToken = devToken,
        _useBrowserWebSocket = useBrowserWebSocket ?? kIsWeb;

  final String baseUrl;
  final CortadoAuthSession? authSession;
  final WebSocketConnector _connector;
  final String _devToken;
  final bool _useBrowserWebSocket;
  final StreamController<MuxFrame> _frames =
      StreamController<MuxFrame>.broadcast();
  final StreamController<Object> _errors = StreamController<Object>.broadcast();

  WebSocketChannel? _ws;
  StreamSubscription<dynamic>? _wsSubscription;
  bool _disposed = false;

  Stream<Object> get errors => _errors.stream;

  Stream<MuxFrame> get frames => _frames.stream;

  Future<void> connect(String workspaceId) async {
    _ensureNotDisposed();

    if (workspaceId.trim().isEmpty) {
      throw ArgumentError.value(
          workspaceId, 'workspaceId', 'Must not be empty.');
    }

    await disconnect();
    final accessToken = await authSession?.accessTokenForWebSocket();

    final channel = _connector.connect(
      _connectUri(workspaceId, accessToken: accessToken),
      protocols: const <String>[cortadoProtocol],
      headers: _headers(accessToken: accessToken),
    );

    try {
      await channel.ready;
    } catch (error, stackTrace) {
      _errors.add(error);
      Error.throwWithStackTrace(error, stackTrace);
    }

    _ws = channel;
    _wsSubscription = channel.stream.listen(
      _onFrame,
      onDone: _onDone,
      onError: _onError,
    );
  }

  Future<void> disconnect() async {
    await _wsSubscription?.cancel();
    _wsSubscription = null;

    await _ws?.sink.close();
    _ws = null;
  }

  Future<void> dispose() async {
    if (_disposed) {
      return;
    }

    _disposed = true;
    await disconnect();
    await _frames.close();
    await _errors.close();
  }

  Stream<MuxFrame> framesForChannel(int channelId) =>
      _frames.stream.where((frame) => frame.channelId == channelId);

  void sendFrame(int channelId, int messageType, Uint8List payload) {
    _ensureNotDisposed();

    final ws = _ws;
    if (ws == null) {
      throw StateError('CortadoClient is not connected.');
    }

    ws.sink.add(MuxFrame(channelId, messageType, payload).encode());
  }

  Uri _connectUri(String workspaceId, {String? accessToken}) {
    final uri = Uri.parse(baseUrl);
    final scheme = switch (uri.scheme) {
      'http' => 'ws',
      'https' => 'wss',
      'ws' || 'wss' => uri.scheme,
      _ => uri.scheme,
    };

    final queryParameters = Map<String, String>.from(uri.queryParameters);
    if (_useBrowserWebSocket) {
      if (accessToken != null) {
        queryParameters['token'] = accessToken;
      } else {
        queryParameters['dev_token'] = _devToken;
      }
    }

    return uri.replace(
      scheme: scheme,
      pathSegments: <String>[
        ...uri.pathSegments.where((segment) => segment.isNotEmpty),
        'v1',
        'workspaces',
        workspaceId,
        'connect',
      ],
      queryParameters: queryParameters.isEmpty ? null : queryParameters,
    );
  }

  Map<String, Object>? _headers({String? accessToken}) {
    if (_useBrowserWebSocket) {
      return null;
    }

    if (accessToken != null) {
      return <String, Object>{
        'Authorization': 'Bearer $accessToken',
      };
    }

    return <String, Object>{
      'X-Cortado-Dev-Token': _devToken,
    };
  }

  void _ensureNotDisposed() {
    if (_disposed) {
      throw StateError('CortadoClient has been disposed.');
    }
  }

  void _onDone() {
    _errors.add(StateError('WebSocket connection closed.'));
  }

  void _onError(Object error) {
    _errors.add(error);
  }

  void _onFrame(dynamic raw) {
    final bytes = switch (raw) {
      Uint8List data => data,
      List<int> data => Uint8List.fromList(data),
      _ => throw FormatException(
          'Unsupported WebSocket frame payload type: ${raw.runtimeType}'),
    };

    _frames.add(MuxFrame.decode(bytes));
  }
}
