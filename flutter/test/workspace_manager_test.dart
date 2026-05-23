import 'dart:async';
import 'dart:convert';
import 'dart:typed_data';

import 'package:cortado/cortado.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;

void main() {
  group('WorkspaceManager — auth and URL building', () {
    test('create uses header auth on non-web and correct collection URL',
        () async {
      final requests = <RecordedRequest>[];
      final client = RecordingClient((request, body) async {
        requests.add(RecordedRequest(DateTime.now(), request, body));
        if (request.method == 'POST' &&
            request.url.path.endsWith('/v1/workspaces')) {
          final payload = jsonEncode({
            'workspace': _sampleWorkspaceJson(id: 'w1'),
          });
          return _stringResponse(200, payload);
        }
        return _stringResponse(404, 'not found');
      });

      final manager = WorkspaceManager(
        baseUrl: 'http://localhost:8080/api',
        httpClient: client,
        useBrowserAuth: false,
      );

      final ws = await manager.create(image: 'ghcr.io/acme/image:tag');
      expect(ws.id, 'w1');

      expect(requests, hasLength(1));
      final req = requests.single.request as http.Request;

      expect(req.method, 'POST');
      expect(req.url, Uri.parse('http://localhost:8080/api/v1/workspaces'));
      // Header auth is used for non-web.
      expect(req.headers['X-Cortado-Dev-Token'], 'dev-bypass');
      expect(req.url.queryParameters.containsKey('dev_token'), isFalse);
      // JSON body present for create
      expect(req.headers['Content-Type'], 'application/json');
      final body =
          jsonDecode(utf8.decode(req.bodyBytes)) as Map<String, dynamic>;
      expect(body['image'], 'ghcr.io/acme/image:tag');
    });

    test(
        'web requests keep query params and use header auth instead of dev_token',
        () async {
      final requests = <RecordedRequest>[];
      final client = RecordingClient((request, body) async {
        requests.add(RecordedRequest(DateTime.now(), request, body));
        return _stringResponse(204, '');
      });

      final manager = WorkspaceManager(
        baseUrl: 'https://api.example.dev/base?foo=bar',
        httpClient: client,
        useBrowserAuth: true,
      );

      await manager.start('ws-123');
      await manager.stop('ws-456');

      expect(requests, hasLength(2));
      final first = requests[0].request;
      final second = requests[1].request;

      expect(first.method, 'POST');
      expect(second.method, 'POST');

      expect(
        first.url,
        Uri.parse(
            'https://api.example.dev/base/v1/workspaces/ws-123/start?foo=bar'),
      );
      expect(
        second.url,
        Uri.parse(
            'https://api.example.dev/base/v1/workspaces/ws-456/stop?foo=bar'),
      );

      expect(first.headers['X-Cortado-Dev-Token'], 'dev-bypass');
      expect(second.headers['X-Cortado-Dev-Token'], 'dev-bypass');
    });

    test('uses bearer auth when a session is present', () async {
      final requests = <RecordedRequest>[];
      final client = RecordingClient((request, body) async {
        requests.add(RecordedRequest(DateTime.now(), request, body));
        return _stringResponse(204, '');
      });

      final accessToken = _jwtExpiringAt(DateTime.utc(2026, 5, 23, 15));
      final authSession = CortadoAuthSession(baseUrl: 'http://localhost:8080')
        ..setTokens(
          accessToken: accessToken,
          refreshToken: 'refresh-token',
        );
      final manager = WorkspaceManager(
        baseUrl: 'http://localhost:8080',
        httpClient: client,
        authSession: authSession,
        useBrowserAuth: true,
      );

      await manager.start('ws-123');

      final request = requests.single.request;
      expect(request.url,
          Uri.parse('http://localhost:8080/v1/workspaces/ws-123/start'));
      expect(request.headers['Authorization'], 'Bearer $accessToken');
      expect(request.headers.containsKey('X-Cortado-Dev-Token'), isFalse);
    });

    test('listDirectory requests the file endpoint and parses entries', () async {
      final requests = <RecordedRequest>[];
      final client = RecordingClient((request, body) async {
        requests.add(RecordedRequest(DateTime.now(), request, body));
        if (request.method == 'GET' &&
            request.url.path.endsWith('/v1/workspaces/ws-123/files')) {
          final payload = jsonEncode({
            'entries': [
              {
                'name': 'lib',
                'size': 0,
                'isDir': true,
                'modTime': DateTime.utc(2026, 5, 23, 21).toIso8601String(),
                'permissions': 493,
              },
            ],
          });
          return _stringResponse(200, payload);
        }
        return _stringResponse(404, 'not found');
      });

      final manager = WorkspaceManager(
        baseUrl: 'http://localhost:8080/api?foo=bar',
        httpClient: client,
        useBrowserAuth: false,
      );

      final entries = await manager.listDirectory('ws-123', path: '/lib');
      expect(entries, hasLength(1));
      expect(entries.single.name, 'lib');
      expect(entries.single.isDir, isTrue);

      final request = requests.single.request as http.Request;
      expect(
        request.url,
        Uri.parse(
          'http://localhost:8080/api/v1/workspaces/ws-123/files?foo=bar&path=lib',
        ),
      );
      expect(request.headers['X-Cortado-Dev-Token'], 'dev-bypass');
    });
  });

  group('WorkspaceManager — watchStatus cadence and completion', () {
    test('transitions from transitional to running cadence', () async {
      final calls = <DateTime>[];
      final seq = <WorkspaceLifecycleState>[
        WorkspaceLifecycleState.creating,
        WorkspaceLifecycleState.running,
        WorkspaceLifecycleState.running,
      ];
      var idx = 0;

      final client = RecordingClient((request, body) async {
        if (request.method == 'GET' &&
            request.url.path.endsWith('/v1/workspaces/ws-1')) {
          calls.add(DateTime.now());
          final status = idx < seq.length ? seq[idx++] : seq.last;
          return _workspaceResponse('ws-1', status);
        }
        return _stringResponse(404, '');
      });

      final manager = WorkspaceManager(
        baseUrl: 'http://localhost:8080',
        httpClient: client,
        useBrowserAuth: false,
        // Short intervals to keep the test quick and observable.
        transitionalPollInterval: const Duration(milliseconds: 10),
        runningPollInterval: const Duration(milliseconds: 80),
      );

      final statuses = <WorkspaceStatus>[];
      final sub = manager.watchStatus('ws-1').listen(statuses.add);

      // Wait enough time for three polls (10ms + 80ms) with slack.
      await Future<void>.delayed(const Duration(milliseconds: 140));
      await sub.cancel();

      // Received creating -> running.
      expect(
          statuses.map((s) => s.status).toList(),
          containsAllInOrder(<WorkspaceLifecycleState>[
            WorkspaceLifecycleState.creating,
            WorkspaceLifecycleState.running,
          ]));

      // Ensure we observed at least 3 GET calls.
      expect(calls.length >= 3, isTrue);
      // Cadence: gap[0->1] ~ transitional (>= 5ms), gap[1->2] ~ running (>= 60ms).
      final gap01 = calls[1].difference(calls[0]);
      final gap12 = calls[2].difference(calls[1]);
      expect(gap01.inMilliseconds >= 5, isTrue,
          reason: 'first gap >= transitional');
      expect(gap12.inMilliseconds >= 60, isTrue,
          reason: 'second gap >= running');
    });

    test('completes when a terminal status is reached', () async {
      final seq = <WorkspaceLifecycleState>[
        WorkspaceLifecycleState.starting,
        WorkspaceLifecycleState.stopped,
      ];
      var idx = 0;

      final client = RecordingClient((request, body) async {
        if (request.method == 'GET' &&
            request.url.path.endsWith('/v1/workspaces/ws-2')) {
          final status = idx < seq.length ? seq[idx++] : seq.last;
          return _workspaceResponse('ws-2', status);
        }
        return _stringResponse(404, '');
      });

      final manager = WorkspaceManager(
        baseUrl: 'http://localhost:8080',
        httpClient: client,
        useBrowserAuth: false,
        transitionalPollInterval: const Duration(milliseconds: 5),
        runningPollInterval: const Duration(milliseconds: 5),
      );

      final done = Completer<void>();
      final statuses = <WorkspaceStatus>[];
      manager.watchStatus('ws-2').listen(
            statuses.add,
            onDone: () => done.complete(),
          );

      await done.future.timeout(const Duration(seconds: 1));
      expect(
          statuses.map((s) => s.status).toList(),
          containsAllInOrder(<WorkspaceLifecycleState>[
            WorkspaceLifecycleState.starting,
            WorkspaceLifecycleState.stopped,
          ]));
    });

    test('cancelling the subscription stops further polling', () async {
      final calls = <int>[];
      final client = RecordingClient((request, body) async {
        if (request.method == 'GET' &&
            request.url.path.endsWith('/v1/workspaces/ws-3')) {
          calls.add(1);
          return _workspaceResponse('ws-3', WorkspaceLifecycleState.creating);
        }
        return _stringResponse(404, '');
      });

      final manager = WorkspaceManager(
        baseUrl: 'http://localhost:8080',
        httpClient: client,
        useBrowserAuth: false,
        transitionalPollInterval: const Duration(milliseconds: 80),
        runningPollInterval: const Duration(milliseconds: 80),
      );

      late final StreamSubscription<WorkspaceStatus> sub;
      sub = manager.watchStatus('ws-3').listen((_) async {
        // Cancel immediately on first event; next poll timer should be cancelled.
        await sub.cancel();
      });

      // Wait longer than transitional interval; no extra calls should have occurred.
      await Future<void>.delayed(const Duration(milliseconds: 160));
      expect(calls, hasLength(1));
    });
  });
}

