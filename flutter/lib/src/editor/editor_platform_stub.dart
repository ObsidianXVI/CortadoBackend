import 'package:flutter/widgets.dart';

typedef CortadoEditorChangedCallback = void Function(String hash);
typedef CortadoEditorLspRequestCallback = void Function(String requestJson);
typedef CortadoEditorInlineCompletionRequestCallback = void Function(
  String requestJson,
);
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

void registerCortadoEditorInlineCompletionRequestHandler({
  required String editorId,
  required CortadoEditorInlineCompletionRequestCallback onRequest,
}) {}

void unregisterCortadoEditorLspRequestHandler(String editorId) {}

void unregisterCortadoEditorInlineCompletionRequestHandler(String editorId) {}

void resolveCortadoEditorLspResponse(int requestId, Object? result) {}

void setCortadoEditorDiagnostics(
  String editorId,
  List<Map<String, Object?>> diagnostics,
) {}

void setCortadoEditorReadOnly(String editorId, bool readOnly) {}

void setCortadoEditorInlineCompletion(
  String editorId, {
  required int requestId,
  required String text,
}) {}

void clearCortadoEditorInlineCompletion(String editorId) {}
