import 'dart:async';
import 'dart:convert';
import 'dart:typed_data';

import 'package:web_socket_channel/web_socket_channel.dart';

import '../cortado_client.dart';
import '../mux_frame.dart';
import 'local_daemon_models.dart';

const String cortadoDaemonProtocol = 'cortado-daemon-v1';
const String defaultLocalDaemonUrl = 'ws://127.0.0.1:9731';

class CortadoLocalDaemonUnavailableException implements Exception {
  const CortadoLocalDaemonUnavailableException({
    required this.installUrl,
    this.message = 'Cortado local daemon is unavailable.',
  });

  final String installUrl;
  final String message;

  @override
  String toString() => message;
}

class CortadoLocalDaemonProtocolException implements Exception {
  const CortadoLocalDaemonProtocolException(this.message);

  final String message;

  @override
  String toString() => message;
}

class CortadoLocalDaemonBridge {
  CortadoLocalDaemonBridge({
    String baseUrl = defaultLocalDaemonUrl,
    WebSocketConnector connector = const DefaultWebSocketConnector(),
    this.installUrl = cortadoDaemonInstallUrl,
  })  : _baseUri = Uri.parse(baseUrl),
        _connector = connector;

  final Uri _baseUri;
  final WebSocketConnector _connector;
  final String installUrl;
  final StreamController<CortadoLocalDaemonAvailability> _availabilityStates =
      StreamController<CortadoLocalDaemonAvailability>.broadcast();
  final StreamController<CortadoLocalDaemonConflict> _conflicts =
      StreamController<CortadoLocalDaemonConflict>.broadcast();
  final StreamController<Object> _errors = StreamController<Object>.broadcast();
  final StreamController<CortadoLocalDaemonSyncStatus> _syncStatuses =
      StreamController<CortadoLocalDaemonSyncStatus>.broadcast();
  final Map<String, Completer<CortadoLocalDaemonSyncStatus>> _pendingRequests =
      <String, Completer<CortadoLocalDaemonSyncStatus>>{};
  final Map<String, CortadoLocalDaemonSyncStatus> _syncStatusByKey =
      <String, CortadoLocalDaemonSyncStatus>{};

  WebSocketChannel? _channel;
  StreamSubscription<dynamic>? _subscription;
  CortadoLocalDaemonAvailability _availability =
      const CortadoLocalDaemonAvailability();
  Completer<void>? _helloCompleter;
  bool _disposed = false;
  bool _disconnectRequested = false;
  int _requestCounter = 0;

  CortadoLocalDaemonAvailability get availability => _availability;

  Map<String, CortadoLocalDaemonSyncStatus> get currentSyncStatuses =>
      Map<String, CortadoLocalDaemonSyncStatus>.unmodifiable(_syncStatusByKey);

  Stream<CortadoLocalDaemonAvailability> get availabilityStates =>
      _availabilityStates.stream;

  Stream<CortadoLocalDaemonConflict> get conflicts => _conflicts.stream;

  Stream<Object> get errors => _errors.stream;

  bool get isConnected =>
      _availability.state == CortadoLocalDaemonAvailabilityState.connected;

  Stream<CortadoLocalDaemonSyncStatus> get syncStatuses => _syncStatuses.stream;

  Future<bool> connect() async {
    _ensureNotDisposed();
    if (isConnected) {
      return true;
    }

    await disconnect();

    final channel = _connector.connect(
      _baseUri,
      protocols: const <String>[cortadoDaemonProtocol],
    );
    _disconnectRequested = false;
    _channel = channel;
    _helloCompleter = Completer<void>();
    _subscription = channel.stream.listen(
      _onMessage,
      onDone: _onDone,
      onError: _onError,
    );

    try {
      await channel.ready;
    } catch (error, stackTrace) {
      await disconnect();
      _setAvailability(
        CortadoLocalDaemonAvailability(
          installUrl: installUrl,
          message: 'Unable to connect to the Cortado daemon on localhost.',
          state: CortadoLocalDaemonAvailabilityState.unavailable,
        ),
      );
      _errors.add(error);
      Error.throwWithStackTrace(error, stackTrace);
    }

    try {
      await _helloCompleter!.future.timeout(const Duration(seconds: 2));
      return true;
    } catch (error) {
      _errors.add(error);
      await disconnect();
      _setAvailability(
        CortadoLocalDaemonAvailability(
          installUrl: installUrl,
          message: 'Connected to localhost but did not receive a daemon hello.',
          state: CortadoLocalDaemonAvailabilityState.unavailable,
        ),
      );
      return false;
    } finally {
      _helloCompleter = null;
    }
  }

