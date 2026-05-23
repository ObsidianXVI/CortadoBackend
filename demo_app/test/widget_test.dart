import 'package:cortado/cortado.dart';
import 'package:demo_app/main.dart';
import 'package:demo_app/src/terminal_smoke_config.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('parses terminal smoke config from query parameters', () {
    final config = TerminalSmokeConfig.fromUri(
      Uri.parse(
        'https://example.test/'
        '?baseUrl=https%3A%2F%2Fcontrol-plane.example.run.app'
        '&workspaceId=ws-123'
        '&shell=%2Fbin%2Fzsh',
      ),
    );

    expect(config.baseUrl, 'https://control-plane.example.run.app');
    expect(config.workspaceId, 'ws-123');
    expect(config.shell, '/bin/zsh');
  });

  testWidgets(
      'renders the smoke harness and connects through the injected client',
      (WidgetTester tester) async {
    late _FakeCortadoClient client;

    await tester.pumpWidget(
      TerminalSmokeApp(
        initialConfig: const TerminalSmokeConfig(
          baseUrl: 'http://localhost:8080',
          workspaceId: 'ws-123',
          shell: '/bin/bash',
        ),
        clientFactory: (String baseUrl) {
          client = _FakeCortadoClient(baseUrl);
          return client;
        },
      ),
    );

    expect(find.text('Cortado Terminal Smoke Test'), findsOneWidget);
    expect(find.text('Connect'), findsOneWidget);

    await tester.tap(find.text('Connect'));
    await tester.pumpAndSettle();

    expect(client.connectedWorkspaceId, 'ws-123');
    expect(find.text('Connected'), findsOneWidget);

    await tester.drag(find.byType(ListView), const Offset(0, -600));
    await tester.pumpAndSettle();

    expect(
      find.text('CortadoTerminal is currently supported on Flutter Web only.'),
      findsOneWidget,
    );
  });
}

class _FakeCortadoClient extends CortadoClient {
  _FakeCortadoClient(String baseUrl) : super(baseUrl: baseUrl);

  String? connectedWorkspaceId;

  @override
  Future<void> connect(String workspaceId) async {
    connectedWorkspaceId = workspaceId;
  }

  @override
  Future<void> dispose() async {}
}
