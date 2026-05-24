// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'local_daemon_models.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

_$CortadoLocalDaemonSyncStatusImpl _$$CortadoLocalDaemonSyncStatusImplFromJson(
        Map<String, dynamic> json) =>
    _$CortadoLocalDaemonSyncStatusImpl(
      localPath: json['localPath'] as String,
      message: json['message'] as String?,
      state: $enumDecode(_$CortadoLocalDaemonSyncStateEnumMap, json['state']),
      workspaceId: json['workspaceId'] as String,
      workspacePath: json['workspacePath'] as String? ?? '/',
    );

Map<String, dynamic> _$$CortadoLocalDaemonSyncStatusImplToJson(
        _$CortadoLocalDaemonSyncStatusImpl instance) =>
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

_$CortadoLocalDaemonConflictImpl _$$CortadoLocalDaemonConflictImplFromJson(
        Map<String, dynamic> json) =>
    _$CortadoLocalDaemonConflictImpl(
      lastSyncedClock: (json['lastSyncedClock'] as num).toInt(),
      localClock: (json['localClock'] as num).toInt(),
      localPath: json['path'] as String,
      reason: json['reason'] as String,
      remoteClock: (json['remoteClock'] as num).toInt(),
      workspaceId: json['workspaceId'] as String?,
      workspacePath: json['workspacePath'] as String?,
    );

Map<String, dynamic> _$$CortadoLocalDaemonConflictImplToJson(
        _$CortadoLocalDaemonConflictImpl instance) =>
    <String, dynamic>{
      'lastSyncedClock': instance.lastSyncedClock,
      'localClock': instance.localClock,
      'path': instance.localPath,
      'reason': instance.reason,
      'remoteClock': instance.remoteClock,
      'workspaceId': instance.workspaceId,
      'workspacePath': instance.workspacePath,
    };
