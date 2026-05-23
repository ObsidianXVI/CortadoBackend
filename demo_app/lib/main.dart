import 'dart:async';
import 'dart:math' as math;

import 'package:cortado/cortado.dart';
import 'package:flutter/material.dart';

import 'src/terminal_smoke_config.dart';

void main() {
  runApp(
      TerminalSmokeApp(initialConfig: TerminalSmokeConfig.fromUri(Uri.base)));
}

typedef CortadoClientFactory = CortadoClient Function(String baseUrl);

class TerminalSmokeApp extends StatelessWidget {
  const TerminalSmokeApp({
    super.key,
    required this.initialConfig,
    this.clientFactory = _defaultClientFactory,
  });

  final TerminalSmokeConfig initialConfig;
  final CortadoClientFactory clientFactory;

  static CortadoClient _defaultClientFactory(String baseUrl) =>
      CortadoClient(baseUrl: baseUrl);

  @override
  Widget build(BuildContext context) {
    const accent = Color(0xFFD97706);

    return MaterialApp(
      title: 'Cortado Terminal Smoke Test',
      debugShowCheckedModeBanner: false,
      theme: ThemeData(
        brightness: Brightness.dark,
        colorScheme: ColorScheme.fromSeed(
          seedColor: accent,
          brightness: Brightness.dark,
        ),
        scaffoldBackgroundColor: const Color(0xFF091018),
        useMaterial3: true,
      ),
      home: TerminalSmokeScreen(
        initialConfig: initialConfig,
        clientFactory: clientFactory,
      ),
    );
  }
}

class TerminalSmokeScreen extends StatefulWidget {
  const TerminalSmokeScreen({
    super.key,
    required this.initialConfig,
    required this.clientFactory,
  });

  final TerminalSmokeConfig initialConfig;
  final CortadoClientFactory clientFactory;

  @override
  State<TerminalSmokeScreen> createState() => _TerminalSmokeScreenState();
}

class _TerminalSmokeScreenState extends State<TerminalSmokeScreen> {
  static const double _minTerminalHeight = 240;
  static const double _maxTerminalHeight = 720;

  late final TextEditingController _baseUrlController =
      TextEditingController(text: widget.initialConfig.baseUrl);
  late final TextEditingController _workspaceIdController =
      TextEditingController(text: widget.initialConfig.workspaceId);
  late final TextEditingController _shellController =
      TextEditingController(text: widget.initialConfig.shell);

  StreamSubscription<Object>? _errorSubscription;
  CortadoClient? _client;
  bool _isConnecting = false;
  double _terminalHeight = 420;
  String _status = 'Idle';
  String? _detail;
  int _connectionAttempt = 0;

  bool get _isConnected => _client != null;

