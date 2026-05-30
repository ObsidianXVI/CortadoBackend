// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'workspace_models.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

_WorkspaceResources _$WorkspaceResourcesFromJson(Map<String, dynamic> json) =>
    _WorkspaceResources(
      cpu: (json['cpu'] as num).toDouble(),
      memoryGb: (json['memoryGb'] as num).toDouble(),
      storageGb: (json['storageGb'] as num).toDouble(),
    );

Map<String, dynamic> _$WorkspaceResourcesToJson(_WorkspaceResources instance) =>
    <String, dynamic>{
      'cpu': instance.cpu,
      'memoryGb': instance.memoryGb,
      'storageGb': instance.storageGb,
    };

_Workspace _$WorkspaceFromJson(Map<String, dynamic> json) => _Workspace(
      id: json['id'] as String,
      tenantId: json['tenantId'] as String,
      userId: json['userId'] as String,
      image: json['image'] as String,
      resources: WorkspaceResources.fromJson(
          json['resources'] as Map<String, dynamic>),
      status: $enumDecode(_$WorkspaceLifecycleStateEnumMap, json['status']),
      createdAt: DateTime.parse(json['createdAt'] as String),
      updatedAt: DateTime.parse(json['updatedAt'] as String),
      lastActiveAt: json['lastActiveAt'] == null
          ? null
          : DateTime.parse(json['lastActiveAt'] as String),
    );

Map<String, dynamic> _$WorkspaceToJson(_Workspace instance) =>
    <String, dynamic>{
      'id': instance.id,
      'tenantId': instance.tenantId,
      'userId': instance.userId,
      'image': instance.image,
      'resources': instance.resources,
      'status': _$WorkspaceLifecycleStateEnumMap[instance.status]!,
      'createdAt': instance.createdAt.toIso8601String(),
      'updatedAt': instance.updatedAt.toIso8601String(),
      'lastActiveAt': instance.lastActiveAt?.toIso8601String(),
    };

const _$WorkspaceLifecycleStateEnumMap = {
  WorkspaceLifecycleState.creating: 'CREATING',
  WorkspaceLifecycleState.starting: 'STARTING',
  WorkspaceLifecycleState.running: 'RUNNING',
  WorkspaceLifecycleState.stopping: 'STOPPING',
  WorkspaceLifecycleState.stopped: 'STOPPED',
  WorkspaceLifecycleState.deleted: 'DELETED',
};

_WorkspaceStatus _$WorkspaceStatusFromJson(Map<String, dynamic> json) =>
    _WorkspaceStatus(
      workspaceId: json['workspaceId'] as String,
      status: $enumDecode(_$WorkspaceLifecycleStateEnumMap, json['status']),
      updatedAt: DateTime.parse(json['updatedAt'] as String),
      lastActiveAt: json['lastActiveAt'] == null
          ? null
          : DateTime.parse(json['lastActiveAt'] as String),
    );

Map<String, dynamic> _$WorkspaceStatusToJson(_WorkspaceStatus instance) =>
    <String, dynamic>{
      'workspaceId': instance.workspaceId,
      'status': _$WorkspaceLifecycleStateEnumMap[instance.status]!,
      'updatedAt': instance.updatedAt.toIso8601String(),
      'lastActiveAt': instance.lastActiveAt?.toIso8601String(),
    };
