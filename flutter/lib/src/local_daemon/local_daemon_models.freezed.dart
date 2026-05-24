// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark

part of 'local_daemon_models.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

T _$identity<T>(T value) => value;

final _privateConstructorUsedError = UnsupportedError(
    'It seems like you constructed your class using `MyClass._()`. This constructor is only meant to be used by freezed and you are not supposed to need it nor use it.\nPlease check the documentation here for more information: https://github.com/rrousselGit/freezed#adding-getters-and-methods-to-our-models');

/// @nodoc
mixin _$CortadoLocalDaemonAvailability {
  String get installUrl => throw _privateConstructorUsedError;
  String? get message => throw _privateConstructorUsedError;
  CortadoLocalDaemonAvailabilityState get state =>
      throw _privateConstructorUsedError;

  /// Create a copy of CortadoLocalDaemonAvailability
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $CortadoLocalDaemonAvailabilityCopyWith<CortadoLocalDaemonAvailability>
      get copyWith => throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $CortadoLocalDaemonAvailabilityCopyWith<$Res> {
  factory $CortadoLocalDaemonAvailabilityCopyWith(
          CortadoLocalDaemonAvailability value,
          $Res Function(CortadoLocalDaemonAvailability) then) =
      _$CortadoLocalDaemonAvailabilityCopyWithImpl<$Res,
          CortadoLocalDaemonAvailability>;
  @useResult
  $Res call(
      {String installUrl,
      String? message,
      CortadoLocalDaemonAvailabilityState state});
}

/// @nodoc
class _$CortadoLocalDaemonAvailabilityCopyWithImpl<$Res,
        $Val extends CortadoLocalDaemonAvailability>
    implements $CortadoLocalDaemonAvailabilityCopyWith<$Res> {
  _$CortadoLocalDaemonAvailabilityCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of CortadoLocalDaemonAvailability
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? installUrl = null,
    Object? message = freezed,
    Object? state = null,
  }) {
    return _then(_value.copyWith(
      installUrl: null == installUrl
          ? _value.installUrl
          : installUrl // ignore: cast_nullable_to_non_nullable
              as String,
      message: freezed == message
          ? _value.message
          : message // ignore: cast_nullable_to_non_nullable
              as String?,
      state: null == state
          ? _value.state
          : state // ignore: cast_nullable_to_non_nullable
              as CortadoLocalDaemonAvailabilityState,
    ) as $Val);
  }
}

/// @nodoc
abstract class _$$CortadoLocalDaemonAvailabilityImplCopyWith<$Res>
    implements $CortadoLocalDaemonAvailabilityCopyWith<$Res> {
  factory _$$CortadoLocalDaemonAvailabilityImplCopyWith(
          _$CortadoLocalDaemonAvailabilityImpl value,
          $Res Function(_$CortadoLocalDaemonAvailabilityImpl) then) =
      __$$CortadoLocalDaemonAvailabilityImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call(
      {String installUrl,
      String? message,
      CortadoLocalDaemonAvailabilityState state});
}

/// @nodoc
class __$$CortadoLocalDaemonAvailabilityImplCopyWithImpl<$Res>
    extends _$CortadoLocalDaemonAvailabilityCopyWithImpl<$Res,
        _$CortadoLocalDaemonAvailabilityImpl>
    implements _$$CortadoLocalDaemonAvailabilityImplCopyWith<$Res> {
  __$$CortadoLocalDaemonAvailabilityImplCopyWithImpl(
      _$CortadoLocalDaemonAvailabilityImpl _value,
      $Res Function(_$CortadoLocalDaemonAvailabilityImpl) _then)
      : super(_value, _then);

  /// Create a copy of CortadoLocalDaemonAvailability
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? installUrl = null,
    Object? message = freezed,
    Object? state = null,
  }) {
    return _then(_$CortadoLocalDaemonAvailabilityImpl(
      installUrl: null == installUrl
          ? _value.installUrl
          : installUrl // ignore: cast_nullable_to_non_nullable
              as String,
      message: freezed == message
          ? _value.message
          : message // ignore: cast_nullable_to_non_nullable
              as String?,
      state: null == state
          ? _value.state
          : state // ignore: cast_nullable_to_non_nullable
              as CortadoLocalDaemonAvailabilityState,
    ));
  }
}

