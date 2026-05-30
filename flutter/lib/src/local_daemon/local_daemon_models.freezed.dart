// GENERATED CODE - DO NOT MODIFY BY HAND
// coverage:ignore-file
// ignore_for_file: type=lint
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark, unreachable_switch_case

part of 'local_daemon_models.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

// dart format off
T _$identity<T>(T value) => value;

/// @nodoc
mixin _$CortadoLocalDaemonAvailability {
  String get installUrl;
  String? get message;
  CortadoLocalDaemonAvailabilityState get state;

  /// Create a copy of CortadoLocalDaemonAvailability
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @pragma('vm:prefer-inline')
  $CortadoLocalDaemonAvailabilityCopyWith<CortadoLocalDaemonAvailability>
      get copyWith => _$CortadoLocalDaemonAvailabilityCopyWithImpl<
              CortadoLocalDaemonAvailability>(
          this as CortadoLocalDaemonAvailability, _$identity);

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is CortadoLocalDaemonAvailability &&
            (identical(other.installUrl, installUrl) ||
                other.installUrl == installUrl) &&
            (identical(other.message, message) || other.message == message) &&
            (identical(other.state, state) || other.state == state));
  }

  @override
  int get hashCode => Object.hash(runtimeType, installUrl, message, state);

  @override
  String toString() {
    return 'CortadoLocalDaemonAvailability(installUrl: $installUrl, message: $message, state: $state)';
  }
}

/// @nodoc
abstract mixin class $CortadoLocalDaemonAvailabilityCopyWith<$Res> {
  factory $CortadoLocalDaemonAvailabilityCopyWith(
          CortadoLocalDaemonAvailability value,
          $Res Function(CortadoLocalDaemonAvailability) _then) =
      _$CortadoLocalDaemonAvailabilityCopyWithImpl;
  @useResult
  $Res call(
      {String installUrl,
      String? message,
      CortadoLocalDaemonAvailabilityState state});
}

/// @nodoc
class _$CortadoLocalDaemonAvailabilityCopyWithImpl<$Res>
    implements $CortadoLocalDaemonAvailabilityCopyWith<$Res> {
  _$CortadoLocalDaemonAvailabilityCopyWithImpl(this._self, this._then);

  final CortadoLocalDaemonAvailability _self;
  final $Res Function(CortadoLocalDaemonAvailability) _then;

  /// Create a copy of CortadoLocalDaemonAvailability
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? installUrl = null,
    Object? message = freezed,
    Object? state = null,
  }) {
    return _then(_self.copyWith(
      installUrl: null == installUrl
          ? _self.installUrl
          : installUrl // ignore: cast_nullable_to_non_nullable
              as String,
      message: freezed == message
          ? _self.message
          : message // ignore: cast_nullable_to_non_nullable
              as String?,
      state: null == state
          ? _self.state
          : state // ignore: cast_nullable_to_non_nullable
              as CortadoLocalDaemonAvailabilityState,
    ));
  }
}

