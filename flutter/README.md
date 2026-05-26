# Cortado Flutter Package

`cortado` is an embeddable Flutter package for talking to the Cortado control plane from a host IDE application.

## First-party browser auth

The package now supports the zero-backend browser path directly:

- `CortadoFirebaseAuthClient` handles Firebase email/password sign-in, Firebase Google popup sign-in on web, and Firebase-to-Cortado session exchange.
- `CortadoEmbeddedAuth` provides a small drop-in auth widget for hosts that want package-owned UI.
- `CortadoAuthSession` still exposes the lower-level session primitives and keeps Cortado access tokens refreshed automatically.

## Low-level usage

```dart
final authClient = CortadoFirebaseAuthClient(
  baseUrl: 'https://cortado.example.com',
  firebaseOptions: const FirebaseOptions(
    apiKey: '...',
    appId: '...',
    messagingSenderId: '...',
    projectId: '...',
  ),
);

final result = await authClient.signInWithEmailPassword(
  email: 'user@example.com',
  password: 'correct horse battery staple',
);

final manager = WorkspaceManager(
  baseUrl: 'https://cortado.example.com',
  authSession: result.session,
);

final client = CortadoClient(
  baseUrl: 'https://cortado.example.com',
  authSession: result.session,
);
```

## Drop-in usage

```dart
CortadoEmbeddedAuth(
  authClient: authClient,
  onAuthenticated: (result) {
    // Reuse result.session with WorkspaceManager and CortadoClient.
  },
)
```

## API-key path

`CortadoAuthSession.createSession(apiKey: ..., userId: ...)` remains available for headless, CLI, and other non-browser bootstrap flows.
