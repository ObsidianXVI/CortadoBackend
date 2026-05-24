import 'dart:async';
import 'dart:convert';
import 'dart:typed_data';

import '../cortado_client.dart';
import '../filesystem/vfs_notifier.dart';
import '../mux_frame.dart';

typedef CortadoLSPDiagnostic = Map<String, Object?>;
typedef CortadoLSPDiagnosticsByUri = Map<String, List<CortadoLSPDiagnostic>>;

class CortadoLSPClient {
  CortadoLSPClient({
    required this.client,
    this.channelId = muxLspChannelStartId,
    this.language = 'dart',
    this.workspaceRoot = '/workspace',
  });

  final CortadoClient client;
  final int channelId;
  final String language;
  final String workspaceRoot;

  final StreamController<void> _stateChanges =
      StreamController<void>.broadcast();
  final StreamController<CortadoLSPDiagnosticsByUri> _diagnosticsChanges =
      StreamController<CortadoLSPDiagnosticsByUri>.broadcast();
  final Map<int, Completer<Object?>> _pendingRequests =
      <int, Completer<Object?>>{};
  final List<_PendingLspOperation<Object?>> _pendingOperations =
      <_PendingLspOperation<Object?>>[];
  final Map<String, int> _documentVersions = <String, int>{};
  final Map<String, List<CortadoLSPDiagnostic>> _diagnostics =
      <String, List<CortadoLSPDiagnostic>>{};

  StreamSubscription<MuxFrame>? _frameSubscription;
  Future<void>? _initializationFuture;
  bool _channelOpened = false;
  bool _disposed = false;
  bool _initialized = false;
  bool _initializing = false;
  bool _flushingQueue = false;
  int _nextRequestId = 0;
  Object? _initializationError;

  Stream<void> get stateChanges => _stateChanges.stream;

  Stream<CortadoLSPDiagnosticsByUri> get diagnosticsStream =>
      _diagnosticsChanges.stream;

  bool get isInitialized => _initialized;

  bool get isInitializing => _initializing && !_initialized;

  Object? get initializationError => _initializationError;

  bool isDocumentOpen(String path) =>
      _documentVersions.containsKey(_documentUriForPath(path));

  Future<void> ensureInitialized() => _ensureStarted();

  Future<Object?> sendRequest(
    String method, {
    Map<String, Object?> params = const <String, Object?>{},
  }) async {
    _ensureNotDisposed();

    if (_initialized) {
      return _sendRequestNow(method, params);
    }

    return _enqueueOperation<Object?>(() => _sendRequestNow(method, params));
  }

  Future<void> didOpenTextDocument({
    required String path,
    required String languageId,
    required String text,
  }) async {
    final uri = _documentUriForPath(path);
    final version = _documentVersions[uri] = 1;

    await _dispatchNotification(
      'textDocument/didOpen',
      <String, Object?>{
        'textDocument': <String, Object?>{
          'uri': uri,
          'languageId': languageId,
          'version': version,
          'text': text,
        },
      },
    );
  }

  Future<void> didChangeTextDocument({
    required String path,
    required String text,
  }) async {
    final uri = _documentUriForPath(path);
    final version = (_documentVersions[uri] ?? 0) + 1;
    _documentVersions[uri] = version;

    await _dispatchNotification(
      'textDocument/didChange',
      <String, Object?>{
        'textDocument': <String, Object?>{
          'uri': uri,
          'version': version,
        },
        'contentChanges': <Object?>[
          <String, Object?>{'text': text},
        ],
      },
    );
  }

  Future<void> didCloseTextDocument({
    required String path,
  }) async {
    final uri = _documentUriForPath(path);
    _documentVersions.remove(uri);
    if (_diagnostics.remove(uri) != null) {
      _emitDiagnostics();
    }

    await _dispatchNotification(
      'textDocument/didClose',
      <String, Object?>{
        'textDocument': <String, Object?>{
          'uri': uri,
        },
      },
    );
  }

  Future<void> dispose() async {
    if (_disposed) {
      return;
    }

    final closeFutures = <Future<void>>[];
    if (_initialized) {
      for (final uri in _documentVersions.keys.toList(growable: false)) {
        closeFutures.add(
          _sendNotificationNow(
            'textDocument/didClose',
            <String, Object?>{
              'textDocument': <String, Object?>{'uri': uri},
            },
          ),
        );
      }
    }
    await Future.wait(closeFutures, eagerError: false);
    _documentVersions.clear();
    _failPendingRequests(
      StateError('CortadoLSPClient was disposed.'),
      StackTrace.current,
    );
    _failQueuedOperations(
      StateError('CortadoLSPClient was disposed.'),
      StackTrace.current,
    );

    await _frameSubscription?.cancel();
    _frameSubscription = null;

    if (_channelOpened) {
      try {
        client.sendFrame(channelId, muxMessageTypeClose, Uint8List(0));
      } on Object {
        // Ignore teardown races with the parent websocket.
      }
      _channelOpened = false;
    }

    _disposed = true;
    await _stateChanges.close();
    await _diagnosticsChanges.close();
  }