/// Adds pattern-matching-related methods to [CortadoLocalDaemonAvailability].
extension CortadoLocalDaemonAvailabilityPatterns
    on CortadoLocalDaemonAvailability {
  /// A variant of `map` that fallback to returning `orElse`.
  ///
  /// It is equivalent to doing:
  /// ```dart
  /// switch (sealedClass) {
  ///   case final Subclass value:
  ///     return ...;
  ///   case _:
  ///     return orElse();
  /// }
  /// ```

  @optionalTypeArgs
  TResult maybeMap<TResult extends Object?>(
    TResult Function(_CortadoLocalDaemonAvailability value)? $default, {
    required TResult orElse(),
  }) {
    final _that = this;
    switch (_that) {
      case _CortadoLocalDaemonAvailability() when $default != null:
        return $default(_that);
      case _:
        return orElse();
    }
  }

  /// A `switch`-like method, using callbacks.
  ///
  /// Callbacks receives the raw object, upcasted.
  /// It is equivalent to doing:
  /// ```dart
  /// switch (sealedClass) {
  ///   case final Subclass value:
  ///     return ...;
  ///   case final Subclass2 value:
  ///     return ...;
  /// }
  /// ```

  @optionalTypeArgs
  TResult map<TResult extends Object?>(
    TResult Function(_CortadoLocalDaemonAvailability value) $default,
  ) {
    final _that = this;
    switch (_that) {
      case _CortadoLocalDaemonAvailability():
        return $default(_that);
      case _:
        throw StateError('Unexpected subclass');
    }
  }

  /// A variant of `map` that fallback to returning `null`.
  ///
  /// It is equivalent to doing:
  /// ```dart
  /// switch (sealedClass) {
  ///   case final Subclass value:
  ///     return ...;
  ///   case _:
  ///     return null;
  /// }
  /// ```

  @optionalTypeArgs
  TResult? mapOrNull<TResult extends Object?>(
    TResult? Function(_CortadoLocalDaemonAvailability value)? $default,
  ) {
    final _that = this;
    switch (_that) {
      case _CortadoLocalDaemonAvailability() when $default != null:
        return $default(_that);
      case _:
        return null;
    }
  }

  /// A variant of `when` that fallback to an `orElse` callback.
  ///
  /// It is equivalent to doing:
  /// ```dart
  /// switch (sealedClass) {
  ///   case Subclass(:final field):
  ///     return ...;
  ///   case _:
  ///     return orElse();
  /// }
  /// ```

  @optionalTypeArgs
  TResult maybeWhen<TResult extends Object?>(
    TResult Function(String installUrl, String? message,
            CortadoLocalDaemonAvailabilityState state)?
        $default, {
    required TResult orElse(),
  }) {
    final _that = this;
    switch (_that) {
      case _CortadoLocalDaemonAvailability() when $default != null:
        return $default(_that.installUrl, _that.message, _that.state);
      case _:
        return orElse();
    }
  }

  /// A `switch`-like method, using callbacks.
  ///
  /// As opposed to `map`, this offers destructuring.
  /// It is equivalent to doing:
  /// ```dart
  /// switch (sealedClass) {
  ///   case Subclass(:final field):
  ///     return ...;
  ///   case Subclass2(:final field2):
  ///     return ...;
  /// }
  /// ```

  @optionalTypeArgs
  TResult when<TResult extends Object?>(
    TResult Function(String installUrl, String? message,
            CortadoLocalDaemonAvailabilityState state)
        $default,
  ) {
    final _that = this;
    switch (_that) {
      case _CortadoLocalDaemonAvailability():
        return $default(_that.installUrl, _that.message, _that.state);
      case _:
        throw StateError('Unexpected subclass');
    }
  }

  /// A variant of `when` that fallback to returning `null`
  ///
  /// It is equivalent to doing:
  /// ```dart
  /// switch (sealedClass) {
  ///   case Subclass(:final field):
  ///     return ...;
  ///   case _:
  ///     return null;
  /// }
  /// ```

  @optionalTypeArgs
  TResult? whenOrNull<TResult extends Object?>(
    TResult? Function(String installUrl, String? message,
            CortadoLocalDaemonAvailabilityState state)?
        $default,
  ) {
    final _that = this;
    switch (_that) {
      case _CortadoLocalDaemonAvailability() when $default != null:
        return $default(_that.installUrl, _that.message, _that.state);
      case _:
        return null;
    }
  }
}

/// @nodoc

class _CortadoLocalDaemonAvailability extends CortadoLocalDaemonAvailability {
  const _CortadoLocalDaemonAvailability(
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

  /// Create a copy of CortadoLocalDaemonAvailability
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  @pragma('vm:prefer-inline')
  _$CortadoLocalDaemonAvailabilityCopyWith<_CortadoLocalDaemonAvailability>
      get copyWith => __$CortadoLocalDaemonAvailabilityCopyWithImpl<
          _CortadoLocalDaemonAvailability>(this, _$identity);

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _CortadoLocalDaemonAvailability &&
            (identical(other.installUrl, installUrl) ||
                other.installUrl == installUrl) &&
            (identical(other.message, message) || other.message == message) &&
            (identical(other.state, state) || other.state == state));
  }

  @override
  int get hashCode => Object.hash(runtimeType, installUrl, message, state);

  @override
  String toString() {
    return 'CortadoLocalDaemonAvailability(installUrl: $installUrl, message: $message, state: $state)';
  }
}