/// @nodoc

class _$CortadoLocalDaemonAvailabilityImpl
    extends _CortadoLocalDaemonAvailability {
  const _$CortadoLocalDaemonAvailabilityImpl(
      {this.installUrl = cortadoDaemonInstallUrl,
      this.message,
      this.state = CortadoLocalDaemonAvailabilityState.unknown})
      : super._();

  @override
  @JsonKey()
  final String installUrl;
  @override
  final String? message;
  @override
  @JsonKey()
  final CortadoLocalDaemonAvailabilityState state;

  @override
  String toString() {
    return 'CortadoLocalDaemonAvailability(installUrl: $installUrl, message: $message, state: $state)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$CortadoLocalDaemonAvailabilityImpl &&
            (identical(other.installUrl, installUrl) ||
                other.installUrl == installUrl) &&
            (identical(other.message, message) || other.message == message) &&
            (identical(other.state, state) || other.state == state));
  }

  @override
  int get hashCode => Object.hash(runtimeType, installUrl, message, state);

  /// Create a copy of CortadoLocalDaemonAvailability
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$CortadoLocalDaemonAvailabilityImplCopyWith<
          _$CortadoLocalDaemonAvailabilityImpl>
      get copyWith => __$$CortadoLocalDaemonAvailabilityImplCopyWithImpl<
          _$CortadoLocalDaemonAvailabilityImpl>(this, _$identity);
}

abstract class _CortadoLocalDaemonAvailability
    extends CortadoLocalDaemonAvailability {
  const factory _CortadoLocalDaemonAvailability(
          {final String installUrl,
          final String? message,
          final CortadoLocalDaemonAvailabilityState state}) =
      _$CortadoLocalDaemonAvailabilityImpl;
  const _CortadoLocalDaemonAvailability._() : super._();

  @override
  String get installUrl;
  @override
  String? get message;
  @override
  CortadoLocalDaemonAvailabilityState get state;

  /// Create a copy of CortadoLocalDaemonAvailability
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$CortadoLocalDaemonAvailabilityImplCopyWith<
          _$CortadoLocalDaemonAvailabilityImpl>
      get copyWith => throw _privateConstructorUsedError;
}

CortadoLocalDaemonSyncStatus _$CortadoLocalDaemonSyncStatusFromJson(
    Map<String, dynamic> json) {
  return _CortadoLocalDaemonSyncStatus.fromJson(json);
}

/// @nodoc
mixin _$CortadoLocalDaemonSyncStatus {
  String get localPath => throw _privateConstructorUsedError;
  String? get message => throw _privateConstructorUsedError;
  CortadoLocalDaemonSyncState get state => throw _privateConstructorUsedError;
  String get workspaceId => throw _privateConstructorUsedError;
  String get workspacePath => throw _privateConstructorUsedError;

  /// Serializes this CortadoLocalDaemonSyncStatus to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of CortadoLocalDaemonSyncStatus
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $CortadoLocalDaemonSyncStatusCopyWith<CortadoLocalDaemonSyncStatus>
      get copyWith => throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $CortadoLocalDaemonSyncStatusCopyWith<$Res> {
  factory $CortadoLocalDaemonSyncStatusCopyWith(
          CortadoLocalDaemonSyncStatus value,
          $Res Function(CortadoLocalDaemonSyncStatus) then) =
      _$CortadoLocalDaemonSyncStatusCopyWithImpl<$Res,
          CortadoLocalDaemonSyncStatus>;
  @useResult
  $Res call(
      {String localPath,
      String? message,
      CortadoLocalDaemonSyncState state,
      String workspaceId,
      String workspacePath});
}

/// @nodoc
class _$CortadoLocalDaemonSyncStatusCopyWithImpl<$Res,
        $Val extends CortadoLocalDaemonSyncStatus>
    implements $CortadoLocalDaemonSyncStatusCopyWith<$Res> {
  _$CortadoLocalDaemonSyncStatusCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of CortadoLocalDaemonSyncStatus
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? localPath = null,
    Object? message = freezed,
    Object? state = null,
    Object? workspaceId = null,
    Object? workspacePath = null,
  }) {
    return _then(_value.copyWith(
      localPath: null == localPath
          ? _value.localPath
          : localPath // ignore: cast_nullable_to_non_nullable
              as String,
      message: freezed == message
          ? _value.message
          : message // ignore: cast_nullable_to_non_nullable
              as String?,
      state: null == state
          ? _value.state
          : state // ignore: cast_nullable_to_non_nullable
              as CortadoLocalDaemonSyncState,
      workspaceId: null == workspaceId
          ? _value.workspaceId
          : workspaceId // ignore: cast_nullable_to_non_nullable
              as String,
      workspacePath: null == workspacePath
          ? _value.workspacePath
          : workspacePath // ignore: cast_nullable_to_non_nullable
              as String,
    ) as $Val);
  }
}

