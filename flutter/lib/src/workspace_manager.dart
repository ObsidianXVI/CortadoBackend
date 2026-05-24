import 'dart:async';
import 'dart:convert';
import 'dart:typed_data';

import 'package:http/http.dart' as http;

import 'cortado_auth_session.dart';
import 'cortado_client.dart';
import 'workspace_models.dart';

class WorkspaceManager {
  WorkspaceManager({
    required this.baseUrl,
    this.authSession,
    String devToken = defaultDevToken,
    http.Client? httpClient,
    bool? useBrowserAuth,
    this.transitionalPollInterval = const Duration(seconds: 3),
    this.runningPollInterval = const Duration(seconds: 30),
  })  : _client = httpClient ?? http.Client(),
        _devToken = devToken,
        _ownsClient = httpClient == null;

  final String baseUrl;
  final CortadoAuthSession? authSession;
  final http.Client _client;
  final String _devToken;
  final bool _ownsClient;
  final Duration transitionalPollInterval;
  final Duration runningPollInterval;

  String get devToken => _devToken;

  Future<Workspace> create({
    required String image,
    WorkspaceResources? resources,
  }) async {
    if (image.trim().isEmpty) {
      throw ArgumentError.value(image, 'image', 'Must not be empty.');
    }

    final payload = <String, Object?>{
      'image': image,
      if (resources != null) 'resources': resources.toJson(),
    };
    final response = await _client.post(
      _collectionUri(),
      headers: await _headers(contentType: 'application/json'),
      body: jsonEncode(payload),
    );

    return _decodeWorkspace(response);
  }

  Future<void> start(String id) async {
    await _transition(id, 'start');
  }

  Future<void> stop(String id) async {
    await _transition(id, 'stop');
  }

  Future<List<WorkspaceDirectoryEntry>> listDirectory(
    String workspaceId, {
    String path = '/',
  }) async {
    _validateWorkspaceId(workspaceId);

    final response = await _client.get(
      _workspaceUri(workspaceId,
          action: 'files',
          queryParameters: <String, String>{
            'path': _fileApiPath(path),
          }),
      headers: await _headers(),
    );

    final body = _decodeJsonObject(response.bodyBytes);
    final entries = body['entries'];
    if (entries is! List<Object?>) {
      throw const FormatException(
        'File list response payload must contain an entries list.',
      );
    }

    return entries.map((entry) {
      if (entry is! Map<String, dynamic>) {
        throw const FormatException(
          'Each file list entry must be a JSON object.',
        );
      }
      return WorkspaceDirectoryEntry.fromJson(entry);
    }).toList(growable: false);
  }

  Future<Uint8List> readFile(
    String workspaceId, {
    required String path,
  }) async {
    _validateWorkspaceId(workspaceId);

    final response = await _client.get(
      _workspaceUri(
        workspaceId,
        action: 'files/content',
        queryParameters: _fileQueryParameters(path),
      ),
      headers: await _headers(),
    );
    _throwIfError(response);
    return Uint8List.fromList(response.bodyBytes);
  }

  Future<WorkspaceWriteFileResult> writeFile(
    String workspaceId, {
    required String path,
    List<int> content = const <int>[],
    bool createMissingDirs = true,
  }) async {
    _validateWorkspaceId(workspaceId);

    final response = await _client.put(
      _workspaceUri(
        workspaceId,
        action: 'files/content',
        queryParameters: <String, String>{
          ..._fileQueryParameters(path),
          if (!createMissingDirs) 'createMissingDirs': 'false',
        },
      ),
      headers: await _headers(contentType: 'application/octet-stream'),
      body: content,
    );
    _throwIfError(response);
    return WorkspaceWriteFileResult.fromJson(
        _decodeJsonObject(response.bodyBytes));
  }

