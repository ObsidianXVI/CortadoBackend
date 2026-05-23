// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark

part of 'workspace_models.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

T _$identity<T>(T value) => value;

final _privateConstructorUsedError = UnsupportedError(
    'It seems like you constructed your class using `MyClass._()`. This constructor is only meant to be used by freezed and you are not supposed to need it nor use it.\nPlease check the documentation here for more information: https://github.com/rrousselGit/freezed#adding-getters-and-methods-to-our-models');

WorkspaceResources _$WorkspaceResourcesFromJson(Map<String, dynamic> json) {
  return _WorkspaceResources.fromJson(json);
}

/// @nodoc
mixin _$WorkspaceResources {
  double get cpu => throw _privateConstructorUsedError;
  @JsonKey(name: 'memoryGb')
  double get memoryGb => throw _privateConstructorUsedError;

  /// Serializes this WorkspaceResources to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of WorkspaceResources
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $WorkspaceResourcesCopyWith<WorkspaceResources> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $WorkspaceResourcesCopyWith<$Res> {
  factory $WorkspaceResourcesCopyWith(
          WorkspaceResources value, $Res Function(WorkspaceResources) then) =
      _$WorkspaceResourcesCopyWithImpl<$Res, WorkspaceResources>;
  @useResult
  $Res call({double cpu, @JsonKey(name: 'memoryGb') double memoryGb});
}

/// @nodoc
class _$WorkspaceResourcesCopyWithImpl<$Res, $Val extends WorkspaceResources>
    implements $WorkspaceResourcesCopyWith<$Res> {
  _$WorkspaceResourcesCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of WorkspaceResources
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? cpu = null,
    Object? memoryGb = null,
  }) {
    return _then(_value.copyWith(
      cpu: null == cpu
          ? _value.cpu
          : cpu // ignore: cast_nullable_to_non_nullable
              as double,
      memoryGb: null == memoryGb
          ? _value.memoryGb
          : memoryGb // ignore: cast_nullable_to_non_nullable
              as double,
    ) as $Val);
  }
}

/// @nodoc
abstract class _$$WorkspaceResourcesImplCopyWith<$Res>
    implements $WorkspaceResourcesCopyWith<$Res> {
  factory _$$WorkspaceResourcesImplCopyWith(_$WorkspaceResourcesImpl value,
          $Res Function(_$WorkspaceResourcesImpl) then) =
      __$$WorkspaceResourcesImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call({double cpu, @JsonKey(name: 'memoryGb') double memoryGb});
}

/// @nodoc
class __$$WorkspaceResourcesImplCopyWithImpl<$Res>
    extends _$WorkspaceResourcesCopyWithImpl<$Res, _$WorkspaceResourcesImpl>
    implements _$$WorkspaceResourcesImplCopyWith<$Res> {
  __$$WorkspaceResourcesImplCopyWithImpl(_$WorkspaceResourcesImpl _value,
      $Res Function(_$WorkspaceResourcesImpl) _then)
      : super(_value, _then);

  /// Create a copy of WorkspaceResources
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? cpu = null,
    Object? memoryGb = null,
  }) {
    return _then(_$WorkspaceResourcesImpl(
      cpu: null == cpu
          ? _value.cpu
          : cpu // ignore: cast_nullable_to_non_nullable
              as double,
      memoryGb: null == memoryGb
          ? _value.memoryGb
          : memoryGb // ignore: cast_nullable_to_non_nullable
              as double,
    ));
  }
}

/// @nodoc
@JsonSerializable()
class _$WorkspaceResourcesImpl implements _WorkspaceResources {
  const _$WorkspaceResourcesImpl(
      {required this.cpu, @JsonKey(name: 'memoryGb') required this.memoryGb});

  factory _$WorkspaceResourcesImpl.fromJson(Map<String, dynamic> json) =>
      _$$WorkspaceResourcesImplFromJson(json);

  @override
  final double cpu;
  @override
  @JsonKey(name: 'memoryGb')
  final double memoryGb;

  @override
  String toString() {
    return 'WorkspaceResources(cpu: $cpu, memoryGb: $memoryGb)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$WorkspaceResourcesImpl &&
            (identical(other.cpu, cpu) || other.cpu == cpu) &&
            (identical(other.memoryGb, memoryGb) ||
                other.memoryGb == memoryGb));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(runtimeType, cpu, memoryGb);

