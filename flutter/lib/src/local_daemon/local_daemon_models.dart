// ignore_for_file: invalid_annotation_target

import 'package:freezed_annotation/freezed_annotation.dart';

part 'local_daemon_models.freezed.dart';
part 'local_daemon_models.g.dart';

const String cortadoDaemonInstallUrl = 'https://install.cortado.dev/daemon';

enum CortadoLocalDaemonAvailabilityState {
  connected,
  disconnected,
  unavailable,
  unknown,
}

enum CortadoLocalDaemonSyncState {
  @JsonValue('CONFLICTED')
  conflicted,
  @JsonValue('IDLE')
  idle,
  @JsonValue('SYNCING')
  syncing,
}

@freezed
class CortadoLocalDaemonAvailability with _$CortadoLocalDaemonAvailability {
  const CortadoLocalDaemonAvailability._();

  const factory CortadoLocalDaemonAvailability({
    @Default(cortadoDaemonInstallUrl) String installUrl,
    String? message,
    @Default(CortadoLocalDaemonAvailabilityState.unknown)
    CortadoLocalDaemonAvailabilityState state,
  }) = _CortadoLocalDaemonAvailability;

  bool get shouldSuggestInstall =>
      state == CortadoLocalDaemonAvailabilityState.unavailable;
}

@freezed
class CortadoLocalDaemonSyncStatus with _$CortadoLocalDaemonSyncStatus {
  const CortadoLocalDaemonSyncStatus._();

  const factory CortadoLocalDaemonSyncStatus({
    required String localPath,
    String? message,
    required CortadoLocalDaemonSyncState state,
    required String workspaceId,
    @Default('/') String workspacePath,
  }) = _CortadoLocalDaemonSyncStatus;

  factory CortadoLocalDaemonSyncStatus.fromJson(Map<String, dynamic> json) =>
      _$CortadoLocalDaemonSyncStatusFromJson(json);
}

@freezed
class CortadoLocalDaemonConflict with _$CortadoLocalDaemonConflict {
  const CortadoLocalDaemonConflict._();

  const factory CortadoLocalDaemonConflict({
    required int lastSyncedClock,
    required int localClock,
    @JsonKey(name: 'path') required String localPath,
    required String reason,
    required int remoteClock,
    String? workspaceId,
    String? workspacePath,
  }) = _CortadoLocalDaemonConflict;

  factory CortadoLocalDaemonConflict.fromJson(Map<String, dynamic> json) =>
      _$CortadoLocalDaemonConflictFromJson(json);
}