  Future<T> _enqueueOperation<T>(Future<T> Function() run) {
    final operation = _PendingLspOperation<T>(run);
    _pendingOperations.add(operation);
    unawaited(_ensureStarted());
    return operation.completer.future;
  }

  Future<void> _dispatchNotification(
    String method,
    Map<String, Object?> params,
  ) async {
    _ensureNotDisposed();

    if (_initialized) {
      await _sendNotificationNow(method, params);
      return;
    }

    await _enqueueOperation<Object?>(
      () => _sendNotificationNow(method, params).then<Object?>((_) => null),
    );
  }

  Future<void> _ensureStarted() async {
    _ensureNotDisposed();
    if (_initialized) {
      return;
    }
    final existing = _initializationFuture;
    if (existing != null) {
      return existing;
    }

    _frameSubscription =
        client.framesForChannel(channelId).listen(_handleFrame);
    _channelOpened = true;
    _initializing = true;
    _initializationError = null;
    _emitStateChange();

    client.sendFrame(
      channelId,
      muxMessageTypeOpen,
      Uint8List.fromList(utf8.encode(language)),
    );

    final future = _sendRequestNow(
      'initialize',
      <String, Object?>{
        'processId': null,
        'rootUri': _workspaceRootUri(),
        'workspaceFolders': <Object?>[
          <String, Object?>{
            'name': 'workspace',
            'uri': _workspaceRootUri(),
          },
        ],
        'clientInfo': const <String, Object?>{
          'name': 'cortado',
        },
        'capabilities': const <String, Object?>{
          'workspace': <String, Object?>{
            'workspaceFolders': true,
          },
          'textDocument': <String, Object?>{
            'publishDiagnostics': <String, Object?>{},
            'synchronization': <String, Object?>{
              'didSave': false,
              'dynamicRegistration': false,
              'willSave': false,
              'willSaveWaitUntil': false,
            },
          },
        },
      },
    ).then((_) async {
      await _sendNotificationNow('initialized', const <String, Object?>{});
      _initialized = true;
      _initializing = false;
      _initializationFuture = null;
      _emitStateChange();
      await _flushQueuedOperations();
    }, onError: (Object error, StackTrace stackTrace) async {
      _initializationError = error;
      _initializing = false;
      _initializationFuture = null;
      _emitStateChange();
      _failQueuedOperations(error, stackTrace);
      Error.throwWithStackTrace(error, stackTrace);
    });

    _initializationFuture = future;
    return future;
  }

  Future<void> _flushQueuedOperations() async {
    if (_flushingQueue || !_initialized) {
      return;
    }
    _flushingQueue = true;
    try {
      while (_pendingOperations.isNotEmpty) {
        final operation = _pendingOperations.removeAt(0);
        await operation.run();
      }
    } finally {
      _flushingQueue = false;
    }
  }

  Future<Object?> _sendRequestNow(
    String method,
    Map<String, Object?> params,
  ) {
    final id = ++_nextRequestId;
    final completer = Completer<Object?>();
    _pendingRequests[id] = completer;
    _sendJson(<String, Object?>{
      'jsonrpc': '2.0',
      'id': id,
      'method': method,
      'params': params,
    });
    return completer.future;
  }

  Future<void> _sendNotificationNow(
    String method,
    Map<String, Object?> params,
  ) async {
    _sendJson(<String, Object?>{
      'jsonrpc': '2.0',
      'method': method,
      'params': params,
    });
  }

  void _sendJson(Map<String, Object?> payload) {
    _ensureNotDisposed();
    client.sendFrame(
      channelId,
      muxMessageTypeData,
      Uint8List.fromList(utf8.encode(jsonEncode(payload))),
    );
  }

  Future<void> _handleFrame(MuxFrame frame) async {
    switch (frame.messageType) {
      case muxMessageTypeData:
        _handleJsonMessage(frame.payload);
        return;
      case muxMessageTypeClose:
      case muxMessageTypeError:
        final message = frame.payload.isEmpty
            ? 'LSP channel closed.'
            : utf8.decode(frame.payload);
        _handleFatalError(StateError(message));
        return;
      default:
        return;
    }
  }

