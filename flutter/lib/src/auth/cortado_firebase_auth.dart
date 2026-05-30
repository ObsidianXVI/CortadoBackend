import 'dart:async';

import 'package:firebase_auth/firebase_auth.dart';
import 'package:firebase_core/firebase_core.dart';
import 'package:flutter/foundation.dart';
import 'package:http/http.dart' as http;

import '../cortado_auth_session.dart';

class CortadoFirebaseUser {
  const CortadoFirebaseUser({
    required this.uid,
    this.displayName,
    this.email,
  });

  final String uid;
  final String? displayName;
  final String? email;

  String get label {
    final email = this.email?.trim();
    if (email != null && email.isNotEmpty) {
      return email;
    }

    final displayName = this.displayName?.trim();
    if (displayName != null && displayName.isNotEmpty) {
      return displayName;
    }

    return uid;
  }
}

class CortadoFirebaseAuthResult {
  const CortadoFirebaseAuthResult({
    required this.session,
    required this.user,
  });

  final CortadoAuthSession session;
  final CortadoFirebaseUser user;
}

abstract class CortadoFirebaseIdentityClient {
  CortadoFirebaseUser? get currentUser;

  Future<String> currentIdToken({bool forceRefresh = false});

  Future<CortadoFirebaseUser> registerWithEmailPassword({
    required String email,
    required String password,
  });

  Future<CortadoFirebaseUser> signInWithEmailPassword({
    required String email,
    required String password,
  });

  Future<CortadoFirebaseUser> signInWithGoogle({
    Iterable<String> scopes = const <String>[],
    String? loginHint,
  });

  Future<void> signOut();
}

class FirebaseCortadoIdentityClient implements CortadoFirebaseIdentityClient {
  FirebaseCortadoIdentityClient({
    FirebaseAuth? firebaseAuth,
    this.firebaseAppName = 'cortado-managed-auth',
    this.firebaseOptions,
  }) : _firebaseAuth = firebaseAuth;

  FirebaseAuth? _firebaseAuth;
  FirebaseApp? _app;
  CortadoFirebaseUser? _currentUser;

  final String firebaseAppName;
  final FirebaseOptions? firebaseOptions;

  @override
  CortadoFirebaseUser? get currentUser => _currentUser;

  @override
  Future<String> currentIdToken({bool forceRefresh = false}) async {
    final auth = await _auth();
    final user = auth.currentUser;
    if (user == null) {
      throw StateError('No Firebase user is currently signed in.');
    }

    final idToken = await user.getIdToken(forceRefresh);
    if (idToken == null || idToken.trim().isEmpty) {
      throw StateError('Firebase did not return an ID token.');
    }

    _currentUser = _mapUser(user);
    return idToken.trim();
  }

  @override
  Future<CortadoFirebaseUser> registerWithEmailPassword({
    required String email,
    required String password,
  }) async {
    final auth = await _auth();
    final credential = await auth.createUserWithEmailAndPassword(
      email: email.trim(),
      password: password,
    );
    return _userFromCredential(credential);
  }

  @override
  Future<CortadoFirebaseUser> signInWithEmailPassword({
    required String email,
    required String password,
  }) async {
    final auth = await _auth();
    final credential = await auth.signInWithEmailAndPassword(
      email: email.trim(),
      password: password,
    );
    return _userFromCredential(credential);
  }

  @override
  Future<CortadoFirebaseUser> signInWithGoogle({
    Iterable<String> scopes = const <String>[],
    String? loginHint,
  }) async {
    if (!kIsWeb) {
      throw UnsupportedError(
        'The built-in Google popup flow is only available on Flutter web. '
        'On native platforms, complete Firebase sign-in in your host app and '
        'then call exchangeCurrentUser().',
      );
    }

    final auth = await _auth();
    final provider = GoogleAuthProvider();
    for (final scope in scopes) {
      final trimmed = scope.trim();
      if (trimmed.isNotEmpty) {
        provider.addScope(trimmed);
      }
    }

    final trimmedLoginHint = loginHint?.trim();
    if (trimmedLoginHint != null && trimmedLoginHint.isNotEmpty) {
      provider.setCustomParameters(<String, String>{
        'login_hint': trimmedLoginHint,
      });
    }

    final credential = await auth.signInWithPopup(provider);
    return _userFromCredential(credential);
  }

  @override
  Future<void> signOut() async {
    final auth = await _auth();
    await auth.signOut();
    _currentUser = null;
  }