  /// Create a copy of WorkspaceResources
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$WorkspaceResourcesImplCopyWith<_$WorkspaceResourcesImpl> get copyWith =>
      __$$WorkspaceResourcesImplCopyWithImpl<_$WorkspaceResourcesImpl>(
          this, _$identity);

  @override
  Map<String, dynamic> toJson() {
    return _$$WorkspaceResourcesImplToJson(
      this,
    );
  }
}

abstract class _WorkspaceResources implements WorkspaceResources {
  const factory _WorkspaceResources(
          {required final double cpu,
          @JsonKey(name: 'memoryGb') required final double memoryGb}) =
      _$WorkspaceResourcesImpl;

  factory _WorkspaceResources.fromJson(Map<String, dynamic> json) =
      _$WorkspaceResourcesImpl.fromJson;

  @override
  double get cpu;
  @override
  @JsonKey(name: 'memoryGb')
  double get memoryGb;

  /// Create a copy of WorkspaceResources
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$WorkspaceResourcesImplCopyWith<_$WorkspaceResourcesImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

Workspace _$WorkspaceFromJson(Map<String, dynamic> json) {
  return _Workspace.fromJson(json);
}

/// @nodoc
mixin _$Workspace {
  String get id => throw _privateConstructorUsedError;
  String get tenantId => throw _privateConstructorUsedError;
  String get userId => throw _privateConstructorUsedError;
  String get image => throw _privateConstructorUsedError;
  WorkspaceResources get resources => throw _privateConstructorUsedError;
  WorkspaceLifecycleState get status => throw _privateConstructorUsedError;
  DateTime get createdAt => throw _privateConstructorUsedError;
  DateTime get updatedAt => throw _privateConstructorUsedError;
  DateTime? get lastActiveAt => throw _privateConstructorUsedError;

  /// Serializes this Workspace to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of Workspace
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $WorkspaceCopyWith<Workspace> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $WorkspaceCopyWith<$Res> {
  factory $WorkspaceCopyWith(Workspace value, $Res Function(Workspace) then) =
      _$WorkspaceCopyWithImpl<$Res, Workspace>;
  @useResult
  $Res call(
      {String id,
      String tenantId,
      String userId,
      String image,
      WorkspaceResources resources,
      WorkspaceLifecycleState status,
      DateTime createdAt,
      DateTime updatedAt,
      DateTime? lastActiveAt});

  $WorkspaceResourcesCopyWith<$Res> get resources;
}

/// @nodoc
class _$WorkspaceCopyWithImpl<$Res, $Val extends Workspace>
    implements $WorkspaceCopyWith<$Res> {
  _$WorkspaceCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of Workspace
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = null,
    Object? tenantId = null,
    Object? userId = null,
    Object? image = null,
    Object? resources = null,
    Object? status = null,
    Object? createdAt = null,
    Object? updatedAt = null,
    Object? lastActiveAt = freezed,
  }) {
    return _then(_value.copyWith(
      id: null == id
          ? _value.id
          : id // ignore: cast_nullable_to_non_nullable
              as String,
      tenantId: null == tenantId
          ? _value.tenantId
          : tenantId // ignore: cast_nullable_to_non_nullable
              as String,
      userId: null == userId
          ? _value.userId
          : userId // ignore: cast_nullable_to_non_nullable
              as String,
      image: null == image
          ? _value.image
          : image // ignore: cast_nullable_to_non_nullable
              as String,
      resources: null == resources
          ? _value.resources
          : resources // ignore: cast_nullable_to_non_nullable
              as WorkspaceResources,
      status: null == status
          ? _value.status
          : status // ignore: cast_nullable_to_non_nullable
              as WorkspaceLifecycleState,
      createdAt: null == createdAt
          ? _value.createdAt
          : createdAt // ignore: cast_nullable_to_non_nullable
              as DateTime,
      updatedAt: null == updatedAt
          ? _value.updatedAt
          : updatedAt // ignore: cast_nullable_to_non_nullable
              as DateTime,
      lastActiveAt: freezed == lastActiveAt
          ? _value.lastActiveAt
          : lastActiveAt // ignore: cast_nullable_to_non_nullable
              as DateTime?,
    ) as $Val);
  }