/// @nodoc
abstract mixin class _$CortadoLocalDaemonAvailabilityCopyWith<$Res>
    implements $CortadoLocalDaemonAvailabilityCopyWith<$Res> {
  factory _$CortadoLocalDaemonAvailabilityCopyWith(
          _CortadoLocalDaemonAvailability value,
          $Res Function(_CortadoLocalDaemonAvailability) _then) =
      __$CortadoLocalDaemonAvailabilityCopyWithImpl;
  @override
  @useResult
  $Res call(
      {String installUrl,
      String? message,
      CortadoLocalDaemonAvailabilityState state});
}

/// @nodoc
class __$CortadoLocalDaemonAvailabilityCopyWithImpl<$Res>
    implements _$CortadoLocalDaemonAvailabilityCopyWith<$Res> {
  __$CortadoLocalDaemonAvailabilityCopyWithImpl(this._self, this._then);

  final _CortadoLocalDaemonAvailability _self;
  final $Res Function(_CortadoLocalDaemonAvailability) _then;

  /// Create a copy of CortadoLocalDaemonAvailability
  /// with the given fields replaced by the non-null parameter values.
  @override
  @pragma('vm:prefer-inline')
  $Res call({
    Object? installUrl = null,
    Object? message = freezed,
    Object? state = null,
  }) {
    return _then(_CortadoLocalDaemonAvailability(
      installUrl: null == installUrl
          ? _self.installUrl
          : installUrl // ignore: cast_nullable_to_non_nullable
              as String,
      message: freezed == message
          ? _self.message
          : message // ignore: cast_nullable_to_non_nullable
              as String?,
      state: null == state
          ? _self.state
          : state // ignore: cast_nullable_to_non_nullable
              as CortadoLocalDaemonAvailabilityState,
    ));
  }
}

/// @nodoc
mixin _$CortadoLocalDaemonSyncStatus {
  String get localPath;
  String? get message;
  CortadoLocalDaemonSyncState get state;
  String get workspaceId;
  String get workspacePath;

  /// Create a copy of CortadoLocalDaemonSyncStatus
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @pragma('vm:prefer-inline')
  $CortadoLocalDaemonSyncStatusCopyWith<CortadoLocalDaemonSyncStatus>
      get copyWith => _$CortadoLocalDaemonSyncStatusCopyWithImpl<
              CortadoLocalDaemonSyncStatus>(
          this as CortadoLocalDaemonSyncStatus, _$identity);

  /// Serializes this CortadoLocalDaemonSyncStatus to a JSON map.
  Map<String, dynamic> toJson();

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is CortadoLocalDaemonSyncStatus &&
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

  @override
  String toString() {
    return 'CortadoLocalDaemonSyncStatus(localPath: $localPath, message: $message, state: $state, workspaceId: $workspaceId, workspacePath: $workspacePath)';
  }
}

/// @nodoc
abstract mixin class $CortadoLocalDaemonSyncStatusCopyWith<$Res> {
  factory $CortadoLocalDaemonSyncStatusCopyWith(
          CortadoLocalDaemonSyncStatus value,
          $Res Function(CortadoLocalDaemonSyncStatus) _then) =
      _$CortadoLocalDaemonSyncStatusCopyWithImpl;
  @useResult
  $Res call(
      {String localPath,
      String? message,
      CortadoLocalDaemonSyncState state,
      String workspaceId,
      String workspacePath});
}