  Future<void> disconnect() async {
    _disconnectRequested = true;
    await _subscription?.cancel();
    _subscription = null;

    await _channel?.sink.close();
    _channel = null;

    if (!_disposed) {
      _setAvailability(
        CortadoLocalDaemonAvailability(
          installUrl: installUrl,
          state: CortadoLocalDaemonAvailabilityState.disconnected,
        ),
      );
    }
  }

  Future<void> dispose() async {
    if (_disposed) {
      return;
    }

    _disposed = true;
    await disconnect();
    await _availabilityStates.close();
    await _conflicts.close();
    await _errors.close();
    await _syncStatuses.close();
  }

  Future<CortadoLocalDaemonSyncStatus> getSyncStatus(
    String localPath,
    String workspaceId,
  ) {
    return _sendCommand(
      type: 'get_sync_status',
      localPath: localPath,
      workspaceId: workspaceId,
    );
  }

  Future<CortadoLocalDaemonSyncStatus> startSync(
    String localPath,
    String workspaceId,
  ) {
    return _sendCommand(
      type: 'start_sync',
      localPath: localPath,
      workspaceId: workspaceId,
    );
  }

  Future<CortadoLocalDaemonSyncStatus> stopSync(
    String localPath,
    String workspaceId,
  ) {
    return _sendCommand(
      type: 'stop_sync',
      localPath: localPath,
      workspaceId: workspaceId,
    );
  }

  void _completePendingRequests(Object error) {
    final requestIds = _pendingRequests.keys.toList(growable: false);
    for (final requestId in requestIds) {
      _pendingRequests.remove(requestId)?.completeError(error);
    }
  }

  String _keyForStatus(String localPath, String workspaceId) =>
      '$workspaceId::$localPath';

  void _ensureNotDisposed() {
    if (_disposed) {
      throw StateError('CortadoLocalDaemonBridge has been disposed.');
    }
  }

  Future<void> _ensureConnected() async {
    if (isConnected) {
      return;
    }
    final connected = await connect();
    if (!connected) {
      throw CortadoLocalDaemonUnavailableException(installUrl: installUrl);
    }
  }

  void _handleBinaryMessage(Uint8List bytes) {
    final frame = MuxFrame.decode(bytes);
    if (frame.channelId != muxConflictNoticeChannelId ||
        frame.messageType != muxMessageTypeData) {
      return;
    }

    final decoded = jsonDecode(utf8.decode(frame.payload));
    if (decoded is! Map<String, dynamic>) {
      throw const FormatException('Unexpected conflict notice payload.');
    }

    final conflict = _resolveConflict(
      CortadoLocalDaemonConflict.fromJson(decoded),
    );
    _conflicts.add(conflict);

    if (conflict.workspaceId != null && conflict.workspacePath != null) {
      final trackedStatus = _matchingSyncRoot(conflict.localPath);
      final status = CortadoLocalDaemonSyncStatus(
        localPath: trackedStatus?.localPath ?? conflict.localPath,
        message: conflict.reason,
        state: CortadoLocalDaemonSyncState.conflicted,
        workspaceId: trackedStatus?.workspaceId ?? conflict.workspaceId!,
        workspacePath: conflict.workspacePath!,
      );
      _syncStatusByKey[_keyForStatus(status.localPath, status.workspaceId)] =
          status;
      _syncStatuses.add(status);
    }
  }

  void _handleTextMessage(String message) {
    final decoded = jsonDecode(message);
    if (decoded is! Map<String, dynamic>) {
      throw const FormatException('Unexpected daemon text frame.');
    }

    switch (decoded['type']) {
      case 'error':
        final requestId = decoded['requestId'] as String?;
        final error = CortadoLocalDaemonProtocolException(
          decoded['message'] as String? ?? 'Unknown daemon error.',
        );
        if (requestId != null) {
          _pendingRequests.remove(requestId)?.completeError(error);
        }
        _errors.add(error);
        return;
      case 'hello':
        _setAvailability(
          CortadoLocalDaemonAvailability(
            installUrl: installUrl,
            state: CortadoLocalDaemonAvailabilityState.connected,
          ),
        );
        _helloCompleter?.complete();
        return;
      case 'sync_status':
        final status = CortadoLocalDaemonSyncStatus.fromJson(decoded);
        _syncStatusByKey[_keyForStatus(status.localPath, status.workspaceId)] =
            status;
        _syncStatuses.add(status);
        final requestId = decoded['requestId'] as String?;
        if (requestId != null) {
          _pendingRequests.remove(requestId)?.complete(status);
        }
        return;
      default:
        return;
    }
  }

