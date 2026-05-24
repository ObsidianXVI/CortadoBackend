import 'dart:async';
import 'dart:convert';
import 'dart:typed_data';

import 'package:cortado/cortado.dart';
import 'package:cortado/src/editor/editor_diagnostics.dart';
import 'package:cortado/src/editor/editor_platform.dart';
import 'package:cortado/src/gen/agent/v1/agent.pbenum.dart' as agentpbenum;
import 'package:cortado/src/gen/agent/v1/agent.pb.dart' as agentpb;
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;

void main() {
  testWidgets('renders a non-web fallback when HtmlElementView is unavailable',
      (WidgetTester tester) async {
    await tester.pumpWidget(
      _wrapEditor(
        CortadoCodeEditor(
          platform: _TestEditorPlatform(supportsPlatformView: false),
          workspaceId: 'ws-123',
          workspaceManager: _TestWorkspaceManager(),
        ),
      ),
    );

    expect(
      find.text(
          'CortadoCodeEditor is currently supported on Flutter Web only.'),
      findsOneWidget,
    );
  });

  testWidgets('loads, saves, and reloads the active tab from file events',
      (WidgetTester tester) async {
    final platform = _TestEditorPlatform();
    final manager = _TestWorkspaceManager(
      files: <String, String>{
        '/lib/main.dart': 'print("hello");',
      },
    );
    final fileEvents = StreamController<agentpb.FileEvent>.broadcast();

    await tester.pumpWidget(
      _wrapEditor(
        CortadoCodeEditor(
          fileEvents: fileEvents.stream,
          path: '/lib/main.dart',
          platform: platform,
          workspaceId: 'ws-123',
          workspaceManager: manager,
        ),
      ),
    );

    await tester.pump();
    await tester.pump();

    expect(platform.currentContent, 'print("hello");');

    platform.currentContent = 'print("updated");';
    platform.emitChanged();
    await tester.pump();

    platform.triggerSave();
    await tester.pump();
    await tester.pump();

    expect(manager.writes.single.path, '/lib/main.dart');
    expect(manager.files['/lib/main.dart'], 'print("updated");');

    manager.files['/lib/main.dart'] = 'print("server");';
    fileEvents.add(
      agentpb.FileEvent(
        path: '/lib/main.dart',
        type: agentpbenum.FileEventType.FILE_EVENT_TYPE_MODIFIED,
      ),
    );

    await tester.pump();
    await tester.pump();

    expect(platform.currentContent, 'print("server");');
    expect(platform.lastPreserveSelection, isTrue);

    await fileEvents.close();
  });

  testWidgets('streams inline completion tokens into the active editor',
      (WidgetTester tester) async {
    final platform = _TestEditorPlatform();
    final manager = _TestWorkspaceManager(
      files: <String, String>{
        '/lib/main.dart': 'print();',
      },
    );
    final requests = <_RecordedHttpRequest>[];
    final responseStream = StreamController<List<int>>();
    final aiService = CortadoAIService(
      baseUrl: 'http://localhost:8080',
      httpClient: _RecordingHttpClient((request, bodyBytes) async {
        requests.add(_RecordedHttpRequest(request, bodyBytes));
        return http.StreamedResponse(
          responseStream.stream,
          200,
          headers: const <String, String>{
            'Content-Type': 'text/event-stream',
          },
        );
      }),
    );

    await tester.pumpWidget(
      _wrapEditor(
        CortadoCodeEditor(
          aiService: aiService,
          path: '/lib/main.dart',
          platform: platform,
          workspaceId: 'ws-123',
          workspaceManager: manager,
        ),
      ),
    );

    await tester.pump();
    await tester.pump();

    platform.emitInlineCompletionRequest(
      <String, Object?>{
        'kind': 'request',
        'requestId': 7,
        'prefix': 'print(',
        'suffix': ');',
      },
    );
    await tester.pump();

    final request = requests.single.request as http.Request;
    final payload = jsonDecode(utf8.decode(requests.single.bodyBytes))
        as Map<String, dynamic>;
    expect(request.url.path, '/v1/workspaces/ws-123/ai/complete');
    expect(payload['path'], '/lib/main.dart');
    expect(payload['prefix'], 'print(');
    expect(payload['suffix'], ');');

    responseStream.add(
      Uint8List.fromList(utf8.encode('data: {"token":"hel"}\n\n')),
    );
    await tester.pump();
    await tester.pump();

    expect(platform.lastInlineCompletionRequestId, 7);
    expect(platform.lastInlineCompletionText, 'hel');

    responseStream.add(
      Uint8List.fromList(utf8.encode('data: {"token":"lo"}\n\n')),
    );
    await tester.pump();
    await tester.pump();

    expect(platform.lastInlineCompletionText, 'hello');

    await responseStream.close();
    await aiService.dispose();
  });

  testWidgets('trims echoed prefix from streamed inline completions',
      (WidgetTester tester) async {
    final platform = _TestEditorPlatform();
    final manager = _TestWorkspaceManager(
      files: <String, String>{
        '/lib/main.dart': 'print();',
      },
    );
    final responseStream = StreamController<List<int>>();
    final aiService = CortadoAIService(
      baseUrl: 'http://localhost:8080',
      httpClient: _RecordingHttpClient((request, bodyBytes) async {
        return http.StreamedResponse(
          responseStream.stream,
          200,
          headers: const <String, String>{
            'Content-Type': 'text/event-stream',
          },
        );
      }),
    );

    await tester.pumpWidget(
      _wrapEditor(
        CortadoCodeEditor(
          aiService: aiService,
          path: '/lib/main.dart',
          platform: platform,
          workspaceId: 'ws-123',
          workspaceManager: manager,
        ),
      ),
    );

    await tester.pump();
    await tester.pump();

    platform.emitInlineCompletionRequest(
      <String, Object?>{
        'kind': 'request',
        'requestId': 8,
        'prefix': 'print(',
        'suffix': ');',
      },
    );
    await tester.pump();

    responseStream.add(
      Uint8List.fromList(utf8.encode('data: {"token":"print(value);"}\n\n')),
    );
    await tester.pump();
    await tester.pump();

    expect(platform.lastInlineCompletionText, 'value);');

    await responseStream.close();
    await aiService.dispose();
  });

  testWidgets('shows the LSP startup overlay and wires document lifecycle',
      (WidgetTester tester) async {
    final platform = _TestEditorPlatform();
    final manager = _TestWorkspaceManager(
      files: <String, String>{
        '/lib/main.dart': 'print("hello");',
      },
    );
    final client = _TestMuxClient();

    await tester.pumpWidget(
      _wrapEditor(
        CortadoCodeEditor(
          client: client,
          path: '/lib/main.dart',
          platform: platform,
          workspaceId: 'ws-123',
          workspaceManager: manager,
        ),
      ),
    );

    await tester.pump();
    await tester.pump();

    expect(find.text('Language server starting...'), findsOneWidget);
    expect(_decodeFrameJson(client.sentFrames[1])['method'], 'initialize');

    final initializeRequest = _decodeFrameJson(client.sentFrames[1]);
    client.emitJson(
      <String, Object?>{
        'jsonrpc': '2.0',
        'id': initializeRequest['id'],
        'result': <String, Object?>{
          'capabilities': <String, Object?>{},
        },
      },
    );
    await tester.pump();
    await tester.pump();

    expect(find.text('Language server starting...'), findsNothing);
    expect(_decodeFrameJson(client.sentFrames[2])['method'], 'initialized');
    expect(_decodeFrameJson(client.sentFrames[3])['method'],
        'textDocument/didOpen');

    platform.currentContent = 'print("updated");';
    platform.emitChanged();
    await tester.pump();

    expect(
      _decodeFrameJson(client.sentFrames.last)['method'],
      'textDocument/didChange',
    );

    await tester.tap(find.byIcon(Icons.close));
    await tester.pump();

    expect(
      _decodeFrameJson(client.sentFrames.last)['method'],
      'textDocument/didClose',
    );
  });

  testWidgets('bridges completion requests through the LSP client',
      (WidgetTester tester) async {
    final platform = _TestEditorPlatform();
    final manager = _TestWorkspaceManager(
      files: <String, String>{
        '/lib/main.dart': 'print("hello");',
      },
    );
    final client = _TestMuxClient();

    await tester.pumpWidget(
      _wrapEditor(
        CortadoCodeEditor(
          client: client,
          path: '/lib/main.dart',
          platform: platform,
          workspaceId: 'ws-123',
          workspaceManager: manager,
        ),
      ),
    );

    await tester.pump();
    await tester.pump();

    final initializeRequest = _decodeFrameJson(client.sentFrames[1]);
    client.emitJson(
      <String, Object?>{
        'jsonrpc': '2.0',
        'id': initializeRequest['id'],
        'result': <String, Object?>{
          'capabilities': <String, Object?>{},
        },
      },
    );
    await tester.pump();
    await tester.pump();

    platform.emitLspRequest(
      <String, Object?>{
        'requestId': 1,
        'position': <String, Object?>{
          'line': 0,
          'character': 5,
        },
      },
    );
    await tester.pump();

    final completionRequest = _decodeFrameJson(client.sentFrames.last);
    expect(completionRequest['method'], 'textDocument/completion');

    client.emitJson(
      <String, Object?>{
        'jsonrpc': '2.0',
        'id': completionRequest['id'],
        'result': <String, Object?>{
          'items': <Object?>[
            <String, Object?>{
              'label': 'print',
              'detail': 'void Function(Object?)',
              'kind': 3,
              'insertText': 'print',
            },
          ],
        },
      },
    );
    await tester.pump();
    await tester.pump();

    expect(platform.lastResolvedRequestId, 1);
    expect(
      platform.lastResolvedItems,
      <Map<String, Object?>>[
        <String, Object?>{
          'label': 'print',
          'detail': 'void Function(Object?)',
          'type': 'function',
        },
      ],
    );
  });

  testWidgets('bridges hover requests through the LSP client',
      (WidgetTester tester) async {
    final platform = _TestEditorPlatform();
    final manager = _TestWorkspaceManager(
      files: <String, String>{
        '/lib/main.dart': 'print("hello");',
      },
    );
    final client = _TestMuxClient();

    await tester.pumpWidget(
      _wrapEditor(
        CortadoCodeEditor(
          client: client,
          path: '/lib/main.dart',
          platform: platform,
          workspaceId: 'ws-123',
          workspaceManager: manager,
        ),
      ),
    );

    await tester.pump();
    await tester.pump();

    final initializeRequest = _decodeFrameJson(client.sentFrames[1]);
    client.emitJson(
      <String, Object?>{
        'jsonrpc': '2.0',
        'id': initializeRequest['id'],
        'result': <String, Object?>{
          'capabilities': <String, Object?>{},
        },
      },
    );
    await tester.pump();
    await tester.pump();

    platform.emitLspRequest(
      <String, Object?>{
        'requestId': 2,
        'kind': 'hover',
        'position': <String, Object?>{
          'line': 0,
          'character': 5,
        },
      },
    );
    await tester.pump();

    final hoverRequest = _decodeFrameJson(client.sentFrames.last);
    expect(hoverRequest['method'], 'textDocument/hover');

    client.emitJson(
      <String, Object?>{
        'jsonrpc': '2.0',
        'id': hoverRequest['id'],
        'result': <String, Object?>{
          'contents': <String, Object?>{
            'kind': 'markdown',
            'value': '**print** docs',
          },
        },
      },
    );
    await tester.pump();
    await tester.pump();

    expect(platform.lastResolvedRequestId, 2);
    expect(
      platform.lastResolvedResult,
      <String, Object?>{'markdown': '**print** docs'},
    );
  });

  testWidgets('opens workspace definition targets in editor tabs',
      (WidgetTester tester) async {
    final platform = _TestEditorPlatform();
    final manager = _TestWorkspaceManager(
      files: <String, String>{
        '/lib/main.dart': 'void main() => helper();',
        '/lib/helper.dart': 'void helper() {}',
      },
    );
    final client = _TestMuxClient();

    await tester.pumpWidget(
      _wrapEditor(
        CortadoCodeEditor(
          client: client,
          path: '/lib/main.dart',
          platform: platform,
          workspaceId: 'ws-123',
          workspaceManager: manager,
        ),
      ),
    );

    await tester.pump();
    await tester.pump();

    final initializeRequest = _decodeFrameJson(client.sentFrames[1]);
    client.emitJson(
      <String, Object?>{
        'jsonrpc': '2.0',
        'id': initializeRequest['id'],
        'result': <String, Object?>{
          'capabilities': <String, Object?>{},
        },
      },
    );
    await tester.pump();
    await tester.pump();

    platform.emitLspRequest(
      <String, Object?>{
        'requestId': 3,
        'kind': 'definition',
        'position': <String, Object?>{
          'line': 0,
          'character': 15,
        },
      },
    );
    await tester.pump();

    final definitionRequest = _decodeFrameJson(client.sentFrames.last);
    expect(definitionRequest['method'], 'textDocument/definition');

    client.emitJson(
      <String, Object?>{
        'jsonrpc': '2.0',
        'id': definitionRequest['id'],
        'result': <String, Object?>{
          'uri': 'file:///workspace/lib/helper.dart',
          'range': <String, Object?>{
            'start': <String, Object?>{
              'line': 0,
              'character': 0,
            },
          },
        },
      },
    );
    await tester.pump();
    await tester.pump();

    expect(platform.currentContent, 'void helper() {}');
    expect(platform.lastReadOnly, isFalse);
  });

  testWidgets('opens SDK definition targets as read-only tabs',
      (WidgetTester tester) async {
    final platform = _TestEditorPlatform();
    final manager = _TestWorkspaceManager(
      files: <String, String>{
        '/lib/main.dart': 'void main() => print("hello");',
      },
    );
    final client = _TestMuxClient();

    await tester.pumpWidget(
      _wrapEditor(
        CortadoCodeEditor(
          client: client,
          path: '/lib/main.dart',
          platform: platform,
          workspaceId: 'ws-123',
          workspaceManager: manager,
        ),
      ),
    );

    await tester.pump();
    await tester.pump();

    final initializeRequest = _decodeFrameJson(client.sentFrames[1]);
    client.emitJson(
      <String, Object?>{
        'jsonrpc': '2.0',
        'id': initializeRequest['id'],
        'result': <String, Object?>{
          'capabilities': <String, Object?>{},
        },
      },
    );
    await tester.pump();
    await tester.pump();

    platform.emitLspRequest(
      <String, Object?>{
        'requestId': 4,
        'kind': 'definition',
        'position': <String, Object?>{
          'line': 0,
          'character': 15,
        },
      },
    );
    await tester.pump();

    final definitionRequest = _decodeFrameJson(client.sentFrames.last);
    expect(definitionRequest['method'], 'textDocument/definition');

    client.emitJson(
      <String, Object?>{
        'jsonrpc': '2.0',
        'id': definitionRequest['id'],
        'result': <String, Object?>{
          'uri': 'file:///usr/local/dart-sdk/lib/core/print.dart',
          'range': <String, Object?>{
            'start': <String, Object?>{
              'line': 0,
              'character': 0,
            },
          },
        },
      },
    );
    await tester.pump();
    await tester.pump();

    expect(platform.lastReadOnly, isTrue);
    expect(
      platform.currentContent,
      contains('Read-only SDK definition target.'),
    );
  });

  testWidgets(
      'publishes diagnostics to the active editor and workspace status state',
      (WidgetTester tester) async {
    final platform = _TestEditorPlatform();
    final manager = _TestWorkspaceManager(
      files: <String, String>{
        '/lib/main.dart': 'print("hello");',
      },
    );
    final client = _TestMuxClient();

    await tester.pumpWidget(
      _wrapEditor(
        CortadoCodeEditor(
          client: client,
          path: '/lib/main.dart',
          platform: platform,
          workspaceId: 'ws-123',
          workspaceManager: manager,
        ),
      ),
    );

    await tester.pump();
    await tester.pump();

    final initializeRequest = _decodeFrameJson(client.sentFrames[1]);
    client.emitJson(
      <String, Object?>{
        'jsonrpc': '2.0',
        'id': initializeRequest['id'],
        'result': <String, Object?>{
          'capabilities': <String, Object?>{},
        },
      },
    );
    await tester.pump();
    await tester.pump();

    client.emitJson(
      <String, Object?>{
        'jsonrpc': '2.0',
        'method': 'textDocument/publishDiagnostics',
        'params': <String, Object?>{
          'uri': 'file:///workspace/lib/main.dart',
          'diagnostics': <Object?>[
            <String, Object?>{
              'message': 'Missing semicolon.',
              'severity': 1,
              'range': <String, Object?>{
                'start': <String, Object?>{
                  'line': 0,
                  'character': 0,
                },
                'end': <String, Object?>{
                  'line': 0,
                  'character': 5,
                },
              },
            },
          ],
        },
      },
    );
    await tester.pump();
    await tester.pump();

    expect(platform.lastDiagnostics, hasLength(1));
    expect(platform.lastDiagnostics.single['message'], 'Missing semicolon.');
    expect(
      find.byKey(const ValueKey('editor-diagnostic-dot:/lib/main.dart')),
      findsOneWidget,
    );

    final container = ProviderScope.containerOf(
      tester.element(find.byType(CortadoCodeEditor)),
    );
    expect(
      container.read(cortadoWorkspaceDiagnosticStatusProvider),
      <String, CortadoFileDiagnosticStatus>{
        '/lib/main.dart': CortadoFileDiagnosticStatus.error,
      },
    );

    client.emitJson(
      <String, Object?>{
        'jsonrpc': '2.0',
        'method': 'textDocument/publishDiagnostics',
        'params': <String, Object?>{
          'uri': 'file:///workspace/lib/main.dart',
          'diagnostics': <Object?>[
            <String, Object?>{
              'message': 'Unused import.',
              'severity': 2,
            },
          ],
        },
      },
    );
    await tester.pump();
    await tester.pump();

    expect(platform.lastDiagnostics, hasLength(1));
    expect(platform.lastDiagnostics.single['message'], 'Unused import.');
    expect(
      container.read(cortadoWorkspaceDiagnosticStatusProvider),
      <String, CortadoFileDiagnosticStatus>{
        '/lib/main.dart': CortadoFileDiagnosticStatus.warning,
      },
    );

    client.emitJson(
      <String, Object?>{
        'jsonrpc': '2.0',
        'method': 'textDocument/publishDiagnostics',
        'params': <String, Object?>{
          'uri': 'file:///workspace/lib/main.dart',
          'diagnostics': const <Object?>[],
        },
      },
    );
    await tester.pump();
    await tester.pump();

    expect(platform.lastDiagnostics, isEmpty);
    expect(
      container.read(cortadoWorkspaceDiagnosticStatusProvider),
      const <String, CortadoFileDiagnosticStatus>{},
    );
    expect(
      find.byKey(const ValueKey('editor-diagnostic-dot:/lib/main.dart')),
      findsNothing,
    );
  });
}

