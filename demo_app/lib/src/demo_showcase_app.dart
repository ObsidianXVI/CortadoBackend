import 'dart:async';
import 'dart:convert' show utf8;

import 'package:code_forge_web/code_forge_web.dart' as code_forge;
import 'package:cortado/cortado.dart';
import 'package:firebase_auth/firebase_auth.dart';
import 'package:flutter/material.dart';
import 'package:flutter_code_editor/flutter_code_editor.dart'
    as flutter_code_editor;
import 'package:flutter_highlight/themes/monokai-sublime.dart';
import 'package:flutter_monaco/flutter_monaco.dart';
import 'package:highlight/languages/dart.dart' as highlight_dart;
import 'package:lite_code_editor/lite_code_editor.dart' as lite_code_editor;
import 'package:re_highlight/languages/dart.dart' as re_highlight_dart;

import 'demo_bootstrap_config.dart';
import 'demo_firebase_bootstrap.dart';

enum DemoEditorPackage {
  flutterCodeEditor,
  flutterMonaco,
  codeForge,
  liteCodeEditor,
}

enum DemoSessionMode {
  firebaseExchange,
  personalApiKey,
  platformApiKey,
}

extension DemoEditorPackagePresentation on DemoEditorPackage {
  String get label => switch (this) {
        DemoEditorPackage.flutterCodeEditor => 'Flutter Code Editor',
        DemoEditorPackage.flutterMonaco => 'Flutter Monaco',
        DemoEditorPackage.codeForge => 'CodeForge Web',
        DemoEditorPackage.liteCodeEditor => 'Lite Code Editor',
      };

  String get packageName => switch (this) {
        DemoEditorPackage.flutterCodeEditor => 'flutter_code_editor',
        DemoEditorPackage.flutterMonaco => 'flutter_monaco',
        DemoEditorPackage.codeForge => 'code_forge_web',
        DemoEditorPackage.liteCodeEditor => 'lite_code_editor',
      };

  List<String> get notes => switch (this) {
        DemoEditorPackage.flutterCodeEditor => const <String>[
            'Pure-Flutter editor with syntax highlighting, folding, gutters, and local analyzers.',
            'Good fit for demonstrating file load/save, keyboard editing, and lightweight embedded IDE UX.',
            'This page keeps the package close to its default CodeField flow and uses the demo shell for backend file writes.',
          ],
        DemoEditorPackage.flutterMonaco => const <String>[
            'Wraps Monaco, the editor engine behind VS Code, with strong language coverage and Monaco-native chrome.',
            'Best package page for showing a familiar IDE editing surface backed by Cortado workspace files.',
            'This demo keeps Monaco focused on content editing while Cortado handles session, workspace, terminal, and file persistence.',
          ],
        DemoEditorPackage.codeForge => const <String>[
            'The upstream code_forge package does not support Flutter Web, so this page uses its official web companion package.',
            'Strong option for demonstrating richer browser-editor behavior like gutters, folding, and future LSP expansion.',
            'This page is the closest CodeForge-family experience available for the required Flutter Web demo app.',
          ],
        DemoEditorPackage.liteCodeEditor => const <String>[
            'Smallest footprint of the four editors with zero external dependencies and a simple Dart-first feature set.',
            'Useful for showing the low-overhead path: open lib/main.dart, edit, save to the workspace, and verify in terminal.',
            'Good contrast against the heavier Monaco and CodeForge experiences.',
          ],
      };
}

extension DemoSessionModePresentation on DemoSessionMode {
  String get label => switch (this) {
        DemoSessionMode.firebaseExchange => 'First-party Firebase session',
        DemoSessionMode.personalApiKey => 'Personal API key session',
        DemoSessionMode.platformApiKey => 'Platform API key session',
      };

  bool get isUserScoped => switch (this) {
        DemoSessionMode.platformApiKey => false,
        DemoSessionMode.firebaseExchange ||
        DemoSessionMode.personalApiKey =>
          true,
      };
}

class CortadoDemoShowcaseApp extends StatelessWidget {
  const CortadoDemoShowcaseApp({
    super.key,
    required this.initialConfig,
  });

  final DemoBootstrapConfig initialConfig;

  @override
  Widget build(BuildContext context) {
    const accent = Color(0xFFD97706);

    return MaterialApp(
      title: 'Cortado Package Showcase',
      debugShowCheckedModeBanner: false,
      theme: ThemeData(
        brightness: Brightness.dark,
        colorScheme: ColorScheme.fromSeed(
          seedColor: accent,
          brightness: Brightness.dark,
        ),
        scaffoldBackgroundColor: const Color(0xFF071018),
        inputDecorationTheme: const InputDecorationTheme(
          filled: true,
          fillColor: Color(0xFF131D28),
          border: OutlineInputBorder(),
        ),
        useMaterial3: true,
      ),
      home: DemoShowcaseScreen(initialConfig: initialConfig),
    );
  }
}

class DemoShowcaseScreen extends StatefulWidget {
  const DemoShowcaseScreen({
    super.key,
    required this.initialConfig,
  });

  final DemoBootstrapConfig initialConfig;

  @override
  State<DemoShowcaseScreen> createState() => _DemoShowcaseScreenState();
}

class _DemoShowcaseScreenState extends State<DemoShowcaseScreen> {
  static const List<String> _bootstrapCommands = <String>[
    'apt-get update',
    'apt-get install -y curl git unzip xz-utils zip libglu1-mesa',
    'git clone https://github.com/flutter/flutter.git -b stable /opt/flutter',
    'export PATH="/opt/flutter/bin:\$PATH"',
    'flutter doctor',
    'flutter create --platforms=web .',
  ];

  late final TextEditingController _baseUrlController =
      TextEditingController(text: widget.initialConfig.baseUrl);
  late final TextEditingController _firebaseEmailController =
      TextEditingController(text: widget.initialConfig.firebaseEmail);
  late final TextEditingController _firebasePasswordController =
      TextEditingController(text: widget.initialConfig.firebasePassword);
  late final TextEditingController _apiKeyController =
      TextEditingController(text: widget.initialConfig.apiKey);
  late final TextEditingController _userIdController =
      TextEditingController(text: widget.initialConfig.userId);
  late final TextEditingController _platformTenantDisplayNameController =
      TextEditingController(text: 'Acme IDE');
  late final TextEditingController _platformTenantIdController =
      TextEditingController();
  late final TextEditingController _workspaceIdController =
      TextEditingController(text: widget.initialConfig.workspaceId);
  late final TextEditingController _imageController =
      TextEditingController(text: widget.initialConfig.image);
  late final TextEditingController _shellController =
      TextEditingController(text: widget.initialConfig.shell);
  late final TextEditingController _filePathController =
      TextEditingController(text: widget.initialConfig.filePath);
  late final TextEditingController _cpuController =
      TextEditingController(text: widget.initialConfig.cpu.toString());
  late final TextEditingController _memoryController =
      TextEditingController(text: widget.initialConfig.memoryGb.toString());
  late final TextEditingController _storageController =
      TextEditingController(text: widget.initialConfig.storageGb.toString());

  CortadoAuthSession? _authSession;
  WorkspaceManager? _workspaceManager;
  CortadoClient? _client;
  StreamSubscription<WorkspaceStatus>? _statusSubscription;
  late final DemoFirebaseBootstrap _firebaseBootstrap =
      DemoFirebaseBootstrap(widget.initialConfig);

  DemoEditorPackage _selectedPackage = DemoEditorPackage.flutterCodeEditor;
  User? _firebaseUser;
  DemoTenantAssignment? _tenantAssignment;
  DemoIssuedApiKey? _issuedApiKey;
  DemoIssuedApiKey? _issuedPlatformApiKey;
  List<DemoApiKeyRecord> _issuedApiKeys = const <DemoApiKeyRecord>[];
  List<DemoApiKeyRecord> _platformApiKeys = const <DemoApiKeyRecord>[];
  List<DemoPlatformTenant> _platformTenants = const <DemoPlatformTenant>[];
  Workspace? _workspace;
  WorkspaceStatus? _workspaceStatus;
  String _draftCode = '';
  String _loadedFilePath = '';
  String? _busyLabel;
  String? _connectedWorkspaceId;
  String? _infoMessage;
  DemoSessionMode? _sessionMode;
  int _documentRevision = 0;