  void _handleJsonMessage(Uint8List payload) {
    final decoded = jsonDecode(utf8.decode(payload));
    if (decoded is! Map<Object?, Object?>) {
      return;
    }

    final id = decoded['id'];
    final method = decoded['method'];
    if (method is String) {
      if (id != null) {
        _handleServerRequest(method, id, decoded['params']);
        return;
      }
      _handleServerNotification(method, decoded['params']);
      return;
    }

    if (id is! int) {
      return;
    }
    final completer = _pendingRequests.remove(id);
    if (completer == null || completer.isCompleted) {
      return;
    }

    if (decoded['error'] case final Map<Object?, Object?> error) {
      final message = error['message'];
      completer.completeError(
        StateError(
          message is String ? message : jsonEncode(error),
        ),
      );
      return;
    }

    completer.complete(decoded['result']);
  }

  void _handleServerRequest(String method, Object id, Object? params) {
    final result = switch (method) {
      'workspace/configuration' => _workspaceConfigurationResult(params),
      'client/registerCapability' => null,
      'client/unregisterCapability' => null,
      'window/workDoneProgress/create' => null,
      'workspace/codeLens/refresh' => null,
      'workspace/diagnostic/refresh' => null,
      'workspace/inlayHint/refresh' => null,
      'workspace/semanticTokens/refresh' => null,
      _ => null,
    };

    _sendJson(<String, Object?>{
      'jsonrpc': '2.0',
      'id': id,
      'result': result,
    });
  }

  Object? _workspaceConfigurationResult(Object? params) {
    if (params is! Map<Object?, Object?>) {
      return const <Object?>[];
    }
    final items = params['items'];
    if (items is! List<Object?>) {
      return const <Object?>[];
    }
    return List<Object?>.filled(items.length, null, growable: false);
  }

  void _handleServerNotification(String method, Object? params) {
    switch (method) {
      case 'textDocument/publishDiagnostics':
        _handlePublishDiagnostics(params);
        return;
      default:
        return;
    }
  }

  void _handlePublishDiagnostics(Object? params) {
    if (params is! Map<Object?, Object?>) {
      return;
    }
    final uri = params['uri'];
    final diagnostics = params['diagnostics'];
    if (uri is! String || diagnostics is! List<Object?>) {
      return;
    }

    _diagnostics[uri] = diagnostics
        .whereType<Map<Object?, Object?>>()
        .map((entry) => Map<String, Object?>.fromEntries(
              entry.entries.where((item) => item.key is String).map(
                    (item) => MapEntry(item.key as String, item.value),
                  ),
            ))
        .toList(growable: false);
    _emitDiagnostics();
  }

  void _emitDiagnostics() {
    if (_diagnosticsChanges.isClosed) {
      return;
    }
    _diagnosticsChanges.add(
      Map<String, List<CortadoLSPDiagnostic>>.unmodifiable(
        _diagnostics.map(
          (uri, entries) => MapEntry(
            uri,
            List<CortadoLSPDiagnostic>.unmodifiable(
              entries.map(Map<String, Object?>.unmodifiable),
            ),
          ),
        ),
      ),
    );
  }

  void _handleFatalError(Object error, [StackTrace? stackTrace]) {
    _initializing = false;
    _initialized = false;
    _initializationError = error;
    _emitStateChange();

    final trace = stackTrace ?? StackTrace.current;
    _failPendingRequests(error, trace);
    _failQueuedOperations(error, trace);
  }

  void _failPendingRequests(Object error, StackTrace stackTrace) {
    final pending = _pendingRequests.values.toList(growable: false);
    _pendingRequests.clear();
    for (final completer in pending) {
      if (!completer.isCompleted) {
        completer.completeError(error, stackTrace);
      }
    }
  }

  void _failQueuedOperations(Object error, StackTrace stackTrace) {
    final pending = _pendingOperations.toList(growable: false);
    _pendingOperations.clear();
    for (final operation in pending) {
      operation.completeError(error, stackTrace);
    }
  }

  void _emitStateChange() {
    if (_stateChanges.isClosed) {
      return;
    }
    _stateChanges.add(null);
  }

  String _workspaceRootUri() => Uri(
        scheme: 'file',
        path: normalizeVfsPath(workspaceRoot),
      ).toString();

  String _documentUriForPath(String path) => Uri(
        scheme: 'file',
        path: '${normalizeVfsPath(workspaceRoot)}${normalizeVfsPath(path)}',
      ).toString();

  void _ensureNotDisposed() {
    if (_disposed) {
      throw StateError('CortadoLSPClient has been disposed.');
    }
  }
}

class _PendingLspOperation<T> {
  _PendingLspOperation(this._action);

  final Future<T> Function() _action;
  final Completer<T> completer = Completer<T>();

  Future<void> run() async {
    if (completer.isCompleted) {
      return;
    }
    try {
      completer.complete(await _action());
    } catch (error, stackTrace) {
      if (!completer.isCompleted) {
        completer.completeError(error, stackTrace);
      }
    }
  }

  void completeError(Object error, StackTrace stackTrace) {
    if (!completer.isCompleted) {
      completer.completeError(error, stackTrace);
    }
  }
}
