import 'dart:async';
import 'dart:convert';
import 'dart:typed_data';

import 'package:cortado/cortado.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;

void main() {
  group('CortadoAIService', () {
    test('streams tokens from the completion SSE endpoint', () async {
      final requests = <RecordedRequest>[];
      final client = RecordingClient((request, body) async {
        requests.add(RecordedRequest(request, body));
        return _streamedResponse(
          200,
          'data: {"token":"hel"}\n\n'
          'data: {"token":"lo"}\n\n',
          headers: const <String, String>{
            'Content-Type': 'text/event-stream',
          },
        );
      });
      final service = CortadoAIService(
        baseUrl: 'http://localhost:8080/api?foo=bar',
        httpClient: client,
      );

      final tokens = await service
          .streamCompletion(const CortadoCompletionContext(
            workspaceId: 'ws-123',
            path: 'lib/main.dart',
            prefix: 'void ',
            suffix: '{}',
          ))
          .toList();

      expect(tokens, <String>['hel', 'lo']);
      final request = requests.single.request as http.Request;
      expect(
        request.url,
        Uri.parse(
          'http://localhost:8080/api/v1/workspaces/ws-123/ai/complete?foo=bar',
        ),
      );
      expect(request.headers['X-Cortado-Dev-Token'], 'dev-bypass');
      expect(request.headers['Content-Type'], 'application/json');

      final payload =
          jsonDecode(utf8.decode(request.bodyBytes)) as Map<String, dynamic>;
      expect(payload['path'], 'lib/main.dart');
      expect(payload['prefix'], 'void ');
      expect(payload['suffix'], '{}');
    });

    test('uses bearer auth when a session is present', () async {
      final requests = <RecordedRequest>[];
      final client = RecordingClient((request, body) async {
        requests.add(RecordedRequest(request, body));
        return _streamedResponse(
          200,
          'data: {"token":"done"}\n\n',
          headers: const <String, String>{
            'Content-Type': 'text/event-stream',
          },
        );
      });
      final accessToken = _jwtExpiringAt(DateTime.utc(2026, 5, 24, 2));
      final authSession = CortadoAuthSession(
        baseUrl: 'http://localhost:8080',
        now: () => DateTime.utc(2026, 5, 24, 1),
      )..setTokens(
          accessToken: accessToken,
          refreshToken: 'refresh-token',
        );
      final service = CortadoAIService(
        baseUrl: 'http://localhost:8080',
        authSession: authSession,
        httpClient: client,
      );

      final tokens = await service
          .streamCompletion(const CortadoCompletionContext(
            workspaceId: 'ws-123',
            prefix: 'void ',
          ))
          .toList();

      expect(tokens, <String>['done']);
      expect(
        requests.single.request.headers['Authorization'],
        'Bearer $accessToken',
      );
      expect(
        requests.single.request.headers.containsKey('X-Cortado-Dev-Token'),
        isFalse,
      );
      await authSession.dispose();
    });

    test('surfaces SSE error events as CortadoAIException', () async {
      final client = RecordingClient((request, body) async {
        return _streamedResponse(
          200,
          'event: error\n'
          'data: {"error":"provider failed"}\n\n',
          headers: const <String, String>{
            'Content-Type': 'text/event-stream',
          },
        );
      });
      final service = CortadoAIService(
        baseUrl: 'http://localhost:8080',
        httpClient: client,
      );

      expect(
        service
            .streamCompletion(const CortadoCompletionContext(
              workspaceId: 'ws-123',
              prefix: 'void ',
            ))
            .toList(),
        throwsA(
          isA<CortadoAIException>().having(
            (error) => error.message,
            'message',
            'provider failed',
          ),
        ),
      );
    });
  });
}

http.StreamedResponse _streamedResponse(
  int status,
  String body, {
  Map<String, String>? headers,
}) {
  final bytes = utf8.encode(body);
  return http.StreamedResponse(
    Stream<List<int>>.fromIterable(<List<int>>[Uint8List.fromList(bytes)]),
    status,
    headers: headers ?? const <String, String>{},
  );
}

class RecordedRequest {
  RecordedRequest(this.request, this.bodyBytes);

  final http.BaseRequest request;
  final List<int> bodyBytes;
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