/// @nodoc
class _$CortadoLocalDaemonSyncStatusCopyWithImpl<$Res>
    implements $CortadoLocalDaemonSyncStatusCopyWith<$Res> {
  _$CortadoLocalDaemonSyncStatusCopyWithImpl(this._self, this._then);

  final CortadoLocalDaemonSyncStatus _self;
  final $Res Function(CortadoLocalDaemonSyncStatus) _then;

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
    return _then(_self.copyWith(
      localPath: null == localPath
          ? _self.localPath
          : localPath // ignore: cast_nullable_to_non_nullable
              as String,
      message: freezed == message
          ? _self.message
          : message // ignore: cast_nullable_to_non_nullable
              as String?,
      state: null == state
          ? _self.state
          : state // ignore: cast_nullable_to_non_nullable
              as CortadoLocalDaemonSyncState,
      workspaceId: null == workspaceId
          ? _self.workspaceId
          : workspaceId // ignore: cast_nullable_to_non_nullable
              as String,
      workspacePath: null == workspacePath
          ? _self.workspacePath
          : workspacePath // ignore: cast_nullable_to_non_nullable
              as String,
    ));
  }
}

/// Adds pattern-matching-related methods to [CortadoLocalDaemonSyncStatus].
extension CortadoLocalDaemonSyncStatusPatterns on CortadoLocalDaemonSyncStatus {
  /// A variant of `map` that fallback to returning `orElse`.
  ///
  /// It is equivalent to doing:
  /// ```dart
  /// switch (sealedClass) {
  ///   case final Subclass value:
  ///     return ...;
  ///   case _:
  ///     return orElse();
  /// }
  /// ```

  @optionalTypeArgs
  TResult maybeMap<TResult extends Object?>(
    TResult Function(_CortadoLocalDaemonSyncStatus value)? $default, {
    required TResult orElse(),
  }) {
    final _that = this;
    switch (_that) {
      case _CortadoLocalDaemonSyncStatus() when $default != null:
        return $default(_that);
      case _:
        return orElse();
    }
  }

  /// A `switch`-like method, using callbacks.
  ///
  /// Callbacks receives the raw object, upcasted.
  /// It is equivalent to doing:
  /// ```dart
  /// switch (sealedClass) {
  ///   case final Subclass value:
  ///     return ...;
  ///   case final Subclass2 value:
  ///     return ...;
  /// }
  /// ```

  @optionalTypeArgs
  TResult map<TResult extends Object?>(
    TResult Function(_CortadoLocalDaemonSyncStatus value) $default,
  ) {
    final _that = this;
    switch (_that) {
      case _CortadoLocalDaemonSyncStatus():
        return $default(_that);
      case _:
        throw StateError('Unexpected subclass');
    }
  }

  /// A variant of `map` that fallback to returning `null`.
  ///
  /// It is equivalent to doing:
  /// ```dart
  /// switch (sealedClass) {
  ///   case final Subclass value:
  ///     return ...;
  ///   case _:
  ///     return null;
  /// }
  /// ```

  @optionalTypeArgs
  TResult? mapOrNull<TResult extends Object?>(
    TResult? Function(_CortadoLocalDaemonSyncStatus value)? $default,
  ) {
    final _that = this;
    switch (_that) {
      case _CortadoLocalDaemonSyncStatus() when $default != null:
        return $default(_that);
      case _:
        return null;
    }
  }

  /// A variant of `when` that fallback to an `orElse` callback.
  ///
  /// It is equivalent to doing:
  /// ```dart
  /// switch (sealedClass) {
  ///   case Subclass(:final field):
  ///     return ...;
  ///   case _:
  ///     return orElse();
  /// }
  /// ```

  @optionalTypeArgs
  TResult maybeWhen<TResult extends Object?>(
    TResult Function(
            String localPath,
            String? message,
            CortadoLocalDaemonSyncState state,
            String workspaceId,
            String workspacePath)?
        $default, {
    required TResult orElse(),
  }) {
    final _that = this;
    switch (_that) {
      case _CortadoLocalDaemonSyncStatus() when $default != null:
        return $default(_that.localPath, _that.message, _that.state,
            _that.workspaceId, _that.workspacePath);
      case _:
        return orElse();
    }
  }

  /// A `switch`-like method, using callbacks.
  ///
  /// As opposed to `map`, this offers destructuring.
  /// It is equivalent to doing:
  /// ```dart
  /// switch (sealedClass) {
  ///   case Subclass(:final field):
  ///     return ...;
  ///   case Subclass2(:final field2):
  ///     return ...;
  /// }
  /// ```

