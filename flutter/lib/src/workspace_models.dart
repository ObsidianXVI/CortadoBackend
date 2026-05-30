// ignore_for_file: invalid_annotation_target

import 'package:freezed_annotation/freezed_annotation.dart';

part 'workspace_models.freezed.dart';
part 'workspace_models.g.dart';

enum WorkspaceLifecycleState {
  @JsonValue('CREATING')
  creating,
  @JsonValue('STARTING')
  starting,
  @JsonValue('RUNNING')
  running,
  @JsonValue('STOPPING')
  stopping,
  @JsonValue('STOPPED')
  stopped,
  @JsonValue('DELETED')
  deleted,
}

@freezed
sealed class WorkspaceResources with _$WorkspaceResources {
  const factory WorkspaceResources({
    required double cpu,
    @JsonKey(name: 'memoryGb') required double memoryGb,
    @JsonKey(name: 'storageGb') required double storageGb,
  }) = _WorkspaceResources;

  factory WorkspaceResources.fromJson(Map<String, dynamic> json) =>
      _$WorkspaceResourcesFromJson(json);
}

@freezed
sealed class Workspace with _$Workspace {
  const Workspace._();

  const factory Workspace({
    required String id,
    required String tenantId,
    required String userId,
    required String image,
    required WorkspaceResources resources,
    required WorkspaceLifecycleState status,
    required DateTime createdAt,
    required DateTime updatedAt,
    DateTime? lastActiveAt,
  }) = _Workspace;

  factory Workspace.fromJson(Map<String, dynamic> json) =>
      _$WorkspaceFromJson(json);

  WorkspaceStatus toStatus() => WorkspaceStatus(
        workspaceId: id,
        status: status,
        updatedAt: updatedAt,
        lastActiveAt: lastActiveAt,
      );
}

@freezed
sealed class WorkspaceStatus with _$WorkspaceStatus {
  const WorkspaceStatus._();

  const factory WorkspaceStatus({
    required String workspaceId,
    required WorkspaceLifecycleState status,
    required DateTime updatedAt,
    DateTime? lastActiveAt,
  }) = _WorkspaceStatus;

  factory WorkspaceStatus.fromJson(Map<String, dynamic> json) =>
      _$WorkspaceStatusFromJson(json);

  factory WorkspaceStatus.fromWorkspace(Workspace workspace) =>
      workspace.toStatus();

  bool get isTerminal => switch (status) {
        WorkspaceLifecycleState.stopped ||
        WorkspaceLifecycleState.deleted =>
          true,
        _ => false,
      };

  Duration? nextPollDelay({
    required Duration transitionalInterval,
    required Duration runningInterval,
  }) =>
      switch (status) {
        WorkspaceLifecycleState.creating ||
        WorkspaceLifecycleState.starting ||
        WorkspaceLifecycleState.stopping =>
          transitionalInterval,
        WorkspaceLifecycleState.running => runningInterval,
        WorkspaceLifecycleState.stopped ||
        WorkspaceLifecycleState.deleted =>
          null,
      };
}
