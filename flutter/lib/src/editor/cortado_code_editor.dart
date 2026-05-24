import 'dart:async';
import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../cortado_client.dart';
import '../cortado_workspace_provider.dart';
import '../filesystem/vfs_notifier.dart';
import '../gen/agent/v1/agent.pb.dart' as agentpb;
import '../mux_frame.dart';
import '../workspace_manager.dart';
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

  void setLanguage(String editorId, String languageId) {
    setCortadoEditorLanguage(editorId, languageId);
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
  final String? workspaceId;
  final WorkspaceManager? workspaceManager;

  @override
  ConsumerState<CortadoCodeEditor> createState() => _CortadoCodeEditorState();
}

class _CortadoCodeEditorState extends ConsumerState<CortadoCodeEditor> {
  static int _instanceCount = 0;

  late final String _editorId = 'cortado-editor-${_instanceCount++}';
  late final String _viewType = 'cortado-editor-view-$_editorId';
  late final TabsNotifier _tabsNotifier = TabsNotifier(maxTabs: widget.maxTabs);
  late final void Function() _removeTabsListener =
      _tabsNotifier.addListener(_handleTabsChanged, fireImmediately: false);

  final Map<String, int> _loadVersions = <String, int>{};

  StreamSubscription<agentpb.FileEvent>? _fileEventSubscription;
  StreamSubscription<void>? _lspStateSubscription;
  CortadoLSPClient? _lspClient;

  @override
  void initState() {
    super.initState();
    if (widget.platform.supportsPlatformView) {
      widget.platform.registerViewFactory(
        viewType: _viewType,
        editorId: _editorId,
        languageId: 'plain',
        onChanged: _handleEditorHashChanged,
        onSave: _handleSaveShortcut,
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
    _lspStateSubscription?.cancel();
    unawaited(_lspClient?.dispose());
    _removeTabsListener();
    _tabsNotifier.dispose();
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
    final normalizedPath = normalizeVfsPath(path);
    final previousActivePath = _tabsNotifier.currentState.activePath;
    if (previousActivePath != null && previousActivePath != normalizedPath) {
      _tabsNotifier.updateContent(
        previousActivePath,
        widget.platform.getContent(_editorId),
      );
    }

    _tabsNotifier.open(normalizedPath);
    final tab = _tabsNotifier.tabForPath(normalizedPath);
    if (tab == null) {
      return;
    }

    if (!reload && tab.loaded) {
      _tabsNotifier.activate(normalizedPath);
      _renderTab(tab, preserveSelection: false);
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
      widget.platform.setContent(_editorId, '', preserveSelection: false);
    }
  }

  Future<void> _loadTab(
    String path, {
    bool preserveSelection = false,
  }) async {
    final normalizedPath = normalizeVfsPath(path);
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
  }) {
    widget.platform.setLanguage(_editorId, editorLanguageIdForPath(path));
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
    widget.platform.setLanguage(_editorId, tab.languageId);
    final hash = widget.platform.setContent(
      _editorId,
      tab.content,
      preserveSelection: preserveSelection,
    );
    _tabsNotifier.markHash(tab.path, hash);
  }

  Future<void> _saveActiveTab() async {
    final activeTab = _tabsNotifier.currentState.activeTab;
    if (activeTab == null || activeTab.isLoading) {
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
    _lspStateSubscription?.cancel();
    _lspStateSubscription = null;
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
      tab.languageId == widget.lspLanguage && _lspClient != null;

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
}

class _EditorTabStrip extends StatelessWidget {
  const _EditorTabStrip({
    required this.activePath,
    required this.tabs,
    required this.onClose,
    required this.onSelect,
  });

  final String? activePath;
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