  @optionalTypeArgs
  TResult when<TResult extends Object?>(
    TResult Function(
            String localPath,
            String? message,
            CortadoLocalDaemonSyncState state,
            String workspaceId,
            String workspacePath)
        $default,
  ) {
    final _that = this;
    switch (_that) {
      case _CortadoLocalDaemonSyncStatus():
        return $default(_that.localPath, _that.message, _that.state,
            _that.workspaceId, _that.workspacePath);
      case _:
        throw StateError('Unexpected subclass');
    }
  }

  /// A variant of `when` that fallback to returning `null`
  ///
  /// It is equivalent to doing:
  /// ```dart
  /// switch (sealedClass) {
  ///   case Subclass(:final field):
  ///     return ...;
  ///   case _:
  ///     return null;
  /// }
  /// ```

  @optionalTypeArgs
  TResult? whenOrNull<TResult extends Object?>(
    TResult? Function(
            String localPath,
            String? message,
            CortadoLocalDaemonSyncState state,
            String workspaceId,
            String workspacePath)?
        $default,
  ) {
    final _that = this;
    switch (_that) {
      case _CortadoLocalDaemonSyncStatus() when $default != null:
        return $default(_that.localPath, _that.message, _that.state,
            _that.workspaceId, _that.workspacePath);
      case _:
        return null;
    }
  }
}

/// @nodoc
@JsonSerializable()
class _CortadoLocalDaemonSyncStatus extends CortadoLocalDaemonSyncStatus {
  const _CortadoLocalDaemonSyncStatus(
      {required this.localPath,
      this.message,
      required this.state,
      required this.workspaceId,
      this.workspacePath = '/'})
      : super._();
  factory _CortadoLocalDaemonSyncStatus.fromJson(Map<String, dynamic> json) =>
      _$CortadoLocalDaemonSyncStatusFromJson(json);

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

  /// Create a copy of CortadoLocalDaemonSyncStatus
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  @pragma('vm:prefer-inline')
  _$CortadoLocalDaemonSyncStatusCopyWith<_CortadoLocalDaemonSyncStatus>
      get copyWith => __$CortadoLocalDaemonSyncStatusCopyWithImpl<
          _CortadoLocalDaemonSyncStatus>(this, _$identity);

