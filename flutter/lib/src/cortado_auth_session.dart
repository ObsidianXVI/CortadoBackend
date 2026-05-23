import 'dart:async';
import 'dart:convert';

import 'package:http/http.dart' as http;

typedef CortadoNow = DateTime Function();
typedef CortadoTimerFactory = Timer Function(
  Duration duration,
  void Function() callback,
);

class CortadoAuthSession {
  CortadoAuthSession({
    required this.baseUrl,
    http.Client? httpClient,
    this.refreshLeadTime = const Duration(minutes: 5),
    CortadoNow? now,
    CortadoTimerFactory? timerFactory,
  })  : _client = httpClient ?? http.Client(),
        _ownsClient = httpClient == null,
        _now = now ?? (() => DateTime.now().toUtc()),
        _timerFactory =
            timerFactory ?? ((duration, callback) => Timer(duration, callback));

  final String baseUrl;
  final http.Client _client;
  final bool _ownsClient;
  final Duration refreshLeadTime;
  final CortadoNow _now;
  final CortadoTimerFactory _timerFactory;

  _SessionState? _state;
  Future<void>? _refreshInFlight;
  Timer? _refreshTimer;

  String? get accessToken => _state?.accessToken;
  String? get refreshToken => _state?.refreshToken;
  DateTime? get expiresAt => _state?.expiresAt;
  bool get hasSession => _state != null;

  Future<void> createSession({
    required String apiKey,
    required String userId,
  }) async {
    final response = await _client.post(
      _sessionUri(const <String>['v1', 'sessions']),
      headers: const <String, String>{
        'Content-Type': 'application/json',
      },
      body: jsonEncode(<String, String>{
        'api_key': apiKey,
        'user_id': userId,
      }),
    );

    final payload = _decodeResponse(response);
    final accessToken = payload['access_token'];
    final refreshToken = payload['refresh_token'];
    if (accessToken is! String || refreshToken is! String) {
      throw const FormatException(
        'Session response must contain string access_token and refresh_token values.',
      );
    }

    setTokens(
      accessToken: accessToken,
      refreshToken: refreshToken,
    );
  }

  void setTokens({
    required String accessToken,
    required String refreshToken,
  }) {
    _setState(_SessionState(
      accessToken: accessToken,
      refreshToken: refreshToken,
      expiresAt: _decodeExpiry(accessToken),
    ));
  }

  Future<String?> accessTokenForHttpRequest() async {
    await _refreshIfExpired();
    return _state?.accessToken;
  }

  Future<String?> accessTokenForWebSocket() async {
    await _refreshIfExpired();
    return _state?.accessToken;
  }

  Future<void> refresh() async {
    final state = _state;
    if (state == null) {
      return;
    }

    final inFlight = _refreshInFlight;
    if (inFlight != null) {
      return inFlight;
    }

    final future = _performRefresh(state.refreshToken);
    _refreshInFlight = future;
    try {
      await future;
    } finally {
      if (identical(_refreshInFlight, future)) {
        _refreshInFlight = null;
      }
    }
  }

  Future<void> dispose() async {
    _refreshTimer?.cancel();
    _refreshTimer = null;
    if (_ownsClient) {
      _client.close();
    }
  }

  Future<void> _performRefresh(String refreshToken) async {
    final response = await _client.post(
      _sessionUri(const <String>['v1', 'sessions', 'refresh']),
      headers: const <String, String>{
        'Content-Type': 'application/json',
      },
      body: jsonEncode(<String, String>{
        'refresh_token': refreshToken,
      }),
    );

    final payload = _decodeResponse(response);
    final accessToken = payload['access_token'];
    if (accessToken is! String) {
      throw const FormatException(
        'Refresh response must contain a string access_token value.',
      );
    }

    setTokens(
      accessToken: accessToken,
      refreshToken: refreshToken,
    );
  }

  Future<void> _refreshIfExpired() async {
    final state = _state;
    if (state == null) {
      return;
    }
    if (state.expiresAt.isAfter(_now())) {
      return;
    }
    await refresh();
  }

  void _setState(_SessionState nextState) {
    _state = nextState;
    _scheduleRefresh(nextState);
  }

  void _scheduleRefresh(_SessionState state) {
    _refreshTimer?.cancel();

    final refreshAt = state.expiresAt.subtract(refreshLeadTime);
    final delay = refreshAt.difference(_now());
    final nextDelay = delay.isNegative ? Duration.zero : delay;
    _refreshTimer = _timerFactory(nextDelay, () {
      unawaited(refresh());
    });
  }

  Uri _sessionUri(List<String> segments) {
    final baseUri = Uri.parse(baseUrl);
    return baseUri.replace(
      pathSegments: <String>[
        ...baseUri.pathSegments.where((segment) => segment.isNotEmpty),
        ...segments,
      ],
    );
  }

  Map<String, dynamic> _decodeResponse(http.Response response) {
    if (response.statusCode < 200 || response.statusCode >= 300) {
      throw CortadoAuthException(
        statusCode: response.statusCode,
        message: utf8.decode(response.bodyBytes).trim(),
      );
    }

    final decoded = jsonDecode(utf8.decode(response.bodyBytes));
    if (decoded is! Map<String, dynamic>) {
      throw const FormatException('Expected a JSON object response body.');
    }
    return decoded;
  }
}

class CortadoAuthException implements Exception {
  const CortadoAuthException({
    required this.statusCode,
    required this.message,
  });

  final int statusCode;
  final String message;

  @override
  String toString() {
    if (message.isEmpty) {
      return 'CortadoAuthException(statusCode: $statusCode)';
    }
    return 'CortadoAuthException(statusCode: $statusCode, message: $message)';
  }
}

class _SessionState {
  const _SessionState({
    required this.accessToken,
    required this.refreshToken,
    required this.expiresAt,
  });

  final String accessToken;
  final String refreshToken;
  final DateTime expiresAt;
}

DateTime _decodeExpiry(String accessToken) {
  final parts = accessToken.split('.');
  if (parts.length != 3) {
    throw const FormatException(
        'JWT access token must contain three segments.');
  }

  final payloadBytes = base64Url.decode(base64Url.normalize(parts[1]));
  final payload = jsonDecode(utf8.decode(payloadBytes));
  if (payload is! Map<String, dynamic>) {
    throw const FormatException('JWT payload must decode to a JSON object.');
  }

  final exp = payload['exp'];
  if (exp is! num) {
    throw const FormatException('JWT payload is missing a numeric exp claim.');
  }

  return DateTime.fromMillisecondsSinceEpoch(
    exp.toInt() * 1000,
    isUtc: true,
  );
}
