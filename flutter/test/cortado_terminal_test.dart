import 'package:cortado/cortado.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('renders a non-web fallback when HtmlElementView is unavailable',
      (WidgetTester tester) async {
    await tester.pumpWidget(
      Directionality(
        textDirection: TextDirection.ltr,
        child: CortadoTerminal(
          client: CortadoClient(baseUrl: 'http://localhost:8080'),
        ),
      ),
    );

    expect(
        find.text(
            'CortadoTerminal is currently supported on Flutter Web only.'),
        findsOneWidget);
  });
}
