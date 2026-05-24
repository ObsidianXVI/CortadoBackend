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
}

class _TestEditorPlatform extends CortadoCodeEditorPlatformAdapter {
  _TestEditorPlatform({
    this.supportsPlatformView = true,
  });

  final bool supportsPlatformView;

  String currentContent = '';
  String currentLanguageId = 'plain';
  bool lastPreserveSelection = false;
  CortadoEditorChangedCallback? _onChanged;
  CortadoEditorSaveCallback? _onSave;

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