  void _onDone() {
    final error = CortadoLocalDaemonUnavailableException(
      installUrl: installUrl,
      message: 'Local Cortado daemon connection closed.',
    );
    _completePendingRequests(error);
    if (_disconnectRequested || _disposed) {
      _setAvailability(
        CortadoLocalDaemonAvailability(
          installUrl: installUrl,
          state: CortadoLocalDaemonAvailabilityState.disconnected,
        ),
      );
      return;
    }

    _setAvailability(
      CortadoLocalDaemonAvailability(
        installUrl: installUrl,
        message: error.message,
        state: CortadoLocalDaemonAvailabilityState.unavailable,
      ),
    );
    _errors.add(error);
  }

  void _onError(Object error) {
    _completePendingRequests(error);
    _setAvailability(
      CortadoLocalDaemonAvailability(
        installUrl: installUrl,
        message: error.toString(),
        state: CortadoLocalDaemonAvailabilityState.unavailable,
      ),
    );
    _errors.add(error);
  }

  CortadoLocalDaemonConflict _resolveConflict(
    CortadoLocalDaemonConflict conflict,
  ) {
    final bestStatus = _matchingSyncRoot(conflict.localPath);
    if (bestStatus == null) {
      return conflict;
    }

    final normalizedConflictPath = _normalizeLocalPath(conflict.localPath);
    final normalizedRootPath = _normalizeLocalPath(bestStatus.localPath);
    final relativePath = normalizedConflictPath == normalizedRootPath
        ? '/'
        : '/${normalizedConflictPath.substring(normalizedRootPath.length + 1)}';
    return conflict.copyWith(
      workspaceId: bestStatus.workspaceId,
      workspacePath: relativePath,
    );
  }

  CortadoLocalDaemonSyncStatus? _matchingSyncRoot(String localPath) {
    final normalizedConflictPath = _normalizeLocalPath(localPath);
    CortadoLocalDaemonSyncStatus? bestStatus;
    for (final status in _syncStatusByKey.values) {
      final normalizedRootPath = _normalizeLocalPath(status.localPath);
      if (normalizedConflictPath == normalizedRootPath ||
          normalizedConflictPath.startsWith('$normalizedRootPath/')) {
        if (bestStatus == null ||
            normalizedRootPath.length >
                _normalizeLocalPath(bestStatus.localPath).length) {
          bestStatus = status;
        }
      }
    }
    return bestStatus;
  }

  Future<CortadoLocalDaemonSyncStatus> _sendCommand({
    required String localPath,
    required String type,
    required String workspaceId,
  }) async {
    _ensureNotDisposed();
    if (localPath.trim().isEmpty) {
      throw ArgumentError.value(localPath, 'localPath', 'Must not be empty.');
    }
    if (workspaceId.trim().isEmpty) {
      throw ArgumentError.value(
        workspaceId,
        'workspaceId',
        'Must not be empty.',
      );
    }

    await _ensureConnected();

    final channel = _channel;
    if (channel == null) {
      throw CortadoLocalDaemonUnavailableException(installUrl: installUrl);
    }

    final requestId = 'req-${++_requestCounter}';
    final completer = Completer<CortadoLocalDaemonSyncStatus>();
    _pendingRequests[requestId] = completer;

    channel.sink.add(
      jsonEncode(<String, String>{
        'localPath': localPath,
        'requestId': requestId,
        'type': type,
        'workspaceId': workspaceId,
      }),
    );

    return completer.future.timeout(
      const Duration(seconds: 5),
      onTimeout: () {
        _pendingRequests.remove(requestId);
        throw TimeoutException(
          'Timed out waiting for local daemon response.',
        );
      },
    );
  }

  void _setAvailability(CortadoLocalDaemonAvailability next) {
    _availability = next;
    if (!_availabilityStates.isClosed) {
      _availabilityStates.add(next);
    }
  }

  String _normalizeLocalPath(String path) => path.replaceAll('\\', '/');

  void _onMessage(dynamic raw) {
    switch (raw) {
      case String text:
        _handleTextMessage(text);
        return;
      case Uint8List bytes:
        _handleBinaryMessage(bytes);
        return;
      case List<int> bytes:
        _handleBinaryMessage(Uint8List.fromList(bytes));
        return;
      default:
        throw FormatException(
          'Unsupported local daemon payload type: ${raw.runtimeType}',
        );
    }
  }
}