  bool get _isBusy => _busyLabel != null;
  bool get _hasUserScopedSession =>
      _authSession != null && (_sessionMode?.isUserScoped ?? false);
  String get _workspaceId => _workspaceIdController.text.trim();
  String get _attachedWorkspaceId => _workspace?.id.trim() ?? '';
  String get _activeWorkspaceId {
    final attached = _attachedWorkspaceId;
    if (attached.isNotEmpty) {
      return attached;
    }
    return _workspaceId;
  }

  String get _filePath => _filePathController.text.trim();
  bool get _hasLoadedFile => _loadedFilePath.isNotEmpty;
  @override
  void initState() {
    super.initState();

    if (_workspaceId.isNotEmpty) {
      _infoMessage =
          'Workspace ID prefilled. Authenticate, refresh status, then load $_filePath.';
    }
  }

  @override
  void dispose() {
    _baseUrlController.dispose();
    _firebaseEmailController.dispose();
    _firebasePasswordController.dispose();
    _apiKeyController.dispose();
    _userIdController.dispose();
    _platformTenantDisplayNameController.dispose();
    _platformTenantIdController.dispose();
    _workspaceIdController.dispose();
    _imageController.dispose();
    _shellController.dispose();
    _filePathController.dispose();
    _cpuController.dispose();
    _memoryController.dispose();
    _storageController.dispose();
    _statusSubscription?.cancel();
    unawaited(_client?.dispose() ?? Future<void>.value());
    unawaited(_workspaceManager?.dispose() ?? Future<void>.value());
    unawaited(_authSession?.dispose() ?? Future<void>.value());
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final width = MediaQuery.sizeOf(context).width;
    final stacked = width < 1080;

    return Scaffold(
      body: DecoratedBox(
        decoration: const BoxDecoration(
          gradient: LinearGradient(
            colors: <Color>[
              Color(0xFF081119),
              Color(0xFF0B1520),
              Color(0xFF04070B),
            ],
            begin: Alignment.topLeft,
            end: Alignment.bottomRight,
          ),
        ),
        child: SafeArea(
          child: Center(
            child: ConstrainedBox(
              constraints: const BoxConstraints(maxWidth: 1440),
              child: ListView(
                padding: const EdgeInsets.all(24),
                children: <Widget>[
                  _buildHeader(context),
                  const SizedBox(height: 20),
                  _buildIdentityCard(context),
                  const SizedBox(height: 16),
                  _buildWorkspaceCard(context),
                  const SizedBox(height: 16),
                  if (stacked)
                    Column(
                      children: <Widget>[
                        _buildPackageCard(context),
                        const SizedBox(height: 16),
                        _buildTerminalCard(context),
                      ],
                    )
                  else
                    Row(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: <Widget>[
                        Expanded(
                          flex: 8,
                          child: _buildPackageCard(context),
                        ),
                        const SizedBox(width: 16),
                        Expanded(
                          flex: 5,
                          child: _buildTerminalCard(context),
                        ),
                      ],
                    ),
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
          'Cortado Package Showcase',
          style: Theme.of(context).textTheme.headlineMedium?.copyWith(
                fontWeight: FontWeight.w800,
                letterSpacing: -0.8,
              ),
        ),
        const SizedBox(height: 8),
        Text(
          'Authenticate with a real session, provision a real Ubuntu workspace, '
          'bootstrap Flutter from the terminal, then open and edit the same '
          'workspace file through four editor packages.',
          style: Theme.of(context).textTheme.bodyLarge?.copyWith(
                color: const Color(0xFFB8C4D2),
              ),
        ),
      ],
    );
  }

  Widget _buildIdentityCard(BuildContext context) {
    final bootstrapReady = widget.initialConfig.hasFirebaseBootstrapConfig;
    final userLabel = _firebaseUser?.email?.trim().isNotEmpty == true
        ? _firebaseUser!.email!
        : (_firebaseUser?.uid ?? 'No Firebase user signed in');

    return _SurfaceCard(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: <Widget>[
          Text('Identity Bootstrap',
              style: Theme.of(context).textTheme.titleLarge),
          const SizedBox(height: 8),
          Text(
            'Register or sign in with Firebase email/password, then either exchange directly into a Cortado session or mint personal and platform API keys from the control plane.',
            style: Theme.of(context).textTheme.bodyMedium?.copyWith(
                  color: const Color(0xFFB8C4D2),
                ),
          ),
          const SizedBox(height: 8),
          Text(
            'In development, the app can assign the `tenant_id` custom claim for you before minting a Cortado API key.',
            style: Theme.of(context).textTheme.bodySmall?.copyWith(
                  color: const Color(0xFF8898AA),
                ),
          ),
          const SizedBox(height: 16),
          if (bootstrapReady)
            Wrap(
              spacing: 12,
              runSpacing: 12,
              children: <Widget>[
                _buildField(_firebaseEmailController, 'Firebase Email', 280),
                _buildField(
                  _firebasePasswordController,
                  'Firebase Password',
                  220,
                  obscure: true,
                ),
              ],
            )
          else
            DecoratedBox(
              decoration: BoxDecoration(
                color: const Color(0xFF121B25),
                borderRadius: BorderRadius.circular(14),
                border: Border.all(color: const Color(0xFF243343)),
              ),
              child: const Padding(
                padding: EdgeInsets.all(14),
                child: Text(
                  'Firebase bootstrap is disabled. Add the Firebase Web config values to demo_app/.env to enable register/login and in-app API key minting.',
                ),
              ),
            ),
          const SizedBox(height: 16),
          Wrap(
            spacing: 12,
            runSpacing: 12,
            children: <Widget>[
              FilledButton.icon(
                onPressed:
                    !bootstrapReady || _isBusy ? null : _registerFirebaseUser,
                icon: const Icon(Icons.person_add_alt_1_rounded),
                label: const Text('Register User'),
              ),
              FilledButton.icon(
                onPressed:
                    !bootstrapReady || _isBusy ? null : _loginFirebaseUser,
                icon: const Icon(Icons.login_rounded),
                label: const Text('Login'),
              ),
              OutlinedButton.icon(
                onPressed: !bootstrapReady || _isBusy || _firebaseUser == null
                    ? null
                    : _signOutFirebaseUser,
                icon: const Icon(Icons.logout_rounded),
                label: const Text('Sign Out'),
              ),
              FilledButton.icon(
                onPressed: !bootstrapReady || _isBusy || _firebaseUser == null
                    ? null
                    : _exchangeFirebaseSession,
                icon: const Icon(Icons.swap_horiz_rounded),
                label: const Text('Exchange Session'),
              ),
              FilledButton.icon(
                onPressed: !bootstrapReady || _isBusy || _firebaseUser == null
                    ? null
                    : _assignDevelopmentTenant,
                icon: const Icon(Icons.verified_user_outlined),
                label: const Text('Assign Dev Tenant'),
              ),
              FilledButton.icon(
                onPressed: !bootstrapReady || _isBusy || _firebaseUser == null
                    ? null
                    : _mintApiKey,
                icon: const Icon(Icons.vpn_key_outlined),
                label: const Text('Mint Personal Key'),
              ),
              OutlinedButton.icon(
                onPressed: !bootstrapReady || _isBusy || _firebaseUser == null
                    ? null
                    : _refreshIssuedApiKeys,
                icon: const Icon(Icons.key_rounded),
                label: const Text('List Personal Keys'),
              ),
            ],
          ),
          const SizedBox(height: 16),
          Wrap(
            spacing: 12,
            runSpacing: 12,
            crossAxisAlignment: WrapCrossAlignment.center,
            children: <Widget>[
              Chip(
                  label: Text(bootstrapReady
                      ? 'Firebase ${widget.initialConfig.firebaseProjectId}'
                      : 'Firebase bootstrap disabled')),
              if (widget.initialConfig.firebaseDevTenantId.isNotEmpty)
                Chip(
                  label: Text(
                    'Dev tenant ${widget.initialConfig.firebaseDevTenantId}',
                  ),
                ),
              Chip(label: Text(userLabel)),
              if (_firebaseUser != null)
                Chip(label: Text('UID ${_firebaseUser!.uid}')),
              if (_tenantAssignment != null)
                Chip(
                  label: Text('Assigned ${_tenantAssignment!.tenantId}'),
                ),
              if (_sessionMode != null) Chip(label: Text(_sessionMode!.label)),
            ],
          ),
          if (_issuedApiKey != null) ...<Widget>[
            const SizedBox(height: 16),
            Text(
              'Latest Minted API Key',
              style: Theme.of(context).textTheme.titleMedium,
            ),
            const SizedBox(height: 10),
            Container(
              width: double.infinity,
              padding: const EdgeInsets.all(16),
              decoration: BoxDecoration(
                color: const Color(0xFF091119),
                borderRadius: BorderRadius.circular(16),
                border: Border.all(color: const Color(0xFF213042)),
              ),
              child: SelectionArea(
                child: Text(
                  _issuedApiKey!.apiKey,
                  style: const TextStyle(
                    fontFamily: 'monospace',
                    fontSize: 13,
                    height: 1.5,
                  ),
                ),
              ),
            ),
          ],
          if (_issuedApiKeys.isNotEmpty) ...<Widget>[
            const SizedBox(height: 16),
            Text(
              'Personal API Keys',
              style: Theme.of(context).textTheme.titleMedium,
            ),
            const SizedBox(height: 12),
            for (final apiKey in _issuedApiKeys.take(4)) ...<Widget>[
              ListTile(
                dense: true,
                contentPadding: EdgeInsets.zero,
                title: Text(apiKey.id),
                subtitle: Text(
                  '${apiKey.kind.isEmpty ? 'personal' : apiKey.kind} · '
                  '${apiKey.tenantId} · ${apiKey.userId} · '
                  '${apiKey.createdAt?.toIso8601String() ?? 'unknown time'}',
                ),
                trailing: Chip(
                  label: Text(apiKey.revoked ? 'Revoked' : 'Active'),
                ),
              ),
            ],
          ],
          const SizedBox(height: 20),
          _buildPlatformCard(context),
        ],
      ),
    );
  }

  Widget _buildPlatformCard(BuildContext context) {
    final platformReady = _hasUserScopedSession;

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: <Widget>[
        Text(
          'Platform Backend Flow',
          style: Theme.of(context).textTheme.titleMedium,
        ),
        const SizedBox(height: 8),
        Text(
          'Use a normal Cortado user session to create a platform tenant, mint a platform API key, then switch the session form below to a platform-scoped backend credential.',
          style: Theme.of(context).textTheme.bodyMedium?.copyWith(
                color: const Color(0xFFB8C4D2),
              ),
        ),
        const SizedBox(height: 8),
        Text(
          platformReady
              ? 'Current session can manage platform tenants.'
              : 'Exchange a Firebase session or create a personal API-key session first. Platform API-key sessions cannot manage platform tenants.',
          style: Theme.of(context).textTheme.bodySmall?.copyWith(
                color: const Color(0xFF8898AA),
              ),
        ),
        const SizedBox(height: 16),
        Wrap(
          spacing: 12,
          runSpacing: 12,
          children: <Widget>[
            _buildField(
              _platformTenantDisplayNameController,
              'Platform Tenant Name',
              260,
            ),
            _buildField(
              _platformTenantIdController,
              'Platform Tenant ID',
              260,
            ),
          ],
        ),
        const SizedBox(height: 16),
        Wrap(
          spacing: 12,
          runSpacing: 12,
          children: <Widget>[
            FilledButton.icon(
              onPressed:
                  !platformReady || _isBusy ? null : _createPlatformTenant,
              icon: const Icon(Icons.apartment_rounded),
              label: const Text('Create Platform Tenant'),
            ),
            OutlinedButton.icon(
              onPressed:
                  !platformReady || _isBusy ? null : _refreshPlatformTenants,
              icon: const Icon(Icons.domain_verification_outlined),
              label: const Text('List Platform Tenants'),
            ),
            FilledButton.icon(
              onPressed: !platformReady || _isBusy ? null : _mintPlatformApiKey,
              icon: const Icon(Icons.key_rounded),
              label: const Text('Mint Platform Key'),
            ),
            OutlinedButton.icon(
              onPressed:
                  !platformReady || _isBusy ? null : _refreshPlatformApiKeys,
              icon: const Icon(Icons.key_off_outlined),
              label: const Text('List Platform Keys'),
            ),
          ],
        ),
        if (_issuedPlatformApiKey != null) ...<Widget>[
          const SizedBox(height: 16),
          Text(
            'Latest Platform API Key',
            style: Theme.of(context).textTheme.titleMedium,
          ),
          const SizedBox(height: 10),
          Container(
            width: double.infinity,
            padding: const EdgeInsets.all(16),
            decoration: BoxDecoration(
              color: const Color(0xFF091119),
              borderRadius: BorderRadius.circular(16),
              border: Border.all(color: const Color(0xFF213042)),
            ),
            child: SelectionArea(
              child: Text(
                _issuedPlatformApiKey!.apiKey,
                style: const TextStyle(
                  fontFamily: 'monospace',
                  fontSize: 13,
                  height: 1.5,
                ),
              ),
            ),
          ),
        ],
        if (_platformTenants.isNotEmpty) ...<Widget>[
          const SizedBox(height: 16),
          Text(
            'Platform Tenants',
            style: Theme.of(context).textTheme.titleMedium,
          ),
          const SizedBox(height: 12),
          for (final tenant in _platformTenants.take(4)) ...<Widget>[
            ListTile(
              dense: true,
              contentPadding: EdgeInsets.zero,
              title: Text(
                tenant.displayName.isEmpty
                    ? tenant.tenantId
                    : tenant.displayName,
              ),
              subtitle: Text(
                '${tenant.kind} · ${tenant.tenantId} · '
                '${tenant.createdAt?.toIso8601String() ?? 'unknown time'}',
              ),
              trailing: OutlinedButton(
                onPressed: _isBusy
                    ? null
                    : () {
                        _platformTenantIdController.text = tenant.tenantId;
                        _setInfoMessage(
                          'Selected platform tenant ${tenant.tenantId}.',
                        );
                      },
                child: const Text('Use Tenant'),
              ),
            ),
          ],
        ],
        if (_platformApiKeys.isNotEmpty) ...<Widget>[
          const SizedBox(height: 16),
          Text(
            'Platform API Keys',
            style: Theme.of(context).textTheme.titleMedium,
          ),
          const SizedBox(height: 12),
          for (final apiKey in _platformApiKeys.take(4)) ...<Widget>[
            ListTile(
              dense: true,
              contentPadding: EdgeInsets.zero,
              title: Text(apiKey.id),
              subtitle: Text(
                '${apiKey.kind.isEmpty ? 'platform' : apiKey.kind} · '
                '${apiKey.tenantId} · '
                '${apiKey.createdAt?.toIso8601String() ?? 'unknown time'}',
              ),
              trailing: Chip(
                label: Text(apiKey.revoked ? 'Revoked' : 'Active'),
              ),
            ),
          ],
        ],
      ],
    );
  }

  Widget _buildWorkspaceCard(BuildContext context) {
    return _SurfaceCard(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: <Widget>[
          Text(
            'Session + Workspace',
            style: Theme.of(context).textTheme.titleLarge,
          ),
          const SizedBox(height: 16),
          Wrap(
            spacing: 12,
            runSpacing: 12,
            children: <Widget>[
              _buildField(_baseUrlController, 'Control Plane Base URL', 360),
              _buildField(_apiKeyController, 'Demo API Key', 280,
                  obscure: true),
              _buildField(
                  _userIdController, 'Demo User ID (personal only)', 220),
              _buildField(_workspaceIdController, 'Workspace ID', 220),
              _buildField(_imageController, 'Workspace Image', 260),
              _buildField(_cpuController, 'CPU', 120),
              _buildField(_memoryController, 'Memory (GB)', 120),
              _buildField(_storageController, 'Storage (GB)', 120),
              _buildField(_shellController, 'Shell', 180),
              _buildField(_filePathController, 'Target File', 220),
            ],
          ),
          const SizedBox(height: 10),
          Text(
            'Leave Demo User ID empty when the API key came from the platform-tenant flow.',
            style: Theme.of(context).textTheme.bodySmall?.copyWith(
                  color: const Color(0xFF8898AA),
                ),
          ),
          const SizedBox(height: 16),
          Wrap(
            spacing: 12,
            runSpacing: 12,
            children: <Widget>[
              FilledButton.icon(
                onPressed: _isBusy ? null : _authenticate,
                icon: const Icon(Icons.key_rounded),
                label: const Text('New Session'),
              ),
              FilledButton.icon(
                onPressed: _isBusy ? null : _createWorkspace,
                icon: const Icon(Icons.cloud_upload_outlined),
                label: const Text('Provision Workspace'),
              ),
              OutlinedButton.icon(
                onPressed: _isBusy ? null : _refreshWorkspace,
                icon: const Icon(Icons.refresh_rounded),
                label: const Text('Refresh Status'),
              ),
              OutlinedButton.icon(
                onPressed: _isBusy ? null : _startWorkspace,
                icon: const Icon(Icons.play_arrow_rounded),
                label: const Text('Start'),
              ),
              OutlinedButton.icon(
                onPressed: _isBusy ? null : _stopWorkspace,
                icon: const Icon(Icons.pause_circle_outline_rounded),
                label: const Text('Stop'),
              ),
              OutlinedButton.icon(
                onPressed: _isBusy ? null : _deleteWorkspace,
                icon: const Icon(Icons.delete_outline_rounded),
                label: const Text('Delete'),
              ),
              OutlinedButton.icon(
                onPressed: _isBusy ? null : _loadFile,
                icon: const Icon(Icons.file_open_outlined),
                label: const Text('Load File'),
              ),
              FilledButton.icon(
                onPressed: _isBusy || !_hasLoadedFile ? null : _saveFile,
                icon: const Icon(Icons.save_outlined),
                label: const Text('Save File'),
              ),
            ],
          ),
          const SizedBox(height: 16),
          Wrap(
            spacing: 12,
            runSpacing: 12,
            crossAxisAlignment: WrapCrossAlignment.center,
            children: <Widget>[
              _StatusChip(
                label: _workspaceStatus == null
                    ? 'No workspace attached'
                    : _workspaceStatus!.status.name.toUpperCase(),
              ),
              if (_busyLabel != null)
                Chip(
                  avatar: const SizedBox(
                    width: 14,
                    height: 14,
                    child: CircularProgressIndicator(strokeWidth: 2),
                  ),
                  label: Text(_busyLabel!),
                ),
              if (_workspace != null)
                Chip(
                  label: Text(
                    'Resources ${_workspace!.resources.cpu.toStringAsFixed(1)} CPU / '
                    '${_workspace!.resources.memoryGb.toStringAsFixed(1)} GB RAM / '
                    '${_workspace!.resources.storageGb.toStringAsFixed(1)} GB disk',
                  ),
                ),
              if (_sessionMode != null) Chip(label: Text(_sessionMode!.label)),
              if (_connectedWorkspaceId == _workspaceId &&
                  _workspaceId.isNotEmpty)
                const Chip(label: Text('Terminal socket attached')),
            ],
          ),
          if (_infoMessage != null) ...<Widget>[
            const SizedBox(height: 14),
            DecoratedBox(
              decoration: BoxDecoration(
                color: const Color(0xFF121B25),
                borderRadius: BorderRadius.circular(14),
                border: Border.all(color: const Color(0xFF243343)),
              ),
              child: Padding(
                padding: const EdgeInsets.all(14),
                child: Text(
                  _infoMessage!,
                  style: Theme.of(context).textTheme.bodyMedium?.copyWith(
                        color: const Color(0xFFD6DEE8),
                      ),
                ),
              ),
            ),
          ],
          const SizedBox(height: 16),
          _buildBootstrapCommands(context),
        ],
      ),
    );
  }

  Widget _buildBootstrapCommands(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: <Widget>[
        Text(
          'Bootstrap Flow',
          style: Theme.of(context).textTheme.titleMedium,
        ),
        const SizedBox(height: 10),
        Text(
          'After the workspace reaches RUNNING, use the shared terminal to install Flutter and generate the demo project at workspace root.',
          style: Theme.of(context).textTheme.bodyMedium?.copyWith(
                color: const Color(0xFFB8C4D2),
              ),
        ),
        const SizedBox(height: 12),
        Container(
          width: double.infinity,
          padding: const EdgeInsets.all(16),
          decoration: BoxDecoration(
            color: const Color(0xFF091119),
            borderRadius: BorderRadius.circular(16),
            border: Border.all(color: const Color(0xFF213042)),
          ),
          child: SelectionArea(
            child: Text(
              _bootstrapCommands.join('\n'),
              style: const TextStyle(
                fontFamily: 'monospace',
                fontSize: 13,
                height: 1.5,
              ),
            ),
          ),
        ),
      ],
    );
  }