Widget _wrapEditor(Widget child) {
  return ProviderScope(
    child: MaterialApp(
      home: SizedBox.expand(child: child),
    ),
  );
}

Map<String, Object?> _decodeFrameJson(MuxFrame frame) =>
    Map<String, Object?>.from(
      jsonDecode(utf8.decode(frame.payload)) as Map<Object?, Object?>,
    );

class _TestEditorPlatform extends CortadoCodeEditorPlatformAdapter {
  _TestEditorPlatform({
    this.supportsPlatformView = true,
  });

  final bool supportsPlatformView;

  String currentContent = '';
  String currentLanguageId = 'plain';
  bool lastPreserveSelection = false;
  bool lastReadOnly = false;
  CortadoEditorChangedCallback? _onChanged;
  CortadoEditorInlineCompletionRequestCallback? _onInlineCompletionRequest;
  CortadoEditorLspRequestCallback? _onLspRequest;
  CortadoEditorSaveCallback? _onSave;
  String? inlineCompletionEditorId;
  String? lspEditorId;
  int clearInlineCompletionCallCount = 0;
  int inlineCompletionSetCallCount = 0;
  List<Map<String, Object?>> lastResolvedItems = <Map<String, Object?>>[];
  List<Map<String, Object?>> lastDiagnostics = <Map<String, Object?>>[];
  String lastInlineCompletionText = '';
  int? lastInlineCompletionRequestId;
  Object? lastResolvedResult;
  int? lastResolvedRequestId;

