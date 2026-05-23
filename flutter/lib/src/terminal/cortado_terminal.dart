import 'dart:async';
import 'dart:convert';
import 'dart:typed_data';

import 'package:flutter/widgets.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../cortado_client.dart';
import '../cortado_workspace_provider.dart';
import '../mux_frame.dart';
import '../workspace_manager.dart';
import '../workspace_models.dart';
import 'terminal_platform.dart';

const String _workspaceResumedBanner =
    '\r\n\x1b[33m--- Workspace resumed ---\x1b[0m\r\n';

class CortadoTerminalReconnectPolicy {
  const CortadoTerminalReconnectPolicy({
    this.statusInitialBackoff = const Duration(seconds: 2),
    this.statusMaxBackoff = const Duration(seconds: 15),
    this.openRetryDelay = const Duration(milliseconds: 500),
    this.maxOpenAttempts = 5,
  }) : assert(maxOpenAttempts > 0, 'maxOpenAttempts must be greater than zero');

  final Duration statusInitialBackoff;
  final Duration statusMaxBackoff;
  final Duration openRetryDelay;
  final int maxOpenAttempts;

  Duration statusDelayForAttempt(int attempt) {
    final cappedAttempt = attempt < 0
        ? 0
        : attempt > 30
            ? 30
            : attempt;
    final multiplier = 1 << cappedAttempt;
    final candidateMs = statusInitialBackoff.inMilliseconds * multiplier;
    final maxMs = statusMaxBackoff.inMilliseconds;
    final nextMs = candidateMs > maxMs ? maxMs : candidateMs;
    return Duration(milliseconds: nextMs);
  }
}

abstract class CortadoTerminalPlatformAdapter {
  const CortadoTerminalPlatformAdapter();

  bool get supportsPlatformView;

  void registerViewFactory({
    required String viewType,
    required String terminalId,
    required CortadoTerminalInputCallback onData,
    required CortadoTerminalResizeCallback onResize,
  });

  Widget buildView(String viewType);

  void writeOutput(String terminalId, String data);

  void disposeView(String terminalId);
}

class DefaultCortadoTerminalPlatformAdapter
    extends CortadoTerminalPlatformAdapter {
  const DefaultCortadoTerminalPlatformAdapter();

  @override
  bool get supportsPlatformView => supportsCortadoTerminalPlatformView;

  @override
  void registerViewFactory({
    required String viewType,
    required String terminalId,
    required CortadoTerminalInputCallback onData,
    required CortadoTerminalResizeCallback onResize,
  }) {
    registerCortadoTerminalViewFactory(
      viewType: viewType,
      terminalId: terminalId,
      onData: onData,
      onResize: onResize,
    );
  }

  @override
  Widget buildView(String viewType) {
    return buildCortadoTerminalPlatformView(viewType);
  }

  @override
  void writeOutput(String terminalId, String data) {
    writeCortadoTerminalOutput(terminalId, data);
  }

  @override
  void disposeView(String terminalId) {
    disposeCortadoTerminalView(terminalId);
  }
}

class CortadoTerminal extends StatefulWidget {
  const CortadoTerminal({
    super.key,
    required this.client,
    this.channelId = muxTerminalChannelId,
    this.shell = '/bin/bash',
    this.autoOpen = true,
    this.onClosed,
    this.onError,
    this.workspaceManager,
    this.workspaceId,
    this.reconnectPolicy = const CortadoTerminalReconnectPolicy(),
    this.platform = const DefaultCortadoTerminalPlatformAdapter(),
  });

  final CortadoClient client;
  final int channelId;
  final String shell;
  final bool autoOpen;
  final ValueChanged<String>? onClosed;
  final ValueChanged<String>? onError;
  final WorkspaceManager? workspaceManager;
  final String? workspaceId;
  final CortadoTerminalReconnectPolicy reconnectPolicy;
  final CortadoTerminalPlatformAdapter platform;

  @override
  State<CortadoTerminal> createState() => _CortadoTerminalState();
}

class _CortadoTerminalState extends State<CortadoTerminal> {
  static int _instanceCount = 0;

