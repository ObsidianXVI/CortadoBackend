import 'dart:convert';

import 'package:firebase_auth/firebase_auth.dart';
import 'package:firebase_core/firebase_core.dart';
import 'package:http/http.dart' as http;

import 'demo_bootstrap_config.dart';

class DemoApiKeyRecord {
  const DemoApiKeyRecord({
    required this.id,
    required this.tenantId,
    required this.userId,
    required this.revoked,
    required this.createdAt,
  });

  final String id;
  final String tenantId;
  final String userId;
  final bool revoked;
  final DateTime? createdAt;

  factory DemoApiKeyRecord.fromJson(Map<String, dynamic> json) {
    return DemoApiKeyRecord(
      id: (json['id'] as String? ?? '').trim(),
      tenantId: (json['tenantId'] as String? ?? '').trim(),
      userId: (json['userId'] as String? ?? '').trim(),
      revoked: json['revoked'] as bool? ?? false,
      createdAt: DateTime.tryParse((json['createdAt'] as String? ?? '').trim()),
    );
  }
}

class DemoIssuedApiKey {
  const DemoIssuedApiKey({
    required this.apiKey,
    required this.record,
  });

  final String apiKey;
  final DemoApiKeyRecord record;

  factory DemoIssuedApiKey.fromJson(Map<String, dynamic> json) {
    return DemoIssuedApiKey(
      apiKey: (json['apiKey'] as String? ?? '').trim(),
      record: DemoApiKeyRecord.fromJson(
        (json['record'] as Map<Object?, Object?>? ?? const <Object?, Object?>{})
            .cast<String, dynamic>(),
      ),
    );
  }
}

class DemoFirebaseBootstrap {
  DemoFirebaseBootstrap(this.config);

  static const String _appName = 'cortado-demo-bootstrap';

  final DemoBootstrapConfig config;

  FirebaseApp? _app;
  FirebaseAuth? _auth;

  bool get isConfigured => config.hasFirebaseBootstrapConfig;

  Future<FirebaseAuth> auth() async {
    if (!isConfigured) {
      throw StateError(
        'Firebase bootstrap is not configured. Set the Firebase values in demo_app/.env first.',
      );
    }
    if (_auth != null) {
      return _auth!;
    }

    final existingApp = Firebase.apps.where((app) => app.name == _appName);
    if (existingApp.isNotEmpty) {
      _app = existingApp.first;
    } else {
      _app = await Firebase.initializeApp(
        name: _appName,
        options: FirebaseOptions(
          apiKey: config.firebaseApiKey,
          appId: config.firebaseAppId,
          messagingSenderId: config.firebaseMessagingSenderId,
          projectId: config.firebaseProjectId,
          authDomain: _nullable(config.firebaseAuthDomain),
          measurementId: _nullable(config.firebaseMeasurementId),
          storageBucket: _nullable(config.firebaseStorageBucket),
        ),
      );
    }

    _auth = FirebaseAuth.instanceFor(app: _app!);
    return _auth!;
  }

  Future<UserCredential> register({
    required String email,
    required String password,
  }) async {
    final firebaseAuth = await auth();
    return firebaseAuth.createUserWithEmailAndPassword(
      email: email.trim(),
      password: password,
    );
  }

  Future<UserCredential> login({
    required String email,
    required String password,
  }) async {
    final firebaseAuth = await auth();
    return firebaseAuth.signInWithEmailAndPassword(
      email: email.trim(),
      password: password,
    );
  }

  Future<void> signOut() async {
    final firebaseAuth = await auth();
    await firebaseAuth.signOut();
  }

  Future<DemoIssuedApiKey> mintApiKey(String baseUrl) async {
    final idToken = await _currentIdToken(forceRefresh: true);
    final response = await http.post(
      _endpoint(baseUrl, '/v1/api-keys'),
      headers: <String, String>{
        'Authorization': 'Bearer $idToken',
      },
    );
    if (response.statusCode != 200 && response.statusCode != 201) {
      throw StateError(_errorMessage(response));
    }

    return DemoIssuedApiKey.fromJson(_decodeObject(response.body));
  }

  Future<List<DemoApiKeyRecord>> listApiKeys(String baseUrl) async {
    final idToken = await _currentIdToken(forceRefresh: true);
    final response = await http.get(
      _endpoint(baseUrl, '/v1/api-keys'),
      headers: <String, String>{
        'Authorization': 'Bearer $idToken',
      },
    );
    if (response.statusCode != 200) {
      throw StateError(_errorMessage(response));
    }

    final payload = _decodeObject(response.body);
    final rawKeys = payload['apiKeys'] as List<Object?>? ?? const <Object?>[];
    return rawKeys
        .whereType<Map<Object?, Object?>>()
        .map((json) => DemoApiKeyRecord.fromJson(json.cast<String, dynamic>()))
        .toList(growable: false);
  }

  Future<String> _currentIdToken({required bool forceRefresh}) async {
    final firebaseAuth = await auth();
    final user = firebaseAuth.currentUser;
    if (user == null) {
      throw StateError('Sign in first before requesting an API key.');
    }

    final idToken = await user.getIdToken(forceRefresh);
    if (idToken == null || idToken.trim().isEmpty) {
      throw StateError('Firebase did not return an ID token.');
    }
    return idToken.trim();
  }

  static Uri _endpoint(String baseUrl, String path) {
    final normalized = baseUrl.trim().replaceAll(RegExp(r'/+$'), '');
    return Uri.parse('$normalized$path');
  }

  static Map<String, dynamic> _decodeObject(String body) {
    final decoded = jsonDecode(body);
    if (decoded is! Map<Object?, Object?>) {
      throw const FormatException('Expected a JSON object response.');
    }
    return decoded.cast<String, dynamic>();
  }

  static String _errorMessage(http.Response response) {
    final body = response.body.trim();
    if (body.isEmpty) {
      return 'Request failed with status ${response.statusCode}.';
    }

    try {
      final payload = _decodeObject(body);
      final message = payload['error'] ?? payload['message'];
      if (message is String && message.trim().isNotEmpty) {
        return message.trim();
      }
    } catch (_) {
      // Fall back to the raw body for plain-text responses.
    }

    return body;
  }

  static String? _nullable(String value) {
    final trimmed = value.trim();
    if (trimmed.isEmpty) {
      return null;
    }
    return trimmed;
  }
}