  @override
  void disposeView(String editorId) {}

  void emitChanged() {
    _onChanged?.call(hashEditorContent(currentContent));
  }

  @override
  String getContent(String editorId) => currentContent;

  @override
  void registerViewFactory({
    required String viewType,
    required String editorId,
    required String languageId,
    required CortadoEditorChangedCallback onChanged,
    required CortadoEditorSaveCallback onSave,
  }) {
    currentLanguageId = languageId;
    _onChanged = onChanged;
    _onSave = onSave;
  }

  @override
  void registerLspRequestHandler({
    required String editorId,
    required CortadoEditorLspRequestCallback onRequest,
  }) {
    lspEditorId = editorId;
    _onLspRequest = onRequest;
  }

  @override
  void registerInlineCompletionRequestHandler({
    required String editorId,
    required CortadoEditorInlineCompletionRequestCallback onRequest,
  }) {
    inlineCompletionEditorId = editorId;
    _onInlineCompletionRequest = onRequest;
  }

  @override
  void resolveLspResult(
    int requestId,
    Object? result,
  ) {
    lastResolvedRequestId = requestId;
    lastResolvedResult = result;
    lastResolvedItems = switch (result) {
      final List<Object?> items => items
          .whereType<Map<Object?, Object?>>()
          .map(
            (item) => Map<String, Object?>.from(
              item.map(
                (key, value) => MapEntry(key.toString(), value),
              ),
            ),
          )
          .toList(growable: false),
      _ => <Map<String, Object?>>[],
    };
  }

