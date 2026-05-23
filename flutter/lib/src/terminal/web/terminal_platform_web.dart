import 'dart:js_interop';
import 'dart:ui_web' as ui_web;

import 'package:flutter/widgets.dart';
import 'package:web/web.dart' as web;

typedef CortadoTerminalInputCallback = void Function(String data);
typedef CortadoTerminalResizeCallback = void Function(int cols, int rows);

final Set<String> _registeredViewTypes = <String>{};

@JS('CortadoXterm.init')
external void _xtermInit(
  web.HTMLDivElement container,
  JSString terminalId,
  JSFunction onData,
  JSFunction onResize,
);

@JS('CortadoXterm.write')
external void _xtermWrite(JSString terminalId, JSString data);

@JS('CortadoXterm.dispose')
external void _xtermDispose(JSString terminalId);

bool get supportsCortadoTerminalPlatformView => true;

void registerCortadoTerminalViewFactory({
  required String viewType,
  required String terminalId,
  required CortadoTerminalInputCallback onData,
  required CortadoTerminalResizeCallback onResize,
}) {
  if (_registeredViewTypes.contains(viewType)) {
    return;
  }

  ui_web.platformViewRegistry.registerViewFactory(viewType, (int viewId) {
    final container = web.HTMLDivElement()
      ..id = 'cortado-terminal-$viewId'
      ..style.width = '100%'
      ..style.height = '100%'
      ..style.backgroundColor = '#101418';

    _xtermInit(
      container,
      terminalId.toJS,
      ((JSString data) {
        onData(data.toDart);
      }).toJS,
      ((JSNumber cols, JSNumber rows) {
        onResize(cols.toDartInt, rows.toDartInt);
      }).toJS,
    );

    return container;
  });

  _registeredViewTypes.add(viewType);
}

Widget buildCortadoTerminalPlatformView(String viewType) {
  return HtmlElementView(viewType: viewType);
}

void writeCortadoTerminalOutput(String terminalId, String data) {
  _xtermWrite(terminalId.toJS, data.toJS);
}

void disposeCortadoTerminalView(String terminalId) {
  _xtermDispose(terminalId.toJS);
}
