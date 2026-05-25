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
    required this.firebaseApiKey,
    required this.firebaseAuthDomain,
    required this.firebaseProjectId,
    required this.firebaseAppId,
    required this.firebaseMessagingSenderId,
    required this.firebaseStorageBucket,
    required this.firebaseMeasurementId,
    required this.firebaseEmail,
    required this.firebasePassword,
    required this.firebaseDevTenantId,
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
  final String firebaseApiKey;
  final String firebaseAuthDomain;
  final String firebaseProjectId;
  final String firebaseAppId;
  final String firebaseMessagingSenderId;
  final String firebaseStorageBucket;
  final String firebaseMeasurementId;
  final String firebaseEmail;
  final String firebasePassword;
  final String firebaseDevTenantId;

  bool get hasFirebaseBootstrapConfig =>
      firebaseApiKey.isNotEmpty &&
      firebaseProjectId.isNotEmpty &&
      firebaseAppId.isNotEmpty &&
      firebaseMessagingSenderId.isNotEmpty;

  factory DemoBootstrapConfig.fromSources({
    required Uri uri,
    required Map<String, String> env,
  }) {
    final query = uri.queryParameters;

    return DemoBootstrapConfig(
      baseUrl: _firstNonEmpty(
            query,
            const <String>[
              'baseUrl',
              'base_url',
              'controlPlaneBaseUrl',
              'control_plane_base_url',
            ],
          ) ??
          _envOrEmpty(env, 'CORTADO_BASE_URL', fallback: defaultBaseUrl),
      apiKey: _firstNonEmpty(
            query,
            const <String>['apiKey', 'api_key', 'demoApiKey', 'demo_api_key'],
          ) ??
          _envOrEmpty(env, 'CORTADO_DEMO_API_KEY'),
      userId: _firstNonEmpty(
            query,
            const <String>['userId', 'user_id', 'demoUserId', 'demo_user_id'],
          ) ??
          _envOrEmpty(env, 'CORTADO_DEMO_USER_ID'),
      workspaceId: _firstNonEmpty(
              query, const <String>['workspaceId', 'workspace_id']) ??
          _envOrEmpty(env, 'CORTADO_WORKSPACE_ID'),
      shell: _firstNonEmpty(query, const <String>['shell']) ??
          _envOrEmpty(env, 'CORTADO_SHELL', fallback: defaultShell),
      image: _firstNonEmpty(
            query,
            const <String>['image', 'workspaceImage', 'workspace_image'],
          ) ??
          _envOrEmpty(env, 'CORTADO_WORKSPACE_IMAGE', fallback: defaultImage),
      filePath: _firstNonEmpty(
            query,
            const <String>['filePath', 'file_path', 'path'],
          ) ??
          _envOrEmpty(env, 'CORTADO_FILE_PATH', fallback: defaultFilePath),
      cpu: _parseDouble(
            _firstNonEmpty(query, const <String>['cpu']) ??
                env['CORTADO_WORKSPACE_CPU'],
          ) ??
          defaultCpu,
      memoryGb: _parseDouble(
            _firstNonEmpty(
                  query,
                  const <String>['memoryGb', 'memory_gb', 'memory'],
                ) ??
                env['CORTADO_WORKSPACE_MEMORY_GB'],
          ) ??
          defaultMemoryGb,
      firebaseApiKey: _firstNonEmpty(
            query,
            const <String>['firebaseApiKey', 'firebase_api_key'],
          ) ??
          _envOrEmpty(env, 'CORTADO_FIREBASE_API_KEY'),
      firebaseAuthDomain: _firstNonEmpty(
            query,
            const <String>['firebaseAuthDomain', 'firebase_auth_domain'],
          ) ??
          _envOrEmpty(env, 'CORTADO_FIREBASE_AUTH_DOMAIN'),
      firebaseProjectId: _firstNonEmpty(
            query,
            const <String>['firebaseProjectId', 'firebase_project_id'],
          ) ??
          _envOrEmpty(env, 'CORTADO_FIREBASE_PROJECT_ID'),
      firebaseAppId: _firstNonEmpty(
            query,
            const <String>['firebaseAppId', 'firebase_app_id'],
          ) ??
          _envOrEmpty(env, 'CORTADO_FIREBASE_APP_ID'),
      firebaseMessagingSenderId: _firstNonEmpty(
            query,
            const <String>[
              'firebaseMessagingSenderId',
              'firebase_messaging_sender_id',
            ],
          ) ??
          _envOrEmpty(env, 'CORTADO_FIREBASE_MESSAGING_SENDER_ID'),
      firebaseStorageBucket: _firstNonEmpty(
            query,
            const <String>['firebaseStorageBucket', 'firebase_storage_bucket'],
          ) ??
          _envOrEmpty(env, 'CORTADO_FIREBASE_STORAGE_BUCKET'),
      firebaseMeasurementId: _firstNonEmpty(
            query,
            const <String>['firebaseMeasurementId', 'firebase_measurement_id'],
          ) ??
          _envOrEmpty(env, 'CORTADO_FIREBASE_MEASUREMENT_ID'),
      firebaseEmail: _firstNonEmpty(
            query,
            const <String>['firebaseEmail', 'firebase_email'],
          ) ??
          _envOrEmpty(env, 'CORTADO_FIREBASE_EMAIL'),
      firebasePassword: _firstNonEmpty(
            query,
            const <String>['firebasePassword', 'firebase_password'],
          ) ??
          _envOrEmpty(env, 'CORTADO_FIREBASE_PASSWORD'),
      firebaseDevTenantId: _firstNonEmpty(
            query,
            const <String>['firebaseDevTenantId', 'firebase_dev_tenant_id'],
          ) ??
          _envOrEmpty(env, 'CORTADO_FIREBASE_DEV_TENANT_ID'),
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
    String? firebaseApiKey,
    String? firebaseAuthDomain,
    String? firebaseProjectId,
    String? firebaseAppId,
    String? firebaseMessagingSenderId,
    String? firebaseStorageBucket,
    String? firebaseMeasurementId,
    String? firebaseEmail,
    String? firebasePassword,
    String? firebaseDevTenantId,
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
      firebaseApiKey: firebaseApiKey ?? this.firebaseApiKey,
      firebaseAuthDomain: firebaseAuthDomain ?? this.firebaseAuthDomain,
      firebaseProjectId: firebaseProjectId ?? this.firebaseProjectId,
      firebaseAppId: firebaseAppId ?? this.firebaseAppId,
      firebaseMessagingSenderId:
          firebaseMessagingSenderId ?? this.firebaseMessagingSenderId,
      firebaseStorageBucket:
          firebaseStorageBucket ?? this.firebaseStorageBucket,
      firebaseMeasurementId:
          firebaseMeasurementId ?? this.firebaseMeasurementId,
      firebaseEmail: firebaseEmail ?? this.firebaseEmail,
      firebasePassword: firebasePassword ?? this.firebasePassword,
      firebaseDevTenantId: firebaseDevTenantId ?? this.firebaseDevTenantId,
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
