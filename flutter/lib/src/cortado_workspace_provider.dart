import 'dart:async';

import 'package:flutter/widgets.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'workspace_manager.dart';
import 'workspace_models.dart';

final cortadoWorkspaceManagerProvider = Provider<WorkspaceManager>((ref) {
  throw StateError('No WorkspaceManager is available in the current scope.');
});

final cortadoWorkspaceIdProvider = Provider<String>((ref) {
  throw StateError('No workspace id is available in the current scope.');
});

final cortadoWorkspaceStatusProvider =
    StreamProvider.autoDispose.family<WorkspaceStatus, String>((ref, id) {
  final manager = ref.watch(cortadoWorkspaceManagerProvider);
  return _workspaceStatusStream(ref, manager, id);
});

final cortadoCurrentWorkspaceStatusProvider =
    StreamProvider.autoDispose<WorkspaceStatus>((ref) {
  final manager = ref.watch(cortadoWorkspaceManagerProvider);
  final workspaceId = ref.watch(cortadoWorkspaceIdProvider);
  return _workspaceStatusStream(ref, manager, workspaceId);
});

class CortadoWorkspaceProvider extends StatefulWidget {
  const CortadoWorkspaceProvider({
    super.key,
    required this.workspaceId,
    required this.manager,
    required this.child,
  });

  final String workspaceId;
  final WorkspaceManager manager;
  final Widget child;

  static CortadoWorkspaceScope of(BuildContext context) {
    final inherited =
        context.dependOnInheritedWidgetOfExactType<_InheritedWorkspaceScope>();
    if (inherited?.notifier case final CortadoWorkspaceScope scope) {
      return scope;
    }

    throw FlutterError(
      'CortadoWorkspaceProvider.of() called with a context that does not '
      'contain a CortadoWorkspaceProvider.',
    );
  }

  @override
  State<CortadoWorkspaceProvider> createState() =>
      _CortadoWorkspaceProviderState();
}

class _CortadoWorkspaceProviderState extends State<CortadoWorkspaceProvider> {
  ProviderContainer? _container;
  ProviderSubscription<String>? _workspaceIdSubscription;
  late final CortadoWorkspaceScope _scope =
      CortadoWorkspaceScope(workspaceId: widget.workspaceId);

  @override
  void initState() {
    super.initState();
    _container = _createContainer();
  }

  @override
  void didUpdateWidget(CortadoWorkspaceProvider oldWidget) {
    super.didUpdateWidget(oldWidget);

    if (oldWidget.workspaceId == widget.workspaceId &&
        oldWidget.manager == widget.manager) {
      return;
    }

    _disposeContainer();
    _container = _createContainer();
  }

  @override
  void dispose() {
    _disposeContainer();
    _scope.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final container = _container;
    if (container == null) {
      throw StateError('Workspace provider container is not initialized.');
    }

    return UncontrolledProviderScope(
      container: container,
      child: _InheritedWorkspaceScope(
        notifier: _scope,
        child: widget.child,
      ),
    );
  }

  ProviderContainer _createContainer() {
    final container = ProviderContainer(
      overrides: <Override>[
        cortadoWorkspaceManagerProvider.overrideWith((ref) => widget.manager),
        cortadoWorkspaceIdProvider.overrideWith((ref) => widget.workspaceId),
      ],
    );

    _workspaceIdSubscription = container.listen<String>(
      cortadoWorkspaceIdProvider,
      (_, next) => _scope.workspaceId = next,
      fireImmediately: true,
    );

    return container;
  }

  void _disposeContainer() {
    _workspaceIdSubscription?.close();
    _workspaceIdSubscription = null;
    _container?.dispose();
    _container = null;
  }
}

class CortadoWorkspaceScope extends ChangeNotifier {
  CortadoWorkspaceScope({required String workspaceId})
      : _workspaceId = workspaceId;

  String _workspaceId;

  String get workspaceId => _workspaceId;

  set workspaceId(String value) {
    if (_workspaceId == value) {
      return;
    }

    _workspaceId = value;
    notifyListeners();
  }
}

class _InheritedWorkspaceScope
    extends InheritedNotifier<CortadoWorkspaceScope> {
  const _InheritedWorkspaceScope({
    required super.notifier,
    required super.child,
  });
}

Stream<WorkspaceStatus> _workspaceStatusStream(
  Ref ref,
  WorkspaceManager manager,
  String workspaceId,
) {
  final controller = StreamController<WorkspaceStatus>();
  final subscription = manager.watchStatus(workspaceId).listen(
    controller.add,
    onError: controller.addError,
    onDone: () {
      if (!controller.isClosed) {
        unawaited(controller.close());
      }
    },
  );

  ref.onDispose(() {
    unawaited(subscription.cancel());
    if (!controller.isClosed) {
      unawaited(controller.close());
    }
  });

  return controller.stream;
}