// Helpers

http.StreamedResponse _stringResponse(int status, String body,
    {Map<String, String>? headers}) {
  final bytes = utf8.encode(body);
  final stream =
      Stream<List<int>>.fromIterable(<List<int>>[Uint8List.fromList(bytes)]);
  return http.StreamedResponse(stream, status,
      headers: headers ?? const <String, String>{});
}

http.StreamedResponse _workspaceResponse(
    String id, WorkspaceLifecycleState state) {
  final payload = jsonEncode({
    'workspace': _sampleWorkspaceJson(id: id, state: state),
  });
  return _stringResponse(200, payload,
      headers: const <String, String>{'Content-Type': 'application/json'});
}

Map<String, Object?> _sampleWorkspaceJson(
        {required String id,
        WorkspaceLifecycleState state = WorkspaceLifecycleState.running}) =>
    <String, Object?>{
      'id': id,
      'tenantId': 't1',
      'userId': 'u1',
      'image': 'img',
      'resources': <String, Object?>{'cpu': 1.0, 'memoryGb': 2.0},
      'status': _encodeState(state),
      'createdAt': DateTime.now().toIso8601String(),
      'updatedAt': DateTime.now().toIso8601String(),
      'lastActiveAt': null,
    };

String _encodeState(WorkspaceLifecycleState s) => switch (s) {
      WorkspaceLifecycleState.creating => 'CREATING',
      WorkspaceLifecycleState.starting => 'STARTING',
      WorkspaceLifecycleState.running => 'RUNNING',
      WorkspaceLifecycleState.stopping => 'STOPPING',
      WorkspaceLifecycleState.stopped => 'STOPPED',
      WorkspaceLifecycleState.deleted => 'DELETED',
    };

class RecordedRequest {
  RecordedRequest(this.timestamp, this.request, this.bodyBytes);
  final DateTime timestamp;
  final http.BaseRequest request;
  final List<int> bodyBytes;
}

class RecordingClient extends http.BaseClient {
  RecordingClient(this._handler);

  final Future<http.StreamedResponse> Function(
      http.BaseRequest request, List<int> bodyBytes) _handler;

  @override
  Future<http.StreamedResponse> send(http.BaseRequest request) async {
    final bodyBytes = await http.ByteStream(request.finalize()).toBytes();
    return _handler(request, bodyBytes);
  }
}

String _jwtExpiringAt(DateTime timestamp) {
  final header = base64Url.encode(utf8.encode(jsonEncode(<String, String>{
    'alg': 'RS256',
    'typ': 'JWT',
  })));
  final payload = base64Url.encode(utf8.encode(jsonEncode(<String, Object>{
    'exp': timestamp.millisecondsSinceEpoch ~/ 1000,
  })));
  return '$header.$payload.signature';
}