  Future<void> makeDir(
    String workspaceId, {
    required String path,
  }) async {
    _validateWorkspaceId(workspaceId);

    final response = await _client.post(
      _workspaceUri(
        workspaceId,
        action: 'files/directory',
        queryParameters: _fileQueryParameters(path),
      ),
      headers: await _headers(),
    );
    _throwIfError(response);
  }

  Future<void> renamePath(
    String workspaceId, {
    required String oldPath,
    required String newPath,
  }) async {
    _validateWorkspaceId(workspaceId);

    final response = await _client.post(
      _workspaceUri(
        workspaceId,
        action: 'files/rename',
        queryParameters: <String, String>{
          'path': _fileApiPath(oldPath),
          'newPath': _fileApiPath(newPath),
        },
      ),
      headers: await _headers(),
    );
    _throwIfError(response);
  }

  Future<void> deletePath(
    String workspaceId, {
    required String path,
  }) async {
    _validateWorkspaceId(workspaceId);

    final response = await _client.delete(
      _workspaceUri(
        workspaceId,
        action: 'files',
        queryParameters: _fileQueryParameters(path),
      ),
      headers: await _headers(),
    );
    _throwIfError(response);
  }

  Stream<WorkspaceStatus> watchStatus(String id) {
    _validateWorkspaceId(id);

    late final StreamController<WorkspaceStatus> controller;
    Timer? nextPollTimer;
    WorkspaceStatus? previousStatus;
    var cancelled = false;

    Future<void> closeController() async {
      if (!controller.isClosed) {
        await controller.close();
      }
    }

    Future<void> poll() async {
      if (cancelled || controller.isClosed) {
        return;
      }

      try {
        final workspace = await _getWorkspace(id);
        if (cancelled || controller.isClosed) {
          return;
        }

        final status = WorkspaceStatus.fromWorkspace(workspace);
        if (status != previousStatus) {
          controller.add(status);
          previousStatus = status;
        }

        final delay = status.nextPollDelay(
          transitionalInterval: transitionalPollInterval,
          runningInterval: runningPollInterval,
        );
        if (delay == null) {
          await closeController();
          return;
        }

        nextPollTimer = Timer(delay, () {
          unawaited(poll());
        });
      } catch (error, stackTrace) {
        if (!controller.isClosed) {
          controller.addError(error, stackTrace);
        }
        await closeController();
      }
    }

    controller = StreamController<WorkspaceStatus>(
      onListen: () {
        unawaited(poll());
      },
      onCancel: () async {
        cancelled = true;
        nextPollTimer?.cancel();
        nextPollTimer = null;
        await closeController();
      },
    );

    return controller.stream;
  }

  Future<void> dispose() async {
    if (_ownsClient) {
      _client.close();
    }
  }

  Future<Workspace> _getWorkspace(String id) async {
    _validateWorkspaceId(id);

    final response = await _client.get(
      _workspaceUri(id),
      headers: await _headers(),
    );

    return _decodeWorkspace(response);
  }

  Future<void> _transition(String id, String action) async {
    _validateWorkspaceId(id);

    final response = await _client.post(
      _workspaceUri(id, action: action),
      headers: await _headers(),
    );
    _throwIfError(response);
  }

  Workspace _decodeWorkspace(http.Response response) {
    _throwIfError(response);

    final body = _decodeJsonObject(response.bodyBytes);
    final workspace = body['workspace'];
    if (workspace is! Map<String, dynamic>) {
      throw const FormatException(
        'Workspace response payload must contain a workspace object.',
      );
    }

    return Workspace.fromJson(workspace);
  }

  Map<String, dynamic> _decodeJsonObject(List<int> bodyBytes) {
    final decoded = jsonDecode(utf8.decode(bodyBytes));
    if (decoded is! Map<String, dynamic>) {
      throw const FormatException('Expected a JSON object response body.');
    }
    return decoded;
  }

  void _throwIfError(http.Response response) {
    if (response.statusCode >= 200 && response.statusCode < 300) {
      return;
    }

    throw WorkspaceRequestException(
      statusCode: response.statusCode,
      message: utf8.decode(response.bodyBytes).trim(),
    );
  }

