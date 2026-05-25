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
      final timers = <FakeTimer>[];
      final client = RecordingClient((request, body) async {
        requests.add(RecordedRequest(DateTime.now(), request, body));
        return _stringResponse(204, '');
      });

      final accessToken = _jwtExpiringAt(DateTime.utc(2026, 5, 23, 15));
      final authSession = CortadoAuthSession(
        baseUrl: 'http://localhost:8080',
        now: () => DateTime.utc(2026, 5, 23, 14),
        timerFactory: (duration, callback) {
          final timer = FakeTimer(duration, callback);
          timers.add(timer);
          return timer;
        },
      )..setTokens(
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
      expect(timers, hasLength(1));

      await authSession.dispose();
    });

    test('listDirectory requests the file endpoint and parses entries',
        () async {
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

    test('list/get/delete workspace helpers use collection and item endpoints',
        () async {
      final requests = <RecordedRequest>[];
      final client = RecordingClient((request, body) async {
        requests.add(RecordedRequest(DateTime.now(), request, body));

        if (request.method == 'GET' &&
            request.url.path.endsWith('/v1/workspaces')) {
          return _stringResponse(
            200,
            jsonEncode({
              'workspaces': <Object?>[
                _sampleWorkspaceJson(
                  id: 'ws-older',
                  state: WorkspaceLifecycleState.stopped,
                ),
                _sampleWorkspaceJson(id: 'ws-123'),
              ],
            }),
            headers: const <String, String>{'Content-Type': 'application/json'},
          );
        }
        if (request.method == 'GET' &&
            request.url.path.endsWith('/v1/workspaces/ws-123')) {
          return _workspaceResponse('ws-123', WorkspaceLifecycleState.running);
        }
        if (request.method == 'DELETE' &&
            request.url.path.endsWith('/v1/workspaces/ws-123')) {
          return _workspaceResponse('ws-123', WorkspaceLifecycleState.deleted);
        }

        return _stringResponse(404, 'not found');
      });

      final manager = WorkspaceManager(
        baseUrl: 'http://localhost:8080/api?foo=bar',
        httpClient: client,
        useBrowserAuth: false,
      );

      final workspaces = await manager.listWorkspaces();
      final workspace = await manager.getWorkspace('ws-123');
      final deleted = await manager.deleteWorkspace('ws-123');

      expect(workspaces, hasLength(2));
      expect(workspaces.first.id, 'ws-older');
      expect(workspaces.last.id, 'ws-123');
      expect(workspace.id, 'ws-123');
      expect(workspace.status, WorkspaceLifecycleState.running);
      expect(deleted.status, WorkspaceLifecycleState.deleted);

      expect(requests, hasLength(3));
      expect(
        requests[0].request.url,
        Uri.parse('http://localhost:8080/api/v1/workspaces?foo=bar'),
      );
      expect(
        requests[1].request.url,
        Uri.parse('http://localhost:8080/api/v1/workspaces/ws-123?foo=bar'),
      );
      expect(
        requests[2].request.url,
        Uri.parse('http://localhost:8080/api/v1/workspaces/ws-123?foo=bar'),
      );
      expect(
        requests.every(
          (request) =>
              request.request.headers['X-Cortado-Dev-Token'] == 'dev-bypass',
        ),
        isTrue,
      );
    });

    test('file content and mutation helpers use the expected endpoints',
        () async {
      final requests = <RecordedRequest>[];
      final client = RecordingClient((request, body) async {
        requests.add(RecordedRequest(DateTime.now(), request, body));

        if (request.method == 'GET' &&
            request.url.path.endsWith('/v1/workspaces/ws-123/files/content')) {
          return _stringResponse(200, 'hello file');
        }
        if (request.method == 'PUT' &&
            request.url.path.endsWith('/v1/workspaces/ws-123/files/content')) {
          return _stringResponse(
            200,
            jsonEncode(<String, Object>{
              'bytesWritten': body.length,
              'checksum': <int>[1, 2, 3],
            }),
            headers: const <String, String>{'Content-Type': 'application/json'},
          );
        }
        if (request.method == 'POST' &&
            request.url.path
                .endsWith('/v1/workspaces/ws-123/files/directory')) {
          return _stringResponse(201, '');
        }
        if (request.method == 'POST' &&
            request.url.path.endsWith('/v1/workspaces/ws-123/files/rename')) {
          return _stringResponse(204, '');
        }
        if (request.method == 'DELETE' &&
            request.url.path.endsWith('/v1/workspaces/ws-123/files')) {
          return _stringResponse(204, '');
        }
        return _stringResponse(404, 'not found');
      });

      final manager = WorkspaceManager(
        baseUrl: 'http://localhost:8080/api?foo=bar',
        httpClient: client,
        useBrowserAuth: false,
      );

      final readBytes =
          await manager.readFile('ws-123', path: '/lib/main.dart');
      expect(utf8.decode(readBytes), 'hello file');

      final writeResult = await manager.writeFile(
        'ws-123',
        path: '/lib/main.dart',
        content: utf8.encode('updated'),
      );
      expect(writeResult.bytesWritten, 7);
      expect(writeResult.checksum, orderedEquals(const <int>[1, 2, 3]));

      await manager.writeFile(
        'ws-123',
        path: '/lib/strict.dart',
        content: utf8.encode('strict'),
        createMissingDirs: false,
      );

      await manager.makeDir('ws-123', path: '/lib/newdir');
      await manager.renamePath(
        'ws-123',
        oldPath: '/lib/main.dart',
        newPath: '/lib/app.dart',
      );
      await manager.deletePath('ws-123', path: '/lib/app.dart');

      expect(requests, hasLength(6));
      expect(
        requests[0].request.url,
        Uri.parse(
          'http://localhost:8080/api/v1/workspaces/ws-123/files/content?foo=bar&path=lib%2Fmain.dart',
        ),
      );
      expect(
        requests[1].request.url,
        Uri.parse(
          'http://localhost:8080/api/v1/workspaces/ws-123/files/content?foo=bar&path=lib%2Fmain.dart',
        ),
      );
      expect(
        requests[2].request.url,
        Uri.parse(
          'http://localhost:8080/api/v1/workspaces/ws-123/files/content?foo=bar&path=lib%2Fstrict.dart&createMissingDirs=false',
        ),
      );
      expect(
        requests[3].request.url,
        Uri.parse(
          'http://localhost:8080/api/v1/workspaces/ws-123/files/directory?foo=bar&path=lib%2Fnewdir',
        ),
      );
      expect(
        requests[4].request.url,
        Uri.parse(
          'http://localhost:8080/api/v1/workspaces/ws-123/files/rename?foo=bar&path=lib%2Fmain.dart&newPath=lib%2Fapp.dart',
        ),
      );
      expect(
        requests[5].request.url,
        Uri.parse(
          'http://localhost:8080/api/v1/workspaces/ws-123/files?foo=bar&path=lib%2Fapp.dart',
        ),
      );
      expect(
        requests[1].request.headers['Content-Type'],
        'application/octet-stream',
      );
      expect(utf8.decode(requests[1].bodyBytes), 'updated');
      expect(
        requests.every(
          (request) =>
              request.request.headers['X-Cortado-Dev-Token'] == 'dev-bypass',
        ),
        isTrue,
      );
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

class FakeTimer implements Timer {
  FakeTimer(this.duration, this._callback);

  final Duration duration;
  final void Function() _callback;
  bool _isActive = true;

  void fire() {
    if (!_isActive) {
      return;
    }
    _callback();
  }

  @override
  void cancel() {
    _isActive = false;
  }

  @override
  bool get isActive => _isActive;

  @override
  int get tick => 0;
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
