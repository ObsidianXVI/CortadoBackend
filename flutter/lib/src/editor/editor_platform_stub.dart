import 'package:flutter/widgets.dart';

typedef CortadoEditorChangedCallback = void Function(String hash);
typedef CortadoEditorLspRequestCallback = void Function(String requestJson);
typedef CortadoEditorSaveCallback = void Function();

bool get supportsCortadoEditorPlatformView => false;

void registerCortadoEditorViewFactory({
  required String viewType,
  required String editorId,
  required String languageId,
  required CortadoEditorChangedCallback onChanged,
  required CortadoEditorSaveCallback onSave,
}) {}

Widget buildCortadoEditorPlatformView(String viewType) {
  return const Center(
    child:
        Text('CortadoCodeEditor is currently supported on Flutter Web only.'),
  );
}

String setCortadoEditorContent(
  String editorId,
  String content, {
  bool preserveSelection = false,
}) {
  return '';
}

String getCortadoEditorContent(String editorId) => '';

void setCortadoEditorLanguage(String editorId, String languageId) {}

void disposeCortadoEditorView(String editorId) {}

void registerCortadoEditorLspRequestHandler({
  required String editorId,
  required CortadoEditorLspRequestCallback onRequest,
}) {}

void unregisterCortadoEditorLspRequestHandler(String editorId) {}

void resolveCortadoEditorLspResult(
  int requestId,
  List<Map<String, Object?>> items,
) {}

void setCortadoEditorDiagnostics(
  String editorId,
  List<Map<String, Object?>> diagnostics,
) {}
