import 'dart:convert';

import 'package:http/http.dart' as http;

import '../cortado_auth_session.dart';

class CortadoPersonalApiKeyRecord {
  const CortadoPersonalApiKeyRecord({
    required this.id,
    required this.tenantId,
    required this.userId,
    required this.revoked,
    this.createdAt,
  });

  final String id;
  final String tenantId;
  final String userId;
  final bool revoked;
  final DateTime? createdAt;

  factory CortadoPersonalApiKeyRecord.fromJson(Map<String, dynamic> json) {
    return CortadoPersonalApiKeyRecord(
      id: (json['id'] as String? ?? '').trim(),
      tenantId: (json['tenantId'] as String? ?? '').trim(),
      userId: (json['userId'] as String? ?? '').trim(),
      revoked: json['revoked'] as bool? ?? false,
      createdAt: DateTime.tryParse((json['createdAt'] as String? ?? '').trim()),
    );
  }
}

class CortadoIssuedPersonalApiKey {
  const CortadoIssuedPersonalApiKey({
    required this.apiKey,
    required this.record,
  });

  final String apiKey;
  final CortadoPersonalApiKeyRecord record;

  factory CortadoIssuedPersonalApiKey.fromJson(Map<String, dynamic> json) {
    return CortadoIssuedPersonalApiKey(
      apiKey: (json['apiKey'] as String? ?? '').trim(),
      record: CortadoPersonalApiKeyRecord.fromJson(
        (json['record'] as Map<Object?, Object?>? ?? const <Object?, Object?>{})
            .cast<String, dynamic>(),
      ),
    );
  }
}

class CortadoPersonalApiKeysClient {
  CortadoPersonalApiKeysClient({
    required this.authSession,
    required this.baseUrl,
    http.Client? httpClient,
  })  : _client = httpClient ?? http.Client(),
        _ownsClient = httpClient == null;

  final CortadoAuthSession authSession;
  final String baseUrl;
  final http.Client _client;
  final bool _ownsClient;

  Future<CortadoIssuedPersonalApiKey> issue() async {
    final response = await _client.post(
      _endpoint(),
      headers: await _authorizedHeaders(),
    );
    final payload = _decodeResponse(response);
    return CortadoIssuedPersonalApiKey.fromJson(payload);
  }

  Future<List<CortadoPersonalApiKeyRecord>> list() async {
    final response = await _client.get(
      _endpoint(),
      headers: await _authorizedHeaders(),
    );
    final payload = _decodeResponse(response);
    final rawKeys = payload['apiKeys'] as List<Object?>? ?? const <Object?>[];
    return rawKeys
        .whereType<Map<Object?, Object?>>()
        .map(
          (json) => CortadoPersonalApiKeyRecord.fromJson(
            json.cast<String, dynamic>(),
          ),
        )
        .toList(growable: false);
  }

  Future<CortadoPersonalApiKeyRecord> revoke(String keyId) async {
    final trimmedKeyId = keyId.trim();
    if (trimmedKeyId.isEmpty) {
      throw ArgumentError.value(keyId, 'keyId', 'Must not be empty.');
    }

    final response = await _client.delete(
      _endpoint(trimmedKeyId),
      headers: await _authorizedHeaders(),
    );
    final payload = _decodeResponse(response);
    return CortadoPersonalApiKeyRecord.fromJson(
      (payload['record'] as Map<Object?, Object?>? ??
              const <Object?, Object?>{})
          .cast<String, dynamic>(),
    );
  }

  Future<void> dispose() async {
    if (_ownsClient) {
      _client.close();
    }
  }

  Uri _endpoint([String? keyId]) {
    final baseUri = Uri.parse(baseUrl);
    return baseUri.replace(
      pathSegments: <String>[
        ...baseUri.pathSegments.where((segment) => segment.isNotEmpty),
        'v1',
        'api-keys',
        if (keyId case final String value) value,
      ],
    );
  }

  Map<String, dynamic> _decodeResponse(http.Response response) {
    if (response.statusCode < 200 || response.statusCode >= 300) {
      throw CortadoAuthException(
        statusCode: response.statusCode,
        message: utf8.decode(response.bodyBytes).trim(),
      );
    }

    final decoded = jsonDecode(utf8.decode(response.bodyBytes));
    if (decoded is! Map<Object?, Object?>) {
      throw const FormatException('Expected a JSON object response body.');
    }
    return decoded.cast<String, dynamic>();
  }

  Future<Map<String, String>> _authorizedHeaders() async {
    final accessToken = await authSession.accessTokenForHttpRequest();
    if (accessToken == null || accessToken.trim().isEmpty) {
      throw StateError(
        'A Cortado session is required before managing personal API keys.',
      );
    }

    return <String, String>{
      'Authorization': 'Bearer ${accessToken.trim()}',
    };
  }
}
