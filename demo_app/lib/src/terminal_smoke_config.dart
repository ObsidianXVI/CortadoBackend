class TerminalSmokeConfig {
  const TerminalSmokeConfig({
    required this.baseUrl,
    required this.workspaceId,
    this.shell = defaultShell,
  });

  static const String defaultBaseUrl = 'http://localhost:8080';
  static const String defaultShell = '/bin/bash';

  final String baseUrl;
  final String workspaceId;
  final String shell;

  factory TerminalSmokeConfig.fromUri(Uri uri) {
    final query = uri.queryParameters;

    return TerminalSmokeConfig(
      baseUrl: _firstNonEmpty(
            query,
            const <String>[
              'baseUrl',
              'base_url',
              'controlPlaneBaseUrl',
              'control_plane_base_url',
            ],
          ) ??
          defaultBaseUrl,
      workspaceId: _firstNonEmpty(
            query,
            const <String>['workspaceId', 'workspace_id'],
          ) ??
          '',
      shell: _firstNonEmpty(query, const <String>['shell']) ?? defaultShell,
    );
  }

  TerminalSmokeConfig copyWith({
    String? baseUrl,
    String? workspaceId,
    String? shell,
  }) {
    return TerminalSmokeConfig(
      baseUrl: baseUrl ?? this.baseUrl,
      workspaceId: workspaceId ?? this.workspaceId,
      shell: shell ?? this.shell,
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
}