  Future<FirebaseAuth> _auth() async {
    if (_firebaseAuth case final FirebaseAuth auth) {
      return auth;
    }

    if (firebaseOptions != null) {
      _app = await initializeOrReuseNamedFirebaseApp<FirebaseApp>(
        initialize: () {
          return Firebase.initializeApp(
            name: firebaseAppName,
            options: firebaseOptions!,
          );
        },
        reuseExisting: () => Firebase.app(firebaseAppName),
      );

      _firebaseAuth = FirebaseAuth.instanceFor(app: _app!);
      return _firebaseAuth!;
    }

    try {
      Firebase.app();
    } on Object {
      throw StateError(
        'Provide firebaseOptions or an initialized FirebaseAuth instance before using CortadoFirebaseAuthClient.',
      );
    }

    _firebaseAuth = FirebaseAuth.instance;
    return _firebaseAuth!;
  }

  CortadoFirebaseUser _userFromCredential(UserCredential credential) {
    final user = credential.user;
    if (user == null) {
      throw StateError('Firebase did not return a user.');
    }

    _currentUser = _mapUser(user);
    return _currentUser!;
  }

  CortadoFirebaseUser _mapUser(User user) {
    return CortadoFirebaseUser(
      uid: user.uid.trim(),
      displayName: user.displayName?.trim(),
      email: user.email?.trim(),
    );
  }
}

Future<T> initializeOrReuseNamedFirebaseApp<T>({
  required Future<T> Function() initialize,
  required T Function() reuseExisting,
}) async {
  try {
    return await initialize();
  } on FirebaseException catch (error) {
    if (error.code != 'duplicate-app') {
      rethrow;
    }
    return reuseExisting();
  }
}

class CortadoFirebaseAuthClient {
  CortadoFirebaseAuthClient({
    required this.baseUrl,
    String firebaseAppName = 'cortado-managed-auth',
    FirebaseAuth? firebaseAuth,
    FirebaseOptions? firebaseOptions,
    http.Client? httpClient,
    CortadoFirebaseIdentityClient? identityClient,
    CortadoAuthSession? session,
    this.googleScopes = const <String>[],
    this.googleLoginHint,
  })  : _identityClient = identityClient ??
            FirebaseCortadoIdentityClient(
              firebaseAuth: firebaseAuth,
              firebaseAppName: firebaseAppName,
              firebaseOptions: firebaseOptions,
            ),
        _ownsSession = session == null,
        session = session ??
            CortadoAuthSession(
              baseUrl: baseUrl,
              httpClient: httpClient,
            );

  final String baseUrl;
  final Iterable<String> googleScopes;
  final String? googleLoginHint;
  final CortadoFirebaseIdentityClient _identityClient;
  final bool _ownsSession;
  final CortadoAuthSession session;

  CortadoFirebaseUser? get currentUser => _identityClient.currentUser;

  Future<CortadoFirebaseAuthResult> exchangeCurrentUser({
    bool forceRefresh = true,
  }) async {
    final idToken = await _identityClient.currentIdToken(
      forceRefresh: forceRefresh,
    );
    await session.exchangeFirebaseSession(firebaseIdToken: idToken);

    final user = _identityClient.currentUser;
    if (user == null) {
      throw StateError('Firebase did not return a current user.');
    }

    return CortadoFirebaseAuthResult(
      session: session,
      user: user,
    );
  }

  Future<CortadoFirebaseAuthResult> registerWithEmailPassword({
    required String email,
    required String password,
  }) async {
    await _identityClient.registerWithEmailPassword(
      email: email,
      password: password,
    );
    return exchangeCurrentUser();
  }

  Future<CortadoFirebaseAuthResult> signInWithEmailPassword({
    required String email,
    required String password,
  }) async {
    await _identityClient.signInWithEmailPassword(
      email: email,
      password: password,
    );
    return exchangeCurrentUser();
  }

  Future<CortadoFirebaseAuthResult> signInWithGoogle() async {
    await _identityClient.signInWithGoogle(
      scopes: googleScopes,
      loginHint: googleLoginHint,
    );
    return exchangeCurrentUser();
  }

  Future<void> signOut({bool clearSession = true}) async {
    await _identityClient.signOut();
    if (clearSession) {
      session.clear();
    }
  }

  Future<void> dispose() async {
    if (_ownsSession) {
      await session.dispose();
    }
  }
}
