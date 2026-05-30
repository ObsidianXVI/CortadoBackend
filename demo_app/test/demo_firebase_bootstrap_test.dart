import 'dart:convert';
import 'dart:typed_data';

import 'package:cortado/cortado.dart';
import 'package:demo_app/src/demo_bootstrap_config.dart';
import 'package:demo_app/src/demo_firebase_bootstrap.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;

void main() {
  test('listApiKeysWithSession uses the Cortado session bearer token',
      () async {
    final authSession = CortadoAuthSession(
      baseUrl: 'https://api.example.dev',
      httpClient: _RecordingClient((request, body) async {
        return _jsonResponse(200, <String, Object?>{
          'access_token': _jwtExpiringAt(DateTime.utc(2030, 5, 30, 12)),
        });
      }),
    );
    authSession.setTokens(
      accessToken: _jwtExpiringAt(DateTime.utc(2030, 5, 30, 12)),
      refreshToken: 'refresh-token',
    );

    final bootstrap = DemoFirebaseBootstrap(_testConfig());
    final apiClient = _RecordingClient((request, body) async {
      expect(request.url, Uri.parse('https://api.example.dev/v1/api-keys'));
      expect(request.method, 'GET');
      expect(
        request.headers['Authorization'],
        'Bearer ${authSession.accessToken}',
      );
      return _jsonResponse(200, <String, Object?>{
        'apiKeys': <Object?>[
          <String, Object?>{
            'id': 'key-1',
            'kind': 'personal',
            'tenantId': 'tenant-1',
            'userId': 'user-1',
            'revoked': false,
            'createdAt': '2026-05-30T12:00:00Z',
          },
        ],
      });
    });

    final listed = await bootstrap.listApiKeysWithSession(
      'https://api.example.dev',
      authSession,
      client: apiClient,
    );

    expect(listed, hasLength(1));
    expect(listed.single.id, 'key-1');
  });

  test('mintApiKeyWithSession uses the Cortado session bearer token', () async {
    final authSession = CortadoAuthSession(baseUrl: 'https://api.example.dev');
    authSession.setTokens(
      accessToken: _jwtExpiringAt(DateTime.utc(2030, 5, 30, 12)),
      refreshToken: 'refresh-token',
    );

    final bootstrap = DemoFirebaseBootstrap(_testConfig());
    final apiClient = _RecordingClient((request, body) async {
      expect(request.url, Uri.parse('https://api.example.dev/v1/api-keys'));
      expect(request.method, 'POST');
      expect(
        request.headers['Authorization'],
        'Bearer ${authSession.accessToken}',
      );
      return _jsonResponse(201, <String, Object?>{
        'apiKey': 'cortado_personal',
        'record': <String, Object?>{
          'id': 'key-2',
          'kind': 'personal',
          'tenantId': 'tenant-1',
          'userId': 'user-1',
          'revoked': false,
          'createdAt': '2026-05-30T12:05:00Z',
        },
      });
    });

    final issued = await bootstrap.mintApiKeyWithSession(
      'https://api.example.dev',
      authSession,
      client: apiClient,
    );

    expect(issued.apiKey, 'cortado_personal');
    expect(issued.record.id, 'key-2');
  });
}

DemoBootstrapConfig _testConfig() {
  return const DemoBootstrapConfig(
    baseUrl: 'https://api.example.dev',
    apiKey: '',
    userId: '',
    workspaceId: '',
    shell: '/bin/bash',
    image:
        'us-central1-docker.pkg.dev/cortado-ide/cortado-dev/cortado-workspace:781d613',
    filePath: 'lib/main.dart',
    cpu: 1,
    memoryGb: 2,
    storageGb: 10,
    firebaseApiKey: 'firebase-api-key',
    firebaseAuthDomain: 'cortado-ide.firebaseapp.com',
    firebaseProjectId: 'cortado-ide',
    firebaseAppId: '1:123:web:abc',
    firebaseMessagingSenderId: '123',
    firebaseStorageBucket: '',
    firebaseMeasurementId: '',
    firebaseEmail: '',
    firebasePassword: '',
    firebaseDevTenantId: 'demo-tenant',
  );
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
  final header = base64Url.encode(utf8.encode(jsonEncode(<String, String>{
    'alg': 'RS256',
    'typ': 'JWT',
  })));
  final payload = base64Url.encode(utf8.encode(jsonEncode(<String, Object>{
    'exp': timestamp.millisecondsSinceEpoch ~/ 1000,
  })));
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
