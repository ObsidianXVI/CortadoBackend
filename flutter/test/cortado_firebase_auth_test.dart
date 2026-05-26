import 'dart:convert';
import 'dart:typed_data';

import 'package:cortado/cortado.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;

void main() {
  group('CortadoFirebaseAuthClient', () {
    test('email sign-in exchanges the Firebase token into a Cortado session',
        () async {
      final accessToken = _jwtExpiringAt(DateTime.utc(2030, 5, 23, 15));
      final identityClient = _FakeIdentityClient(
        idToken: 'firebase-id-token',
        user: const CortadoFirebaseUser(
          uid: 'firebase-user-1',
          email: 'user@example.com',
        ),
      );
      final authClient = CortadoFirebaseAuthClient(
        baseUrl: 'https://api.example.dev',
        httpClient: _RecordingClient((request, body) async {
          expect(
            request.url,
            Uri.parse(
              'https://api.example.dev/v1/sessions/exchange/firebase',
            ),
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
        }),
        identityClient: identityClient,
      );
      addTearDown(authClient.dispose);

      final result = await authClient.signInWithEmailPassword(
        email: 'user@example.com',
        password: 'hunter2',
      );

      expect(result.user.email, 'user@example.com');
      expect(result.session.accessToken, accessToken);
      expect(result.session.refreshToken, 'refresh-token');
      expect(
        identityClient.lastEmailPasswordSignIn,
        ('user@example.com', 'hunter2'),
      );
    });

    test('signOut clears the reusable Cortado session', () async {
      final accessToken = _jwtExpiringAt(DateTime.utc(2030, 5, 23, 15));
      final identityClient = _FakeIdentityClient(
        idToken: 'firebase-id-token',
        user: const CortadoFirebaseUser(
          uid: 'firebase-user-1',
          email: 'user@example.com',
        ),
      );
      final authClient = CortadoFirebaseAuthClient(
        baseUrl: 'https://api.example.dev',
        httpClient: _RecordingClient((request, body) async {
          return _jsonResponse(200, <String, Object?>{
            'access_token': accessToken,
            'refresh_token': 'refresh-token',
          });
        }),
        identityClient: identityClient,
      );
      addTearDown(authClient.dispose);

      await authClient.exchangeCurrentUser();
      expect(authClient.session.hasSession, isTrue);

      await authClient.signOut();

      expect(authClient.session.hasSession, isFalse);
      expect(identityClient.signOutCalls, 1);
    });
  });

  testWidgets('CortadoEmbeddedAuth signs in and reports the result',
      (WidgetTester tester) async {
    final accessToken = _jwtExpiringAt(DateTime.utc(2030, 5, 23, 15));
    final identityClient = _FakeIdentityClient(
      idToken: 'firebase-id-token',
      user: const CortadoFirebaseUser(
        uid: 'firebase-user-1',
        email: 'user@example.com',
      ),
    );
    final authClient = CortadoFirebaseAuthClient(
      baseUrl: 'https://api.example.dev',
      httpClient: _RecordingClient((request, body) async {
        return _jsonResponse(200, <String, Object?>{
          'access_token': accessToken,
          'refresh_token': 'refresh-token',
        });
      }),
      identityClient: identityClient,
    );
    addTearDown(authClient.dispose);

    CortadoFirebaseAuthResult? seenResult;

    await tester.pumpWidget(
      MaterialApp(
        home: Scaffold(
          body: CortadoEmbeddedAuth(
            authClient: authClient,
            onAuthenticated: (result) {
              seenResult = result;
            },
          ),
        ),
      ),
    );

    await tester.enterText(find.byType(TextField).at(0), 'user@example.com');
    await tester.enterText(find.byType(TextField).at(1), 'hunter2');
    await tester.tap(find.text('Sign in'));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 10));

    expect(seenResult, isNotNull);
    expect(seenResult!.user.email, 'user@example.com');
    expect(seenResult!.session.accessToken, accessToken);
    expect(find.text('Signed in as user@example.com'), findsOneWidget);

    await tester.pumpWidget(const SizedBox.shrink());
    await authClient.dispose();
    await tester.pump();
  });
}

class _FakeIdentityClient implements CortadoFirebaseIdentityClient {
  _FakeIdentityClient({
    required this.idToken,
    this.user,
  });

  final String idToken;
  (String, String)? lastEmailPasswordRegister;
  (String, String)? lastEmailPasswordSignIn;
  int signOutCalls = 0;
  CortadoFirebaseUser? user;

  @override
  CortadoFirebaseUser? get currentUser => user;

  @override
  Future<String> currentIdToken({bool forceRefresh = false}) async {
    if (user == null) {
      throw StateError('No Firebase user is currently signed in.');
    }
    return idToken;
  }

  @override
  Future<CortadoFirebaseUser> registerWithEmailPassword({
    required String email,
    required String password,
  }) async {
    lastEmailPasswordRegister = (email, password);
    return user!;
  }

  @override
  Future<CortadoFirebaseUser> signInWithEmailPassword({
    required String email,
    required String password,
  }) async {
    lastEmailPasswordSignIn = (email, password);
    return user!;
  }

  @override
  Future<CortadoFirebaseUser> signInWithGoogle({
    Iterable<String> scopes = const <String>[],
    String? loginHint,
  }) async {
    return user!;
  }

  @override
  Future<void> signOut() async {
    signOutCalls++;
    user = null;
  }
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
