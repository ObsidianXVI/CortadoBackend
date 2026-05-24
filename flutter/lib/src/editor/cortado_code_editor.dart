import 'dart:async';
import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../ai/cortado_ai_service.dart';
import '../cortado_client.dart';
import '../cortado_workspace_provider.dart';
import '../filesystem/vfs_notifier.dart';
import '../gen/agent/v1/agent.pb.dart' as agentpb;
import '../mux_frame.dart';
import '../workspace_manager.dart';
import 'editor_diagnostics.dart';
import 'cortado_lsp_client.dart';
import 'editor_models.dart';
import 'editor_platform.dart';
import 'tabs_notifier.dart';

class CortadoCodeEditorPlatformAdapter {
  const CortadoCodeEditorPlatformAdapter();

  bool get supportsPlatformView => supportsCortadoEditorPlatformView;

  void registerViewFactory({
    required String viewType,
    required String editorId,
    required String languageId,
    required CortadoEditorChangedCallback onChanged,
    required CortadoEditorSaveCallback onSave,
  }) {
    registerCortadoEditorViewFactory(
      viewType: viewType,
      editorId: editorId,
      languageId: languageId,
      onChanged: onChanged,
      onSave: onSave,
    );
  }

  Widget buildView(String viewType) {
    return buildCortadoEditorPlatformView(viewType);
  }

  String getContent(String editorId) {
    return getCortadoEditorContent(editorId);
  }

  void disposeView(String editorId) {
    disposeCortadoEditorView(editorId);
  }

  void registerLspRequestHandler({
    required String editorId,
    required CortadoEditorLspRequestCallback onRequest,
  }) {
    registerCortadoEditorLspRequestHandler(
      editorId: editorId,
      onRequest: onRequest,
    );
  }

  void unregisterLspRequestHandler(String editorId) {
    unregisterCortadoEditorLspRequestHandler(editorId);
  }

  void registerInlineCompletionRequestHandler({
    required String editorId,
    required CortadoEditorInlineCompletionRequestCallback onRequest,
  }) {
    registerCortadoEditorInlineCompletionRequestHandler(
      editorId: editorId,
      onRequest: onRequest,
    );
  }

  void unregisterInlineCompletionRequestHandler(String editorId) {
    unregisterCortadoEditorInlineCompletionRequestHandler(editorId);
  }

  void resolveLspResult(
    int requestId,
    Object? result,
  ) {
    resolveCortadoEditorLspResponse(requestId, result);
  }

  void setDiagnostics(
    String editorId,
    List<Map<String, Object?>> diagnostics,
  ) {
    setCortadoEditorDiagnostics(editorId, diagnostics);
  }

  void setLanguage(String editorId, String languageId) {
    setCortadoEditorLanguage(editorId, languageId);
  }

  void setReadOnly(String editorId, bool readOnly) {
    setCortadoEditorReadOnly(editorId, readOnly);
  }

  void setInlineCompletion(
    String editorId, {
    required int requestId,
    required String text,
  }) {
    setCortadoEditorInlineCompletion(
      editorId,
      requestId: requestId,
      text: text,
    );
  }

  void clearInlineCompletion(String editorId) {
    clearCortadoEditorInlineCompletion(editorId);
  }

  String setContent(
    String editorId,
    String content, {
    bool preserveSelection = false,
  }) {
    return setCortadoEditorContent(
      editorId,
      content,
      preserveSelection: preserveSelection,
    );
  }
}

class CortadoCodeEditor extends ConsumerStatefulWidget {
  const CortadoCodeEditor({
    super.key,
    this.client,
    this.path,
    this.fileEvents,
    this.lspChannelId = muxLspChannelStartId,
    this.lspLanguage = 'dart',
    this.maxTabs = 15,
    this.onError,
    this.onTabChanged,
    this.platform = const CortadoCodeEditorPlatformAdapter(),
    this.aiService,
    this.workspaceId,
    this.workspaceManager,
  }) : assert(maxTabs > 0, 'maxTabs must be greater than zero.');

  final CortadoClient? client;
  final Stream<agentpb.FileEvent>? fileEvents;
  final int lspChannelId;
  final String lspLanguage;
  final int maxTabs;
  final ValueChanged<String>? onError;
  final ValueChanged<OpenTab?>? onTabChanged;
  final String? path;
  final CortadoCodeEditorPlatformAdapter platform;
  final CortadoAIService? aiService;
  final String? workspaceId;
  final WorkspaceManager? workspaceManager;

  @override
  ConsumerState<CortadoCodeEditor> createState() => _CortadoCodeEditorState();
}

class _CortadoCodeEditorState extends ConsumerState<CortadoCodeEditor> {
  static int _instanceCount = 0;
  static const String _dartSdkRoot = '/usr/local/dart-sdk';

  late final String _editorId = 'cortado-editor-${_instanceCount++}';
  late final String _viewType = 'cortado-editor-view-$_editorId';
  late final TabsNotifier _tabsNotifier = TabsNotifier(maxTabs: widget.maxTabs);
  late final StateController<Map<String, CortadoFileDiagnosticStatus>>
      _diagnosticStatusController;
  late final void Function() _removeTabsListener =
      _tabsNotifier.addListener(_handleTabsChanged, fireImmediately: false);

  final Map<String, int> _loadVersions = <String, int>{};

  StreamSubscription<agentpb.FileEvent>? _fileEventSubscription;
  StreamSubscription<String>? _inlineCompletionSubscription;
  StreamSubscription<CortadoLSPDiagnosticsByUri>? _lspDiagnosticsSubscription;
  StreamSubscription<void>? _lspStateSubscription;
  CortadoAIService? _ownedInlineCompletionService;
  CortadoLSPDiagnosticsByUri _diagnosticsByUri =
      const <String, List<CortadoLSPDiagnostic>>{};
  Map<String, CortadoFileDiagnosticStatus> _diagnosticStatusesByPath =
      const <String, CortadoFileDiagnosticStatus>{};
  int? _activeInlineCompletionRequestId;
  CortadoLSPClient? _lspClient;

