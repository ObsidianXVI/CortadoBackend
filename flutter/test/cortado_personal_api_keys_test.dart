import 'dart:convert';
import 'dart:typed_data';

import 'package:cortado/cortado.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;

void main() {
  group('CortadoPersonalApiKeysClient', () {
    test('issues a personal API key with session auth', () async {
      final session = CortadoAuthSession(
        baseUrl: 'https://api.example.dev',
        now: () => DateTime.utc(2026, 5, 26, 2),
      );
      session.setTokens(
        accessToken: _jwtExpiringAt(DateTime.utc(2026, 5, 26, 4)),
        refreshToken: 'refresh-token',
      );
      addTearDown(session.dispose);

      final client = CortadoPersonalApiKeysClient(
        baseUrl: 'https://api.example.dev',
        authSession: session,
        httpClient: _RecordingClient((request, body) async {
          expect(request.method, 'POST');
          expect(request.url, Uri.parse('https://api.example.dev/v1/api-keys'));
          expect(request.headers['authorization'], startsWith('Bearer '));
          expect(body, isEmpty);

          return _jsonResponse(201, <String, Object?>{
            'apiKey': 'cortado_secret',
            'record': <String, Object?>{
              'id': 'key-1',
              'tenantId': 'tenant-1',
              'userId': 'user-1',
              'revoked': false,
              'createdAt': '2026-05-26T02:00:00Z',
            },
          });
        }),
      );

      final issued = await client.issue();

      expect(issued.apiKey, 'cortado_secret');
      expect(issued.record.id, 'key-1');
      expect(issued.record.tenantId, 'tenant-1');
      expect(issued.record.userId, 'user-1');
    });

    test('lists and revokes personal API keys with session auth', () async {
      final session = CortadoAuthSession(
        baseUrl: 'https://api.example.dev',
        now: () => DateTime.utc(2026, 5, 26, 2),
      );
      session.setTokens(
        accessToken: _jwtExpiringAt(DateTime.utc(2026, 5, 26, 4)),
        refreshToken: 'refresh-token',
      );
      addTearDown(session.dispose);

      final seenRequests = <Uri>[];
      final client = CortadoPersonalApiKeysClient(
        baseUrl: 'https://api.example.dev',
        authSession: session,
        httpClient: _RecordingClient((request, _) async {
          seenRequests.add(request.url);
          if (request.method == 'GET') {
            return _jsonResponse(200, <String, Object?>{
              'apiKeys': <Object?>[
                <String, Object?>{
                  'id': 'key-1',
                  'tenantId': 'tenant-1',
                  'userId': 'user-1',
                  'revoked': false,
                  'createdAt': '2026-05-26T02:00:00Z',
                },
              ],
            });
          }

          return _jsonResponse(200, <String, Object?>{
            'record': <String, Object?>{
              'id': 'key-1',
              'tenantId': 'tenant-1',
              'userId': 'user-1',
              'revoked': true,
              'createdAt': '2026-05-26T02:00:00Z',
            },
          });
        }),
      );

      final listed = await client.list();
      final revoked = await client.revoke('key-1');

      expect(listed, hasLength(1));
      expect(listed.single.id, 'key-1');
      expect(revoked.revoked, isTrue);
      expect(
        seenRequests,
        <Uri>[
          Uri.parse('https://api.example.dev/v1/api-keys'),
          Uri.parse('https://api.example.dev/v1/api-keys/key-1'),
        ],
      );
    });

    test('rejects management calls without a Cortado session', () async {
      final session = CortadoAuthSession(
        baseUrl: 'https://api.example.dev',
        now: () => DateTime.utc(2026, 5, 26, 2),
      );
      addTearDown(session.dispose);

      final client = CortadoPersonalApiKeysClient(
        baseUrl: 'https://api.example.dev',
        authSession: session,
      );
      addTearDown(client.dispose);

      await expectLater(
        client.issue(),
        throwsA(
          isA<StateError>().having(
            (error) => error.message,
            'message',
            contains('Cortado session is required'),
          ),
        ),
      );
    });
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

String _jwtExpiringAt(DateTime timestamp) {
  final header = base64Url.encode(
    utf8.encode(jsonEncode(<String, String>{'alg': 'RS256', 'typ': 'JWT'})),
  );
  final payload = base64Url.encode(
    utf8.encode(
      jsonEncode(<String, Object>{
        'exp': timestamp.millisecondsSinceEpoch ~/ 1000,
      }),
    ),
  );
  return '$header.$payload.signature';
}

class _RecordingClient extends http.BaseClient {
  _RecordingClient(this._handler);

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