/// @nodoc
abstract class _$$CortadoLocalDaemonSyncStatusImplCopyWith<$Res>
    implements $CortadoLocalDaemonSyncStatusCopyWith<$Res> {
  factory _$$CortadoLocalDaemonSyncStatusImplCopyWith(
          _$CortadoLocalDaemonSyncStatusImpl value,
          $Res Function(_$CortadoLocalDaemonSyncStatusImpl) then) =
      __$$CortadoLocalDaemonSyncStatusImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call(
      {String localPath,
      String? message,
      CortadoLocalDaemonSyncState state,
      String workspaceId,
      String workspacePath});
}

/// @nodoc
class __$$CortadoLocalDaemonSyncStatusImplCopyWithImpl<$Res>
    extends _$CortadoLocalDaemonSyncStatusCopyWithImpl<$Res,
        _$CortadoLocalDaemonSyncStatusImpl>
    implements _$$CortadoLocalDaemonSyncStatusImplCopyWith<$Res> {
  __$$CortadoLocalDaemonSyncStatusImplCopyWithImpl(
      _$CortadoLocalDaemonSyncStatusImpl _value,
      $Res Function(_$CortadoLocalDaemonSyncStatusImpl) _then)
      : super(_value, _then);

  /// Create a copy of CortadoLocalDaemonSyncStatus
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? localPath = null,
    Object? message = freezed,
    Object? state = null,
    Object? workspaceId = null,
    Object? workspacePath = null,
  }) {
    return _then(_$CortadoLocalDaemonSyncStatusImpl(
      localPath: null == localPath
          ? _value.localPath
          : localPath // ignore: cast_nullable_to_non_nullable
              as String,
      message: freezed == message
          ? _value.message
          : message // ignore: cast_nullable_to_non_nullable
              as String?,
      state: null == state
          ? _value.state
          : state // ignore: cast_nullable_to_non_nullable
              as CortadoLocalDaemonSyncState,
      workspaceId: null == workspaceId
          ? _value.workspaceId
          : workspaceId // ignore: cast_nullable_to_non_nullable
              as String,
      workspacePath: null == workspacePath
          ? _value.workspacePath
          : workspacePath // ignore: cast_nullable_to_non_nullable
              as String,
    ));
  }
}

/// @nodoc
@JsonSerializable()
class _$CortadoLocalDaemonSyncStatusImpl extends _CortadoLocalDaemonSyncStatus {
  const _$CortadoLocalDaemonSyncStatusImpl(
      {required this.localPath,
      this.message,
      required this.state,
      required this.workspaceId,
      this.workspacePath = '/'})
      : super._();

  factory _$CortadoLocalDaemonSyncStatusImpl.fromJson(
          Map<String, dynamic> json) =>
      _$$CortadoLocalDaemonSyncStatusImplFromJson(json);

  @override
  final String localPath;
  @override
  final String? message;
  @override
  final CortadoLocalDaemonSyncState state;
  @override
  final String workspaceId;
  @override
  @JsonKey()
  final String workspacePath;