  /// Create a copy of Workspace
  /// with the given fields replaced by the non-null parameter values.
  @override
  @pragma('vm:prefer-inline')
  $WorkspaceResourcesCopyWith<$Res> get resources {
    return $WorkspaceResourcesCopyWith<$Res>(_value.resources, (value) {
      return _then(_value.copyWith(resources: value) as $Val);
    });
  }
}

/// @nodoc
abstract class _$$WorkspaceImplCopyWith<$Res>
    implements $WorkspaceCopyWith<$Res> {
  factory _$$WorkspaceImplCopyWith(
          _$WorkspaceImpl value, $Res Function(_$WorkspaceImpl) then) =
      __$$WorkspaceImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call(
      {String id,
      String tenantId,
      String userId,
      String image,
      WorkspaceResources resources,
      WorkspaceLifecycleState status,
      DateTime createdAt,
      DateTime updatedAt,
      DateTime? lastActiveAt});

  @override
  $WorkspaceResourcesCopyWith<$Res> get resources;
}

/// @nodoc
class __$$WorkspaceImplCopyWithImpl<$Res>
    extends _$WorkspaceCopyWithImpl<$Res, _$WorkspaceImpl>
    implements _$$WorkspaceImplCopyWith<$Res> {
  __$$WorkspaceImplCopyWithImpl(
      _$WorkspaceImpl _value, $Res Function(_$WorkspaceImpl) _then)
      : super(_value, _then);

  /// Create a copy of Workspace
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? id = null,
    Object? tenantId = null,
    Object? userId = null,
    Object? image = null,
    Object? resources = null,
    Object? status = null,
    Object? createdAt = null,
    Object? updatedAt = null,
    Object? lastActiveAt = freezed,
  }) {
    return _then(_$WorkspaceImpl(
      id: null == id
          ? _value.id
          : id // ignore: cast_nullable_to_non_nullable
              as String,
      tenantId: null == tenantId
          ? _value.tenantId
          : tenantId // ignore: cast_nullable_to_non_nullable
              as String,
      userId: null == userId
          ? _value.userId
          : userId // ignore: cast_nullable_to_non_nullable
              as String,
      image: null == image
          ? _value.image
          : image // ignore: cast_nullable_to_non_nullable
              as String,
      resources: null == resources
          ? _value.resources
          : resources // ignore: cast_nullable_to_non_nullable
              as WorkspaceResources,
      status: null == status
          ? _value.status
          : status // ignore: cast_nullable_to_non_nullable
              as WorkspaceLifecycleState,
      createdAt: null == createdAt
          ? _value.createdAt
          : createdAt // ignore: cast_nullable_to_non_nullable
              as DateTime,
      updatedAt: null == updatedAt
          ? _value.updatedAt
          : updatedAt // ignore: cast_nullable_to_non_nullable
              as DateTime,
      lastActiveAt: freezed == lastActiveAt
          ? _value.lastActiveAt
          : lastActiveAt // ignore: cast_nullable_to_non_nullable
              as DateTime?,
    ));
  }
}

/// @nodoc
@JsonSerializable()
class _$WorkspaceImpl extends _Workspace {
  const _$WorkspaceImpl(
      {required this.id,
      required this.tenantId,
      required this.userId,
      required this.image,
      required this.resources,
      required this.status,
      required this.createdAt,
      required this.updatedAt,
      this.lastActiveAt})
      : super._();

  factory _$WorkspaceImpl.fromJson(Map<String, dynamic> json) =>
      _$$WorkspaceImplFromJson(json);

  @override
  final String id;
  @override
  final String tenantId;
  @override
  final String userId;
  @override
  final String image;
  @override
  final WorkspaceResources resources;
  @override
  final WorkspaceLifecycleState status;
  @override
  final DateTime createdAt;
  @override
  final DateTime updatedAt;
  @override
  final DateTime? lastActiveAt;