  Widget _buildPackageCard(BuildContext context) {
    return _SurfaceCard(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: <Widget>[
          Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: <Widget>[
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: <Widget>[
                    Text(
                      'Editor Packages',
                      style: Theme.of(context).textTheme.titleLarge,
                    ),
                    const SizedBox(height: 6),
                    Text(
                      'Each page below edits the same workspace file using a different package surface.',
                      style: Theme.of(context).textTheme.bodyMedium?.copyWith(
                            color: const Color(0xFFB8C4D2),
                          ),
                    ),
                  ],
                ),
              ),
            ],
          ),
          const SizedBox(height: 16),
          SingleChildScrollView(
            scrollDirection: Axis.horizontal,
            child: SegmentedButton<DemoEditorPackage>(
              segments: DemoEditorPackage.values
                  .map(
                    (pkg) => ButtonSegment<DemoEditorPackage>(
                      value: pkg,
                      label: Text(pkg.label),
                    ),
                  )
                  .toList(growable: false),
              selected: <DemoEditorPackage>{_selectedPackage},
              onSelectionChanged: (selection) {
                setState(() {
                  _selectedPackage = selection.first;
                });
              },
              showSelectedIcon: false,
            ),
          ),
          const SizedBox(height: 16),
          _EditorExperienceCard(
            package: _selectedPackage,
            loadedFilePath: _loadedFilePath,
            hasLoadedFile: _hasLoadedFile,
            draftCode: _draftCode,
            documentKey:
                '${_selectedPackage.name}-${_loadedFilePath.isEmpty ? 'empty' : _loadedFilePath}-$_documentRevision',
            onChanged: (value) {
              _draftCode = value;
            },
          ),
        ],
      ),
    );
  }

  Widget _buildTerminalCard(BuildContext context) {
    final canShowTerminal = _client != null &&
        _workspaceManager != null &&
        _workspaceId.isNotEmpty &&
        _connectedWorkspaceId == _workspaceId;

    return _SurfaceCard(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: <Widget>[
          Text('Terminal', style: Theme.of(context).textTheme.titleLarge),
          const SizedBox(height: 8),
          Text(
            'Shared Cortado terminal for provisioning the Ubuntu workspace, installing Flutter, and verifying edits from every package page.',
            style: Theme.of(context).textTheme.bodyMedium?.copyWith(
                  color: const Color(0xFFB8C4D2),
                ),
          ),
          const SizedBox(height: 16),
          SizedBox(
            height: 640,
            child: DecoratedBox(
              decoration: BoxDecoration(
                color: const Color(0xFF071018),
                borderRadius: BorderRadius.circular(16),
                border: Border.all(color: const Color(0xFF253443)),
              ),
              child: ClipRRect(
                borderRadius: BorderRadius.circular(16),
                child: canShowTerminal
                    ? CortadoTerminal(
                        key:
                            ValueKey<String>('terminal-$_connectedWorkspaceId'),
                        client: _client!,
                        workspaceManager: _workspaceManager!,
                        workspaceId: _workspaceId,
                        shell: _shellController.text.trim().isEmpty
                            ? DemoBootstrapConfig.defaultShell
                            : _shellController.text.trim(),
                        onError: _setInfoMessage,
                        onClosed: _setInfoMessage,
                      )
                    : Center(
                        child: Padding(
                          padding: const EdgeInsets.all(24),
                          child: Text(
                            'Authenticate, provision or attach a workspace, wait until it is RUNNING, then refresh status to attach the terminal socket.',
                            style:
                                Theme.of(context).textTheme.bodyLarge?.copyWith(
                                      color: const Color(0xFFB8C4D2),
                                    ),
                            textAlign: TextAlign.center,
                          ),
                        ),
                      ),
              ),
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildField(
    TextEditingController controller,
    String label,
    double width, {
    bool obscure = false,
  }) {
    return SizedBox(
      width: width,
      child: TextField(
        controller: controller,
        obscureText: obscure,
        decoration: InputDecoration(labelText: label),
      ),
    );
  }

  Future<void> _registerFirebaseUser() async {
    await _runBusy('Registering Firebase user', () async {
      final credential = await _firebaseBootstrap.register(
        email: _firebaseEmailController.text.trim(),
        password: _firebasePasswordController.text,
      );
      await _afterFirebaseCredential(
        credential,
        successMessage:
            'Registered ${credential.user?.email ?? credential.user?.uid ?? 'Firebase user'}.',
      );
    });
  }

  Future<void> _loginFirebaseUser() async {
    await _runBusy('Logging into Firebase', () async {
      final credential = await _firebaseBootstrap.login(
        email: _firebaseEmailController.text.trim(),
        password: _firebasePasswordController.text,
      );
      await _afterFirebaseCredential(
        credential,
        successMessage:
            'Logged in as ${credential.user?.email ?? credential.user?.uid ?? 'Firebase user'}.',
      );
    });
  }

  Future<void> _signOutFirebaseUser() async {
    await _runBusy('Signing out', () async {
      await _firebaseBootstrap.signOut();
      if (!mounted) {
        return;
      }

      setState(() {
        _firebaseUser = null;
        _tenantAssignment = null;
        _issuedApiKey = null;
        _issuedApiKeys = const <DemoApiKeyRecord>[];
        _issuedPlatformApiKey = null;
        _platformApiKeys = const <DemoApiKeyRecord>[];
        _platformTenants = const <DemoPlatformTenant>[];
      });
      _setInfoMessage('Firebase user signed out.');
    });
  }

  Future<void> _exchangeFirebaseSession() async {
    await _runBusy('Exchanging Firebase session', () async {
      final baseUrl = _baseUrlController.text.trim();
      if (baseUrl.isEmpty) {
        throw StateError('Base URL is required.');
      }

      final firebaseIdToken = await _firebaseBootstrap.currentIdToken();
      await _disposeTransports();

      final session = CortadoAuthSession(baseUrl: baseUrl);
      await session.exchangeFirebaseSession(firebaseIdToken: firebaseIdToken);
      await _bindSession(
        session,
        mode: DemoSessionMode.firebaseExchange,
        infoMessage: 'Exchanged Firebase sign-in into a Cortado session.',
      );
    });
  }

  Future<void> _mintApiKey() async {
    await _runBusy('Minting Cortado API key', () async {
      final baseUrl = _baseUrlController.text.trim();
      DemoIssuedApiKey issued;
      if (_hasUserScopedSession && _authSession != null) {
        issued = await _firebaseBootstrap.mintApiKeyWithSession(
          baseUrl,
          _authSession!,
        );
      } else {
        try {
          issued = await _firebaseBootstrap.mintApiKey(baseUrl);
        } on StateError catch (error) {
          if (!error.toString().contains('firebase tenant claim is required')) {
            rethrow;
          }
          await _assignDevelopmentTenantInternal(baseUrl);
          issued = await _firebaseBootstrap.mintApiKey(baseUrl);
        }
      }
      final listed = await _loadPersonalApiKeys(baseUrl);

      _apiKeyController.text = issued.apiKey;
      _userIdController.text = issued.record.userId;

      if (!mounted) {
        return;
      }

      setState(() {
        _issuedApiKey = issued;
        _issuedApiKeys = listed;
      });
      _setInfoMessage(
        'Minted Cortado API key for ${issued.record.userId}. The session form was updated automatically.',
      );
    });
  }

  Future<void> _createPlatformTenant() async {
    if (!_ensureUserScopedSession()) {
      return;
    }

    await _runBusy('Creating platform tenant', () async {
      final baseUrl = _baseUrlController.text.trim();
      final displayName = _platformTenantDisplayNameController.text.trim();
      if (baseUrl.isEmpty) {
        throw StateError('Base URL is required.');
      }
      if (displayName.isEmpty) {
        throw StateError('Platform tenant name is required.');
      }

      final tenant = await _firebaseBootstrap.createPlatformTenant(
        baseUrl,
        _authSession!,
        displayName: displayName,
      );
      final tenants = await _firebaseBootstrap.listPlatformTenants(
        baseUrl,
        _authSession!,
      );

      _platformTenantIdController.text = tenant.tenantId;
      if (!mounted) {
        return;
      }

      setState(() {
        _platformTenants = tenants;
      });
      _setInfoMessage(
        'Created platform tenant ${tenant.tenantId}. You can mint a platform API key for it now.',
      );
    });
  }

  Future<void> _refreshPlatformTenants() async {
    if (!_ensureUserScopedSession()) {
      return;
    }

    await _runBusy('Loading platform tenants', () async {
      final tenants = await _firebaseBootstrap.listPlatformTenants(
        _baseUrlController.text.trim(),
        _authSession!,
      );
      if (!mounted) {
        return;
      }

      setState(() {
        _platformTenants = tenants;
      });
      _setInfoMessage('Loaded ${tenants.length} platform tenant(s).');
    });
  }

  Future<void> _mintPlatformApiKey() async {
    if (!_ensureUserScopedSession()) {
      return;
    }

    await _runBusy('Minting platform API key', () async {
      final baseUrl = _baseUrlController.text.trim();
      final tenantId = _platformTenantIdController.text.trim();
      if (baseUrl.isEmpty || tenantId.isEmpty) {
        throw StateError('Base URL and platform tenant ID are required.');
      }

      final issued = await _firebaseBootstrap.mintPlatformApiKey(
        baseUrl,
        _authSession!,
        tenantId: tenantId,
      );
      final listed = await _firebaseBootstrap.listPlatformApiKeys(
        baseUrl,
        _authSession!,
        tenantId: tenantId,
      );

      _apiKeyController.text = issued.apiKey;
      _userIdController.clear();

      if (!mounted) {
        return;
      }

      setState(() {
        _issuedPlatformApiKey = issued;
        _platformApiKeys = listed;
      });
      _setInfoMessage(
        'Minted a platform API key for $tenantId. The session form was updated; leave Demo User ID empty before pressing New Session.',
      );
    });
  }

  Future<void> _refreshPlatformApiKeys() async {
    if (!_ensureUserScopedSession()) {
      return;
    }

    await _runBusy('Loading platform API keys', () async {
      final tenantId = _platformTenantIdController.text.trim();
      if (tenantId.isEmpty) {
        throw StateError('Enter a platform tenant ID first.');
      }

      final listed = await _firebaseBootstrap.listPlatformApiKeys(
        _baseUrlController.text.trim(),
        _authSession!,
        tenantId: tenantId,
      );
      if (!mounted) {
        return;
      }

      setState(() {
        _platformApiKeys = listed;
      });
      _setInfoMessage('Loaded ${listed.length} platform API key record(s).');
    });
  }

  Future<void> _assignDevelopmentTenant() async {
    await _runBusy('Assigning development tenant', () async {
      final assignment = await _assignDevelopmentTenantInternal(
          _baseUrlController.text.trim());
      _setInfoMessage(
        'Assigned dev tenant ${assignment.tenantId} to ${assignment.userId}.',
      );
    });
  }

  Future<void> _refreshIssuedApiKeys() async {
    await _runBusy('Loading issued API keys', () async {
      final listed = await _loadPersonalApiKeys(_baseUrlController.text.trim());
      if (!mounted) {
        return;
      }

      setState(() {
        _issuedApiKeys = listed;
      });
      _setInfoMessage('Loaded ${listed.length} issued API key record(s).');
    });
  }

  Future<void> _afterFirebaseCredential(
    UserCredential credential, {
    required String successMessage,
  }) async {
    final user = credential.user;
    if (user == null) {
      throw StateError('Firebase did not return a user.');
    }

    _userIdController.text = user.uid;
    List<DemoApiKeyRecord> listed = _issuedApiKeys;
    try {
      listed = await _loadPersonalApiKeys(_baseUrlController.text.trim());
    } catch (_) {
      listed = const <DemoApiKeyRecord>[];
    }

    if (!mounted) {
      return;
    }

    setState(() {
      _firebaseUser = user;
      _issuedApiKeys = listed;
    });
    _setInfoMessage(successMessage);
  }

  Future<DemoTenantAssignment> _assignDevelopmentTenantInternal(
    String baseUrl,
  ) async {
    DemoTenantAssignment assignment;
    try {
      assignment = await _firebaseBootstrap.assignDevelopmentTenant(
        baseUrl,
        tenantId: widget.initialConfig.firebaseDevTenantId,
      );
    } on StateError catch (error) {
      if (!error.toString().contains('status 404')) {
        rethrow;
      }
      throw StateError(
        'The development tenant-claim route is not mounted. Restart the control plane with CORTADO_ENV=development, or skip this step and use Exchange Session instead.',
      );
    }
    if (!mounted) {
      return assignment;
    }

    setState(() {
      _tenantAssignment = assignment;
    });
    return assignment;
  }

  Future<List<DemoApiKeyRecord>> _loadPersonalApiKeys(String baseUrl) async {
    if (_hasUserScopedSession && _authSession != null) {
      return _firebaseBootstrap.listApiKeysWithSession(baseUrl, _authSession!);
    }
    return _firebaseBootstrap.listApiKeys(baseUrl);
  }

  Future<void> _authenticate() async {
    await _runBusy('Authenticating session', () async {
      final baseUrl = _baseUrlController.text.trim();
      final apiKey = _apiKeyController.text.trim();
      final userId = _userIdController.text.trim();

      if (baseUrl.isEmpty || apiKey.isEmpty) {
        throw StateError('Base URL and API key are required.');
      }

      await _disposeTransports();

      final session = CortadoAuthSession(baseUrl: baseUrl);
      final sessionMode = userId.isEmpty
          ? DemoSessionMode.platformApiKey
          : DemoSessionMode.personalApiKey;
      await session.createSession(
        apiKey: apiKey,
        userId: userId.isEmpty ? null : userId,
      );
      await _bindSession(
        session,
        mode: sessionMode,
        infoMessage: userId.isEmpty
            ? 'Platform API key session established.'
            : 'Session established for $userId.',
      );
    });
  }

  Future<void> _createWorkspace() async {
    if (!_ensureWorkspaceManager()) {
      return;
    }

    await _runBusy('Provisioning workspace', () async {
      final cpu = double.tryParse(_cpuController.text.trim()) ??
          DemoBootstrapConfig.defaultCpu;
      final memory = double.tryParse(_memoryController.text.trim()) ??
          DemoBootstrapConfig.defaultMemoryGb;
      final storage = double.tryParse(_storageController.text.trim()) ??
          DemoBootstrapConfig.defaultStorageGb;
      final image = _imageController.text.trim().isEmpty
          ? DemoBootstrapConfig.defaultImage
          : _imageController.text.trim();

      final workspace = await _workspaceManager!.create(
        image: image,
        resources: WorkspaceResources(
          cpu: cpu,
          memoryGb: memory,
          storageGb: storage,
        ),
      );

      _workspace = workspace;
      _workspaceStatus = workspace.toStatus();
      _workspaceIdController.text = workspace.id;
      _connectedWorkspaceId = null;
      _clearLoadedFile();
      await _watchWorkspace(workspace.id);
      await _maybeConnectWorkspaceSocket();

      _setInfoMessage(
        'Workspace ${workspace.id} created with $image.',
      );
    });
  }

  Future<void> _refreshWorkspace() async {
    if (!_ensureWorkspaceManager()) {
      return;
    }
    final workspaceId = _workspaceId;
    if (workspaceId.isEmpty) {
      _setInfoMessage('Enter a workspace ID first.');
      return;
    }

    await _runBusy('Refreshing workspace status', () async {
      final refreshed = await _guardWorkspaceLookup(
        workspaceId,
        () async {
          await _refreshWorkspaceState();
          return true;
        },
      );
      if (refreshed != true) {
        return;
      }
      _setInfoMessage(
        'Attached to workspace $workspaceId in '
        '${_workspaceStatus?.status.name.toUpperCase()}.',
      );
    });
  }

  Future<void> _startWorkspace() async {
    if (!_ensureWorkspaceManager()) {
      return;
    }
    final workspaceId = _workspaceId;
    if (workspaceId.isEmpty) {
      _setInfoMessage('Enter a workspace ID first.');
      return;
    }

    await _runBusy('Starting workspace', () async {
      final started = await _guardWorkspaceLookup(
        workspaceId,
        () async {
          await _workspaceManager!.start(workspaceId);
          await _refreshWorkspaceState();
          return true;
        },
      );
      if (started != true) {
        return;
      }
      _setInfoMessage('Start requested for workspace $workspaceId.');
    });
  }

  Future<void> _stopWorkspace() async {
    if (!_ensureWorkspaceManager()) {
      return;
    }
    final workspaceId = _workspaceId;
    if (workspaceId.isEmpty) {
      _setInfoMessage('Enter a workspace ID first.');
      return;
    }

    await _runBusy('Stopping workspace', () async {
      final stopped = await _guardWorkspaceLookup(
        workspaceId,
        () async {
          await _workspaceManager!.stop(workspaceId);
          return true;
        },
      );
      if (stopped != true) {
        return;
      }
      await _disconnectWorkspaceSocket();
      await _refreshWorkspaceState();
      _setInfoMessage('Stop requested for workspace $workspaceId.');
    });
  }

  Future<void> _deleteWorkspace() async {
    if (!_ensureWorkspaceManager()) {
      return;
    }
    final workspaceId = _workspaceId;
    if (workspaceId.isEmpty) {
      _setInfoMessage('Enter a workspace ID first.');
      return;
    }

    await _runBusy('Deleting workspace', () async {
      final deletedWorkspace = await _guardWorkspaceLookup(
        workspaceId,
        () => _workspaceManager!.deleteWorkspace(
          workspaceId,
        ),
      );
      if (deletedWorkspace == null) {
        return;
      }

      await _statusSubscription?.cancel();
      _statusSubscription = null;
      await _disconnectWorkspaceSocket();

      setState(() {
        _workspace = null;
        _workspaceStatus = null;
        _workspaceIdController.clear();
        _clearLoadedFile();
      });

      _setInfoMessage('Workspace ${deletedWorkspace.id} deleted.');
    });
  }

  Future<void> _loadFile() async {
    if (!_ensureWorkspaceManager()) {
      return;
    }
    final workspaceId = _activeWorkspaceId;
    if (workspaceId.isEmpty) {
      _setInfoMessage('Enter a workspace ID first.');
      return;
    }
    if (_filePath.isEmpty) {
      _setInfoMessage('Enter a target file path first.');
      return;
    }

    await _runBusy('Loading file', () async {
      final bytes = await _guardWorkspaceLookup(
        workspaceId,
        () => _workspaceManager!.readFile(
          workspaceId,
          path: _filePath,
        ),
      );
      if (bytes == null) {
        return;
      }
      setState(() {
        _draftCode = utf8.decode(bytes, allowMalformed: true);
        _loadedFilePath = _filePath;
        _documentRevision++;
      });

      _setInfoMessage('Loaded $_filePath from workspace $workspaceId.');
    });
  }

  Future<void> _saveFile() async {
    if (!_ensureWorkspaceManager()) {
      return;
    }
    final workspaceId = _activeWorkspaceId;
    if (workspaceId.isEmpty || _filePath.isEmpty) {
      _setInfoMessage('Workspace ID and target file are required.');
      return;
    }

    await _runBusy('Saving file', () async {
      final saved = await _guardWorkspaceLookup(
        workspaceId,
        () => _workspaceManager!.writeFile(
          workspaceId,
          path: _filePath,
          content: utf8.encode(_draftCode),
        ),
      );
      if (saved == null) {
        return;
      }
      setState(() {
        _loadedFilePath = _filePath;
      });
      _setInfoMessage('Saved $_filePath to workspace $workspaceId.');
    });
  }

  Future<void> _watchWorkspace(String workspaceId) async {
    await _statusSubscription?.cancel();
    _statusSubscription = _workspaceManager!.watchStatus(workspaceId).listen(
      (status) {
        if (!mounted) {
          return;
        }

        setState(() {
          _workspaceStatus = status;
        });

        if (status.status == WorkspaceLifecycleState.running) {
          unawaited(_maybeConnectWorkspaceSocket());
        } else if (status.isTerminal) {
          unawaited(_disconnectWorkspaceSocket());
        }
      },
      onError: (Object error, StackTrace stackTrace) {
        unawaited(_handleWorkspaceLookupError(workspaceId, error));
      },
    );
  }

  Future<void> _refreshWorkspaceState() async {
    if (mounted) {
      setState(() {
        _workspace = null;
        _workspaceStatus = null;
        _connectedWorkspaceId = null;
      });
      _clearLoadedFile();
    }

    final workspace = await _workspaceManager!.getWorkspace(_workspaceId);
    if (!mounted) {
      return;
    }

    setState(() {
      _workspace = workspace;
      _workspaceStatus = workspace.toStatus();
    });

    await _watchWorkspace(workspace.id);
    await _maybeConnectWorkspaceSocket();
  }

  Future<void> _maybeConnectWorkspaceSocket() async {
    final workspaceId = _activeWorkspaceId;
    if (_client == null || workspaceId.isEmpty) {
      return;
    }
    if (_workspaceStatus?.status != WorkspaceLifecycleState.running) {
      return;
    }
    if (_connectedWorkspaceId == workspaceId) {
      return;
    }

    try {
      await _client!.connect(workspaceId);
      if (!mounted) {
        return;
      }
      setState(() {
        _connectedWorkspaceId = workspaceId;
      });
    } catch (error) {
      _setInfoMessage('Terminal socket attach failed: $error');
    }
  }

  Future<void> _disconnectWorkspaceSocket() async {
    await _client?.disconnect();
    if (!mounted) {
      return;
    }
    setState(() {
      _connectedWorkspaceId = null;
    });
  }

  Future<void> _disposeTransports() async {
    await _statusSubscription?.cancel();
    _statusSubscription = null;
    await _client?.dispose();
    await _workspaceManager?.dispose();
    await _authSession?.dispose();
    _client = null;
    _workspaceManager = null;
    _authSession = null;
    _workspace = null;
    _workspaceStatus = null;
    _connectedWorkspaceId = null;
    _sessionMode = null;
    _clearLoadedFile();
  }

  Future<T?> _guardWorkspaceLookup<T>(
    String workspaceId,
    Future<T> Function() action,
  ) async {
    try {
      return await action();
    } catch (error) {
      final handled = await _handleWorkspaceLookupError(workspaceId, error);
      if (handled) {
        return null;
      }
      rethrow;
    }
  }

  Future<bool> _handleWorkspaceLookupError(
    String workspaceId,
    Object error,
  ) async {
    if (error is! WorkspaceRequestException || error.statusCode != 404) {
      _setInfoMessage(error.toString());
      return false;
    }

    await _statusSubscription?.cancel();
    _statusSubscription = null;
    await _disconnectWorkspaceSocket();
    if (!mounted) {
      return true;
    }

    setState(() {
      _workspace = null;
      _workspaceStatus = null;
      _connectedWorkspaceId = null;
      _clearLoadedFile();
    });
    _setInfoMessage(
      'Workspace $workspaceId was not found. It may have been deleted, or it belongs to a different Cortado session. Provision a new workspace or refresh using a workspace ID owned by the current session.',
    );
    return true;
  }

  bool _ensureWorkspaceManager() {
    if (_workspaceManager != null && _client != null && _authSession != null) {
      return true;
    }
    _setInfoMessage('Authenticate the demo session first.');
    return false;
  }

  Future<void> _runBusy(
    String label,
    Future<void> Function() action,
  ) async {
    if (_isBusy) {
      return;
    }

    setState(() {
      _busyLabel = label;
    });

    try {
      await action();
    } catch (error) {
      _setInfoMessage(error.toString());
    } finally {
      if (mounted) {
        setState(() {
          _busyLabel = null;
        });
      }
    }
  }

  void _setInfoMessage(String message) {
    if (!mounted) {
      return;
    }

    setState(() {
      _infoMessage = message;
    });

    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(content: Text(message)),
    );
  }

  void _clearLoadedFile() {
    _draftCode = '';
    _loadedFilePath = '';
    _documentRevision++;
  }

  bool _ensureUserScopedSession() {
    if (_hasUserScopedSession) {
      return true;
    }

    _setInfoMessage(
      'Create a first-party Firebase session or personal API-key session first. Platform API-key sessions cannot manage platform tenants.',
    );
    return false;
  }

  Future<void> _bindSession(
    CortadoAuthSession session, {
    required DemoSessionMode mode,
    required String infoMessage,
  }) async {
    final baseUrl = _baseUrlController.text.trim();
    _authSession = session;
    _workspaceManager = WorkspaceManager(
      baseUrl: baseUrl,
      authSession: session,
    );
    _client = CortadoClient(
      baseUrl: baseUrl,
      authSession: session,
    );
    if (mounted) {
      setState(() {
        _workspace = null;
        _workspaceStatus = null;
        _connectedWorkspaceId = null;
        _sessionMode = mode;
      });
      _clearLoadedFile();
    } else {
      _connectedWorkspaceId = null;
      _sessionMode = mode;
    }
    _setInfoMessage(infoMessage);

    if (_workspaceId.isNotEmpty) {
      await _refreshWorkspaceState();
    }
  }
}

class _EditorExperienceCard extends StatelessWidget {
  const _EditorExperienceCard({
    required this.package,
    required this.loadedFilePath,
    required this.hasLoadedFile,
    required this.draftCode,
    required this.documentKey,
    required this.onChanged,
  });

  final DemoEditorPackage package;
  final String loadedFilePath;
  final bool hasLoadedFile;
  final String draftCode;
  final String documentKey;
  final ValueChanged<String> onChanged;

  @override
  Widget build(BuildContext context) {
    final noteStyle = Theme.of(context).textTheme.bodyMedium?.copyWith(
          color: const Color(0xFFB8C4D2),
        );

    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(18),
      decoration: BoxDecoration(
        color: const Color(0xFF0A1118),
        borderRadius: BorderRadius.circular(18),
        border: Border.all(color: const Color(0xFF223142)),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: <Widget>[
          Text(
            package.label,
            style: Theme.of(context).textTheme.titleMedium,
          ),
          const SizedBox(height: 4),
          Text(
            package.packageName,
            style: Theme.of(context).textTheme.bodySmall?.copyWith(
                  color: const Color(0xFF8898AA),
                ),
          ),
          const SizedBox(height: 12),
          for (final note in package.notes) ...<Widget>[
            Text('• $note', style: noteStyle),
            const SizedBox(height: 6),
          ],
          const SizedBox(height: 10),
          if (!hasLoadedFile)
            Container(
              height: 460,
              alignment: Alignment.center,
              decoration: BoxDecoration(
                color: const Color(0xFF101823),
                borderRadius: BorderRadius.circular(16),
                border: Border.all(color: const Color(0xFF243240)),
              ),
              child: Padding(
                padding: const EdgeInsets.all(24),
                child: Text(
                  'Load the target workspace file first. After you install Flutter in the Ubuntu workspace and run `flutter create --platforms=web .`, load `lib/main.dart` to start switching between editor packages.',
                  style: noteStyle,
                  textAlign: TextAlign.center,
                ),
              ),
            )
          else ...<Widget>[
            Text(
              'Loaded file: $loadedFilePath',
              style: Theme.of(context).textTheme.bodyMedium?.copyWith(
                    color: const Color(0xFFE5ECF3),
                    fontWeight: FontWeight.w600,
                  ),
            ),
            const SizedBox(height: 12),
            SizedBox(
              height: 520,
              child: _EditorViewport(
                key: ValueKey<String>(documentKey),
                package: package,
                filePath: loadedFilePath,
                initialCode: draftCode,
                onChanged: onChanged,
              ),
            ),
          ],
        ],
      ),
    );
  }
}

class _EditorViewport extends StatelessWidget {
  const _EditorViewport({
    super.key,
    required this.package,
    required this.filePath,
    required this.initialCode,
    required this.onChanged,
  });

  final DemoEditorPackage package;
  final String filePath;
  final String initialCode;
  final ValueChanged<String> onChanged;

  @override
  Widget build(BuildContext context) {
    return switch (package) {
      DemoEditorPackage.flutterCodeEditor => _FlutterCodeEditorPane(
          initialCode: initialCode,
          onChanged: onChanged,
        ),
      DemoEditorPackage.flutterMonaco => _MonacoPane(
          initialCode: initialCode,
          onChanged: onChanged,
        ),
      DemoEditorPackage.codeForge => _CodeForgePane(
          initialCode: initialCode,
          filePath: filePath,
          onChanged: onChanged,
        ),
      DemoEditorPackage.liteCodeEditor => _LiteCodeEditorPane(
          initialCode: initialCode,
          onChanged: onChanged,
        ),
    };
  }
}

class _FlutterCodeEditorPane extends StatefulWidget {
  const _FlutterCodeEditorPane({
    required this.initialCode,
    required this.onChanged,
  });

  final String initialCode;
  final ValueChanged<String> onChanged;

  @override
  State<_FlutterCodeEditorPane> createState() => _FlutterCodeEditorPaneState();
}

class _FlutterCodeEditorPaneState extends State<_FlutterCodeEditorPane> {
  late final flutter_code_editor.CodeController _controller =
      flutter_code_editor.CodeController(
    text: widget.initialCode,
    language: highlight_dart.dart,
  );

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return DecoratedBox(
      decoration: BoxDecoration(
        color: const Color(0xFF10161F),
        borderRadius: BorderRadius.circular(16),
      ),
      child: flutter_code_editor.CodeTheme(
        data: flutter_code_editor.CodeThemeData(styles: monokaiSublimeTheme),
        child: flutter_code_editor.CodeField(
          controller: _controller,
          wrap: false,
          textStyle: const TextStyle(
            fontFamily: 'monospace',
            fontSize: 14,
            height: 1.5,
          ),
          onChanged: widget.onChanged,
        ),
      ),
    );
  }
}

class _LiteCodeEditorPane extends StatefulWidget {
  const _LiteCodeEditorPane({
    required this.initialCode,
    required this.onChanged,
  });

  final String initialCode;
  final ValueChanged<String> onChanged;

  @override
  State<_LiteCodeEditorPane> createState() => _LiteCodeEditorPaneState();
}

class _LiteCodeEditorPaneState extends State<_LiteCodeEditorPane> {
  late final lite_code_editor.CodeEditorController _controller =
      lite_code_editor.CodeEditorController(
    initialCode: widget.initialCode,
    language: lite_code_editor.CodeLanguage.dart,
  );

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return ClipRRect(
      borderRadius: BorderRadius.circular(16),
      child: lite_code_editor.CodeEditor(
        controller: _controller,
        theme: lite_code_editor.EditorTheme.dark(),
        onChanged: widget.onChanged,
      ),
    );
  }
}

