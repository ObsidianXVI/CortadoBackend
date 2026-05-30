// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'local_daemon_models.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

_CortadoLocalDaemonSyncStatus _$CortadoLocalDaemonSyncStatusFromJson(
        Map<String, dynamic> json) =>
    _CortadoLocalDaemonSyncStatus(
      localPath: json['localPath'] as String,
      message: json['message'] as String?,
      state: $enumDecode(_$CortadoLocalDaemonSyncStateEnumMap, json['state']),
      workspaceId: json['workspaceId'] as String,
      workspacePath: json['workspacePath'] as String? ?? '/',
    );

Map<String, dynamic> _$CortadoLocalDaemonSyncStatusToJson(
        _CortadoLocalDaemonSyncStatus instance) =>
    <String, dynamic>{
      'localPath': instance.localPath,
      'message': instance.message,
      'state': _$CortadoLocalDaemonSyncStateEnumMap[instance.state]!,
      'workspaceId': instance.workspaceId,
      'workspacePath': instance.workspacePath,
    };

const _$CortadoLocalDaemonSyncStateEnumMap = {
  CortadoLocalDaemonSyncState.conflicted: 'CONFLICTED',
  CortadoLocalDaemonSyncState.idle: 'IDLE',
  CortadoLocalDaemonSyncState.syncing: 'SYNCING',
};

_CortadoLocalDaemonConflict _$CortadoLocalDaemonConflictFromJson(
        Map<String, dynamic> json) =>
    _CortadoLocalDaemonConflict(
      lastSyncedClock: (json['lastSyncedClock'] as num).toInt(),
      localClock: (json['localClock'] as num).toInt(),
      localPath: json['path'] as String,
      reason: json['reason'] as String,
      remoteClock: (json['remoteClock'] as num).toInt(),
      workspaceId: json['workspaceId'] as String?,
      workspacePath: json['workspacePath'] as String?,
    );

Map<String, dynamic> _$CortadoLocalDaemonConflictToJson(
        _CortadoLocalDaemonConflict instance) =>
    <String, dynamic>{
      'lastSyncedClock': instance.lastSyncedClock,
      'localClock': instance.localClock,
      'path': instance.localPath,
      'reason': instance.reason,
      'remoteClock': instance.remoteClock,
      'workspaceId': instance.workspaceId,
      'workspacePath': instance.workspacePath,
    };