  @override
  Map<String, dynamic> toJson() {
    return _$CortadoLocalDaemonSyncStatusToJson(
      this,
    );
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _CortadoLocalDaemonSyncStatus &&
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

  @override
  String toString() {
    return 'CortadoLocalDaemonSyncStatus(localPath: $localPath, message: $message, state: $state, workspaceId: $workspaceId, workspacePath: $workspacePath)';
  }
}

/// @nodoc
abstract mixin class _$CortadoLocalDaemonSyncStatusCopyWith<$Res>
    implements $CortadoLocalDaemonSyncStatusCopyWith<$Res> {
  factory _$CortadoLocalDaemonSyncStatusCopyWith(
          _CortadoLocalDaemonSyncStatus value,
          $Res Function(_CortadoLocalDaemonSyncStatus) _then) =
      __$CortadoLocalDaemonSyncStatusCopyWithImpl;
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
class __$CortadoLocalDaemonSyncStatusCopyWithImpl<$Res>
    implements _$CortadoLocalDaemonSyncStatusCopyWith<$Res> {
  __$CortadoLocalDaemonSyncStatusCopyWithImpl(this._self, this._then);

  final _CortadoLocalDaemonSyncStatus _self;
  final $Res Function(_CortadoLocalDaemonSyncStatus) _then;

  /// Create a copy of CortadoLocalDaemonSyncStatus
  /// with the given fields replaced by the non-null parameter values.
  @override
  @pragma('vm:prefer-inline')
  $Res call({
    Object? localPath = null,
    Object? message = freezed,
    Object? state = null,
    Object? workspaceId = null,
    Object? workspacePath = null,
  }) {
    return _then(_CortadoLocalDaemonSyncStatus(
      localPath: null == localPath
          ? _self.localPath
          : localPath // ignore: cast_nullable_to_non_nullable
              as String,
      message: freezed == message
          ? _self.message
          : message // ignore: cast_nullable_to_non_nullable
              as String?,
      state: null == state
          ? _self.state
          : state // ignore: cast_nullable_to_non_nullable
              as CortadoLocalDaemonSyncState,
      workspaceId: null == workspaceId
          ? _self.workspaceId
          : workspaceId // ignore: cast_nullable_to_non_nullable
              as String,
      workspacePath: null == workspacePath
          ? _self.workspacePath
          : workspacePath // ignore: cast_nullable_to_non_nullable
              as String,
    ));
  }
}

/// @nodoc
mixin _$CortadoLocalDaemonConflict {
  int get lastSyncedClock;
  int get localClock;
  @JsonKey(name: 'path')
  String get localPath;
  String get reason;
  int get remoteClock;
  String? get workspaceId;
  String? get workspacePath;

  /// Create a copy of CortadoLocalDaemonConflict
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @pragma('vm:prefer-inline')
  $CortadoLocalDaemonConflictCopyWith<CortadoLocalDaemonConflict>
      get copyWith =>
          _$CortadoLocalDaemonConflictCopyWithImpl<CortadoLocalDaemonConflict>(
              this as CortadoLocalDaemonConflict, _$identity);

  /// Serializes this CortadoLocalDaemonConflict to a JSON map.
  Map<String, dynamic> toJson();

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is CortadoLocalDaemonConflict &&
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

  @override
  String toString() {
    return 'CortadoLocalDaemonConflict(lastSyncedClock: $lastSyncedClock, localClock: $localClock, localPath: $localPath, reason: $reason, remoteClock: $remoteClock, workspaceId: $workspaceId, workspacePath: $workspacePath)';
  }
}

/// @nodoc
abstract mixin class $CortadoLocalDaemonConflictCopyWith<$Res> {
  factory $CortadoLocalDaemonConflictCopyWith(CortadoLocalDaemonConflict value,
          $Res Function(CortadoLocalDaemonConflict) _then) =
      _$CortadoLocalDaemonConflictCopyWithImpl;
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
class _$CortadoLocalDaemonConflictCopyWithImpl<$Res>
    implements $CortadoLocalDaemonConflictCopyWith<$Res> {
  _$CortadoLocalDaemonConflictCopyWithImpl(this._self, this._then);

  final CortadoLocalDaemonConflict _self;
  final $Res Function(CortadoLocalDaemonConflict) _then;

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
    return _then(_self.copyWith(
      lastSyncedClock: null == lastSyncedClock
          ? _self.lastSyncedClock
          : lastSyncedClock // ignore: cast_nullable_to_non_nullable
              as int,
      localClock: null == localClock
          ? _self.localClock
          : localClock // ignore: cast_nullable_to_non_nullable
              as int,
      localPath: null == localPath
          ? _self.localPath
          : localPath // ignore: cast_nullable_to_non_nullable
              as String,
      reason: null == reason
          ? _self.reason
          : reason // ignore: cast_nullable_to_non_nullable
              as String,
      remoteClock: null == remoteClock
          ? _self.remoteClock
          : remoteClock // ignore: cast_nullable_to_non_nullable
              as int,
      workspaceId: freezed == workspaceId
          ? _self.workspaceId
          : workspaceId // ignore: cast_nullable_to_non_nullable
              as String?,
      workspacePath: freezed == workspacePath
          ? _self.workspacePath
          : workspacePath // ignore: cast_nullable_to_non_nullable
              as String?,
    ));
  }
}

/// Adds pattern-matching-related methods to [CortadoLocalDaemonConflict].
extension CortadoLocalDaemonConflictPatterns on CortadoLocalDaemonConflict {
  /// A variant of `map` that fallback to returning `orElse`.
  ///
  /// It is equivalent to doing:
  /// ```dart
  /// switch (sealedClass) {
  ///   case final Subclass value:
  ///     return ...;
  ///   case _:
  ///     return orElse();
  /// }
  /// ```

  @optionalTypeArgs
  TResult maybeMap<TResult extends Object?>(
    TResult Function(_CortadoLocalDaemonConflict value)? $default, {
    required TResult orElse(),
  }) {
    final _that = this;
    switch (_that) {
      case _CortadoLocalDaemonConflict() when $default != null:
        return $default(_that);
      case _:
        return orElse();
    }
  }

  /// A `switch`-like method, using callbacks.
  ///
  /// Callbacks receives the raw object, upcasted.
  /// It is equivalent to doing:
  /// ```dart
  /// switch (sealedClass) {
  ///   case final Subclass value:
  ///     return ...;
  ///   case final Subclass2 value:
  ///     return ...;
  /// }
  /// ```

  @optionalTypeArgs
  TResult map<TResult extends Object?>(
    TResult Function(_CortadoLocalDaemonConflict value) $default,
  ) {
    final _that = this;
    switch (_that) {
      case _CortadoLocalDaemonConflict():
        return $default(_that);
      case _:
        throw StateError('Unexpected subclass');
    }
  }

  /// A variant of `map` that fallback to returning `null`.
  ///
  /// It is equivalent to doing:
  /// ```dart
  /// switch (sealedClass) {
  ///   case final Subclass value:
  ///     return ...;
  ///   case _:
  ///     return null;
  /// }
  /// ```

  @optionalTypeArgs
  TResult? mapOrNull<TResult extends Object?>(
    TResult? Function(_CortadoLocalDaemonConflict value)? $default,
  ) {
    final _that = this;
    switch (_that) {
      case _CortadoLocalDaemonConflict() when $default != null:
        return $default(_that);
      case _:
        return null;
    }
  }

  /// A variant of `when` that fallback to an `orElse` callback.
  ///
  /// It is equivalent to doing:
  /// ```dart
  /// switch (sealedClass) {
  ///   case Subclass(:final field):
  ///     return ...;
  ///   case _:
  ///     return orElse();
  /// }
  /// ```

  @optionalTypeArgs
  TResult maybeWhen<TResult extends Object?>(
    TResult Function(
            int lastSyncedClock,
            int localClock,
            @JsonKey(name: 'path') String localPath,
            String reason,
            int remoteClock,
            String? workspaceId,
            String? workspacePath)?
        $default, {
    required TResult orElse(),
  }) {
    final _that = this;
    switch (_that) {
      case _CortadoLocalDaemonConflict() when $default != null:
        return $default(
            _that.lastSyncedClock,
            _that.localClock,
            _that.localPath,
            _that.reason,
            _that.remoteClock,
            _that.workspaceId,
            _that.workspacePath);
      case _:
        return orElse();
    }
  }

  /// A `switch`-like method, using callbacks.
  ///
  /// As opposed to `map`, this offers destructuring.
  /// It is equivalent to doing:
  /// ```dart
  /// switch (sealedClass) {
  ///   case Subclass(:final field):
  ///     return ...;
  ///   case Subclass2(:final field2):
  ///     return ...;
  /// }
  /// ```

  @optionalTypeArgs
  TResult when<TResult extends Object?>(
    TResult Function(
            int lastSyncedClock,
            int localClock,
            @JsonKey(name: 'path') String localPath,
            String reason,
            int remoteClock,
            String? workspaceId,
            String? workspacePath)
        $default,
  ) {
    final _that = this;
    switch (_that) {
      case _CortadoLocalDaemonConflict():
        return $default(
            _that.lastSyncedClock,
            _that.localClock,
            _that.localPath,
            _that.reason,
            _that.remoteClock,
            _that.workspaceId,
            _that.workspacePath);
      case _:
        throw StateError('Unexpected subclass');
    }
  }

  /// A variant of `when` that fallback to returning `null`
  ///
  /// It is equivalent to doing:
  /// ```dart
  /// switch (sealedClass) {
  ///   case Subclass(:final field):
  ///     return ...;
  ///   case _:
  ///     return null;
  /// }
  /// ```

  @optionalTypeArgs
  TResult? whenOrNull<TResult extends Object?>(
    TResult? Function(
            int lastSyncedClock,
            int localClock,
            @JsonKey(name: 'path') String localPath,
            String reason,
            int remoteClock,
            String? workspaceId,
            String? workspacePath)?
        $default,
  ) {
    final _that = this;
    switch (_that) {
      case _CortadoLocalDaemonConflict() when $default != null:
        return $default(
            _that.lastSyncedClock,
            _that.localClock,
            _that.localPath,
            _that.reason,
            _that.remoteClock,
            _that.workspaceId,
            _that.workspacePath);
      case _:
        return null;
    }
  }
}

/// @nodoc
@JsonSerializable()
class _CortadoLocalDaemonConflict extends CortadoLocalDaemonConflict {
  const _CortadoLocalDaemonConflict(
      {required this.lastSyncedClock,
      required this.localClock,
      @JsonKey(name: 'path') required this.localPath,
      required this.reason,
      required this.remoteClock,
      this.workspaceId,
      this.workspacePath})
      : super._();
  factory _CortadoLocalDaemonConflict.fromJson(Map<String, dynamic> json) =>
      _$CortadoLocalDaemonConflictFromJson(json);

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

  /// Create a copy of CortadoLocalDaemonConflict
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  @pragma('vm:prefer-inline')
  _$CortadoLocalDaemonConflictCopyWith<_CortadoLocalDaemonConflict>
      get copyWith => __$CortadoLocalDaemonConflictCopyWithImpl<
          _CortadoLocalDaemonConflict>(this, _$identity);

  @override
  Map<String, dynamic> toJson() {
    return _$CortadoLocalDaemonConflictToJson(
      this,
    );
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _CortadoLocalDaemonConflict &&
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

  @override
  String toString() {
    return 'CortadoLocalDaemonConflict(lastSyncedClock: $lastSyncedClock, localClock: $localClock, localPath: $localPath, reason: $reason, remoteClock: $remoteClock, workspaceId: $workspaceId, workspacePath: $workspacePath)';
  }
}

/// @nodoc
abstract mixin class _$CortadoLocalDaemonConflictCopyWith<$Res>
    implements $CortadoLocalDaemonConflictCopyWith<$Res> {
  factory _$CortadoLocalDaemonConflictCopyWith(
          _CortadoLocalDaemonConflict value,
          $Res Function(_CortadoLocalDaemonConflict) _then) =
      __$CortadoLocalDaemonConflictCopyWithImpl;
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
class __$CortadoLocalDaemonConflictCopyWithImpl<$Res>
    implements _$CortadoLocalDaemonConflictCopyWith<$Res> {
  __$CortadoLocalDaemonConflictCopyWithImpl(this._self, this._then);

  final _CortadoLocalDaemonConflict _self;
  final $Res Function(_CortadoLocalDaemonConflict) _then;

  /// Create a copy of CortadoLocalDaemonConflict
  /// with the given fields replaced by the non-null parameter values.
  @override
  @pragma('vm:prefer-inline')
  $Res call({
    Object? lastSyncedClock = null,
    Object? localClock = null,
    Object? localPath = null,
    Object? reason = null,
    Object? remoteClock = null,
    Object? workspaceId = freezed,
    Object? workspacePath = freezed,
  }) {
    return _then(_CortadoLocalDaemonConflict(
      lastSyncedClock: null == lastSyncedClock
          ? _self.lastSyncedClock
          : lastSyncedClock // ignore: cast_nullable_to_non_nullable
              as int,
      localClock: null == localClock
          ? _self.localClock
          : localClock // ignore: cast_nullable_to_non_nullable
              as int,
      localPath: null == localPath
          ? _self.localPath
          : localPath // ignore: cast_nullable_to_non_nullable
              as String,
      reason: null == reason
          ? _self.reason
          : reason // ignore: cast_nullable_to_non_nullable
              as String,
      remoteClock: null == remoteClock
          ? _self.remoteClock
          : remoteClock // ignore: cast_nullable_to_non_nullable
              as int,
      workspaceId: freezed == workspaceId
          ? _self.workspaceId
          : workspaceId // ignore: cast_nullable_to_non_nullable
              as String?,
      workspacePath: freezed == workspacePath
          ? _self.workspacePath
          : workspacePath // ignore: cast_nullable_to_non_nullable
              as String?,
    ));
  }
}

// dart format on
