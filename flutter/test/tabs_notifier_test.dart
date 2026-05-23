import 'package:cortado/cortado.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('caps open tabs at fifteen and activates the newest tab', () {
    final notifier = TabsNotifier(maxTabs: 15);

    for (var index = 0; index < 16; index++) {
      notifier.open('/lib/file_$index.dart');
      notifier.setLoaded(
        '/lib/file_$index.dart',
        content: 'file $index',
        hash: hashEditorContent('file $index'),
      );
    }

    expect(notifier.state.tabs, hasLength(15));
    expect(
      notifier.state.tabs.first.path,
      '/lib/file_1.dart',
    );
    expect(notifier.state.activePath, '/lib/file_15.dart');
  });

  test('tracks dirty state from current and saved hashes', () {
    final notifier = TabsNotifier();

    notifier.open('/lib/main.dart');
    notifier.setLoaded(
      '/lib/main.dart',
      content: 'hello',
      hash: hashEditorContent('hello'),
    );

    expect(notifier.state.activeTab?.isDirty, isFalse);

    notifier.markHash('/lib/main.dart', hashEditorContent('hello world'));
    expect(notifier.state.activeTab?.isDirty, isTrue);

    notifier.markSaved(
      '/lib/main.dart',
      content: 'hello world',
      hash: hashEditorContent('hello world'),
    );
    expect(notifier.state.activeTab?.isDirty, isFalse);
  });
}