  @override
  String toString() {
    return 'Workspace(id: $id, tenantId: $tenantId, userId: $userId, image: $image, resources: $resources, status: $status, createdAt: $createdAt, updatedAt: $updatedAt, lastActiveAt: $lastActiveAt)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$WorkspaceImpl &&
            (identical(other.id, id) || other.id == id) &&
            (identical(other.tenantId, tenantId) ||
                other.tenantId == tenantId) &&
            (identical(other.userId, userId) || other.userId == userId) &&
            (identical(other.image, image) || other.image == image) &&
            (identical(other.resources, resources) ||
                other.resources == resources) &&
            (identical(other.status, status) || other.status == status) &&
            (identical(other.createdAt, createdAt) ||
                other.createdAt == createdAt) &&
            (identical(other.updatedAt, updatedAt) ||
                other.updatedAt == updatedAt) &&
            (identical(other.lastActiveAt, lastActiveAt) ||
                other.lastActiveAt == lastActiveAt));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(runtimeType, id, tenantId, userId, image,
      resources, status, createdAt, updatedAt, lastActiveAt);

  /// Create a copy of Workspace
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$WorkspaceImplCopyWith<_$WorkspaceImpl> get copyWith =>
      __$$WorkspaceImplCopyWithImpl<_$WorkspaceImpl>(this, _$identity);

  @override
  Map<String, dynamic> toJson() {
    return _$$WorkspaceImplToJson(
      this,
    );
  }
}

abstract class _Workspace extends Workspace {
  const factory _Workspace(
      {required final String id,
      required final String tenantId,
      required final String userId,
      required final String image,
      required final WorkspaceResources resources,
      required final WorkspaceLifecycleState status,
      required final DateTime createdAt,
      required final DateTime updatedAt,
      final DateTime? lastActiveAt}) = _$WorkspaceImpl;
  const _Workspace._() : super._();

  factory _Workspace.fromJson(Map<String, dynamic> json) =
      _$WorkspaceImpl.fromJson;

  @override
  String get id;
  @override
  String get tenantId;
  @override
  String get userId;
  @override
  String get image;
  @override
  WorkspaceResources get resources;
  @override
  WorkspaceLifecycleState get status;
  @override
  DateTime get createdAt;
  @override
  DateTime get updatedAt;
  @override
  DateTime? get lastActiveAt;

  /// Create a copy of Workspace
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$WorkspaceImplCopyWith<_$WorkspaceImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

WorkspaceStatus _$WorkspaceStatusFromJson(Map<String, dynamic> json) {
  return _WorkspaceStatus.fromJson(json);
}

/// @nodoc
mixin _$WorkspaceStatus {
  String get workspaceId => throw _privateConstructorUsedError;
  WorkspaceLifecycleState get status => throw _privateConstructorUsedError;
  DateTime get updatedAt => throw _privateConstructorUsedError;
  DateTime? get lastActiveAt => throw _privateConstructorUsedError;

  /// Serializes this WorkspaceStatus to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of WorkspaceStatus
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $WorkspaceStatusCopyWith<WorkspaceStatus> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $WorkspaceStatusCopyWith<$Res> {
  factory $WorkspaceStatusCopyWith(
          WorkspaceStatus value, $Res Function(WorkspaceStatus) then) =
      _$WorkspaceStatusCopyWithImpl<$Res, WorkspaceStatus>;
  @useResult
  $Res call(
      {String workspaceId,
      WorkspaceLifecycleState status,
      DateTime updatedAt,
      DateTime? lastActiveAt});
}

/// @nodoc
class _$WorkspaceStatusCopyWithImpl<$Res, $Val extends WorkspaceStatus>
    implements $WorkspaceStatusCopyWith<$Res> {
  _$WorkspaceStatusCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of WorkspaceStatus
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? workspaceId = null,
    Object? status = null,
    Object? updatedAt = null,
    Object? lastActiveAt = freezed,
  }) {
    return _then(_value.copyWith(
      workspaceId: null == workspaceId
          ? _value.workspaceId
          : workspaceId // ignore: cast_nullable_to_non_nullable
              as String,
      status: null == status
          ? _value.status
          : status // ignore: cast_nullable_to_non_nullable
              as WorkspaceLifecycleState,
      updatedAt: null == updatedAt
          ? _value.updatedAt
          : updatedAt // ignore: cast_nullable_to_non_nullable
              as DateTime,
      lastActiveAt: freezed == lastActiveAt
          ? _value.lastActiveAt
          : lastActiveAt // ignore: cast_nullable_to_non_nullable
              as DateTime?,
    ) as $Val);
  }
}

