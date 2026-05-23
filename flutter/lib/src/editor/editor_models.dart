import 'package:freezed_annotation/freezed_annotation.dart';

part 'editor_models.freezed.dart';

@freezed
class OpenTab with _$OpenTab {
  const OpenTab._();

  const factory OpenTab({
    required String path,
    required String title,
    required String languageId,
    @Default('') String content,
    @Default('') String savedHash,
    @Default('') String currentHash,
    @Default(false) bool isLoading,
    @Default(false) bool isSaving,
    @Default(false) bool loaded,
    String? errorMessage,
  }) = _OpenTab;

  bool get isDirty => currentHash != savedHash;
}

@freezed
class TabsState with _$TabsState {
  const TabsState._();

  const factory TabsState({
    @Default(<OpenTab>[]) List<OpenTab> tabs,
    String? activePath,
  }) = _TabsState;

  OpenTab? get activeTab {
    final selectedPath = activePath;
    if (selectedPath == null) {
      return null;
    }
    for (final tab in tabs) {
      if (tab.path == selectedPath) {
        return tab;
      }
    }
    return null;
  }
}
