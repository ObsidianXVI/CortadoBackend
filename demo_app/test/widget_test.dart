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
        'CORTADO_WORKSPACE_IMAGE':
            'us-central1-docker.pkg.dev/cortado-ide/cortado-dev/cortado-workspace:781d613',
        'CORTADO_FIREBASE_API_KEY': 'firebase-api-key',
        'CORTADO_FIREBASE_PROJECT_ID': 'demo-firebase-project',
        'CORTADO_FIREBASE_APP_ID': '1:123:web:abc',
        'CORTADO_FIREBASE_MESSAGING_SENDER_ID': '123',
        'CORTADO_FIREBASE_EMAIL': 'demo@example.com',
        'CORTADO_FIREBASE_DEV_TENANT_ID': 'demo-tenant',
      },
    );

    expect(config.baseUrl, 'https://control-plane.example.run.app');
    expect(config.apiKey, 'local-demo-key');
    expect(config.userId, 'demo-user');
    expect(config.workspaceId, 'ws-123');
    expect(config.shell, '/bin/zsh');
    expect(
      config.image,
      'us-central1-docker.pkg.dev/cortado-ide/cortado-dev/cortado-workspace:781d613',
    );
    expect(config.filePath, 'lib/main.dart');
    expect(config.cpu, 1.5);
    expect(config.memoryGb, 3);
    expect(config.firebaseApiKey, 'firebase-api-key');
    expect(config.firebaseProjectId, 'demo-firebase-project');
    expect(config.firebaseAppId, '1:123:web:abc');
    expect(config.firebaseMessagingSenderId, '123');
    expect(config.firebaseEmail, 'demo@example.com');
    expect(config.firebaseDevTenantId, 'demo-tenant');
    expect(config.hasFirebaseBootstrapConfig, isTrue);
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
            image:
                'us-central1-docker.pkg.dev/cortado-ide/cortado-dev/cortado-workspace:781d613',
            filePath: 'lib/main.dart',
            cpu: 1,
            memoryGb: 2,
            firebaseApiKey: '',
            firebaseAuthDomain: '',
            firebaseProjectId: '',
            firebaseAppId: '',
            firebaseMessagingSenderId: '',
            firebaseStorageBucket: '',
            firebaseMeasurementId: '',
            firebaseEmail: '',
            firebasePassword: '',
            firebaseDevTenantId: '',
          ),
        ),
      ),
    );

    expect(find.text('Cortado Package Showcase'), findsOneWidget);
    expect(find.text('Identity Bootstrap'), findsOneWidget);
    expect(find.text('Exchange Session'), findsOneWidget);
    expect(find.text('Platform Backend Flow'), findsOneWidget);
  });
}
