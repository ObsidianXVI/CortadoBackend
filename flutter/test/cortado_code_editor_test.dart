import 'dart:async';
import 'dart:convert';
import 'dart:typed_data';

import 'package:cortado/cortado.dart';
import 'package:cortado/src/editor/editor_platform.dart';
import 'package:cortado/src/gen/agent/v1/agent.pbenum.dart' as agentpbenum;
import 'package:cortado/src/gen/agent/v1/agent.pb.dart' as agentpb;
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('renders a non-web fallback when HtmlElementView is unavailable',
      (WidgetTester tester) async {
    await tester.pumpWidget(
      MaterialApp(
        home: SizedBox.expand(
          child: CortadoCodeEditor(
            platform: _TestEditorPlatform(supportsPlatformView: false),
            workspaceId: 'ws-123',
            workspaceManager: _TestWorkspaceManager(),
          ),
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
      MaterialApp(
        home: SizedBox.expand(
          child: CortadoCodeEditor(
            fileEvents: fileEvents.stream,
            path: '/lib/main.dart',
            platform: platform,
            workspaceId: 'ws-123',
            workspaceManager: manager,
          ),
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
      MaterialApp(
        home: SizedBox.expand(
          child: CortadoCodeEditor(
            client: client,
            path: '/lib/main.dart',
            platform: platform,
            workspaceId: 'ws-123',
            workspaceManager: manager,
          ),
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
      MaterialApp(
        home: SizedBox.expand(
          child: CortadoCodeEditor(
            client: client,
            path: '/lib/main.dart',
            platform: platform,
            workspaceId: 'ws-123',
            workspaceManager: manager,
          ),
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
  CortadoEditorChangedCallback? _onChanged;
  CortadoEditorLspRequestCallback? _onLspRequest;
  CortadoEditorSaveCallback? _onSave;
  String? lspEditorId;
  List<Map<String, Object?>> lastResolvedItems = <Map<String, Object?>>[];
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
  void resolveLspResult(
    int requestId,
    List<Map<String, Object?>> items,
  ) {
    lastResolvedRequestId = requestId;
    lastResolvedItems =
        items.map(Map<String, Object?>.from).toList(growable: false);
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

  @override
  void unregisterLspRequestHandler(String editorId) {
    if (lspEditorId == editorId) {
      _onLspRequest = null;
      lspEditorId = null;
    }
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