  @override
  void setDiagnostics(
    String editorId,
    List<Map<String, Object?>> diagnostics,
  ) {
    lastDiagnostics =
        diagnostics.map(Map<String, Object?>.from).toList(growable: false);
  }

  @override
  String setContent(
    String editorId,
    String content, {
    bool preserveSelection = false,
  }) {
    currentContent = content;
    lastPreserveSelection = preserveSelection;
    return hashEditorContent(content);
  }

  @override
  void setLanguage(String editorId, String languageId) {
    currentLanguageId = languageId;
  }

  @override
  void setReadOnly(String editorId, bool readOnly) {
    lastReadOnly = readOnly;
  }

  @override
  void setInlineCompletion(
    String editorId, {
    required int requestId,
    required String text,
  }) {
    lastInlineCompletionRequestId = requestId;
    lastInlineCompletionText = text;
    inlineCompletionSetCallCount += 1;
  }

  @override
  void clearInlineCompletion(String editorId) {
    lastInlineCompletionRequestId = null;
    lastInlineCompletionText = '';
    clearInlineCompletionCallCount += 1;
  }

  void triggerSave() {
    _onSave?.call();
  }

  void emitLspRequest(Map<String, Object?> payload) {
    _onLspRequest?.call(
      jsonEncode(
        <String, Object?>{
          'editorId': lspEditorId,
          ...payload,
        },
      ),
    );
  }

