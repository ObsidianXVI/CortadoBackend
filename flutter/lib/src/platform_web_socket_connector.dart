import 'package:web_socket_channel/web_socket_channel.dart';

import 'platform_web_socket_connector_stub.dart'
    if (dart.library.io) 'platform_web_socket_connector_io.dart'
    if (dart.library.js_interop) 'platform_web_socket_connector_web.dart'
    as platform;

WebSocketChannel connectPlatformWebSocket(
  Uri uri, {
  Iterable<String> protocols = const <String>[],
  Map<String, Object>? headers,
}) {
  return platform.connectPlatformWebSocket(
    uri,
    protocols: protocols,
    headers: headers,
  );
}
