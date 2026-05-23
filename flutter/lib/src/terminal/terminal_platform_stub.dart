import 'package:flutter/widgets.dart';

typedef CortadoTerminalInputCallback = void Function(String data);
typedef CortadoTerminalResizeCallback = void Function(int cols, int rows);

bool get supportsCortadoTerminalPlatformView => false;

void registerCortadoTerminalViewFactory({
  required String viewType,
  required String terminalId,
  required CortadoTerminalInputCallback onData,
  required CortadoTerminalResizeCallback onResize,
}) {}

Widget buildCortadoTerminalPlatformView(String viewType) {
  return const Center(
    child: Text('CortadoTerminal is currently supported on Flutter Web only.'),
  );
}

void writeCortadoTerminalOutput(String terminalId, String data) {}

void disposeCortadoTerminalView(String terminalId) {}