  @override
  void dispose() {
    _baseUrlController.dispose();
    _workspaceIdController.dispose();
    _shellController.dispose();
    unawaited(_disposeClient());
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final screenWidth = MediaQuery.sizeOf(context).width;
    final isCompact = screenWidth < 960;

    return Scaffold(
      body: DecoratedBox(
        decoration: const BoxDecoration(
          gradient: LinearGradient(
            colors: <Color>[
              Color(0xFF0B1520),
              Color(0xFF091018),
              Color(0xFF05080C),
            ],
            begin: Alignment.topLeft,
            end: Alignment.bottomRight,
          ),
        ),
        child: SafeArea(
          child: Center(
            child: ConstrainedBox(
              constraints: const BoxConstraints(maxWidth: 1240),
              child: ListView(
                padding: const EdgeInsets.all(24),
                children: <Widget>[
                  _buildHeader(context),
                  const SizedBox(height: 20),
                  if (isCompact)
                    Column(
                      crossAxisAlignment: CrossAxisAlignment.stretch,
                      children: <Widget>[
                        _buildControlsCard(context),
                        const SizedBox(height: 16),
                        _buildChecklistCard(context),
                      ],
                    )
                  else
                    Row(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: <Widget>[
                        Expanded(
                          flex: 8,
                          child: _buildControlsCard(context),
                        ),
                        const SizedBox(width: 16),
                        Expanded(
                          flex: 5,
                          child: _buildChecklistCard(context),
                        ),
                      ],
                    ),
                  const SizedBox(height: 16),
                  _buildTerminalCard(context),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }

  Widget _buildHeader(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: <Widget>[
        Text(
          'Cortado Terminal Smoke Test',
          style: Theme.of(context).textTheme.headlineMedium?.copyWith(
                fontWeight: FontWeight.w700,
                letterSpacing: -0.6,
              ),
        ),
        const SizedBox(height: 8),
        Text(
          'Use this page to verify the full terminal path from Flutter Web to '
          'the deployed control plane and workspace shell.',
          style: Theme.of(context).textTheme.bodyLarge?.copyWith(
                color: const Color(0xFFB7C0CC),
              ),
        ),
      ],
    );
  }

  Widget _buildControlsCard(BuildContext context) {
    final connectLabel = _isConnected ? 'Reconnect' : 'Connect';

    return _SurfaceCard(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: <Widget>[
          Text(
            'Connection',
            style: Theme.of(context).textTheme.titleLarge,
          ),
          const SizedBox(height: 16),
          Wrap(
            spacing: 12,
            runSpacing: 12,
            children: <Widget>[
              SizedBox(
                width: 420,
                child: TextField(
                  controller: _baseUrlController,
                  decoration: const InputDecoration(
                    labelText: 'Control plane base URL',
                    hintText: 'https://control-plane.example.run.app',
                  ),
                ),
              ),
              SizedBox(
                width: 280,
                child: TextField(
                  controller: _workspaceIdController,
                  decoration: const InputDecoration(
                    labelText: 'Workspace ID',
                    hintText: 'ws-123',
                  ),
                ),
              ),
              SizedBox(
                width: 220,
                child: TextField(
                  controller: _shellController,
                  decoration: const InputDecoration(
                    labelText: 'Shell',
                    hintText: '/bin/bash',
                  ),
                ),
              ),
            ],
          ),
          const SizedBox(height: 16),
          Wrap(
            spacing: 12,
            runSpacing: 12,
            crossAxisAlignment: WrapCrossAlignment.center,
            children: <Widget>[
              FilledButton.icon(
                onPressed: _isConnecting ? null : _connect,
                icon: _isConnecting
                    ? const SizedBox(
                        width: 16,
                        height: 16,
                        child: CircularProgressIndicator(strokeWidth: 2),
                      )
                    : const Icon(Icons.play_arrow_rounded),
                label: Text(connectLabel),
              ),
              OutlinedButton.icon(
                onPressed: _isConnected && !_isConnecting ? _disconnect : null,
                icon: const Icon(Icons.stop_circle_outlined),
                label: const Text('Disconnect'),
              ),
              Chip(
                label: Text(_status),
                side: BorderSide.none,
                backgroundColor: const Color(0xFF18212D),
              ),
            ],
          ),
          if (_detail != null) ...<Widget>[
            const SizedBox(height: 16),
            Text(
              _detail!,
              style: Theme.of(context).textTheme.bodyMedium?.copyWith(
                    color: const Color(0xFFB7C0CC),
                  ),
            ),
          ],
          const SizedBox(height: 16),
          Text(
            'Optional query parameters: '
            '`?baseUrl=https://...&workspaceId=ws-123&shell=/bin/bash`',
            style: Theme.of(context).textTheme.bodySmall?.copyWith(
                  color: const Color(0xFF8F99A8),
                ),
          ),
        ],
      ),
    );
  }

  Widget _buildChecklistCard(BuildContext context) {
    final textTheme = Theme.of(context).textTheme;

    return _SurfaceCard(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: <Widget>[
          Text('Manual checklist', style: textTheme.titleLarge),
          const SizedBox(height: 16),
          for (final item in const <String>[
            'Run `echo hello_v0_1` and confirm the echoed line renders cleanly.',
            'Run `vim` and verify the full-screen TUI redraw behaves correctly.',
            'Run `python3` and confirm an interactive REPL prompt works.',
            'Drag the terminal resize handle, then run `tput cols` to confirm width changes.',
            'Measure a keystroke round trip in Chrome DevTools WebSocket frames.',
          ])
            Padding(
              padding: const EdgeInsets.only(bottom: 10),
              child: Row(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: <Widget>[
                  const Padding(
                    padding: EdgeInsets.only(top: 3),
                    child: Icon(Icons.chevron_right_rounded, size: 18),
                  ),
                  const SizedBox(width: 8),
                  Expanded(
                    child: Text(
                      item,
                      style: textTheme.bodyMedium?.copyWith(
                        color: const Color(0xFFCDD6E2),
                      ),
                    ),
                  ),
                ],
              ),
            ),
          const SizedBox(height: 8),
          Text(
            'If latency is consistently above 200 ms from Singapore to '
            '`us-central1`, log it as a v0.1 limitation for later regional rollout.',
            style: textTheme.bodySmall?.copyWith(
              color: const Color(0xFF8F99A8),
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildTerminalCard(BuildContext context) {
    return _SurfaceCard(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: <Widget>[
          Row(
            children: <Widget>[
              Text('Terminal', style: Theme.of(context).textTheme.titleLarge),
              const Spacer(),
              Text(
                '${_terminalHeight.round()} px',
                style: Theme.of(context).textTheme.bodySmall?.copyWith(
                      color: const Color(0xFF8F99A8),
                    ),
              ),
            ],
          ),
          const SizedBox(height: 16),
          AnimatedContainer(
            duration: const Duration(milliseconds: 160),
            curve: Curves.easeOutCubic,
            height: _terminalHeight,
            decoration: BoxDecoration(
              borderRadius: BorderRadius.circular(20),
              border: Border.all(color: const Color(0xFF223042)),
              color: const Color(0xFF0B1118),
              boxShadow: const <BoxShadow>[
                BoxShadow(
                  blurRadius: 26,
                  color: Color(0x22000000),
                  offset: Offset(0, 18),
                ),
              ],
            ),
            child: ClipRRect(
              borderRadius: BorderRadius.circular(20),
              child: _isConnected && _client != null
                  ? CortadoTerminal(
                      client: _client!,
                      shell: _shellController.text.trim().isEmpty
                          ? TerminalSmokeConfig.defaultShell
                          : _shellController.text.trim(),
                      onClosed: (String message) {
                        if (!mounted) {
                          return;
                        }
                        setState(() {
                          _status = 'Terminal closed';
                          _detail = message;
                        });
                      },
                      onError: (String message) {
                        if (!mounted) {
                          return;
                        }
                        setState(() {
                          _status = 'Terminal error';
                          _detail = message;
                        });
                      },
                    )
                  : _buildEmptyTerminalState(context),
            ),
          ),
          const SizedBox(height: 12),
          GestureDetector(
            behavior: HitTestBehavior.opaque,
            onVerticalDragUpdate: (DragUpdateDetails details) {
              setState(() {
                _terminalHeight = _clampTerminalHeight(
                  _terminalHeight + details.delta.dy,
                );
              });
            },
            child: Center(
              child: Container(
                width: 124,
                height: 10,
                decoration: BoxDecoration(
                  borderRadius: BorderRadius.circular(999),
                  color: const Color(0xFF243345),
                ),
              ),
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildEmptyTerminalState(BuildContext context) {
    return Container(
      width: double.infinity,
      height: double.infinity,
      padding: const EdgeInsets.all(24),
      color: const Color(0xFF0B1118),
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: <Widget>[
          const Icon(Icons.terminal_rounded,
              size: 42, color: Color(0xFFD97706)),
          const SizedBox(height: 16),
          Text(
            'Connect to a workspace to start the smoke test.',
            style: Theme.of(context).textTheme.titleMedium,
          ),
          const SizedBox(height: 8),
          Text(
            'The terminal widget will open the configured shell as soon as the '
            'WebSocket connection succeeds.',
            textAlign: TextAlign.center,
            style: Theme.of(context).textTheme.bodyMedium?.copyWith(
                  color: const Color(0xFF8F99A8),
                ),
          ),
        ],
      ),
    );
  }

  Future<void> _connect() async {
    final baseUrl = _baseUrlController.text.trim();
    final workspaceId = _workspaceIdController.text.trim();

    if (baseUrl.isEmpty || workspaceId.isEmpty) {
      setState(() {
        _status = 'Missing input';
        _detail = 'Enter both a control-plane base URL and a workspace ID.';
      });
      return;
    }

    final shell = _shellController.text.trim().isEmpty
        ? TerminalSmokeConfig.defaultShell
        : _shellController.text.trim();
    final attempt = ++_connectionAttempt;
    final client = widget.clientFactory(baseUrl);
    final errorSubscription = client.errors.listen(_handleClientError);

    setState(() {
      _isConnecting = true;
      _status = 'Connecting';
      _detail = 'Opening $shell on workspace $workspaceId...';
    });

    try {
      await client.connect(workspaceId);

      if (!mounted || attempt != _connectionAttempt) {
        await errorSubscription.cancel();
        await client.dispose();
        return;
      }

      final previousClient = _client;
      final previousSubscription = _errorSubscription;

      setState(() {
        _client = client;
        _errorSubscription = errorSubscription;
        _status = 'Connected';
        _detail = 'Connected to $workspaceId via $baseUrl';
      });

      await previousSubscription?.cancel();
      await previousClient?.dispose();
    } catch (error) {
      await errorSubscription.cancel();
      await client.dispose();

      if (!mounted || attempt != _connectionAttempt) {
        return;
      }

      setState(() {
        _status = 'Connection failed';
        _detail = '$error';
      });
    } finally {
      if (mounted && attempt == _connectionAttempt) {
        setState(() {
          _isConnecting = false;
        });
      }
    }
  }

  Future<void> _disconnect() async {
    _connectionAttempt++;
    await _disposeClient();

    if (!mounted) {
      return;
    }

    setState(() {
      _status = 'Disconnected';
      _detail = 'Connection closed.';
      _isConnecting = false;
    });
  }

  void _handleClientError(Object error) {
    if (!mounted) {
      return;
    }

    setState(() {
      _status = 'Connection error';
      _detail = '$error';
    });
  }

  Future<void> _disposeClient() async {
    final subscription = _errorSubscription;
    final client = _client;

    _errorSubscription = null;
    _client = null;

    await subscription?.cancel();
    await client?.dispose();
  }

  double _clampTerminalHeight(double value) {
    return math.max(_minTerminalHeight, math.min(_maxTerminalHeight, value));
  }
}

class _SurfaceCard extends StatelessWidget {
  const _SurfaceCard({required this.child});

  final Widget child;

  @override
  Widget build(BuildContext context) {
    return DecoratedBox(
      decoration: BoxDecoration(
        borderRadius: BorderRadius.circular(28),
        border: Border.all(color: const Color(0xFF18222D)),
        color: const Color(0xCC0F1620),
      ),
      child: Padding(
        padding: const EdgeInsets.all(20),
        child: child,
      ),
    );
  }
}