  Future<Map<String, String>> _headers({String? contentType}) async {
    final headers = <String, String>{};
    final accessToken = await authSession?.accessTokenForHttpRequest();
    if (accessToken != null) {
      headers['Authorization'] = 'Bearer $accessToken';
    } else {
      headers['X-Cortado-Dev-Token'] = _devToken;
    }
    if (contentType != null) {
      headers['Content-Type'] = contentType;
    }
    return headers;
  }

  Uri _collectionUri() => _buildUri(const <String>['v1', 'workspaces']);

  Uri _workspaceUri(
    String id, {
    String? action,
    Map<String, String>? queryParameters,
  }) =>
      _buildUri(
        <String>[
          'v1',
          'workspaces',
          id,
          if (action != null)
            ...action.split('/').where((segment) => segment.isNotEmpty),
        ],
        queryParameters: queryParameters,
      );

  Uri _buildUri(List<String> segments, {Map<String, String>? queryParameters}) {
    final baseUri = Uri.parse(baseUrl);
    final mergedQueryParameters = <String, String>{
      ...baseUri.queryParameters,
      if (queryParameters != null) ...queryParameters,
    };

    return baseUri.replace(
      pathSegments: <String>[
        ...baseUri.pathSegments.where((segment) => segment.isNotEmpty),
        ...segments,
      ],
      queryParameters:
          mergedQueryParameters.isEmpty ? null : mergedQueryParameters,
    );
  }

  void _validateWorkspaceId(String id) {
    if (id.trim().isEmpty) {
      throw ArgumentError.value(id, 'id', 'Must not be empty.');
    }
  }

  String _fileApiPath(String path) {
    final trimmed = path.trim();
    if (trimmed.isEmpty || trimmed == '/' || trimmed == '.') {
      return '';
    }
    return trimmed.startsWith('/') ? trimmed.substring(1) : trimmed;
  }

  Map<String, String> _fileQueryParameters(String path) {
    return <String, String>{
      'path': _fileApiPath(path),
    };
  }
}

class WorkspaceDirectoryEntry {
  const WorkspaceDirectoryEntry({
    required this.isDir,
    required this.modTime,
    required this.name,
    required this.permissions,
    required this.size,
  });

  factory WorkspaceDirectoryEntry.fromJson(Map<String, dynamic> json) {
    return WorkspaceDirectoryEntry(
      name: json['name'] as String,
      size: (json['size'] as num).toInt(),
      isDir: json['isDir'] as bool,
      modTime: DateTime.parse(json['modTime'] as String),
      permissions: (json['permissions'] as num).toInt(),
    );
  }

  final bool isDir;
  final DateTime modTime;
  final String name;
  final int permissions;
  final int size;
}

class WorkspaceWriteFileResult {
  const WorkspaceWriteFileResult({
    required this.bytesWritten,
    required this.checksum,
  });

  factory WorkspaceWriteFileResult.fromJson(Map<String, dynamic> json) {
    final checksum = json['checksum'];
    return WorkspaceWriteFileResult(
      bytesWritten: (json['bytesWritten'] as num).toInt(),
      checksum: checksum is List<Object?>
          ? Uint8List.fromList(
              checksum
                  .map((value) => (value as num).toInt())
                  .toList(growable: false),
            )
          : Uint8List(0),
    );
  }

  final int bytesWritten;
  final Uint8List checksum;
}

class WorkspaceRequestException implements Exception {
  const WorkspaceRequestException({
    required this.statusCode,
    required this.message,
  });

  final int statusCode;
  final String message;

  @override
  String toString() {
    if (message.isEmpty) {
      return 'WorkspaceRequestException(statusCode: $statusCode)';
    }
    return 'WorkspaceRequestException(statusCode: $statusCode, message: $message)';
  }
}