  void emitInlineCompletionRequest(Map<String, Object?> payload) {
    if ((payload['kind'] as String?) == 'cancel') {
      lastInlineCompletionRequestId = null;
      lastInlineCompletionText = '';
    }

    _onInlineCompletionRequest?.call(
      jsonEncode(
        <String, Object?>{
          'editorId': inlineCompletionEditorId,
          ...payload,
        },
      ),
    );
  }

  @override
  void unregisterLspRequestHandler(String editorId) {
    if (lspEditorId == editorId) {
      _onLspRequest = null;
      lspEditorId = null;
    }
  }

  @override
  void unregisterInlineCompletionRequestHandler(String editorId) {
    if (inlineCompletionEditorId == editorId) {
      _onInlineCompletionRequest = null;
      inlineCompletionEditorId = null;
    }
  }
}

class _RecordedHttpRequest {
  const _RecordedHttpRequest(this.request, this.bodyBytes);

  final http.BaseRequest request;
  final List<int> bodyBytes;
}

class _RecordingHttpClient extends http.BaseClient {
  _RecordingHttpClient(this._handler);

  final Future<http.StreamedResponse> Function(
    http.BaseRequest request,
    List<int> bodyBytes,
  ) _handler;

  @override
  Future<http.StreamedResponse> send(http.BaseRequest request) async {
    final bodyBytes = await http.ByteStream(request.finalize()).toBytes();
    return _handler(request, bodyBytes);
  }
}

