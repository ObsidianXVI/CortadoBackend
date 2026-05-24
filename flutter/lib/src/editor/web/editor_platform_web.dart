import 'dart:convert';
import 'dart:js_interop';
import 'dart:ui_web' as ui_web;

import 'package:flutter/widgets.dart';
import 'package:web/web.dart' as web;

typedef CortadoEditorChangedCallback = void Function(String hash);
typedef CortadoEditorLspRequestCallback = void Function(String requestJson);
typedef CortadoEditorSaveCallback = void Function();

final Set<String> _registeredViewTypes = <String>{};
final Map<String, CortadoEditorLspRequestCallback> _lspRequestHandlers =
    <String, CortadoEditorLspRequestCallback>{};
final Map<int, String> _lspRequestKindsById = <int, String>{};
bool _didRegisterLspBridge = false;

@JS('CortadoEditor.init')
external void _editorInit(
  web.HTMLDivElement container,
  JSString editorId,
  JSString languageId,
  JSFunction onChanged,
  JSFunction onSave,
);

@JS('CortadoEditor.setContent')
external JSString _editorSetContent(
  JSString editorId,
  JSString content,
  JSBoolean preserveSelection,
);

@JS('CortadoEditor.getContent')
external JSString _editorGetContent(JSString editorId);

@JS('CortadoEditor.setLanguage')
external void _editorSetLanguage(JSString editorId, JSString languageId);

@JS('CortadoEditor.dispose')
external void _editorDispose(JSString editorId);

@JS('CortadoEditor.setDiagnostics')
external void _editorSetDiagnostics(JSString editorId, JSAny diagnostics);

@JS('CortadoEditor.setReadOnly')
external void _editorSetReadOnly(JSString editorId, JSBoolean readOnly);

@JS('window._cortadoLSPRequest')
external set _editorLspRequestHandler(JSFunction? handler);

@JS('window._cortadoLSPHoverRequest')
external set _editorLspHoverRequestHandler(JSFunction? handler);

@JS('window._cortadoLSPDefinitionRequest')
external set _editorLspDefinitionRequestHandler(JSFunction? handler);

@JS('window._cortadoLSPResult')
external void _editorLspResult(JSNumber requestId, JSAny result);

@JS('window._cortadoLSPHoverResult')
external void _editorLspHoverResult(JSNumber requestId, JSAny result);

@JS('window._cortadoLSPDefinitionResult')
external void _editorLspDefinitionResult(JSNumber requestId, JSAny result);

@JS('JSON.parse')
external JSAny _jsonParse(JSString input);

@JS('JSON.stringify')
external JSString _jsonStringify(JSAny input);

bool get supportsCortadoEditorPlatformView => true;

void registerCortadoEditorViewFactory({
  required String viewType,
  required String editorId,
  required String languageId,
  required CortadoEditorChangedCallback onChanged,
  required CortadoEditorSaveCallback onSave,
}) {
  if (_registeredViewTypes.contains(viewType)) {
    return;
  }

  ui_web.platformViewRegistry.registerViewFactory(viewType, (int viewId) {
    final container = web.HTMLDivElement()
      ..id = 'cortado-editor-$viewId'
      ..style.width = '100%'
      ..style.height = '100%'
      ..style.backgroundColor = '#0B1220';

    _editorInit(
      container,
      editorId.toJS,
      languageId.toJS,
      ((JSString hash) {
        onChanged(hash.toDart);
      }).toJS,
      (() {
        onSave();
      }).toJS,
    );

    return container;
  });

  _registeredViewTypes.add(viewType);
}

Widget buildCortadoEditorPlatformView(String viewType) {
  return HtmlElementView(viewType: viewType);
}

String setCortadoEditorContent(
  String editorId,
  String content, {
  bool preserveSelection = false,
}) {
  return _editorSetContent(
    editorId.toJS,
    content.toJS,
    preserveSelection.toJS,
  ).toDart;
}

String getCortadoEditorContent(String editorId) {
  return _editorGetContent(editorId.toJS).toDart;
}

void setCortadoEditorLanguage(String editorId, String languageId) {
  _editorSetLanguage(editorId.toJS, languageId.toJS);
}

void disposeCortadoEditorView(String editorId) {
  _editorDispose(editorId.toJS);
}

void registerCortadoEditorLspRequestHandler({
  required String editorId,
  required CortadoEditorLspRequestCallback onRequest,
}) {
  _lspRequestHandlers[editorId] = onRequest;
  if (_didRegisterLspBridge) {
    return;
  }

  _editorLspRequestHandler = ((JSAny request) {
    _dispatchLspRequest(request, fallbackKind: 'completion');
  }).toJS;
  _editorLspHoverRequestHandler = ((JSAny request) {
    _dispatchLspRequest(request, fallbackKind: 'hover');
  }).toJS;
  _editorLspDefinitionRequestHandler = ((JSAny request) {
    _dispatchLspRequest(request, fallbackKind: 'definition');
  }).toJS;
  _didRegisterLspBridge = true;
}

void unregisterCortadoEditorLspRequestHandler(String editorId) {
  _lspRequestHandlers.remove(editorId);
  if (_lspRequestHandlers.isNotEmpty || !_didRegisterLspBridge) {
    return;
  }

  _editorLspRequestHandler = null;
  _editorLspHoverRequestHandler = null;
  _editorLspDefinitionRequestHandler = null;
  _didRegisterLspBridge = false;
}

void resolveCortadoEditorLspResponse(int requestId, Object? result) {
  final jsResult = _jsonParse(jsonEncode(result).toJS);
  final requestKind = _lspRequestKindsById.remove(requestId) ?? 'completion';
  switch (requestKind) {
    case 'hover':
      _editorLspHoverResult(requestId.toJS, jsResult);
      return;
    case 'definition':
      _editorLspDefinitionResult(requestId.toJS, jsResult);
      return;
    default:
      _editorLspResult(requestId.toJS, jsResult);
      return;
  }
}

void setCortadoEditorDiagnostics(
  String editorId,
  List<Map<String, Object?>> diagnostics,
) {
  _editorSetDiagnostics(
    editorId.toJS,
    _jsonParse(jsonEncode(diagnostics).toJS),
  );
}

void setCortadoEditorReadOnly(String editorId, bool readOnly) {
  _editorSetReadOnly(editorId.toJS, readOnly.toJS);
}

void _dispatchLspRequest(
  JSAny request, {
  required String fallbackKind,
}) {
  final requestJson = _jsonStringify(request).toDart;
  final decoded = jsonDecode(requestJson);
  if (decoded is! Map<String, Object?>) {
    return;
  }

  final editorId = decoded['editorId'];
  if (editorId is! String) {
    return;
  }

  final requestId = (decoded['requestId'] as num?)?.toInt();
  final kind = (decoded['kind'] as String?) ?? fallbackKind;
  if (requestId != null) {
    _lspRequestKindsById[requestId] = kind;
  }
  _lspRequestHandlers[editorId]?.call(
    jsonEncode(<String, Object?>{
      ...decoded,
      'kind': kind,
    }),
  );
}
