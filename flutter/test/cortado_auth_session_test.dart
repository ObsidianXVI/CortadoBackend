import 'dart:async';
import 'dart:convert';
import 'dart:typed_data';

import 'package:cortado/cortado.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;

void main() {
  group('CortadoAuthSession', () {
    test('createSession stores tokens and parses the exp claim', () async {
      final accessToken = _jwtExpiringAt(DateTime.utc(2026, 5, 23, 15));
      final client = RecordingClient((request, body) async {
        expect(request.method, 'POST');
        expect(request.url, Uri.parse('https://api.example.dev/v1/sessions'));
        expect(
          jsonDecode(utf8.decode(body)),
          <String, Object?>{
            'api_key': 'secret-api-key',
            'user_id': 'user-1',
          },
        );

        return _jsonResponse(200, <String, Object?>{
          'access_token': accessToken,
          'refresh_token': 'refresh-token',
        });
      });

      // Use a fixed clock so the test doesn't race wall-clock time.
      // If the current time is already within the 5-minute refresh lead,
      // CortadoAuthSession would schedule an immediate refresh that triggers
      // an unexpected extra HTTP call during this test.
      final session = CortadoAuthSession(
        baseUrl: 'https://api.example.dev',
        httpClient: client,
        now: () => DateTime.utc(2026, 5, 23, 12), // well before expiry
      );

      await session.createSession(
        apiKey: 'secret-api-key',
        userId: 'user-1',
      );

      expect(session.accessToken, accessToken);
      expect(session.refreshToken, 'refresh-token');
      expect(session.expiresAt, DateTime.utc(2026, 5, 23, 15));
    });

    test('schedules a refresh five minutes before expiry', () async {
      final now = DateTime.utc(2026, 5, 23, 13);
      final timers = <FakeTimer>[];
      final refreshedToken = _jwtExpiringAt(DateTime.utc(2026, 5, 23, 16));
      final client = RecordingClient((request, body) async {
        expect(request.method, 'POST');
        expect(request.url,
            Uri.parse('https://api.example.dev/base/v1/sessions/refresh'));
        expect(
          jsonDecode(utf8.decode(body)),
          <String, Object?>{
            'refresh_token': 'refresh-token',
          },
        );

        return _jsonResponse(200, <String, Object?>{
          'access_token': refreshedToken,
        });
      });

      final session = CortadoAuthSession(
        baseUrl: 'https://api.example.dev/base',
        httpClient: client,
        now: () => now,
        timerFactory: (duration, callback) {
          final timer = FakeTimer(duration, callback);
          timers.add(timer);
          return timer;
        },
      );

      session.setTokens(
        accessToken: _jwtExpiringAt(DateTime.utc(2026, 5, 23, 13, 10)),
        refreshToken: 'refresh-token',
      );

      expect(timers, hasLength(1));
      expect(timers.single.duration, const Duration(minutes: 5));

      timers.single.fire();
      await Future<void>.delayed(Duration.zero);

      expect(session.accessToken, refreshedToken);
    });

    test('refreshes synchronously when the current token is expired', () async {
      final refreshedToken = _jwtExpiringAt(DateTime.utc(2026, 5, 23, 16));
      final client = RecordingClient((request, body) async {
        expect(request.url,
            Uri.parse('https://api.example.dev/v1/sessions/refresh'));
        return _jsonResponse(200, <String, Object?>{
          'access_token': refreshedToken,
        });
      });

      final session = CortadoAuthSession(
        baseUrl: 'https://api.example.dev',
        httpClient: client,
        now: () => DateTime.utc(2026, 5, 23, 15),
      );
      session.setTokens(
        accessToken: _jwtExpiringAt(DateTime.utc(2026, 5, 23, 14, 59)),
        refreshToken: 'refresh-token',
      );

      final token = await session.accessTokenForHttpRequest();

      expect(token, refreshedToken);
      expect(session.accessToken, refreshedToken);
    });

    test('exchangeFirebaseSession stores exchanged tokens', () async {
      final accessToken = _jwtExpiringAt(DateTime.utc(2026, 5, 23, 15));
      final client = RecordingClient((request, body) async {
        expect(request.method, 'POST');
        expect(
          request.url,
          Uri.parse('https://api.example.dev/v1/sessions/exchange/firebase'),
        );
        expect(
          jsonDecode(utf8.decode(body)),
          <String, Object?>{
            'firebase_id_token': 'firebase-id-token',
          },
        );

        return _jsonResponse(200, <String, Object?>{
          'access_token': accessToken,
          'refresh_token': 'refresh-token',
        });
      });

      final session = CortadoAuthSession(
        baseUrl: 'https://api.example.dev',
        httpClient: client,
        now: () => DateTime.utc(2026, 5, 23, 12),
      );

      await session.exchangeFirebaseSession(
        firebaseIdToken: 'firebase-id-token',
      );

      expect(session.accessToken, accessToken);
      expect(session.refreshToken, 'refresh-token');
      expect(session.expiresAt, DateTime.utc(2026, 5, 23, 15));
    });

    test('createSession omits user_id for platform api keys', () async {
      final accessToken = _jwtExpiringAt(DateTime.utc(2026, 5, 23, 15));
      final client = RecordingClient((request, body) async {
        expect(request.method, 'POST');
        expect(request.url, Uri.parse('https://api.example.dev/v1/sessions'));
        expect(
          jsonDecode(utf8.decode(body)),
          <String, Object?>{
            'api_key': 'platform-api-key',
          },
        );

        return _jsonResponse(200, <String, Object?>{
          'access_token': accessToken,
          'refresh_token': 'refresh-token',
        });
      });

      final session = CortadoAuthSession(
        baseUrl: 'https://api.example.dev',
        httpClient: client,
        now: () => DateTime.utc(2026, 5, 23, 12),
      );

      await session.createSession(apiKey: 'platform-api-key');

      expect(session.accessToken, accessToken);
      expect(session.refreshToken, 'refresh-token');
    });

    test('clear removes the current session state', () async {
      final session = CortadoAuthSession(
        baseUrl: 'https://api.example.dev',
        now: () => DateTime.utc(2026, 5, 23, 12),
      );

      session.setTokens(
        accessToken: _jwtExpiringAt(DateTime.utc(2026, 5, 23, 15)),
        refreshToken: 'refresh-token',
      );

      session.clear();

      expect(session.hasSession, isFalse);
      expect(session.accessToken, isNull);
      expect(session.refreshToken, isNull);
      expect(session.expiresAt, isNull);
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
  final header = base64Url.encode(utf8.encode(jsonEncode(<String, String>{
    'alg': 'RS256',
    'typ': 'JWT',
  })));
  final payload = base64Url.encode(utf8.encode(jsonEncode(<String, Object>{
    'exp': timestamp.millisecondsSinceEpoch ~/ 1000,
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