class _TestWorkspaceManager extends WorkspaceManager {
  _TestWorkspaceManager({
    Map<String, String>? files,
  })  : files = files ?? <String, String>{},
        super(baseUrl: 'http://localhost:8080', useBrowserAuth: false);

  final Map<String, String> files;
  final List<_WriteCall> writes = <_WriteCall>[];

  @override
  Future<Uint8List> readFile(
    String workspaceId, {
    required String path,
  }) async {
    final content = files[path];
    if (content == null) {
      throw StateError('Missing test file for $path');
    }
    return Uint8List.fromList(utf8.encode(content));
  }

  @override
  Future<WorkspaceWriteFileResult> writeFile(
    String workspaceId, {
    required String path,
    List<int> content = const <int>[],
    bool createMissingDirs = true,
  }) async {
    final text = utf8.decode(content);
    files[path] = text;
    writes.add(_WriteCall(path: path, content: text));
    return WorkspaceWriteFileResult(
      bytesWritten: content.length,
      checksum: Uint8List(0),
    );
  }
}

class _WriteCall {
  const _WriteCall({
    required this.content,
    required this.path,
  });

  final String content;
  final String path;
}

class _TestMuxClient extends CortadoClient {
  _TestMuxClient()
      : _frames = StreamController<MuxFrame>.broadcast(),
        super(baseUrl: 'http://localhost:8080', useBrowserWebSocket: false);

  final StreamController<MuxFrame> _frames;
  final List<MuxFrame> sentFrames = <MuxFrame>[];

  @override
  Stream<MuxFrame> get frames => _frames.stream;

  @override
  Stream<MuxFrame> framesForChannel(int channelId) =>
      _frames.stream.where((frame) => frame.channelId == channelId);

  @override
  void sendFrame(int channelId, int messageType, Uint8List payload) {
    sentFrames.add(MuxFrame(channelId, messageType, payload));
  }

  void emitJson(Map<String, Object?> payload) {
    _frames.add(
      MuxFrame(
        muxLspChannelStartId,
        muxMessageTypeData,
        Uint8List.fromList(utf8.encode(jsonEncode(payload))),
      ),
    );
  }
}
