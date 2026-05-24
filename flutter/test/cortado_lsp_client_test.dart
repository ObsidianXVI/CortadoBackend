import 'dart:async';
import 'dart:convert';
import 'dart:typed_data';

import 'package:cortado/cortado.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('CortadoLSPClient', () {
    test('opens the mux channel, initializes, and flushes queued work in order',
        () async {
      final client = _FakeMuxClient();
      final lspClient = CortadoLSPClient(client: client);

      final didOpenFuture = lspClient.didOpenTextDocument(
        path: '/lib/main.dart',
        languageId: 'dart',
        text: 'void main() {}',
      );
      final completionFuture = lspClient.sendRequest(
        'textDocument/completion',
        params: <String, Object?>{
          'textDocument': <String, Object?>{
            'uri': 'file:///workspace/lib/main.dart',
          },
          'position': const <String, Object?>{
            'line': 0,
            'character': 4,
          },
        },
      );

      expect(client.sentFrames, hasLength(2));
      expect(client.sentFrames.first.messageType, muxMessageTypeOpen);
      expect(utf8.decode(client.sentFrames.first.payload), 'dart');

      final initializePayload = _decodeJson(client.sentFrames[1].payload);
      expect(initializePayload['method'], 'initialize');
      expect(client.sentFrames, hasLength(2));

      client.emitJson(
        channelId: muxLspChannelStartId,
        payload: <String, Object?>{
          'jsonrpc': '2.0',
          'id': initializePayload['id'],
          'result': <String, Object?>{
            'capabilities': <String, Object?>{},
          },
        },
      );
      await Future<void>.delayed(Duration.zero);

      expect(client.sentFrames, hasLength(5));
      expect(
          _decodeJson(client.sentFrames[2].payload)['method'], 'initialized');
      expect(
        _decodeJson(client.sentFrames[3].payload)['method'],
        'textDocument/didOpen',
      );
      expect(
        _decodeJson(client.sentFrames[4].payload)['method'],
        'textDocument/completion',
      );

      final completionRequest = _decodeJson(client.sentFrames[4].payload);
      client.emitJson(
        channelId: muxLspChannelStartId,
        payload: <String, Object?>{
          'jsonrpc': '2.0',
          'id': completionRequest['id'],
          'result': <String, Object?>{
            'items': <Object?>[],
          },
        },
      );

      await didOpenFuture;
      expect(
        await completionFuture,
        <String, Object?>{
          'items': <Object?>[],
        },
      );
      expect(lspClient.isInitialized, isTrue);

      await lspClient.dispose();
    });

    test('publishes diagnostics updates to listeners', () async {
      final client = _FakeMuxClient();
      final lspClient = CortadoLSPClient(client: client);
      final diagnosticsUpdates = <CortadoLSPDiagnosticsByUri>[];
      final diagnosticsSub =
          lspClient.diagnosticsStream.listen(diagnosticsUpdates.add);

      final initializeFuture = lspClient.ensureInitialized();
      final initializePayload = _decodeJson(client.sentFrames[1].payload);
      client.emitJson(
        channelId: muxLspChannelStartId,
        payload: <String, Object?>{
          'jsonrpc': '2.0',
          'id': initializePayload['id'],
          'result': <String, Object?>{
            'capabilities': <String, Object?>{},
          },
        },
      );
      await initializeFuture;

      client.emitJson(
        channelId: muxLspChannelStartId,
        payload: <String, Object?>{
          'jsonrpc': '2.0',
          'method': 'textDocument/publishDiagnostics',
          'params': <String, Object?>{
            'uri': 'file:///workspace/lib/main.dart',
            'diagnostics': <Object?>[
              <String, Object?>{
                'message': 'Missing semicolon.',
                'severity': 1,
              },
            ],
          },
        },
      );
      await Future<void>.delayed(Duration.zero);

      expect(diagnosticsUpdates, hasLength(1));
      expect(
        diagnosticsUpdates.single['file:///workspace/lib/main.dart'],
        <Map<String, Object?>>[
          <String, Object?>{
            'message': 'Missing semicolon.',
            'severity': 1,
          },
        ],
      );

      await diagnosticsSub.cancel();
      await lspClient.dispose();
    });
  });
}

Map<String, Object?> _decodeJson(Uint8List payload) =>
    Map<String, Object?>.from(
      jsonDecode(utf8.decode(payload)) as Map<Object?, Object?>,
    );

class _FakeMuxClient extends CortadoClient {
  _FakeMuxClient()
      : super(
          baseUrl: 'http://localhost:8080',
          useBrowserWebSocket: false,
        );

  final List<MuxFrame> sentFrames = <MuxFrame>[];
  final Map<int, StreamController<MuxFrame>> _controllers =
      <int, StreamController<MuxFrame>>{};

  @override
  Stream<MuxFrame> framesForChannel(int channelId) =>
      _controller(channelId).stream;

  @override
  void sendFrame(int channelId, int messageType, Uint8List payload) {
    sentFrames.add(MuxFrame(channelId, messageType, payload));
  }

  void emitJson({
    required int channelId,
    required Map<String, Object?> payload,
  }) {
    _controller(channelId).add(
      MuxFrame(
        channelId,
        muxMessageTypeData,
        Uint8List.fromList(utf8.encode(jsonEncode(payload))),
      ),
    );
  }

  StreamController<MuxFrame> _controller(int channelId) {
    return _controllers.putIfAbsent(
      channelId,
      () => StreamController<MuxFrame>.broadcast(),
    );
  }
}