  @override
  void initState() {
    super.initState();
    _diagnosticStatusController =
        ref.read(cortadoWorkspaceDiagnosticStatusProvider.notifier);
    if (widget.platform.supportsPlatformView) {
      widget.platform.registerViewFactory(
        viewType: _viewType,
        editorId: _editorId,
        languageId: 'plain',
        onChanged: _handleEditorHashChanged,
        onSave: _handleSaveShortcut,
      );
      widget.platform.registerLspRequestHandler(
        editorId: _editorId,
        onRequest: (requestJson) => unawaited(_handleLspRequest(requestJson)),
      );
      widget.platform.registerInlineCompletionRequestHandler(
        editorId: _editorId,
        onRequest: (requestJson) =>
            unawaited(_handleInlineCompletionRequest(requestJson)),
      );
    }
    _configureLspClient();
    _subscribeToFileEvents();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted) {
        return;
      }
      unawaited(_openRequestedPath());
    });
  }

  @override
  void didUpdateWidget(CortadoCodeEditor oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.client != widget.client ||
        oldWidget.lspChannelId != widget.lspChannelId ||
        oldWidget.lspLanguage != widget.lspLanguage) {
      _configureLspClient();
    }
    if (oldWidget.fileEvents != widget.fileEvents) {
      _subscribeToFileEvents();
    }
    if (oldWidget.path != widget.path) {
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (!mounted) {
          return;
        }
        unawaited(_openRequestedPath());
      });
    }
  }

  @override
  void dispose() {
    _fileEventSubscription?.cancel();
    _inlineCompletionSubscription?.cancel();
    _lspDiagnosticsSubscription?.cancel();
    _lspStateSubscription?.cancel();
    _publishDiagnosticStatuses(
      const <String, CortadoFileDiagnosticStatus>{},
    );
    widget.platform.clearInlineCompletion(_editorId);
    widget.platform.setDiagnostics(_editorId, const <Map<String, Object?>>[]);
    unawaited(_ownedInlineCompletionService?.dispose());
    unawaited(_lspClient?.dispose());
    _removeTabsListener();
    _tabsNotifier.dispose();
    widget.platform.unregisterInlineCompletionRequestHandler(_editorId);
    widget.platform.unregisterLspRequestHandler(_editorId);
    widget.platform.disposeView(_editorId);
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    if (!widget.platform.supportsPlatformView) {
      return widget.platform.buildView(_viewType);
    }

    final state = _tabsNotifier.currentState;
    final activeTab = state.activeTab;

    return Material(
      color: Colors.transparent,
      child: DecoratedBox(
        decoration: BoxDecoration(
          color: const Color(0xFF0B1220),
          border: Border.all(color: const Color(0x1F94A3B8)),
          borderRadius: BorderRadius.circular(14),
        ),
        child: Column(
          children: <Widget>[
            _EditorTabStrip(
              activePath: state.activePath,
              diagnosticStatusesByPath: _diagnosticStatusesByPath,
              tabs: state.tabs,
              onClose: _closeTab,
              onSelect: _activateTab,
            ),
            Expanded(
              child: Stack(
                children: <Widget>[
                  Positioned.fill(child: widget.platform.buildView(_viewType)),
                  if (activeTab == null)
                    const Positioned.fill(child: _EditorEmptyState())
                  else if (activeTab.isLoading)
                    Positioned.fill(
                      child: _EditorOverlay(
                        message: 'Loading ${activeTab.title}...',
                      ),
                    )
                  else if (activeTab.errorMessage case final String message?)
                    Positioned.fill(
                      child: _EditorOverlay(
                        actionLabel: 'Retry',
                        message: message,
                        onAction: () => unawaited(_loadTab(activeTab.path)),
                      ),
                    )
                  else if (_shouldShowLspOverlay(activeTab))
                    const Positioned.fill(
                      child: _EditorOverlay(
                        message: 'Language server starting...',
                      ),
                    )
                  else if (activeTab.isSaving)
                    const Positioned(
                      right: 16,
                      top: 12,
                      child: _SavingBadge(),
                    ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }

  void _handleTabsChanged(TabsState next) {
    if (!mounted) {
      return;
    }
    _syncPlatformDiagnostics();
    setState(() {});
    widget.onTabChanged?.call(next.activeTab);
  }

  void _handleEditorHashChanged(String hash) {
    final activePath = _tabsNotifier.currentState.activePath;
    if (activePath == null) {
      return;
    }
    _tabsNotifier.markHash(activePath, hash);

    final activeTab = _tabsNotifier.tabForPath(activePath);
    final lspClient = _lspClient;
    if (activeTab == null ||
        lspClient == null ||
        !_shouldUseLspForTab(activeTab) ||
        !lspClient.isDocumentOpen(activePath)) {
      return;
    }

    _runLspAction(
      lspClient.didChangeTextDocument(
        path: activePath,
        text: widget.platform.getContent(_editorId),
      ),
    );
  }

  void _handleSaveShortcut() {
    unawaited(_saveActiveTab());
  }

  Future<void> _openRequestedPath() async {
    final requestedPath = widget.path;
    if (requestedPath == null || requestedPath.trim().isEmpty) {
      return;
    }

    final normalizedPath = normalizeVfsPath(requestedPath);
    final existingTab = _tabsNotifier.tabForPath(normalizedPath);
    if (existingTab != null) {
      await _activateTab(normalizedPath);
      return;
    }

    await _activatePath(
      normalizedPath,
      preserveSelection: false,
      reload: true,
    );
  }

  Future<void> _activatePath(
    String path, {
    required bool preserveSelection,
    required bool reload,
  }) async {
    _clearInlineCompletion();
    final normalizedPath = normalizeVfsPath(path);
    final previousActivePath = _tabsNotifier.currentState.activePath;
    if (previousActivePath != null && previousActivePath != normalizedPath) {
      _tabsNotifier.updateContent(
        previousActivePath,
        widget.platform.getContent(_editorId),
      );
    }

    final readOnly = _isReadOnlyExternalPath(normalizedPath);
    _tabsNotifier.open(
      normalizedPath,
      readOnly: readOnly,
    );
    final tab = _tabsNotifier.tabForPath(normalizedPath);
    if (tab == null) {
      return;
    }

    if (!reload && tab.loaded) {
      _tabsNotifier.activate(normalizedPath);
      _renderTab(tab, preserveSelection: false);
      return;
    }

    if (tab.readOnly && tab.loaded) {
      _tabsNotifier.activate(normalizedPath);
      _renderTab(
        tab,
        preserveSelection: preserveSelection,
      );
      return;
    }

    await _loadTab(
      normalizedPath,
      preserveSelection: preserveSelection,
    );
  }

  Future<void> _activateTab(String path) async {
    await _activatePath(
      path,
      preserveSelection: false,
      reload: false,
    );
  }

  Future<void> _closeTab(String path) async {
    _clearInlineCompletion();
    final normalizedPath = normalizeVfsPath(path);
    final closingTab = _tabsNotifier.tabForPath(normalizedPath);
    final activePath = _tabsNotifier.currentState.activePath;
    if (activePath == normalizedPath) {
      _tabsNotifier.updateContent(
        normalizedPath,
        widget.platform.getContent(_editorId),
      );
    }

    _tabsNotifier.close(normalizedPath);
    _closeLspDocumentIfNeeded(closingTab);
    final nextActiveTab = _tabsNotifier.currentState.activeTab;
    if (nextActiveTab != null) {
      _renderTab(nextActiveTab, preserveSelection: false);
    } else {
      widget.platform.setLanguage(_editorId, 'plain');
      widget.platform.setReadOnly(_editorId, false);
      widget.platform.setContent(_editorId, '', preserveSelection: false);
    }
  }

  Future<void> _loadTab(
    String path, {
    bool preserveSelection = false,
  }) async {
    final normalizedPath = normalizeVfsPath(path);
    if (!_isWorkspacePath(normalizedPath)) {
      return;
    }
    final wasOpenInLsp = _lspClient?.isDocumentOpen(normalizedPath) ?? false;
    _tabsNotifier.markLoading(normalizedPath);

    final loadVersion = (_loadVersions[normalizedPath] ?? 0) + 1;
    _loadVersions[normalizedPath] = loadVersion;

    try {
      final bytes = await _manager.readFile(
        _workspaceId,
        path: normalizedPath,
      );
      final content = utf8.decode(bytes);
      if (!mounted || _loadVersions[normalizedPath] != loadVersion) {
        return;
      }

      final hash = _tabsNotifier.currentState.activePath == normalizedPath
          ? _renderLoadedContent(
              normalizedPath,
              content,
              preserveSelection: preserveSelection,
            )
          : hashEditorContent(content);

      _tabsNotifier.setLoaded(
        normalizedPath,
        content: content,
        hash: hash,
      );

      if (_tabsNotifier.currentState.activePath == normalizedPath) {
        final refreshedTab = _tabsNotifier.tabForPath(normalizedPath);
        if (refreshedTab != null) {
          _renderTab(
            refreshedTab,
            preserveSelection: preserveSelection,
          );
        }
      }

      final loadedTab = _tabsNotifier.tabForPath(normalizedPath);
      if (loadedTab != null) {
        _syncLoadedTabWithLsp(
          loadedTab,
          content,
          wasAlreadyOpen: wasOpenInLsp,
        );
      }
    } catch (error) {
      if (!mounted || _loadVersions[normalizedPath] != loadVersion) {
        return;
      }
      _tabsNotifier.setError(normalizedPath, error.toString());
      widget.onError?.call(error.toString());
    }
  }

  String _renderLoadedContent(
    String path,
    String content, {
    required bool preserveSelection,
    bool readOnly = false,
  }) {
    _clearInlineCompletion();
    widget.platform.setLanguage(_editorId, editorLanguageIdForPath(path));
    widget.platform.setReadOnly(_editorId, readOnly);
    return widget.platform.setContent(
      _editorId,
      content,
      preserveSelection: preserveSelection,
    );
  }

  void _renderTab(
    OpenTab tab, {
    required bool preserveSelection,
  }) {
    _clearInlineCompletion();
    widget.platform.setLanguage(_editorId, tab.languageId);
    widget.platform.setReadOnly(_editorId, tab.readOnly);
    final hash = widget.platform.setContent(
      _editorId,
      tab.content,
      preserveSelection: preserveSelection,
    );
    _tabsNotifier.markHash(tab.path, hash);
  }

  Future<void> _saveActiveTab() async {
    final activeTab = _tabsNotifier.currentState.activeTab;
    if (activeTab == null || activeTab.isLoading || activeTab.readOnly) {
      return;
    }

    final content = widget.platform.getContent(_editorId);
    final hash = hashEditorContent(content);
    _tabsNotifier.markSaving(activeTab.path, true);

    try {
      await _manager.writeFile(
        _workspaceId,
        path: activeTab.path,
        content: utf8.encode(content),
      );
      _tabsNotifier.markSaved(
        activeTab.path,
        content: content,
        hash: hash,
      );
    } catch (error) {
      _tabsNotifier.setError(activeTab.path, error.toString());
      widget.onError?.call(error.toString());
    }
  }

  void _subscribeToFileEvents() {
    _fileEventSubscription?.cancel();
    final fileEvents = widget.fileEvents;
    if (fileEvents == null) {
      _fileEventSubscription = null;
      return;
    }

    _fileEventSubscription = fileEvents.listen(_handleExternalFileEvent);
  }

  void _handleExternalFileEvent(agentpb.FileEvent event) {
    final path = normalizeVfsPath(event.path);
    final openTab = _tabsNotifier.tabForPath(path);
    if (openTab == null) {
      return;
    }

    _clearInlineCompletion();

    switch (event.type) {
      case agentpb.FileEventType.FILE_EVENT_TYPE_MODIFIED:
      case agentpb.FileEventType.FILE_EVENT_TYPE_CREATED:
        if (openTab.isDirty) {
          return;
        }
        unawaited(
          _loadTab(
            path,
            preserveSelection: _tabsNotifier.currentState.activePath == path,
          ),
        );
        return;
      case agentpb.FileEventType.FILE_EVENT_TYPE_DELETED:
      case agentpb.FileEventType.FILE_EVENT_TYPE_RENAMED:
        unawaited(_closeTab(path));
        return;
      case agentpb.FileEventType.FILE_EVENT_TYPE_UNSPECIFIED:
        return;
    }
  }

  WorkspaceManager get _manager =>
      widget.workspaceManager ?? ref.read(cortadoWorkspaceManagerProvider);

  String get _workspaceId =>
      widget.workspaceId ?? ref.read(cortadoWorkspaceIdProvider);

  void _configureLspClient() {
    _clearInlineCompletion();
    _lspDiagnosticsSubscription?.cancel();
    _lspDiagnosticsSubscription = null;
    _lspStateSubscription?.cancel();
    _lspStateSubscription = null;
    _diagnosticsByUri = const <String, List<CortadoLSPDiagnostic>>{};
    _publishDiagnosticStatuses(
      const <String, CortadoFileDiagnosticStatus>{},
    );
    widget.platform.setDiagnostics(_editorId, const <Map<String, Object?>>[]);
    unawaited(_lspClient?.dispose());
    _lspClient = null;

    final client = widget.client;
    if (client == null) {
      if (mounted) {
        setState(() {});
      }
      return;
    }

    final lspClient = CortadoLSPClient(
      client: client,
      channelId: widget.lspChannelId,
      language: widget.lspLanguage,
    );
    _lspClient = lspClient;
    _lspStateSubscription = lspClient.stateChanges.listen((_) {
      if (mounted) {
        setState(() {});
      }
    });
    _lspDiagnosticsSubscription =
        lspClient.diagnosticsStream.listen(_handleDiagnosticsChanged);

    _resyncOpenTabsWithLsp(lspClient);
  }

  void _resyncOpenTabsWithLsp(CortadoLSPClient client) {
    for (final tab in _tabsNotifier.currentState.tabs) {
      if (!tab.loaded || !_shouldUseLspForTab(tab)) {
        continue;
      }
      _runLspAction(
        client.didOpenTextDocument(
          path: tab.path,
          languageId: tab.languageId,
          text: tab.content,
        ),
      );
    }
  }

  bool _shouldShowLspOverlay(OpenTab? activeTab) {
    if (activeTab == null || !_shouldUseLspForTab(activeTab)) {
      return false;
    }
    return _lspClient?.isInitializing ?? false;
  }

  bool _shouldUseLspForTab(OpenTab tab) =>
      !tab.readOnly &&
      _isWorkspacePath(tab.path) &&
      tab.languageId == widget.lspLanguage &&
      _lspClient != null;

  void _handleDiagnosticsChanged(CortadoLSPDiagnosticsByUri diagnosticsByUri) {
    _diagnosticsByUri = diagnosticsByUri;
    _publishDiagnosticStatuses(
      summarizeWorkspaceDiagnosticStatuses(diagnosticsByUri),
    );
    _syncPlatformDiagnostics();
    if (mounted) {
      setState(() {});
    }
  }

  void _publishDiagnosticStatuses(
    Map<String, CortadoFileDiagnosticStatus> statuses,
  ) {
    _diagnosticStatusesByPath = statuses;
    _diagnosticStatusController.state = statuses;
  }

  void _syncPlatformDiagnostics() {
    final activeTab = _tabsNotifier.currentState.activeTab;
    if (activeTab == null || !_shouldUseLspForTab(activeTab)) {
      widget.platform.setDiagnostics(_editorId, const <Map<String, Object?>>[]);
      return;
    }

    widget.platform.setDiagnostics(
      _editorId,
      _diagnosticsByUri[workspaceDocumentUriForPath(activeTab.path)] ??
          const <Map<String, Object?>>[],
    );
  }

  void _syncLoadedTabWithLsp(
    OpenTab tab,
    String content, {
    required bool wasAlreadyOpen,
  }) {
    final lspClient = _lspClient;
    if (lspClient == null || !_shouldUseLspForTab(tab)) {
      return;
    }

    _runLspAction(
      wasAlreadyOpen
          ? lspClient.didChangeTextDocument(
              path: tab.path,
              text: content,
            )
          : lspClient.didOpenTextDocument(
              path: tab.path,
              languageId: tab.languageId,
              text: content,
            ),
    );
  }

  void _closeLspDocumentIfNeeded(OpenTab? tab) {
    final lspClient = _lspClient;
    if (tab == null ||
        lspClient == null ||
        !_shouldUseLspForTab(tab) ||
        !lspClient.isDocumentOpen(tab.path)) {
      return;
    }

    _runLspAction(lspClient.didCloseTextDocument(path: tab.path));
  }

  void _runLspAction(Future<void> future) {
    future.catchError((Object error, StackTrace _) {
      widget.onError?.call(error.toString());
    });
  }

  Future<void> _handleInlineCompletionRequest(String requestJson) async {
    final decoded = jsonDecode(requestJson);
    if (decoded is! Map<String, Object?>) {
      return;
    }

    final kind = (decoded['kind'] as String? ?? 'request').trim().toLowerCase();
    switch (kind) {
      case 'cancel':
        await _cancelInlineCompletion(
          requestId: (decoded['requestId'] as num?)?.toInt(),
          clearGhost: false,
        );
        return;
      case 'request':
        await _startInlineCompletion(decoded);
        return;
      default:
        return;
    }
  }

  Future<void> _startInlineCompletion(Map<String, Object?> request) async {
    final requestId = (request['requestId'] as num?)?.toInt();
    final prefix = request['prefix'];
    final suffix = request['suffix'];
    final activeTab = _tabsNotifier.currentState.activeTab;
    if (requestId == null ||
        prefix is! String ||
        suffix is! String ||
        activeTab == null ||
        activeTab.readOnly ||
        !_isWorkspacePath(activeTab.path)) {
      _clearInlineCompletion();
      return;
    }

    await _cancelInlineCompletion(clearGhost: false);

    final ownedService = widget.aiService == null
        ? CortadoAIService(
            baseUrl: _manager.baseUrl,
            authSession: _manager.authSession,
            devToken: _manager.devToken,
          )
        : null;
    final service = widget.aiService ?? ownedService!;
    _ownedInlineCompletionService = ownedService;
    _activeInlineCompletionRequestId = requestId;

    final buffer = StringBuffer();
    _inlineCompletionSubscription = service
        .streamCompletion(
      CortadoCompletionContext(
        workspaceId: _workspaceId,
        path: activeTab.path,
        prefix: prefix,
        suffix: suffix,
      ),
    )
        .listen(
      (token) {
        if (!mounted || _activeInlineCompletionRequestId != requestId) {
          return;
        }

        buffer.write(token);
        final ghostText = _trimInlineCompletionPrefixOverlap(
          completion: buffer.toString(),
          prefix: prefix,
        );
        if (ghostText.isEmpty) {
          return;
        }

        widget.platform.setInlineCompletion(
          _editorId,
          requestId: requestId,
          text: ghostText,
        );
      },
      onError: (Object error, StackTrace _) async {
        if (_activeInlineCompletionRequestId != requestId) {
          return;
        }

        await _cancelInlineCompletion(
          requestId: requestId,
          clearGhost: true,
        );
        widget.onError?.call(error.toString());
      },
      onDone: () {
        if (_activeInlineCompletionRequestId != requestId) {
          return;
        }
        unawaited(_finishInlineCompletionStream(requestId));
      },
      cancelOnError: false,
    );
  }

  Future<void> _finishInlineCompletionStream(int requestId) async {
    if (_activeInlineCompletionRequestId != requestId) {
      return;
    }

    _activeInlineCompletionRequestId = null;
    _inlineCompletionSubscription = null;
    await _disposeOwnedInlineCompletionService();
  }

  Future<void> _cancelInlineCompletion({
    int? requestId,
    required bool clearGhost,
  }) async {
    if (requestId != null &&
        _activeInlineCompletionRequestId != null &&
        _activeInlineCompletionRequestId != requestId) {
      return;
    }

    _activeInlineCompletionRequestId = null;

    final subscription = _inlineCompletionSubscription;
    _inlineCompletionSubscription = null;
    unawaited(subscription?.cancel());
    await _disposeOwnedInlineCompletionService();

    if (clearGhost) {
      widget.platform.clearInlineCompletion(_editorId);
    }
  }

  void _clearInlineCompletion() {
    unawaited(
      _cancelInlineCompletion(
        clearGhost: true,
      ),
    );
  }

  Future<void> _disposeOwnedInlineCompletionService() async {
    final service = _ownedInlineCompletionService;
    _ownedInlineCompletionService = null;
    await service?.dispose();
  }

  String _trimInlineCompletionPrefixOverlap({
    required String completion,
    required String prefix,
  }) {
    if (completion.isEmpty || prefix.isEmpty) {
      return completion;
    }

    final maxOverlap =
        completion.length < prefix.length ? completion.length : prefix.length;
    for (var overlap = maxOverlap; overlap > 0; overlap -= 1) {
      if (prefix.endsWith(completion.substring(0, overlap))) {
        return completion.substring(overlap);
      }
    }
    return completion;
  }

  Future<void> _handleLspRequest(String requestJson) async {
    final decoded = jsonDecode(requestJson);
    if (decoded is! Map<String, Object?>) {
      return;
    }

    final requestId = (decoded['requestId'] as num?)?.toInt();
    if (requestId == null) {
      return;
    }

    final requestKind = _lspRequestKind(decoded);
    try {
      final result = switch (requestKind) {
        _CortadoEditorLspRequestKind.hover => await _hoverResultForRequest(
            decoded,
          ),
        _CortadoEditorLspRequestKind.definition =>
          await _definitionResultForRequest(decoded),
        _CortadoEditorLspRequestKind.completion =>
          await _completionItemsForRequest(decoded),
      };
      widget.platform.resolveLspResult(requestId, result);
    } catch (error) {
      widget.onError?.call(error.toString());
      widget.platform.resolveLspResult(
        requestId,
        switch (requestKind) {
          _CortadoEditorLspRequestKind.hover => null,
          _CortadoEditorLspRequestKind.definition => null,
          _CortadoEditorLspRequestKind.completion =>
            const <Map<String, Object?>>[],
        },
      );
    }
  }

  Future<List<Map<String, Object?>>> _completionItemsForRequest(
    Map<String, Object?> request,
  ) async {
    final activeTab = _tabsNotifier.currentState.activeTab;
    final lspClient = _lspClient;
    final position = _requestPosition(request);
    if (activeTab == null ||
        lspClient == null ||
        !_shouldUseLspForTab(activeTab) ||
        position == null) {
      return const <Map<String, Object?>>[];
    }

    final response = await lspClient.sendRequest(
      'textDocument/completion',
      params: <String, Object?>{
        'textDocument': <String, Object?>{
          'uri': workspaceDocumentUriForPath(activeTab.path),
        },
        'position': <String, Object?>{
          'line': position.line,
          'character': position.character,
        },
      },
    );
    return _mapCompletionItems(response);
  }

  Future<Map<String, Object?>?> _hoverResultForRequest(
    Map<String, Object?> request,
  ) async {
    final activeTab = _tabsNotifier.currentState.activeTab;
    final lspClient = _lspClient;
    final position = _requestPosition(request);
    if (activeTab == null ||
        lspClient == null ||
        !_shouldUseLspForTab(activeTab) ||
        position == null) {
      return null;
    }

    final response = await lspClient.sendRequest(
      'textDocument/hover',
      params: <String, Object?>{
        'textDocument': <String, Object?>{
          'uri': workspaceDocumentUriForPath(activeTab.path),
        },
        'position': <String, Object?>{
          'line': position.line,
          'character': position.character,
        },
      },
    );
    return _hoverPayloadFromResponse(response);
  }

  Future<Map<String, Object?>?> _definitionResultForRequest(
    Map<String, Object?> request,
  ) async {
    final activeTab = _tabsNotifier.currentState.activeTab;
    final lspClient = _lspClient;
    final position = _requestPosition(request);
    if (activeTab == null ||
        lspClient == null ||
        !_shouldUseLspForTab(activeTab) ||
        position == null) {
      return null;
    }

    final response = await lspClient.sendRequest(
      'textDocument/definition',
      params: <String, Object?>{
        'textDocument': <String, Object?>{
          'uri': workspaceDocumentUriForPath(activeTab.path),
        },
        'position': <String, Object?>{
          'line': position.line,
          'character': position.character,
        },
      },
    );
    final target = _definitionTargetFromResponse(response);
    if (target == null) {
      return null;
    }
    final selection = _definitionSelectionFromResponse(response);

    if (target.isWorkspacePath) {
      await _activatePath(
        target.path,
        preserveSelection: false,
        reload: false,
      );
      return _definitionResultPayload(selection);
    }

    await _openReadOnlyExternalTab(
      target.path,
      content: _readOnlyExternalContent(target.path),
    );
    return _definitionResultPayload(selection);
  }

  List<Map<String, Object?>> _mapCompletionItems(Object? response) {
    final items = switch (response) {
      final List<Object?> entries => entries,
      final Map<Object?, Object?> payload
          when payload['items'] is List<Object?> =>
        payload['items'] as List<Object?>,
      _ => const <Object?>[],
    };

    return items
        .whereType<Map<Object?, Object?>>()
        .map(_mapCompletionItem)
        .whereType<Map<String, Object?>>()
        .toList(growable: false);
  }

  Map<String, Object?>? _mapCompletionItem(Map<Object?, Object?> item) {
    final label = item['label'];
    if (label is! String || label.isEmpty) {
      return null;
    }

    final mapped = <String, Object?>{
      'label': label,
    };
    final detail = item['detail'];
    if (detail is String && detail.isNotEmpty) {
      mapped['detail'] = detail;
    }

    final type = _completionTypeForKind((item['kind'] as num?)?.toInt());
    if (type != null) {
      mapped['type'] = type;
    }

    final apply = _completionApplyText(item);
    if (apply != null && apply != label) {
      mapped['apply'] = apply;
    }
    return mapped;
  }

  String? _completionApplyText(Map<Object?, Object?> item) {
    final textEdit = item['textEdit'];
    if (textEdit is Map<Object?, Object?>) {
      final newText = textEdit['newText'];
      if (newText is String && newText.isNotEmpty) {
        return newText;
      }
    }

    final insertText = item['insertText'];
    if (insertText is String && insertText.isNotEmpty) {
      return insertText;
    }
    return null;
  }

  String? _completionTypeForKind(int? kind) {
    return switch (kind) {
      2 || 3 => 'function',
      4 => 'class',
      5 || 10 => 'property',
      6 => 'variable',
      7 => 'class',
      8 => 'interface',
      9 => 'namespace',
      11 => 'unit',
      12 => 'text',
      13 => 'enum',
      14 => 'keyword',
      15 => 'snippet',
      16 => 'constant',
      17 => 'class',
      18 => 'constant',
      19 => 'constant',
      20 => 'enum',
      21 => 'constant',
      22 => 'struct',
      23 => 'function',
      24 => 'operator',
      25 => 'type',
      _ => null,
    };
  }

  Future<void> _openReadOnlyExternalTab(
    String path, {
    required String content,
  }) async {
    final normalizedPath = normalizeVfsPath(path);
    final previousActivePath = _tabsNotifier.currentState.activePath;
    if (previousActivePath != null && previousActivePath != normalizedPath) {
      _tabsNotifier.updateContent(
        previousActivePath,
        widget.platform.getContent(_editorId),
      );
    }

    final existing = _tabsNotifier.tabForPath(normalizedPath);
    if (existing != null && existing.loaded) {
      _clearInlineCompletion();
      _tabsNotifier.activate(normalizedPath);
      _renderTab(existing, preserveSelection: false);
      return;
    }

    _tabsNotifier.open(
      normalizedPath,
      readOnly: true,
    );
    final hash = _tabsNotifier.currentState.activePath == normalizedPath
        ? _renderLoadedContent(
            normalizedPath,
            content,
            preserveSelection: false,
            readOnly: true,
          )
        : hashEditorContent(content);
    _tabsNotifier.setLoaded(
      normalizedPath,
      content: content,
      hash: hash,
      readOnly: true,
    );
    final tab = _tabsNotifier.tabForPath(normalizedPath);
    if (tab != null) {
      _renderTab(tab, preserveSelection: false);
    }
  }

  _EditorRequestPosition? _requestPosition(Map<String, Object?> request) {
    final position = request['position'];
    final line = switch (position) {
      final Map<Object?, Object?> payload => (payload['line'] as num?)?.toInt(),
      _ => (request['line'] as num?)?.toInt(),
    };
    final character = switch (position) {
      final Map<Object?, Object?> payload =>
        (payload['character'] as num?)?.toInt(),
      _ => (request['character'] as num?)?.toInt(),
    };
    if (line == null || character == null) {
      return null;
    }
    return _EditorRequestPosition(line: line, character: character);
  }

  _CortadoEditorLspRequestKind _lspRequestKind(Map<String, Object?> request) {
    final rawKind =
        request['kind'] ?? request['type'] ?? request['method'] ?? 'completion';
    final normalizedKind = rawKind.toString().trim().toLowerCase();
    return switch (normalizedKind) {
      'hover' || 'textdocument/hover' => _CortadoEditorLspRequestKind.hover,
      'definition' ||
      'goto-definition' ||
      'gotodefinition' ||
      'textdocument/definition' =>
        _CortadoEditorLspRequestKind.definition,
      _ => _CortadoEditorLspRequestKind.completion,
    };
  }

  Map<String, Object?>? _hoverPayloadFromResponse(Object? response) {
    if (response is! Map<Object?, Object?>) {
      return null;
    }
    final markdown = _hoverContentsToMarkdown(response['contents']);
    if (markdown != null) {
      return <String, Object?>{'markdown': markdown};
    }
    return null;
  }

  String? _hoverContentsToMarkdown(Object? contents) {
    switch (contents) {
      case null:
        return null;
      case final String text when text.trim().isNotEmpty:
        return text.trim();
      case final List<Object?> entries:
        final segments = entries
            .map(_hoverContentsToMarkdown)
            .whereType<String>()
            .map((entry) => entry.trim())
            .where((entry) => entry.isNotEmpty)
            .toList(growable: false);
        return segments.isEmpty ? null : segments.join('\n\n');
      case final Map<Object?, Object?> payload:
        return _hoverMarkupFromMap(payload);
      default:
        return null;
    }
  }

  String? _hoverMarkupFromMap(Map<Object?, Object?> payload) {
    final kind = payload['kind'];
    final value = payload['value'];
    if (kind is String && value is String && value.trim().isNotEmpty) {
      return value.trim();
    }

    final language = payload['language'];
    if (language is String && language.isNotEmpty && value is String) {
      return '```$language\n${value.trim()}\n```';
    }
    return null;
  }

  _DefinitionTarget? _definitionTargetFromResponse(Object? response) {
    final payload = switch (response) {
      final List<Object?> entries when entries.isNotEmpty => entries.first,
      final Map<Object?, Object?> map => map,
      _ => null,
    };
    if (payload is! Map<Object?, Object?>) {
      return null;
    }

    final uri = payload['targetUri'] ?? payload['uri'];
    if (uri is! String || uri.trim().isEmpty) {
      return null;
    }
    final workspacePath = workspacePathFromDocumentUri(uri);
    if (workspacePath != null) {
      return _DefinitionTarget(path: workspacePath, isWorkspacePath: true);
    }

    final parsedUri = Uri.tryParse(uri);
    if (parsedUri == null || parsedUri.scheme != 'file') {
      return null;
    }
    return _DefinitionTarget(
      path: normalizeVfsPath(parsedUri.path),
      isWorkspacePath: false,
    );
  }

  _DefinitionSelection? _definitionSelectionFromResponse(Object? response) {
    final payload = switch (response) {
      final List<Object?> entries when entries.isNotEmpty => entries.first,
      final Map<Object?, Object?> map => map,
      _ => null,
    };
    if (payload is! Map<Object?, Object?>) {
      return null;
    }

    final range = payload['targetSelectionRange'] ??
        payload['range'] ??
        payload['targetRange'];
    if (range is! Map<Object?, Object?>) {
      return null;
    }
    final start = range['start'];
    if (start is! Map<Object?, Object?>) {
      return null;
    }
    final line = (start['line'] as num?)?.toInt();
    final character = (start['character'] as num?)?.toInt();
    if (line == null || character == null) {
      return null;
    }
    return _DefinitionSelection(line: line, character: character);
  }

  Map<String, Object?> _definitionResultPayload(
    _DefinitionSelection? selection,
  ) {
    return <String, Object?>{
      'editorId': _editorId,
      if (selection != null)
        'range': <String, Object?>{
          'start': <String, Object?>{
            'line': selection.line,
            'character': selection.character,
          },
        },
    };
  }

  bool _isWorkspacePath(String path) => !_isReadOnlyExternalPath(path);

  bool _isReadOnlyExternalPath(String path) {
    final normalizedPath = normalizeVfsPath(path);
    return normalizedPath == _dartSdkRoot ||
        normalizedPath.startsWith('$_dartSdkRoot/');
  }

  String _readOnlyExternalContent(String path) {
    if (_isReadOnlyExternalPath(path)) {
      return '// Read-only SDK definition target.\n'
          '// SDK source loading is not yet wired through the workspace file API.\n'
          '// Target: $path\n';
    }
    return '// Read-only definition target.\n// Target: $path\n';
  }
}

enum _CortadoEditorLspRequestKind {
  completion,
  hover,
  definition,
}

class _EditorRequestPosition {
  const _EditorRequestPosition({
    required this.line,
    required this.character,
  });

  final int line;
  final int character;
}

class _DefinitionTarget {
  const _DefinitionTarget({
    required this.path,
    required this.isWorkspacePath,
  });

  final String path;
  final bool isWorkspacePath;
}

class _DefinitionSelection {
  const _DefinitionSelection({
    required this.line,
    required this.character,
  });

  final int line;
  final int character;
}

class _EditorTabStrip extends StatelessWidget {
  const _EditorTabStrip({
    required this.activePath,
    required this.diagnosticStatusesByPath,
    required this.tabs,
    required this.onClose,
    required this.onSelect,
  });

  final String? activePath;
  final Map<String, CortadoFileDiagnosticStatus> diagnosticStatusesByPath;
  final List<OpenTab> tabs;
  final Future<void> Function(String path) onClose;
  final Future<void> Function(String path) onSelect;

  @override
  Widget build(BuildContext context) {
    return Container(
      height: 44,
      decoration: const BoxDecoration(
        border: Border(
          bottom: BorderSide(color: Color(0x1F94A3B8)),
        ),
      ),
      child: ListView.separated(
        itemCount: tabs.length,
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
        scrollDirection: Axis.horizontal,
        separatorBuilder: (_, __) => const SizedBox(width: 8),
        itemBuilder: (context, index) {
          final tab = tabs[index];
          final diagnosticStatus = diagnosticStatusesByPath[tab.path] ??
              CortadoFileDiagnosticStatus.none;
          final selected = tab.path == activePath;
          return InkWell(
            borderRadius: BorderRadius.circular(10),
            onTap: () => unawaited(onSelect(tab.path)),
            child: Container(
              padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
              decoration: BoxDecoration(
                color: selected
                    ? const Color(0xFF162235)
                    : const Color(0xFF101827),
                border: Border.all(
                  color: selected
                      ? const Color(0xFF2F6FEB)
                      : const Color(0x1F94A3B8),
                ),
                borderRadius: BorderRadius.circular(10),
              ),
              child: Row(
                mainAxisSize: MainAxisSize.min,
                children: <Widget>[
                  if (diagnosticStatus != CortadoFileDiagnosticStatus.none)
                    Padding(
                      padding: const EdgeInsets.only(right: 8),
                      child: _DiagnosticStatusDot(
                        key: ValueKey('editor-diagnostic-dot:${tab.path}'),
                        status: diagnosticStatus,
                      ),
                    ),
                  if (tab.isDirty)
                    const Padding(
                      padding: EdgeInsets.only(right: 8),
                      child: _DirtyDot(),
                    ),
                  Text(
                    tab.title,
                    style: const TextStyle(
                      color: Color(0xFFE2E8F0),
                      fontSize: 13,
                      fontWeight: FontWeight.w600,
                    ),
                  ),
                  const SizedBox(width: 8),
                  InkWell(
                    borderRadius: BorderRadius.circular(999),
                    onTap: () => unawaited(onClose(tab.path)),
                    child: const Padding(
                      padding: EdgeInsets.all(2),
                      child: Icon(
                        Icons.close,
                        size: 14,
                        color: Color(0xFF94A3B8),
                      ),
                    ),
                  ),
                ],
              ),
            ),
          );
        },
      ),
    );
  }
}

class _DirtyDot extends StatelessWidget {
  const _DirtyDot();

  @override
  Widget build(BuildContext context) {
    return const DecoratedBox(
      decoration: BoxDecoration(
        color: Color(0xFFFACC15),
        shape: BoxShape.circle,
      ),
      child: SizedBox(width: 8, height: 8),
    );
  }
}

class _DiagnosticStatusDot extends StatelessWidget {
  const _DiagnosticStatusDot({
    required this.status,
    super.key,
  });

  final CortadoFileDiagnosticStatus status;

  @override
  Widget build(BuildContext context) {
    final color = switch (status) {
      CortadoFileDiagnosticStatus.error => const Color(0xFFEF4444),
      CortadoFileDiagnosticStatus.warning => const Color(0xFFF59E0B),
      CortadoFileDiagnosticStatus.none => Colors.transparent,
    };

    return DecoratedBox(
      decoration: BoxDecoration(
        color: color,
        shape: BoxShape.circle,
      ),
      child: const SizedBox(width: 8, height: 8),
    );
  }
}

class _EditorEmptyState extends StatelessWidget {
  const _EditorEmptyState();

  @override
  Widget build(BuildContext context) {
    return const ColoredBox(
      color: Color(0xFF0B1220),
      child: Center(
        child: Text(
          'Open a file to start editing.',
          style: TextStyle(
            color: Color(0xFF94A3B8),
            fontSize: 14,
          ),
        ),
      ),
    );
  }
}

class _EditorOverlay extends StatelessWidget {
  const _EditorOverlay({
    required this.message,
    this.actionLabel,
    this.onAction,
  });

  final String? actionLabel;
  final String message;
  final VoidCallback? onAction;

  @override
  Widget build(BuildContext context) {
    return ColoredBox(
      color: const Color(0xDD0B1220),
      child: Center(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: <Widget>[
            Text(
              message,
              textAlign: TextAlign.center,
              style: const TextStyle(
                color: Color(0xFFE2E8F0),
                fontSize: 14,
              ),
            ),
            if (actionLabel != null && onAction != null) ...<Widget>[
              const SizedBox(height: 12),
              OutlinedButton(
                onPressed: onAction,
                child: Text(actionLabel!),
              ),
            ],
          ],
        ),
      ),
    );
  }
}

class _SavingBadge extends StatelessWidget {
  const _SavingBadge();

  @override
  Widget build(BuildContext context) {
    return DecoratedBox(
      decoration: BoxDecoration(
        color: const Color(0xE6101827),
        border: Border.all(color: const Color(0x1F94A3B8)),
        borderRadius: BorderRadius.circular(999),
      ),
      child: const Padding(
        padding: EdgeInsets.symmetric(horizontal: 10, vertical: 6),
        child: Text(
          'Saving...',
          style: TextStyle(
            color: Color(0xFFE2E8F0),
            fontSize: 12,
            fontWeight: FontWeight.w600,
          ),
        ),
      ),
    );
  }
}
