import 'dart:async';

import 'package:firebase_auth/firebase_auth.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';

import 'cortado_firebase_auth.dart';

typedef CortadoAuthResultCallback = FutureOr<void> Function(
  CortadoFirebaseAuthResult result,
);

class CortadoEmbeddedAuth extends StatefulWidget {
  const CortadoEmbeddedAuth({
    super.key,
    required this.authClient,
    this.onAuthenticated,
    this.enableRegistration = true,
    this.showGoogleButton,
    this.subtitle,
    this.title = 'Sign in to Cortado',
  });

  final CortadoFirebaseAuthClient authClient;
  final CortadoAuthResultCallback? onAuthenticated;
  final bool enableRegistration;
  final bool? showGoogleButton;
  final String? subtitle;
  final String title;

  @override
  State<CortadoEmbeddedAuth> createState() => _CortadoEmbeddedAuthState();
}

class _CortadoEmbeddedAuthState extends State<CortadoEmbeddedAuth> {
  late final TextEditingController _emailController = TextEditingController();
  late final TextEditingController _passwordController =
      TextEditingController();

  String? _errorMessage;
  String? _statusMessage;
  bool _busy = false;

  @override
  void dispose() {
    _emailController.dispose();
    _passwordController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final user = widget.authClient.currentUser;
    final showGoogleButton = widget.showGoogleButton ?? kIsWeb;

    return Card(
      clipBehavior: Clip.antiAlias,
      child: Padding(
        padding: const EdgeInsets.all(20),
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 420),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            mainAxisSize: MainAxisSize.min,
            children: <Widget>[
              Text(widget.title, style: theme.textTheme.titleLarge),
              if (widget.subtitle case final String subtitle
                  when subtitle.trim().isNotEmpty) ...<Widget>[
                const SizedBox(height: 8),
                Text(
                  subtitle,
                  style: theme.textTheme.bodyMedium,
                ),
              ],
              const SizedBox(height: 20),
              TextField(
                controller: _emailController,
                enabled: !_busy,
                keyboardType: TextInputType.emailAddress,
                decoration: const InputDecoration(
                  labelText: 'Email',
                ),
              ),
              const SizedBox(height: 12),
              TextField(
                controller: _passwordController,
                enabled: !_busy,
                obscureText: true,
                decoration: const InputDecoration(
                  labelText: 'Password',
                ),
              ),
              const SizedBox(height: 16),
              Wrap(
                spacing: 12,
                runSpacing: 12,
                children: <Widget>[
                  ElevatedButton(
                    onPressed: _busy ? null : _signInWithEmailPassword,
                    child: const Text('Sign in'),
                  ),
                  if (widget.enableRegistration)
                    OutlinedButton(
                      onPressed: _busy ? null : _registerWithEmailPassword,
                      child: const Text('Create account'),
                    ),
                  if (showGoogleButton)
                    OutlinedButton(
                      onPressed: _busy ? null : _signInWithGoogle,
                      child: const Text('Continue with Google'),
                    ),
                  if (user != null || widget.authClient.session.hasSession)
                    TextButton(
                      onPressed: _busy ? null : _signOut,
                      child: const Text('Sign out'),
                    ),
                ],
              ),
              if (_busy) ...<Widget>[
                const SizedBox(height: 16),
                const LinearProgressIndicator(),
              ],
              if (user != null) ...<Widget>[
                const SizedBox(height: 16),
                Text(
                  'Signed in as ${user.label}',
                  style: theme.textTheme.bodyMedium,
                ),
              ],
              if (_statusMessage case final String message
                  when message.trim().isNotEmpty) ...<Widget>[
                const SizedBox(height: 12),
                Text(
                  message,
                  style: theme.textTheme.bodyMedium?.copyWith(
                    color: theme.colorScheme.primary,
                  ),
                ),
              ],
              if (_errorMessage case final String message
                  when message.trim().isNotEmpty) ...<Widget>[
                const SizedBox(height: 12),
                Text(
                  message,
                  style: theme.textTheme.bodyMedium?.copyWith(
                    color: theme.colorScheme.error,
                  ),
                ),
              ],
            ],
          ),
        ),
      ),
    );
  }

  Future<void> _registerWithEmailPassword() async {
    await _runAction(
      () => widget.authClient.registerWithEmailPassword(
        email: _emailController.text.trim(),
        password: _passwordController.text,
      ),
      successLabel: 'Account ready',
    );
  }

  Future<void> _signInWithEmailPassword() async {
    await _runAction(
      () => widget.authClient.signInWithEmailPassword(
        email: _emailController.text.trim(),
        password: _passwordController.text,
      ),
      successLabel: 'Signed in',
    );
  }

  Future<void> _signInWithGoogle() async {
    await _runAction(
      widget.authClient.signInWithGoogle,
      successLabel: 'Signed in',
    );
  }

  Future<void> _signOut() async {
    await _setBusy(true);
    try {
      await widget.authClient.signOut();
      if (!mounted) {
        return;
      }

      setState(() {
        _errorMessage = null;
        _statusMessage = 'Signed out.';
      });
    } catch (error) {
      if (!mounted) {
        return;
      }

      setState(() {
        _errorMessage = _errorText(error);
        _statusMessage = null;
      });
    } finally {
      await _setBusy(false);
    }
  }

  Future<void> _runAction(
    Future<CortadoFirebaseAuthResult> Function() action, {
    required String successLabel,
  }) async {
    await _setBusy(true);
    try {
      final result = await action();
      await widget.onAuthenticated?.call(result);
      if (!mounted) {
        return;
      }

      setState(() {
        _errorMessage = null;
        _statusMessage = '$successLabel as ${result.user.label}.';
      });
    } catch (error) {
      if (!mounted) {
        return;
      }

      setState(() {
        _errorMessage = _errorText(error);
        _statusMessage = null;
      });
    } finally {
      await _setBusy(false);
    }
  }

  Future<void> _setBusy(bool value) async {
    if (!mounted) {
      return;
    }

    setState(() {
      _busy = value;
    });
  }

  String _errorText(Object error) {
    if (error case final FirebaseAuthException authError
        when authError.message != null &&
            authError.message!.trim().isNotEmpty) {
      return authError.message!.trim();
    }

    final message = error.toString().trim();
    if (message.startsWith('Exception: ')) {
      return message.substring('Exception: '.length).trim();
    }
    if (message.startsWith('StateError: ')) {
      return message.substring('StateError: '.length).trim();
    }
    return message;
  }
}
