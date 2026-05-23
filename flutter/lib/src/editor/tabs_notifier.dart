import 'package:riverpod/riverpod.dart';

import '../filesystem/vfs_notifier.dart';
import 'editor_models.dart';

class TabsNotifier extends StateNotifier<TabsState> {
  TabsNotifier({this.maxTabs = 15}) : super(const TabsState());

  final int maxTabs;

  TabsState get currentState => state;

  OpenTab? tabForPath(String path) {
    final normalizedPath = normalizeVfsPath(path);
    for (final tab in state.tabs) {
      if (tab.path == normalizedPath) {
        return tab;
      }
    }
    return null;
  }

  void open(String path) {
    final normalizedPath = normalizeVfsPath(path);
    final existing = tabForPath(normalizedPath);
    if (existing != null) {
      activate(normalizedPath);
      return;
    }

    final nextTab = OpenTab(
      path: normalizedPath,
      title: _tabTitleForPath(normalizedPath),
      languageId: editorLanguageIdForPath(normalizedPath),
      isLoading: true,
    );
    final nextTabs = <OpenTab>[
      ...state.tabs,
      nextTab,
    ];
    while (nextTabs.length > maxTabs) {
      nextTabs.removeAt(0);
    }

    state = state.copyWith(
      tabs: nextTabs,
      activePath: normalizedPath,
    );
  }

  void activate(String path) {
    final normalizedPath = normalizeVfsPath(path);
    if (tabForPath(normalizedPath) == null) {
      return;
    }
    state = state.copyWith(activePath: normalizedPath);
  }

  void close(String path) {
    final normalizedPath = normalizeVfsPath(path);
    final nextTabs = state.tabs
        .where((tab) => tab.path != normalizedPath)
        .toList(growable: false);
    final nextActivePath = switch (state.activePath == normalizedPath) {
      true when nextTabs.isNotEmpty => nextTabs.last.path,
      true => null,
      false => state.activePath,
    };
    state = state.copyWith(
      tabs: nextTabs,
      activePath: nextActivePath,
    );
  }

  void updateContent(String path, String content) {
    _patchTab(path, (tab) => tab.copyWith(content: content));
  }

  void markLoading(String path) {
    _patchTab(
      path,
      (tab) => tab.copyWith(
        isLoading: true,
        errorMessage: null,
      ),
    );
  }

  void setLoaded(
    String path, {
    required String content,
    required String hash,
  }) {
    final normalizedPath = normalizeVfsPath(path);
    _patchTab(
      normalizedPath,
      (tab) => tab.copyWith(
        content: content,
        currentHash: hash,
        errorMessage: null,
        isLoading: false,
        languageId: editorLanguageIdForPath(normalizedPath),
        loaded: true,
        savedHash: hash,
        title: _tabTitleForPath(normalizedPath),
      ),
    );
  }

  void markHash(String path, String hash) {
    _patchTab(path, (tab) => tab.copyWith(currentHash: hash));
  }

  void markSaving(String path, bool isSaving) {
    _patchTab(path, (tab) => tab.copyWith(isSaving: isSaving));
  }

  void markSaved(
    String path, {
    required String content,
    required String hash,
  }) {
    _patchTab(
      path,
      (tab) => tab.copyWith(
        content: content,
        currentHash: hash,
        errorMessage: null,
        isSaving: false,
        loaded: true,
        savedHash: hash,
      ),
    );
  }

  void setError(String path, String message) {
    _patchTab(
      path,
      (tab) => tab.copyWith(
        errorMessage: message,
        isLoading: false,
        isSaving: false,
      ),
    );
  }

  void _patchTab(String path, OpenTab Function(OpenTab tab) patch) {
    final normalizedPath = normalizeVfsPath(path);
    state = state.copyWith(
      tabs: state.tabs
          .map(
            (tab) => tab.path == normalizedPath ? patch(tab) : tab,
          )
          .toList(growable: false),
    );
  }
}

String editorLanguageIdForPath(String path) {
  final normalizedPath = normalizeVfsPath(path);
  final dotIndex = normalizedPath.lastIndexOf('.');
  if (dotIndex < 0 || dotIndex == normalizedPath.length - 1) {
    return 'plain';
  }

  final extension = normalizedPath.substring(dotIndex + 1).toLowerCase();
  return switch (extension) {
    'dart' => 'dart',
    'js' || 'mjs' || 'cjs' || 'ts' || 'jsx' || 'tsx' => 'javascript',
    'py' => 'python',
    'go' => 'go',
    'yaml' || 'yml' => 'yaml',
    'json' => 'json',
    _ => 'plain',
  };
}

String hashEditorContent(String content) {
  var hash = 0x811C9DC5;
  for (final codeUnit in content.codeUnits) {
    hash ^= codeUnit;
    hash = (hash * 0x01000193) & 0xFFFFFFFF;
  }
  return hash.toRadixString(16).padLeft(8, '0');
}

String _tabTitleForPath(String path) {
  final normalizedPath = normalizeVfsPath(path);
  if (normalizedPath == vfsRootPath) {
    return '/';
  }

  final slashIndex = normalizedPath.lastIndexOf('/');
  return slashIndex < 0
      ? normalizedPath
      : normalizedPath.substring(slashIndex + 1);
}
