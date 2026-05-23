// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'workspace_models.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

_$WorkspaceResourcesImpl _$$WorkspaceResourcesImplFromJson(
        Map<String, dynamic> json) =>
    _$WorkspaceResourcesImpl(
      cpu: (json['cpu'] as num).toDouble(),
      memoryGb: (json['memoryGb'] as num).toDouble(),
    );

Map<String, dynamic> _$$WorkspaceResourcesImplToJson(
        _$WorkspaceResourcesImpl instance) =>
    <String, dynamic>{
      'cpu': instance.cpu,
      'memoryGb': instance.memoryGb,
    };

_$WorkspaceImpl _$$WorkspaceImplFromJson(Map<String, dynamic> json) =>
    _$WorkspaceImpl(
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

Map<String, dynamic> _$$WorkspaceImplToJson(_$WorkspaceImpl instance) =>
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

_$WorkspaceStatusImpl _$$WorkspaceStatusImplFromJson(
        Map<String, dynamic> json) =>
    _$WorkspaceStatusImpl(
      workspaceId: json['workspaceId'] as String,
      status: $enumDecode(_$WorkspaceLifecycleStateEnumMap, json['status']),
      updatedAt: DateTime.parse(json['updatedAt'] as String),
      lastActiveAt: json['lastActiveAt'] == null
          ? null
          : DateTime.parse(json['lastActiveAt'] as String),
    );

Map<String, dynamic> _$$WorkspaceStatusImplToJson(
        _$WorkspaceStatusImpl instance) =>
    <String, dynamic>{
      'workspaceId': instance.workspaceId,
      'status': _$WorkspaceLifecycleStateEnumMap[instance.status]!,
      'updatedAt': instance.updatedAt.toIso8601String(),
      'lastActiveAt': instance.lastActiveAt?.toIso8601String(),
    };