  @override
  String toString() {
    return 'CortadoLocalDaemonSyncStatus(localPath: $localPath, message: $message, state: $state, workspaceId: $workspaceId, workspacePath: $workspacePath)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$CortadoLocalDaemonSyncStatusImpl &&
            (identical(other.localPath, localPath) ||
                other.localPath == localPath) &&
            (identical(other.message, message) || other.message == message) &&
            (identical(other.state, state) || other.state == state) &&
            (identical(other.workspaceId, workspaceId) ||
                other.workspaceId == workspaceId) &&
            (identical(other.workspacePath, workspacePath) ||
                other.workspacePath == workspacePath));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(
      runtimeType, localPath, message, state, workspaceId, workspacePath);

  /// Create a copy of CortadoLocalDaemonSyncStatus
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$CortadoLocalDaemonSyncStatusImplCopyWith<
          _$CortadoLocalDaemonSyncStatusImpl>
      get copyWith => __$$CortadoLocalDaemonSyncStatusImplCopyWithImpl<
          _$CortadoLocalDaemonSyncStatusImpl>(this, _$identity);

  @override
  Map<String, dynamic> toJson() {
    return _$$CortadoLocalDaemonSyncStatusImplToJson(
      this,
    );
  }
}

abstract class _CortadoLocalDaemonSyncStatus
    extends CortadoLocalDaemonSyncStatus {
  const factory _CortadoLocalDaemonSyncStatus(
      {required final String localPath,
      final String? message,
      required final CortadoLocalDaemonSyncState state,
      required final String workspaceId,
      final String workspacePath}) = _$CortadoLocalDaemonSyncStatusImpl;
  const _CortadoLocalDaemonSyncStatus._() : super._();

  factory _CortadoLocalDaemonSyncStatus.fromJson(Map<String, dynamic> json) =
      _$CortadoLocalDaemonSyncStatusImpl.fromJson;

  @override
  String get localPath;
  @override
  String? get message;
  @override
  CortadoLocalDaemonSyncState get state;
  @override
  String get workspaceId;
  @override
  String get workspacePath;

  /// Create a copy of CortadoLocalDaemonSyncStatus
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$CortadoLocalDaemonSyncStatusImplCopyWith<
          _$CortadoLocalDaemonSyncStatusImpl>
      get copyWith => throw _privateConstructorUsedError;
}

CortadoLocalDaemonConflict _$CortadoLocalDaemonConflictFromJson(
    Map<String, dynamic> json) {
  return _CortadoLocalDaemonConflict.fromJson(json);
}