class _MonacoPane extends StatelessWidget {
  const _MonacoPane({
    required this.initialCode,
    required this.onChanged,
  });

  final String initialCode;
  final ValueChanged<String> onChanged;

  @override
  Widget build(BuildContext context) {
    return ClipRRect(
      borderRadius: BorderRadius.circular(16),
      child: MonacoEditor(
        initialValue: initialCode,
        options: const EditorOptions(
          language: MonacoLanguage.dart,
          theme: MonacoTheme.vsDark,
          fontSize: 14,
          minimap: false,
          lineNumbers: true,
          wordWrap: true,
        ),
        showStatusBar: true,
        onContentChanged: onChanged,
      ),
    );
  }
}

class _CodeForgePane extends StatefulWidget {
  const _CodeForgePane({
    required this.initialCode,
    required this.filePath,
    required this.onChanged,
  });

  final String initialCode;
  final String filePath;
  final ValueChanged<String> onChanged;

  @override
  State<_CodeForgePane> createState() => _CodeForgePaneState();
}

class _CodeForgePaneState extends State<_CodeForgePane> {
  late final code_forge.CodeForgeWebController _controller =
      code_forge.CodeForgeWebController()..text = widget.initialCode;

  @override
  void initState() {
    super.initState();
    _controller.addListener(_handleChanged);
  }