/// @nodoc
abstract class _$$WorkspaceStatusImplCopyWith<$Res>
    implements $WorkspaceStatusCopyWith<$Res> {
  factory _$$WorkspaceStatusImplCopyWith(_$WorkspaceStatusImpl value,
          $Res Function(_$WorkspaceStatusImpl) then) =
      __$$WorkspaceStatusImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call(
      {String workspaceId,
      WorkspaceLifecycleState status,
      DateTime updatedAt,
      DateTime? lastActiveAt});
}

/// @nodoc
class __$$WorkspaceStatusImplCopyWithImpl<$Res>
    extends _$WorkspaceStatusCopyWithImpl<$Res, _$WorkspaceStatusImpl>
    implements _$$WorkspaceStatusImplCopyWith<$Res> {
  __$$WorkspaceStatusImplCopyWithImpl(
      _$WorkspaceStatusImpl _value, $Res Function(_$WorkspaceStatusImpl) _then)
      : super(_value, _then);

  /// Create a copy of WorkspaceStatus
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? workspaceId = null,
    Object? status = null,
    Object? updatedAt = null,
    Object? lastActiveAt = freezed,
  }) {
    return _then(_$WorkspaceStatusImpl(
      workspaceId: null == workspaceId
          ? _value.workspaceId
          : workspaceId // ignore: cast_nullable_to_non_nullable
              as String,
      status: null == status
          ? _value.status
          : status // ignore: cast_nullable_to_non_nullable
              as WorkspaceLifecycleState,
      updatedAt: null == updatedAt
          ? _value.updatedAt
          : updatedAt // ignore: cast_nullable_to_non_nullable
              as DateTime,
      lastActiveAt: freezed == lastActiveAt
          ? _value.lastActiveAt
          : lastActiveAt // ignore: cast_nullable_to_non_nullable
              as DateTime?,
    ));
  }
}

/// @nodoc
@JsonSerializable()
class _$WorkspaceStatusImpl extends _WorkspaceStatus {
  const _$WorkspaceStatusImpl(
      {required this.workspaceId,
      required this.status,
      required this.updatedAt,
      this.lastActiveAt})
      : super._();

  factory _$WorkspaceStatusImpl.fromJson(Map<String, dynamic> json) =>
      _$$WorkspaceStatusImplFromJson(json);

  @override
  final String workspaceId;
  @override
  final WorkspaceLifecycleState status;
  @override
  final DateTime updatedAt;
  @override
  final DateTime? lastActiveAt;

  @override
  String toString() {
    return 'WorkspaceStatus(workspaceId: $workspaceId, status: $status, updatedAt: $updatedAt, lastActiveAt: $lastActiveAt)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$WorkspaceStatusImpl &&
            (identical(other.workspaceId, workspaceId) ||
                other.workspaceId == workspaceId) &&
            (identical(other.status, status) || other.status == status) &&
            (identical(other.updatedAt, updatedAt) ||
                other.updatedAt == updatedAt) &&
            (identical(other.lastActiveAt, lastActiveAt) ||
                other.lastActiveAt == lastActiveAt));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode =>
      Object.hash(runtimeType, workspaceId, status, updatedAt, lastActiveAt);

  /// Create a copy of WorkspaceStatus
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$WorkspaceStatusImplCopyWith<_$WorkspaceStatusImpl> get copyWith =>
      __$$WorkspaceStatusImplCopyWithImpl<_$WorkspaceStatusImpl>(
          this, _$identity);

  @override
  Map<String, dynamic> toJson() {
    return _$$WorkspaceStatusImplToJson(
      this,
    );
  }
}

abstract class _WorkspaceStatus extends WorkspaceStatus {
  const factory _WorkspaceStatus(
      {required final String workspaceId,
      required final WorkspaceLifecycleState status,
      required final DateTime updatedAt,
      final DateTime? lastActiveAt}) = _$WorkspaceStatusImpl;
  const _WorkspaceStatus._() : super._();

  factory _WorkspaceStatus.fromJson(Map<String, dynamic> json) =
      _$WorkspaceStatusImpl.fromJson;

  @override
  String get workspaceId;
  @override
  WorkspaceLifecycleState get status;
  @override
  DateTime get updatedAt;
  @override
  DateTime? get lastActiveAt;

  /// Create a copy of WorkspaceStatus
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$WorkspaceStatusImplCopyWith<_$WorkspaceStatusImpl> get copyWith =>
      throw _privateConstructorUsedError;
}