/// @nodoc
mixin _$CortadoLocalDaemonConflict {
  int get lastSyncedClock => throw _privateConstructorUsedError;
  int get localClock => throw _privateConstructorUsedError;
  @JsonKey(name: 'path')
  String get localPath => throw _privateConstructorUsedError;
  String get reason => throw _privateConstructorUsedError;
  int get remoteClock => throw _privateConstructorUsedError;
  String? get workspaceId => throw _privateConstructorUsedError;
  String? get workspacePath => throw _privateConstructorUsedError;

  /// Serializes this CortadoLocalDaemonConflict to a JSON map.
  Map<String, dynamic> toJson() => throw _privateConstructorUsedError;

  /// Create a copy of CortadoLocalDaemonConflict
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $CortadoLocalDaemonConflictCopyWith<CortadoLocalDaemonConflict>
      get copyWith => throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $CortadoLocalDaemonConflictCopyWith<$Res> {
  factory $CortadoLocalDaemonConflictCopyWith(CortadoLocalDaemonConflict value,
          $Res Function(CortadoLocalDaemonConflict) then) =
      _$CortadoLocalDaemonConflictCopyWithImpl<$Res,
          CortadoLocalDaemonConflict>;
  @useResult
  $Res call(
      {int lastSyncedClock,
      int localClock,
      @JsonKey(name: 'path') String localPath,
      String reason,
      int remoteClock,
      String? workspaceId,
      String? workspacePath});
}

/// @nodoc
class _$CortadoLocalDaemonConflictCopyWithImpl<$Res,
        $Val extends CortadoLocalDaemonConflict>
    implements $CortadoLocalDaemonConflictCopyWith<$Res> {
  _$CortadoLocalDaemonConflictCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

  /// Create a copy of CortadoLocalDaemonConflict
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? lastSyncedClock = null,
    Object? localClock = null,
    Object? localPath = null,
    Object? reason = null,
    Object? remoteClock = null,
    Object? workspaceId = freezed,
    Object? workspacePath = freezed,
  }) {
    return _then(_value.copyWith(
      lastSyncedClock: null == lastSyncedClock
          ? _value.lastSyncedClock
          : lastSyncedClock // ignore: cast_nullable_to_non_nullable
              as int,
      localClock: null == localClock
          ? _value.localClock
          : localClock // ignore: cast_nullable_to_non_nullable
              as int,
      localPath: null == localPath
          ? _value.localPath
          : localPath // ignore: cast_nullable_to_non_nullable
              as String,
      reason: null == reason
          ? _value.reason
          : reason // ignore: cast_nullable_to_non_nullable
              as String,
      remoteClock: null == remoteClock
          ? _value.remoteClock
          : remoteClock // ignore: cast_nullable_to_non_nullable
              as int,
      workspaceId: freezed == workspaceId
          ? _value.workspaceId
          : workspaceId // ignore: cast_nullable_to_non_nullable
              as String?,
      workspacePath: freezed == workspacePath
          ? _value.workspacePath
          : workspacePath // ignore: cast_nullable_to_non_nullable
              as String?,
    ) as $Val);
  }
}

/// @nodoc
abstract class _$$CortadoLocalDaemonConflictImplCopyWith<$Res>
    implements $CortadoLocalDaemonConflictCopyWith<$Res> {
  factory _$$CortadoLocalDaemonConflictImplCopyWith(
          _$CortadoLocalDaemonConflictImpl value,
          $Res Function(_$CortadoLocalDaemonConflictImpl) then) =
      __$$CortadoLocalDaemonConflictImplCopyWithImpl<$Res>;
  @override
  @useResult
  $Res call(
      {int lastSyncedClock,
      int localClock,
      @JsonKey(name: 'path') String localPath,
      String reason,
      int remoteClock,
      String? workspaceId,
      String? workspacePath});
}

/// @nodoc
class __$$CortadoLocalDaemonConflictImplCopyWithImpl<$Res>
    extends _$CortadoLocalDaemonConflictCopyWithImpl<$Res,
        _$CortadoLocalDaemonConflictImpl>
    implements _$$CortadoLocalDaemonConflictImplCopyWith<$Res> {
  __$$CortadoLocalDaemonConflictImplCopyWithImpl(
      _$CortadoLocalDaemonConflictImpl _value,
      $Res Function(_$CortadoLocalDaemonConflictImpl) _then)
      : super(_value, _then);

  /// Create a copy of CortadoLocalDaemonConflict
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? lastSyncedClock = null,
    Object? localClock = null,
    Object? localPath = null,
    Object? reason = null,
    Object? remoteClock = null,
    Object? workspaceId = freezed,
    Object? workspacePath = freezed,
  }) {
    return _then(_$CortadoLocalDaemonConflictImpl(
      lastSyncedClock: null == lastSyncedClock
          ? _value.lastSyncedClock
          : lastSyncedClock // ignore: cast_nullable_to_non_nullable
              as int,
      localClock: null == localClock
          ? _value.localClock
          : localClock // ignore: cast_nullable_to_non_nullable
              as int,
      localPath: null == localPath
          ? _value.localPath
          : localPath // ignore: cast_nullable_to_non_nullable
              as String,
      reason: null == reason
          ? _value.reason
          : reason // ignore: cast_nullable_to_non_nullable
              as String,
      remoteClock: null == remoteClock
          ? _value.remoteClock
          : remoteClock // ignore: cast_nullable_to_non_nullable
              as int,
      workspaceId: freezed == workspaceId
          ? _value.workspaceId
          : workspaceId // ignore: cast_nullable_to_non_nullable
              as String?,
      workspacePath: freezed == workspacePath
          ? _value.workspacePath
          : workspacePath // ignore: cast_nullable_to_non_nullable
              as String?,
    ));
  }
}

