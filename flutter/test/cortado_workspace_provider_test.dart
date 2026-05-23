import 'dart:async';

import 'package:cortado/cortado.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('CortadoWorkspaceProvider.of(context).workspaceId exposes id',
      (WidgetTester tester) async {
    final manager = WorkspaceManager(
      baseUrl: 'http://localhost:8080',
      useBrowserAuth: false,
    );

    String? seenWorkspaceId;

    await tester.pumpWidget(
      Directionality(
        textDirection: TextDirection.ltr,
        child: CortadoWorkspaceProvider(
          workspaceId: 'ws-abc',
          manager: manager,
          child: Builder(
            builder: (BuildContext context) {
              seenWorkspaceId =
                  CortadoWorkspaceProvider.of(context).workspaceId;
              return const SizedBox.shrink();
            },
          ),
        ),
      ),
    );

    expect(seenWorkspaceId, 'ws-abc');
    await manager.dispose();
  });

  testWidgets('disposal cancels the status subscription',
      (WidgetTester tester) async {
    final fake = TestWorkspaceManager();

    await tester.pumpWidget(
      Directionality(
        textDirection: TextDirection.ltr,
        child: CortadoWorkspaceProvider(
          workspaceId: 'ws-xyz',
          manager: fake,
          child: const _StatusReader(),
        ),
      ),
    );

    // Ensure provider has time to subscribe.
    await tester.pump(const Duration(milliseconds: 10));
    expect(fake.watchCalledWith, 'ws-xyz');
    expect(fake.cancelled, isFalse);

    // Dispose the subtree, which should dispose the provider and cancel.
    await tester.pumpWidget(const SizedBox.shrink());
    await tester.pump(const Duration(milliseconds: 10));

    expect(fake.cancelled, isTrue);
    await fake.dispose();
  });
}

class _StatusReader extends ConsumerWidget {
  const _StatusReader();

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    // Reading this provider triggers the underlying status subscription.
    ref.watch(cortadoCurrentWorkspaceStatusProvider);
    return const SizedBox.shrink();
  }
}

class TestWorkspaceManager extends WorkspaceManager {
  TestWorkspaceManager()
      : _controller = StreamController<WorkspaceStatus>(),
        super(baseUrl: 'http://localhost:8080', useBrowserAuth: false);

  final StreamController<WorkspaceStatus> _controller;
  String? watchCalledWith;
  bool cancelled = false;

  @override
  Stream<WorkspaceStatus> watchStatus(String id) {
    watchCalledWith = id;
    _controller.onCancel = () {
      cancelled = true;
    };
    return _controller.stream;
  }
}
