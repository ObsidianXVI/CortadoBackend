import 'dart:async';
import 'dart:convert';

import 'package:http/http.dart' as http;

import '../cortado_auth_session.dart';
import '../cortado_client.dart';

class CortadoCompletionContext {
  const CortadoCompletionContext({
    required this.prefix,
    required this.workspaceId,
    this.path,
    this.suffix = '',
  });

  final String workspaceId;
  final String prefix;
  final String suffix;
  final String? path;
}

class CortadoAIService {
  CortadoAIService({
    required this.baseUrl,
    this.authSession,
    String devToken = defaultDevToken,
    http.Client? httpClient,
  })  : _client = httpClient ?? http.Client(),
        _devToken = devToken,
        _ownsClient = httpClient == null;

  final String baseUrl;
  final CortadoAuthSession? authSession;
  final http.Client _client;
  final String _devToken;
  final bool _ownsClient;

  Stream<String> streamCompletion(CortadoCompletionContext context) async* {
    _validateContext(context);

    final request = http.Request(
      'POST',
      _completionUri(context.workspaceId),
    )
      ..headers.addAll(await _headers(contentType: 'application/json'))
      ..body = jsonEncode(<String, Object?>{
        'path': context.path,
        'prefix': context.prefix,
        'suffix': context.suffix,
      });

    final response = await _client.send(request);
    if (response.statusCode < 200 || response.statusCode >= 300) {
      final message = (await response.stream.bytesToString()).trim();
      throw CortadoAIException(
        message: message,
        statusCode: response.statusCode,
      );
    }

    final eventData = <String>[];
    await for (final line in response.stream
        .transform(utf8.decoder)
        .transform(const LineSplitter())) {
      if (line.isEmpty) {
        final token = _decodeEvent(eventData);
        eventData.clear();
        if (token != null) {
          yield token;
        }
        continue;
      }

      if (line.startsWith('data:')) {
        eventData.add(line.substring(5).trimLeft());
      }
    }

    final token = _decodeEvent(eventData);
    if (token != null) {
      yield token;
    }
  }

  Future<void> dispose() async {
    if (_ownsClient) {
      _client.close();
    }
  }

  Uri _completionUri(String workspaceId) {
    final baseUri = Uri.parse(baseUrl);
    return baseUri.replace(
      pathSegments: <String>[
        ...baseUri.pathSegments.where((segment) => segment.isNotEmpty),
        'v1',
        'workspaces',
        workspaceId,
        'ai',
        'complete',
      ],
      queryParameters:
          baseUri.queryParameters.isEmpty ? null : baseUri.queryParameters,
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

  String? _decodeEvent(List<String> eventData) {
    if (eventData.isEmpty) {
      return null;
    }

    final decoded = jsonDecode(eventData.join('\n'));
    if (decoded is! Map<String, dynamic>) {
      throw const FormatException(
          'Completion SSE event must decode to an object.');
    }

    final error = decoded['error'];
    if (error is String && error.trim().isNotEmpty) {
      throw CortadoAIException(message: error.trim());
    }

    final token = decoded['token'];
    if (token == null) {
      return null;
    }
    if (token is! String) {
      throw const FormatException(
        'Completion SSE token payload must contain a string token.',
      );
    }
    return token;
  }

  void _validateContext(CortadoCompletionContext context) {
    if (context.workspaceId.trim().isEmpty) {
      throw ArgumentError.value(
        context.workspaceId,
        'workspaceId',
        'Must not be empty.',
      );
    }
    if (context.prefix.isEmpty && context.suffix.isEmpty) {
      throw ArgumentError(
        'Completion context must include a prefix or suffix.',
      );
    }
  }
}

class CortadoAIException implements Exception {
  const CortadoAIException({
    required this.message,
    this.statusCode,
  });

  final String message;
  final int? statusCode;

  @override
  String toString() {
    if (statusCode == null) {
      return 'CortadoAIException(message: $message)';
    }
    return 'CortadoAIException(statusCode: $statusCode, message: $message)';
  }
}