/// @nodoc
@JsonSerializable()
class _$CortadoLocalDaemonConflictImpl extends _CortadoLocalDaemonConflict {
  const _$CortadoLocalDaemonConflictImpl(
      {required this.lastSyncedClock,
      required this.localClock,
      @JsonKey(name: 'path') required this.localPath,
      required this.reason,
      required this.remoteClock,
      this.workspaceId,
      this.workspacePath})
      : super._();

  factory _$CortadoLocalDaemonConflictImpl.fromJson(
          Map<String, dynamic> json) =>
      _$$CortadoLocalDaemonConflictImplFromJson(json);

  @override
  final int lastSyncedClock;
  @override
  final int localClock;
  @override
  @JsonKey(name: 'path')
  final String localPath;
  @override
  final String reason;
  @override
  final int remoteClock;
  @override
  final String? workspaceId;
  @override
  final String? workspacePath;

  @override
  String toString() {
    return 'CortadoLocalDaemonConflict(lastSyncedClock: $lastSyncedClock, localClock: $localClock, localPath: $localPath, reason: $reason, remoteClock: $remoteClock, workspaceId: $workspaceId, workspacePath: $workspacePath)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$CortadoLocalDaemonConflictImpl &&
            (identical(other.lastSyncedClock, lastSyncedClock) ||
                other.lastSyncedClock == lastSyncedClock) &&
            (identical(other.localClock, localClock) ||
                other.localClock == localClock) &&
            (identical(other.localPath, localPath) ||
                other.localPath == localPath) &&
            (identical(other.reason, reason) || other.reason == reason) &&
            (identical(other.remoteClock, remoteClock) ||
                other.remoteClock == remoteClock) &&
            (identical(other.workspaceId, workspaceId) ||
                other.workspaceId == workspaceId) &&
            (identical(other.workspacePath, workspacePath) ||
                other.workspacePath == workspacePath));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(runtimeType, lastSyncedClock, localClock,
      localPath, reason, remoteClock, workspaceId, workspacePath);

  /// Create a copy of CortadoLocalDaemonConflict
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$CortadoLocalDaemonConflictImplCopyWith<_$CortadoLocalDaemonConflictImpl>
      get copyWith => __$$CortadoLocalDaemonConflictImplCopyWithImpl<
          _$CortadoLocalDaemonConflictImpl>(this, _$identity);

  @override
  Map<String, dynamic> toJson() {
    return _$$CortadoLocalDaemonConflictImplToJson(
      this,
    );
  }
}

abstract class _CortadoLocalDaemonConflict extends CortadoLocalDaemonConflict {
  const factory _CortadoLocalDaemonConflict(
      {required final int lastSyncedClock,
      required final int localClock,
      @JsonKey(name: 'path') required final String localPath,
      required final String reason,
      required final int remoteClock,
      final String? workspaceId,
      final String? workspacePath}) = _$CortadoLocalDaemonConflictImpl;
  const _CortadoLocalDaemonConflict._() : super._();

  factory _CortadoLocalDaemonConflict.fromJson(Map<String, dynamic> json) =
      _$CortadoLocalDaemonConflictImpl.fromJson;

  @override
  int get lastSyncedClock;
  @override
  int get localClock;
  @override
  @JsonKey(name: 'path')
  String get localPath;
  @override
  String get reason;
  @override
  int get remoteClock;
  @override
  String? get workspaceId;
  @override
  String? get workspacePath;

  /// Create a copy of CortadoLocalDaemonConflict
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$CortadoLocalDaemonConflictImplCopyWith<_$CortadoLocalDaemonConflictImpl>
      get copyWith => throw _privateConstructorUsedError;
}
