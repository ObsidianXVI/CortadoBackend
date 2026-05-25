class DemoBootstrapConfig {
  const DemoBootstrapConfig({
    required this.baseUrl,
    required this.apiKey,
    required this.userId,
    required this.workspaceId,
    required this.shell,
    required this.image,
    required this.filePath,
    required this.cpu,
    required this.memoryGb,
  });

  static const String defaultBaseUrl = 'http://localhost:8080';
  static const String defaultShell = '/bin/bash';
  static const String defaultImage = 'ubuntu:24.04';
  static const String defaultFilePath = 'lib/main.dart';
  static const double defaultCpu = 1;
  static const double defaultMemoryGb = 2;

  final String baseUrl;
  final String apiKey;
  final String userId;
  final String workspaceId;
  final String shell;
  final String image;
  final String filePath;
  final double cpu;
  final double memoryGb;

  factory DemoBootstrapConfig.fromSources({
    required Uri uri,
    required Map<String, String> env,
  }) {
    final query = uri.queryParameters;

    return DemoBootstrapConfig(
      baseUrl:
          _firstNonEmpty(
            query,
            const <String>[
              'baseUrl',
              'base_url',
              'controlPlaneBaseUrl',
              'control_plane_base_url',
            ],
          ) ??
          _envOrEmpty(env, 'CORTADO_BASE_URL', fallback: defaultBaseUrl),
      apiKey:
          _firstNonEmpty(
            query,
            const <String>['apiKey', 'api_key', 'demoApiKey', 'demo_api_key'],
          ) ??
          _envOrEmpty(env, 'CORTADO_DEMO_API_KEY'),
      userId:
          _firstNonEmpty(
            query,
            const <String>['userId', 'user_id', 'demoUserId', 'demo_user_id'],
          ) ??
          _envOrEmpty(env, 'CORTADO_DEMO_USER_ID'),
      workspaceId:
          _firstNonEmpty(query, const <String>['workspaceId', 'workspace_id']) ??
          _envOrEmpty(env, 'CORTADO_WORKSPACE_ID'),
      shell:
          _firstNonEmpty(query, const <String>['shell']) ??
          _envOrEmpty(env, 'CORTADO_SHELL', fallback: defaultShell),
      image:
          _firstNonEmpty(
            query,
            const <String>['image', 'workspaceImage', 'workspace_image'],
          ) ??
          _envOrEmpty(env, 'CORTADO_WORKSPACE_IMAGE', fallback: defaultImage),
      filePath:
          _firstNonEmpty(
            query,
            const <String>['filePath', 'file_path', 'path'],
          ) ??
          _envOrEmpty(env, 'CORTADO_FILE_PATH', fallback: defaultFilePath),
      cpu:
          _parseDouble(
            _firstNonEmpty(query, const <String>['cpu']) ??
                env['CORTADO_WORKSPACE_CPU'],
          ) ??
          defaultCpu,
      memoryGb:
          _parseDouble(
            _firstNonEmpty(
                  query,
                  const <String>['memoryGb', 'memory_gb', 'memory'],
                ) ??
                env['CORTADO_WORKSPACE_MEMORY_GB'],
          ) ??
          defaultMemoryGb,
    );
  }

  DemoBootstrapConfig copyWith({
    String? baseUrl,
    String? apiKey,
    String? userId,
    String? workspaceId,
    String? shell,
    String? image,
    String? filePath,
    double? cpu,
    double? memoryGb,
  }) {
    return DemoBootstrapConfig(
      baseUrl: baseUrl ?? this.baseUrl,
      apiKey: apiKey ?? this.apiKey,
      userId: userId ?? this.userId,
      workspaceId: workspaceId ?? this.workspaceId,
      shell: shell ?? this.shell,
      image: image ?? this.image,
      filePath: filePath ?? this.filePath,
      cpu: cpu ?? this.cpu,
      memoryGb: memoryGb ?? this.memoryGb,
    );
  }

  static String? _firstNonEmpty(
    Map<String, String> query,
    List<String> keys,
  ) {
    for (final key in keys) {
      final value = query[key]?.trim();
      if (value != null && value.isNotEmpty) {
        return value;
      }
    }
    return null;
  }

  static String _envOrEmpty(
    Map<String, String> env,
    String key, {
    String fallback = '',
  }) {
    final value = env[key]?.trim();
    if (value == null || value.isEmpty) {
      return fallback;
    }
    return value;
  }

  static double? _parseDouble(String? raw) {
    final value = raw?.trim();
    if (value == null || value.isEmpty) {
      return null;
    }
    return double.tryParse(value);
  }
}
