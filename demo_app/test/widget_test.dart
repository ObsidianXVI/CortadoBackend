import 'package:demo_app/src/demo_bootstrap_config.dart';
import 'package:demo_app/src/demo_showcase_app.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('parses showcase bootstrap config from query params and env', () {
    final config = DemoBootstrapConfig.fromSources(
      uri: Uri.parse(
        'https://example.test/'
        '?workspaceId=ws-123'
        '&shell=%2Fbin%2Fzsh'
        '&cpu=1.5'
        '&memoryGb=3',
      ),
      env: const <String, String>{
        'CORTADO_BASE_URL': 'https://control-plane.example.run.app',
        'CORTADO_DEMO_API_KEY': 'local-demo-key',
        'CORTADO_DEMO_USER_ID': 'demo-user',
        'CORTADO_WORKSPACE_IMAGE': 'ubuntu:24.04',
      },
    );

    expect(config.baseUrl, 'https://control-plane.example.run.app');
    expect(config.apiKey, 'local-demo-key');
    expect(config.userId, 'demo-user');
    expect(config.workspaceId, 'ws-123');
    expect(config.shell, '/bin/zsh');
    expect(config.image, 'ubuntu:24.04');
    expect(config.filePath, 'lib/main.dart');
    expect(config.cpu, 1.5);
    expect(config.memoryGb, 3);
  });

  testWidgets('renders showcase shell', (WidgetTester tester) async {
    await tester.pumpWidget(
      const MaterialApp(
        home: DemoShowcaseScreen(
          initialConfig: DemoBootstrapConfig(
            baseUrl: 'http://localhost:8080',
            apiKey: '',
            userId: '',
            workspaceId: '',
            shell: '/bin/bash',
            image: 'ubuntu:24.04',
            filePath: 'lib/main.dart',
            cpu: 1,
            memoryGb: 2,
          ),
        ),
      ),
    );

    expect(find.text('Cortado Package Showcase'), findsOneWidget);
    expect(find.text('Workspace Shell'), findsOneWidget);
    expect(find.text('Authenticate'), findsOneWidget);
  });
}