  @override
  void dispose() {
    _controller.removeListener(_handleChanged);
    _controller.dispose();
    super.dispose();
  }

  void _handleChanged() {
    widget.onChanged(_controller.text);
  }

  @override
  Widget build(BuildContext context) {
    return ClipRRect(
      borderRadius: BorderRadius.circular(16),
      child: code_forge.CodeForgeWeb(
        controller: _controller,
        fileUri: 'file:///workspace/${widget.filePath}',
        initialText: widget.initialCode,
        language: re_highlight_dart.langDart,
        enableFolding: true,
        enableGuideLines: true,
        enableGutter: true,
        enableSuggestions: true,
        textStyle: const TextStyle(
          fontFamily: 'monospace',
          fontSize: 14,
          color: Colors.white,
        ),
      ),
    );
  }
}

class _SurfaceCard extends StatelessWidget {
  const _SurfaceCard({required this.child});

  final Widget child;

  @override
  Widget build(BuildContext context) {
    return DecoratedBox(
      decoration: BoxDecoration(
        color: const Color(0xCC101824),
        borderRadius: BorderRadius.circular(24),
        border: Border.all(color: const Color(0xFF223142)),
        boxShadow: const <BoxShadow>[
          BoxShadow(
            color: Color(0x55000000),
            blurRadius: 32,
            offset: Offset(0, 18),
          ),
        ],
      ),
      child: Padding(
        padding: const EdgeInsets.all(20),
        child: child,
      ),
    );
  }
}

class _StatusChip extends StatelessWidget {
  const _StatusChip({required this.label});

  final String label;

  @override
  Widget build(BuildContext context) {
    return Chip(
      label: Text(label),
      side: BorderSide.none,
      backgroundColor: const Color(0xFF182432),
    );
  }
}
