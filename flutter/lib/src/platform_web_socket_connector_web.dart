import 'package:web_socket_channel/html.dart';
import 'package:web_socket_channel/web_socket_channel.dart';

WebSocketChannel connectPlatformWebSocket(
  Uri uri, {
  Iterable<String> protocols = const <String>[],
  Map<String, Object>? headers,
}) {
  return HtmlWebSocketChannel.connect(
    uri,
    protocols: protocols,
  );
}