  late final String _terminalId = 'cortado-terminal-${_instanceCount++}';
  late final String _viewType = 'cortado-terminal-view-$_terminalId';
  StreamSubscription<MuxFrame>? _frameSubscription;
  StreamSubscription<Object>? _clientErrorSubscription;
  bool _didOpenTerminal = false;
  bool _hasRequestedTerminalOpen = false;
  bool _isReconnectOverlayVisible = false;
  bool _isReconnectInProgress = false;
  String? _terminalErrorMessage;
  MuxTerminalResize? _lastKnownResize;
  _PendingTerminalOpen? _pendingOpen;
  final List<String> _pendingInput = <String>[];
  int _sessionVersion = 0;
  WorkspaceManager? _resolvedWorkspaceManager;
  String? _resolvedWorkspaceId;

  @override
  void initState() {
    super.initState();

    if (!widget.platform.supportsPlatformView) {
      return;
    }

    widget.platform.registerViewFactory(
      viewType: _viewType,
      terminalId: _terminalId,
      onData: _handleTerminalInput,
      onResize: _handleTerminalResize,
    );
    _subscribeToClientStreams();

    if (widget.autoOpen) {
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (!mounted) {
          return;
        }
        unawaited(_openTerminalWithRetry());
      });
    }
  }

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    _resolveWorkspaceContext();
  }

  @override
  void didUpdateWidget(CortadoTerminal oldWidget) {
    super.didUpdateWidget(oldWidget);

    _resolveWorkspaceContext();

    if (!widget.platform.supportsPlatformView) {
      return;
    }

    final platformChanged = oldWidget.platform != widget.platform;
    final clientOrChannelChanged = oldWidget.client != widget.client ||
        oldWidget.channelId != widget.channelId;
    if (platformChanged || clientOrChannelChanged) {
      _sessionVersion++;
      _frameSubscription?.cancel();
      _clientErrorSubscription?.cancel();
      _frameSubscription = null;
      _clientErrorSubscription = null;
      _completePendingOpenError(StateError('Terminal configuration changed.'));
      _didOpenTerminal = false;
      _hasRequestedTerminalOpen = false;
      _terminalErrorMessage = null;
      _isReconnectOverlayVisible = false;
      _isReconnectInProgress = false;
      _subscribeToClientStreams();
      if (widget.autoOpen) {
        WidgetsBinding.instance.addPostFrameCallback((_) {
          if (!mounted) {
            return;
          }
          unawaited(_openTerminalWithRetry());
        });
      }
    }
  }

  @override
  void dispose() {
    _sessionVersion++;
    _completePendingOpenError(StateError('Terminal disposed.'));
    _frameSubscription?.cancel();
    _clientErrorSubscription?.cancel();
    if (widget.platform.supportsPlatformView) {
      widget.platform.disposeView(_terminalId);
    }
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Stack(
      fit: StackFit.expand,
      children: <Widget>[
        widget.platform.buildView(_viewType),
        if (_isReconnectOverlayVisible || _terminalErrorMessage != null)
          _buildStatusOverlay(context),
      ],
    );
  }

  void _subscribeToClientStreams() {
    _frameSubscription =
        widget.client.framesForChannel(widget.channelId).listen(_handleFrame);
    _clientErrorSubscription = widget.client.errors.listen(_handleClientError);
  }

  Widget _buildStatusOverlay(BuildContext context) {
    final message = _terminalErrorMessage ??
        (_isReconnectOverlayVisible ? 'Reconnecting...' : null);
    if (message == null) {
      return const SizedBox.shrink();
    }

    return IgnorePointer(
      child: ColoredBox(
        color: const Color(0xB8141B24),
        child: Center(
          child: DecoratedBox(
            decoration: BoxDecoration(
              color: const Color(0xE61B2530),
              borderRadius: BorderRadius.circular(16),
              border: Border.all(color: const Color(0xFF3C4A59)),
            ),
            child: Padding(
              padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 14),
              child: DefaultTextStyle(
                style: const TextStyle(
                  color: Color(0xFFF5F7FA),
                  fontSize: 14,
                ),
                child: Text(message),
              ),
            ),
          ),
        ),
      ),
    );
  }

  Future<void> _openTerminalWithRetry({bool resumed = false}) async {
    if (_didOpenTerminal || _pendingOpen != null) {
      return;
    }

    _hasRequestedTerminalOpen = true;
    _clearTerminalError();

    for (var attempt = 1;
        attempt <= widget.reconnectPolicy.maxOpenAttempts;
        attempt++) {
      try {
        await _sendOpenFrame();
        if (!_isCurrentSessionActive) {
          return;
        }
        _didOpenTerminal = true;
        _flushResize();
        if (resumed) {
          widget.platform.writeOutput(_terminalId, _workspaceResumedBanner);
        }
        return;
      } on _TerminalOpenFailure catch (error) {
        if (!_isCurrentSessionActive) {
          return;
        }
        _didOpenTerminal = false;
        if (!error.retryable ||
            attempt >= widget.reconnectPolicy.maxOpenAttempts) {
          throw StateError(error.message);
        }
        await Future<void>.delayed(widget.reconnectPolicy.openRetryDelay);
      }
    }
  }

  void _handleFrame(MuxFrame frame) {
    switch (frame.messageType) {
      case muxMessageTypeData:
        _completePendingOpenSuccess();
        widget.platform.writeOutput(
          _terminalId,
          utf8.decode(frame.payload, allowMalformed: true),
        );
        break;
      case muxMessageTypeClose:
        final message = utf8.decode(frame.payload, allowMalformed: true);
        if (_pendingOpen != null) {
          _completePendingOpenError(
            _TerminalOpenFailure(
              message,
              retryable: _isRetryableOpenFailure(message),
            ),
          );
          break;
        }
        _didOpenTerminal = false;
        _hasRequestedTerminalOpen = false;
        widget.onClosed?.call(message);
        break;
      case muxMessageTypeError:
        final message = utf8.decode(frame.payload, allowMalformed: true);
        if (_pendingOpen != null) {
          if (_isIgnorablePendingOpenError(message)) {
            break;
          }
          _completePendingOpenError(
            _TerminalOpenFailure(
              message,
              retryable: _isRetryableOpenFailure(message),
            ),
          );
          break;
        }
        widget.onError?.call(message);
        break;
      default:
        break;
    }
  }

  void _handleTerminalInput(String data) {
    if (!_didOpenTerminal) {
      _pendingInput.add(data);
      unawaited(_openTerminalWithRetry());
      return;
    }

    _sendInput(data);
  }

  void _handleTerminalResize(int cols, int rows) {
    final resize = MuxTerminalResize(cols: cols, rows: rows);
    _lastKnownResize = resize;
    if (_didOpenTerminal) {
      _sendResize(resize);
    }
  }

  void _sendResize(MuxTerminalResize resize) {
    widget.client.sendFrame(
      widget.channelId,
      muxMessageTypeResize,
      resize.encode(),
    );
  }

  void _handleClientError(Object error) {
    _completePendingOpenError(error);
    _didOpenTerminal = false;
    if (!_hasRequestedTerminalOpen || _isReconnectInProgress) {
      return;
    }

    final resumeContext = _resumeContext;
    if (resumeContext == null) {
      widget.onError?.call(error.toString());
      return;
    }

    unawaited(_resumeAfterDisconnect(resumeContext));
  }

  Future<void> _resumeAfterDisconnect(_ResumeContext context) async {
    if (_isReconnectInProgress) {
      return;
    }

    final version = _sessionVersion;
    _isReconnectInProgress = true;
    _showReconnectOverlay();

    try {
      await context.manager.start(context.workspaceId);
      await _waitForWorkspaceRunning(context.workspaceId, version);
      if (!_matchesSessionVersion(version)) {
        return;
      }

      await widget.client.connect(context.workspaceId);
      if (!_matchesSessionVersion(version)) {
        return;
      }

      await _openTerminalWithRetry(resumed: true);
      if (!_matchesSessionVersion(version)) {
        return;
      }

      _hideReconnectOverlay();
      _clearTerminalError();
    } catch (error) {
      if (!_matchesSessionVersion(version)) {
        return;
      }
      _hideReconnectOverlay();
      _setTerminalError(error.toString());
      widget.onError?.call(error.toString());
    } finally {
      if (_matchesSessionVersion(version)) {
        _isReconnectInProgress = false;
      }
    }
  }

  Future<void> _waitForWorkspaceRunning(String workspaceId, int version) async {
    for (var attempt = 0; _matchesSessionVersion(version); attempt++) {
      final status =
          await _resolvedWorkspaceManager!.watchStatus(workspaceId).first;
      if (status.status == WorkspaceLifecycleState.running) {
        return;
      }
      if (status.isTerminal) {
        throw StateError(
          'Workspace entered ${status.status.name} while reconnecting.',
        );
      }

      await Future<void>.delayed(
        widget.reconnectPolicy.statusDelayForAttempt(attempt),
      );
    }
  }

  Future<void> _sendOpenFrame() {
    final completer = Completer<void>();
    final pending = _PendingTerminalOpen(completer);
    _pendingOpen = pending;

    pending.stabilityTimer = Timer(widget.reconnectPolicy.openRetryDelay, () {
      if (!completer.isCompleted) {
        completer.complete();
      }
    });

    try {
      widget.client.sendFrame(
        widget.channelId,
        muxMessageTypeOpen,
        Uint8List.fromList(utf8.encode(widget.shell)),
      );
    } catch (error, stackTrace) {
      _completePendingOpenError(error);
      return Future<void>.error(error, stackTrace);
    }

    return completer.future.whenComplete(() {
      pending.stabilityTimer?.cancel();
      if (identical(_pendingOpen, pending)) {
        _pendingOpen = null;
      }
    });
  }

  void _completePendingOpenSuccess() {
    final pending = _pendingOpen;
    if (pending == null || pending.completer.isCompleted) {
      return;
    }

    pending.completer.complete();
  }

  void _completePendingOpenError(Object error) {
    final pending = _pendingOpen;
    if (pending == null || pending.completer.isCompleted) {
      return;
    }

    pending.completer.completeError(error);
  }

  void _flushResize() {
    final resize = _lastKnownResize;
    if (resize != null) {
      _sendResize(resize);
    }
    if (_pendingInput.isEmpty) {
      return;
    }
    final pendingInput = List<String>.from(_pendingInput);
    _pendingInput.clear();
    for (final data in pendingInput) {
      _sendInput(data);
    }
  }

  void _resolveWorkspaceContext() {
    _resolvedWorkspaceManager = widget.workspaceManager;
    _resolvedWorkspaceId = widget.workspaceId;

    if (_resolvedWorkspaceManager != null && _resolvedWorkspaceId != null) {
      return;
    }

    try {
      final container = ProviderScope.containerOf(context, listen: false);
      _resolvedWorkspaceManager ??=
          container.read(cortadoWorkspaceManagerProvider);
      _resolvedWorkspaceId ??= container.read(cortadoWorkspaceIdProvider);
    } on Object {
      // Resume support is optional when no workspace scope is available.
    }
  }

  _ResumeContext? get _resumeContext {
    final manager = _resolvedWorkspaceManager;
    final workspaceId = _resolvedWorkspaceId;
    if (manager == null || workspaceId == null || workspaceId.trim().isEmpty) {
      return null;
    }
    return _ResumeContext(manager: manager, workspaceId: workspaceId);
  }

  bool get _isCurrentSessionActive => mounted;

  bool _matchesSessionVersion(int version) {
    return mounted && version == _sessionVersion;
  }

  void _showReconnectOverlay() {
    if (!mounted) {
      return;
    }
    setState(() {
      _isReconnectOverlayVisible = true;
      _terminalErrorMessage = null;
    });
  }

  void _hideReconnectOverlay() {
    if (!mounted) {
      return;
    }
    setState(() {
      _isReconnectOverlayVisible = false;
    });
  }

  void _setTerminalError(String message) {
    if (!mounted) {
      return;
    }
    setState(() {
      _terminalErrorMessage = message;
    });
  }

  void _clearTerminalError() {
    if (!mounted || _terminalErrorMessage == null) {
      return;
    }
    setState(() {
      _terminalErrorMessage = null;
    });
  }

  void _sendInput(String data) {
    widget.client.sendFrame(
      widget.channelId,
      muxMessageTypeData,
      Uint8List.fromList(utf8.encode(data)),
    );
  }

  bool _isRetryableOpenFailure(String message) {
    final normalized = message.toLowerCase();
    if (normalized.contains('code = invalidargument') ||
        normalized.contains('not found in image')) {
      return false;
    }

    return normalized.contains('code = unavailable') ||
        normalized.contains('code = deadlineexceeded') ||
        normalized.contains('workspace starting');
  }

  bool _isIgnorablePendingOpenError(String message) {
    return message.toLowerCase().contains('terminal channel is not open');
  }
}

class _PendingTerminalOpen {
  _PendingTerminalOpen(this.completer);

  final Completer<void> completer;
  Timer? stabilityTimer;
}

class _ResumeContext {
  const _ResumeContext({
    required this.manager,
    required this.workspaceId,
  });

  final WorkspaceManager manager;
  final String workspaceId;
}

class _TerminalOpenFailure implements Exception {
  const _TerminalOpenFailure(
    this.message, {
    this.retryable = true,
  });

  final String message;
  final bool retryable;

  @override
  String toString() => message;
}
