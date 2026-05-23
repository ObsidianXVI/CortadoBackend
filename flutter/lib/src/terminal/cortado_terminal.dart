import 'dart:async';
import 'dart:convert';
import 'dart:typed_data';

import 'package:flutter/widgets.dart';

import '../cortado_client.dart';
import '../mux_frame.dart';
import 'terminal_platform.dart';

class CortadoTerminal extends StatefulWidget {
  const CortadoTerminal({
    super.key,
    required this.client,
    this.channelId = muxTerminalChannelId,
    this.shell = '/bin/bash',
    this.autoOpen = true,
    this.onClosed,
    this.onError,
  });

  final CortadoClient client;
  final int channelId;
  final String shell;
  final bool autoOpen;
  final ValueChanged<String>? onClosed;
  final ValueChanged<String>? onError;

  @override
  State<CortadoTerminal> createState() => _CortadoTerminalState();
}

class _CortadoTerminalState extends State<CortadoTerminal> {
  static int _instanceCount = 0;

  late final String _terminalId = 'cortado-terminal-${_instanceCount++}';
  late final String _viewType = 'cortado-terminal-view-$_terminalId';
  StreamSubscription<MuxFrame>? _frameSubscription;
  bool _didOpenTerminal = false;
  MuxTerminalResize? _pendingResize;

  @override
  void initState() {
    super.initState();

    if (!supportsCortadoTerminalPlatformView) {
      return;
    }

    registerCortadoTerminalViewFactory(
      viewType: _viewType,
      terminalId: _terminalId,
      onData: _handleTerminalInput,
      onResize: _handleTerminalResize,
    );
    _subscribeToTerminalFrames();

    if (widget.autoOpen) {
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (!mounted) {
          return;
        }
        _openTerminal();
      });
    }
  }

  @override
  void didUpdateWidget(CortadoTerminal oldWidget) {
    super.didUpdateWidget(oldWidget);

    if (!supportsCortadoTerminalPlatformView) {
      return;
    }

    if (oldWidget.client != widget.client ||
        oldWidget.channelId != widget.channelId) {
      _frameSubscription?.cancel();
      _frameSubscription = null;
      _didOpenTerminal = false;
      _subscribeToTerminalFrames();
      if (widget.autoOpen) {
        WidgetsBinding.instance.addPostFrameCallback((_) {
          if (!mounted) {
            return;
          }
          _openTerminal();
        });
      }
    }
  }

  @override
  void dispose() {
    _frameSubscription?.cancel();
    if (supportsCortadoTerminalPlatformView) {
      disposeCortadoTerminalView(_terminalId);
    }
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return buildCortadoTerminalPlatformView(_viewType);
  }

  void _subscribeToTerminalFrames() {
    _frameSubscription =
        widget.client.framesForChannel(widget.channelId).listen(_handleFrame);
  }

  void _openTerminal() {
    if (_didOpenTerminal) {
      return;
    }

    _didOpenTerminal = true;
    widget.client.sendFrame(
      widget.channelId,
      muxMessageTypeOpen,
      Uint8List.fromList(utf8.encode(widget.shell)),
    );

    final pendingResize = _pendingResize;
    if (pendingResize != null) {
      _pendingResize = null;
      _sendResize(pendingResize);
    }
  }

  void _handleFrame(MuxFrame frame) {
    switch (frame.messageType) {
      case muxMessageTypeData:
        writeCortadoTerminalOutput(
          _terminalId,
          utf8.decode(frame.payload, allowMalformed: true),
        );
        break;
      case muxMessageTypeClose:
        widget.onClosed?.call(utf8.decode(frame.payload, allowMalformed: true));
        break;
      case muxMessageTypeError:
        widget.onError?.call(utf8.decode(frame.payload, allowMalformed: true));
        break;
      default:
        break;
    }
  }

  void _handleTerminalInput(String data) {
    if (!_didOpenTerminal) {
      _openTerminal();
    }

    widget.client.sendFrame(
      widget.channelId,
      muxMessageTypeData,
      Uint8List.fromList(utf8.encode(data)),
    );
  }

  void _handleTerminalResize(int cols, int rows) {
    final resize = MuxTerminalResize(cols: cols, rows: rows);
    if (!_didOpenTerminal) {
      _pendingResize = resize;
      return;
    }

    _sendResize(resize);
  }

  void _sendResize(MuxTerminalResize resize) {
    widget.client.sendFrame(
      widget.channelId,
      muxMessageTypeResize,
      resize.encode(),
    );
  }
}
