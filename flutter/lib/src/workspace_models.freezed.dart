// GENERATED CODE - DO NOT MODIFY BY HAND
// coverage:ignore-file
// ignore_for_file: type=lint
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark, unreachable_switch_case

part of 'workspace_models.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

// dart format off
T _$identity<T>(T value) => value;

/// @nodoc
mixin _$WorkspaceResources {
  double get cpu;
  @JsonKey(name: 'memoryGb')
  double get memoryGb;
  @JsonKey(name: 'storageGb')
  double get storageGb;

  /// Create a copy of WorkspaceResources
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @pragma('vm:prefer-inline')
  $WorkspaceResourcesCopyWith<WorkspaceResources> get copyWith =>
      _$WorkspaceResourcesCopyWithImpl<WorkspaceResources>(
          this as WorkspaceResources, _$identity);

  /// Serializes this WorkspaceResources to a JSON map.
  Map<String, dynamic> toJson();

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is WorkspaceResources &&
            (identical(other.cpu, cpu) || other.cpu == cpu) &&
            (identical(other.memoryGb, memoryGb) ||
                other.memoryGb == memoryGb) &&
            (identical(other.storageGb, storageGb) ||
                other.storageGb == storageGb));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(runtimeType, cpu, memoryGb, storageGb);

  @override
  String toString() {
    return 'WorkspaceResources(cpu: $cpu, memoryGb: $memoryGb, storageGb: $storageGb)';
  }
}

/// @nodoc
abstract mixin class $WorkspaceResourcesCopyWith<$Res> {
  factory $WorkspaceResourcesCopyWith(
          WorkspaceResources value, $Res Function(WorkspaceResources) _then) =
      _$WorkspaceResourcesCopyWithImpl;
  @useResult
  $Res call(
      {double cpu,
      @JsonKey(name: 'memoryGb') double memoryGb,
      @JsonKey(name: 'storageGb') double storageGb});
}

/// @nodoc
class _$WorkspaceResourcesCopyWithImpl<$Res>
    implements $WorkspaceResourcesCopyWith<$Res> {
  _$WorkspaceResourcesCopyWithImpl(this._self, this._then);

  final WorkspaceResources _self;
  final $Res Function(WorkspaceResources) _then;

  /// Create a copy of WorkspaceResources
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? cpu = null,
    Object? memoryGb = null,
    Object? storageGb = null,
  }) {
    return _then(_self.copyWith(
      cpu: null == cpu
          ? _self.cpu
          : cpu // ignore: cast_nullable_to_non_nullable
              as double,
      memoryGb: null == memoryGb
          ? _self.memoryGb
          : memoryGb // ignore: cast_nullable_to_non_nullable
              as double,
      storageGb: null == storageGb
          ? _self.storageGb
          : storageGb // ignore: cast_nullable_to_non_nullable
              as double,
    ));
  }
}

/// Adds pattern-matching-related methods to [WorkspaceResources].
extension WorkspaceResourcesPatterns on WorkspaceResources {
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
    TResult Function(_WorkspaceResources value)? $default, {
    required TResult orElse(),
  }) {
    final _that = this;
    switch (_that) {
      case _WorkspaceResources() when $default != null:
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
    TResult Function(_WorkspaceResources value) $default,
  ) {
    final _that = this;
    switch (_that) {
      case _WorkspaceResources():
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
    TResult? Function(_WorkspaceResources value)? $default,
  ) {
    final _that = this;
    switch (_that) {
      case _WorkspaceResources() when $default != null:
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
    TResult Function(double cpu, @JsonKey(name: 'memoryGb') double memoryGb,
            @JsonKey(name: 'storageGb') double storageGb)?
        $default, {
    required TResult orElse(),
  }) {
    final _that = this;
    switch (_that) {
      case _WorkspaceResources() when $default != null:
        return $default(_that.cpu, _that.memoryGb, _that.storageGb);
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
    TResult Function(double cpu, @JsonKey(name: 'memoryGb') double memoryGb,
            @JsonKey(name: 'storageGb') double storageGb)
        $default,
  ) {
    final _that = this;
    switch (_that) {
      case _WorkspaceResources():
        return $default(_that.cpu, _that.memoryGb, _that.storageGb);
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
    TResult? Function(double cpu, @JsonKey(name: 'memoryGb') double memoryGb,
            @JsonKey(name: 'storageGb') double storageGb)?
        $default,
  ) {
    final _that = this;
    switch (_that) {
      case _WorkspaceResources() when $default != null:
        return $default(_that.cpu, _that.memoryGb, _that.storageGb);
      case _:
        return null;
    }
  }
}

/// @nodoc
@JsonSerializable()
class _WorkspaceResources implements WorkspaceResources {
  const _WorkspaceResources(
      {required this.cpu,
      @JsonKey(name: 'memoryGb') required this.memoryGb,
      @JsonKey(name: 'storageGb') required this.storageGb});
  factory _WorkspaceResources.fromJson(Map<String, dynamic> json) =>
      _$WorkspaceResourcesFromJson(json);

  @override
  final double cpu;
  @override
  @JsonKey(name: 'memoryGb')
  final double memoryGb;
  @override
  @JsonKey(name: 'storageGb')
  final double storageGb;

  /// Create a copy of WorkspaceResources
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  @pragma('vm:prefer-inline')
  _$WorkspaceResourcesCopyWith<_WorkspaceResources> get copyWith =>
      __$WorkspaceResourcesCopyWithImpl<_WorkspaceResources>(this, _$identity);

  @override
  Map<String, dynamic> toJson() {
    return _$WorkspaceResourcesToJson(
      this,
    );
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _WorkspaceResources &&
            (identical(other.cpu, cpu) || other.cpu == cpu) &&
            (identical(other.memoryGb, memoryGb) ||
                other.memoryGb == memoryGb) &&
            (identical(other.storageGb, storageGb) ||
                other.storageGb == storageGb));
  }

  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  int get hashCode => Object.hash(runtimeType, cpu, memoryGb, storageGb);

  @override
  String toString() {
    return 'WorkspaceResources(cpu: $cpu, memoryGb: $memoryGb, storageGb: $storageGb)';
  }
}

/// @nodoc
abstract mixin class _$WorkspaceResourcesCopyWith<$Res>
    implements $WorkspaceResourcesCopyWith<$Res> {
  factory _$WorkspaceResourcesCopyWith(
          _WorkspaceResources value, $Res Function(_WorkspaceResources) _then) =
      __$WorkspaceResourcesCopyWithImpl;
  @override
  @useResult
  $Res call(
      {double cpu,
      @JsonKey(name: 'memoryGb') double memoryGb,
      @JsonKey(name: 'storageGb') double storageGb});
}

/// @nodoc
class __$WorkspaceResourcesCopyWithImpl<$Res>
    implements _$WorkspaceResourcesCopyWith<$Res> {
  __$WorkspaceResourcesCopyWithImpl(this._self, this._then);

  final _WorkspaceResources _self;
  final $Res Function(_WorkspaceResources) _then;

  /// Create a copy of WorkspaceResources
  /// with the given fields replaced by the non-null parameter values.
  @override
  @pragma('vm:prefer-inline')
  $Res call({
    Object? cpu = null,
    Object? memoryGb = null,
    Object? storageGb = null,
  }) {
    return _then(_WorkspaceResources(
      cpu: null == cpu
          ? _self.cpu
          : cpu // ignore: cast_nullable_to_non_nullable
              as double,
      memoryGb: null == memoryGb
          ? _self.memoryGb
          : memoryGb // ignore: cast_nullable_to_non_nullable
              as double,
      storageGb: null == storageGb
          ? _self.storageGb
          : storageGb // ignore: cast_nullable_to_non_nullable
              as double,
    ));
  }
}

/// @nodoc
mixin _$Workspace {
  String get id;
  String get tenantId;
  String get userId;
  String get image;
  WorkspaceResources get resources;
  WorkspaceLifecycleState get status;
  DateTime get createdAt;
  DateTime get updatedAt;
  DateTime? get lastActiveAt;

  /// Create a copy of Workspace
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @pragma('vm:prefer-inline')
  $WorkspaceCopyWith<Workspace> get copyWith =>
      _$WorkspaceCopyWithImpl<Workspace>(this as Workspace, _$identity);

  /// Serializes this Workspace to a JSON map.
  Map<String, dynamic> toJson();

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is Workspace &&
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

  @override
  String toString() {
    return 'Workspace(id: $id, tenantId: $tenantId, userId: $userId, image: $image, resources: $resources, status: $status, createdAt: $createdAt, updatedAt: $updatedAt, lastActiveAt: $lastActiveAt)';
  }
}

/// @nodoc
abstract mixin class $WorkspaceCopyWith<$Res> {
  factory $WorkspaceCopyWith(Workspace value, $Res Function(Workspace) _then) =
      _$WorkspaceCopyWithImpl;
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
class _$WorkspaceCopyWithImpl<$Res> implements $WorkspaceCopyWith<$Res> {
  _$WorkspaceCopyWithImpl(this._self, this._then);

  final Workspace _self;
  final $Res Function(Workspace) _then;

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
    return _then(_self.copyWith(
      id: null == id
          ? _self.id
          : id // ignore: cast_nullable_to_non_nullable
              as String,
      tenantId: null == tenantId
          ? _self.tenantId
          : tenantId // ignore: cast_nullable_to_non_nullable
              as String,
      userId: null == userId
          ? _self.userId
          : userId // ignore: cast_nullable_to_non_nullable
              as String,
      image: null == image
          ? _self.image
          : image // ignore: cast_nullable_to_non_nullable
              as String,
      resources: null == resources
          ? _self.resources
          : resources // ignore: cast_nullable_to_non_nullable
              as WorkspaceResources,
      status: null == status
          ? _self.status
          : status // ignore: cast_nullable_to_non_nullable
              as WorkspaceLifecycleState,
      createdAt: null == createdAt
          ? _self.createdAt
          : createdAt // ignore: cast_nullable_to_non_nullable
              as DateTime,
      updatedAt: null == updatedAt
          ? _self.updatedAt
          : updatedAt // ignore: cast_nullable_to_non_nullable
              as DateTime,
      lastActiveAt: freezed == lastActiveAt
          ? _self.lastActiveAt
          : lastActiveAt // ignore: cast_nullable_to_non_nullable
              as DateTime?,
    ));
  }

  /// Create a copy of Workspace
  /// with the given fields replaced by the non-null parameter values.
  @override
  @pragma('vm:prefer-inline')
  $WorkspaceResourcesCopyWith<$Res> get resources {
    return $WorkspaceResourcesCopyWith<$Res>(_self.resources, (value) {
      return _then(_self.copyWith(resources: value));
    });
  }
}

/// Adds pattern-matching-related methods to [Workspace].
extension WorkspacePatterns on Workspace {
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
    TResult Function(_Workspace value)? $default, {
    required TResult orElse(),
  }) {
    final _that = this;
    switch (_that) {
      case _Workspace() when $default != null:
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
    TResult Function(_Workspace value) $default,
  ) {
    final _that = this;
    switch (_that) {
      case _Workspace():
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
    TResult? Function(_Workspace value)? $default,
  ) {
    final _that = this;
    switch (_that) {
      case _Workspace() when $default != null:
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
            String id,
            String tenantId,
            String userId,
            String image,
            WorkspaceResources resources,
            WorkspaceLifecycleState status,
            DateTime createdAt,
            DateTime updatedAt,
            DateTime? lastActiveAt)?
        $default, {
    required TResult orElse(),
  }) {
    final _that = this;
    switch (_that) {
      case _Workspace() when $default != null:
        return $default(
            _that.id,
            _that.tenantId,
            _that.userId,
            _that.image,
            _that.resources,
            _that.status,
            _that.createdAt,
            _that.updatedAt,
            _that.lastActiveAt);
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
            String id,
            String tenantId,
            String userId,
            String image,
            WorkspaceResources resources,
            WorkspaceLifecycleState status,
            DateTime createdAt,
            DateTime updatedAt,
            DateTime? lastActiveAt)
        $default,
  ) {
    final _that = this;
    switch (_that) {
      case _Workspace():
        return $default(
            _that.id,
            _that.tenantId,
            _that.userId,
            _that.image,
            _that.resources,
            _that.status,
            _that.createdAt,
            _that.updatedAt,
            _that.lastActiveAt);
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
            String id,
            String tenantId,
            String userId,
            String image,
            WorkspaceResources resources,
            WorkspaceLifecycleState status,
            DateTime createdAt,
            DateTime updatedAt,
            DateTime? lastActiveAt)?
        $default,
  ) {
    final _that = this;
    switch (_that) {
      case _Workspace() when $default != null:
        return $default(
            _that.id,
            _that.tenantId,
            _that.userId,
            _that.image,
            _that.resources,
            _that.status,
            _that.createdAt,
            _that.updatedAt,
            _that.lastActiveAt);
      case _:
        return null;
    }
  }
}

/// @nodoc
@JsonSerializable()
class _Workspace extends Workspace {
  const _Workspace(
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
  factory _Workspace.fromJson(Map<String, dynamic> json) =>
      _$WorkspaceFromJson(json);

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

  /// Create a copy of Workspace
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  @pragma('vm:prefer-inline')
  _$WorkspaceCopyWith<_Workspace> get copyWith =>
      __$WorkspaceCopyWithImpl<_Workspace>(this, _$identity);

  @override
  Map<String, dynamic> toJson() {
    return _$WorkspaceToJson(
      this,
    );
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _Workspace &&
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

  @override
  String toString() {
    return 'Workspace(id: $id, tenantId: $tenantId, userId: $userId, image: $image, resources: $resources, status: $status, createdAt: $createdAt, updatedAt: $updatedAt, lastActiveAt: $lastActiveAt)';
  }
}

/// @nodoc
abstract mixin class _$WorkspaceCopyWith<$Res>
    implements $WorkspaceCopyWith<$Res> {
  factory _$WorkspaceCopyWith(
          _Workspace value, $Res Function(_Workspace) _then) =
      __$WorkspaceCopyWithImpl;
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
class __$WorkspaceCopyWithImpl<$Res> implements _$WorkspaceCopyWith<$Res> {
  __$WorkspaceCopyWithImpl(this._self, this._then);

  final _Workspace _self;
  final $Res Function(_Workspace) _then;

  /// Create a copy of Workspace
  /// with the given fields replaced by the non-null parameter values.
  @override
  @pragma('vm:prefer-inline')
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
    return _then(_Workspace(
      id: null == id
          ? _self.id
          : id // ignore: cast_nullable_to_non_nullable
              as String,
      tenantId: null == tenantId
          ? _self.tenantId
          : tenantId // ignore: cast_nullable_to_non_nullable
              as String,
      userId: null == userId
          ? _self.userId
          : userId // ignore: cast_nullable_to_non_nullable
              as String,
      image: null == image
          ? _self.image
          : image // ignore: cast_nullable_to_non_nullable
              as String,
      resources: null == resources
          ? _self.resources
          : resources // ignore: cast_nullable_to_non_nullable
              as WorkspaceResources,
      status: null == status
          ? _self.status
          : status // ignore: cast_nullable_to_non_nullable
              as WorkspaceLifecycleState,
      createdAt: null == createdAt
          ? _self.createdAt
          : createdAt // ignore: cast_nullable_to_non_nullable
              as DateTime,
      updatedAt: null == updatedAt
          ? _self.updatedAt
          : updatedAt // ignore: cast_nullable_to_non_nullable
              as DateTime,
      lastActiveAt: freezed == lastActiveAt
          ? _self.lastActiveAt
          : lastActiveAt // ignore: cast_nullable_to_non_nullable
              as DateTime?,
    ));
  }

  /// Create a copy of Workspace
  /// with the given fields replaced by the non-null parameter values.
  @override
  @pragma('vm:prefer-inline')
  $WorkspaceResourcesCopyWith<$Res> get resources {
    return $WorkspaceResourcesCopyWith<$Res>(_self.resources, (value) {
      return _then(_self.copyWith(resources: value));
    });
  }
}

/// @nodoc
mixin _$WorkspaceStatus {
  String get workspaceId;
  WorkspaceLifecycleState get status;
  DateTime get updatedAt;
  DateTime? get lastActiveAt;

  /// Create a copy of WorkspaceStatus
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @pragma('vm:prefer-inline')
  $WorkspaceStatusCopyWith<WorkspaceStatus> get copyWith =>
      _$WorkspaceStatusCopyWithImpl<WorkspaceStatus>(
          this as WorkspaceStatus, _$identity);

  /// Serializes this WorkspaceStatus to a JSON map.
  Map<String, dynamic> toJson();

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is WorkspaceStatus &&
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

  @override
  String toString() {
    return 'WorkspaceStatus(workspaceId: $workspaceId, status: $status, updatedAt: $updatedAt, lastActiveAt: $lastActiveAt)';
  }
}

/// @nodoc
abstract mixin class $WorkspaceStatusCopyWith<$Res> {
  factory $WorkspaceStatusCopyWith(
          WorkspaceStatus value, $Res Function(WorkspaceStatus) _then) =
      _$WorkspaceStatusCopyWithImpl;
  @useResult
  $Res call(
      {String workspaceId,
      WorkspaceLifecycleState status,
      DateTime updatedAt,
      DateTime? lastActiveAt});
}

/// @nodoc
class _$WorkspaceStatusCopyWithImpl<$Res>
    implements $WorkspaceStatusCopyWith<$Res> {
  _$WorkspaceStatusCopyWithImpl(this._self, this._then);

  final WorkspaceStatus _self;
  final $Res Function(WorkspaceStatus) _then;

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
    return _then(_self.copyWith(
      workspaceId: null == workspaceId
          ? _self.workspaceId
          : workspaceId // ignore: cast_nullable_to_non_nullable
              as String,
      status: null == status
          ? _self.status
          : status // ignore: cast_nullable_to_non_nullable
              as WorkspaceLifecycleState,
      updatedAt: null == updatedAt
          ? _self.updatedAt
          : updatedAt // ignore: cast_nullable_to_non_nullable
              as DateTime,
      lastActiveAt: freezed == lastActiveAt
          ? _self.lastActiveAt
          : lastActiveAt // ignore: cast_nullable_to_non_nullable
              as DateTime?,
    ));
  }
}

/// Adds pattern-matching-related methods to [WorkspaceStatus].
extension WorkspaceStatusPatterns on WorkspaceStatus {
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
    TResult Function(_WorkspaceStatus value)? $default, {
    required TResult orElse(),
  }) {
    final _that = this;
    switch (_that) {
      case _WorkspaceStatus() when $default != null:
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
    TResult Function(_WorkspaceStatus value) $default,
  ) {
    final _that = this;
    switch (_that) {
      case _WorkspaceStatus():
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
    TResult? Function(_WorkspaceStatus value)? $default,
  ) {
    final _that = this;
    switch (_that) {
      case _WorkspaceStatus() when $default != null:
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
    TResult Function(String workspaceId, WorkspaceLifecycleState status,
            DateTime updatedAt, DateTime? lastActiveAt)?
        $default, {
    required TResult orElse(),
  }) {
    final _that = this;
    switch (_that) {
      case _WorkspaceStatus() when $default != null:
        return $default(_that.workspaceId, _that.status, _that.updatedAt,
            _that.lastActiveAt);
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
    TResult Function(String workspaceId, WorkspaceLifecycleState status,
            DateTime updatedAt, DateTime? lastActiveAt)
        $default,
  ) {
    final _that = this;
    switch (_that) {
      case _WorkspaceStatus():
        return $default(_that.workspaceId, _that.status, _that.updatedAt,
            _that.lastActiveAt);
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
    TResult? Function(String workspaceId, WorkspaceLifecycleState status,
            DateTime updatedAt, DateTime? lastActiveAt)?
        $default,
  ) {
    final _that = this;
    switch (_that) {
      case _WorkspaceStatus() when $default != null:
        return $default(_that.workspaceId, _that.status, _that.updatedAt,
            _that.lastActiveAt);
      case _:
        return null;
    }
  }
}

/// @nodoc
@JsonSerializable()
class _WorkspaceStatus extends WorkspaceStatus {
  const _WorkspaceStatus(
      {required this.workspaceId,
      required this.status,
      required this.updatedAt,
      this.lastActiveAt})
      : super._();
  factory _WorkspaceStatus.fromJson(Map<String, dynamic> json) =>
      _$WorkspaceStatusFromJson(json);

  @override
  final String workspaceId;
  @override
  final WorkspaceLifecycleState status;
  @override
  final DateTime updatedAt;
  @override
  final DateTime? lastActiveAt;

  /// Create a copy of WorkspaceStatus
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  @pragma('vm:prefer-inline')
  _$WorkspaceStatusCopyWith<_WorkspaceStatus> get copyWith =>
      __$WorkspaceStatusCopyWithImpl<_WorkspaceStatus>(this, _$identity);

  @override
  Map<String, dynamic> toJson() {
    return _$WorkspaceStatusToJson(
      this,
    );
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _WorkspaceStatus &&
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

  @override
  String toString() {
    return 'WorkspaceStatus(workspaceId: $workspaceId, status: $status, updatedAt: $updatedAt, lastActiveAt: $lastActiveAt)';
  }
}

/// @nodoc
abstract mixin class _$WorkspaceStatusCopyWith<$Res>
    implements $WorkspaceStatusCopyWith<$Res> {
  factory _$WorkspaceStatusCopyWith(
          _WorkspaceStatus value, $Res Function(_WorkspaceStatus) _then) =
      __$WorkspaceStatusCopyWithImpl;
  @override
  @useResult
  $Res call(
      {String workspaceId,
      WorkspaceLifecycleState status,
      DateTime updatedAt,
      DateTime? lastActiveAt});
}

/// @nodoc
class __$WorkspaceStatusCopyWithImpl<$Res>
    implements _$WorkspaceStatusCopyWith<$Res> {
  __$WorkspaceStatusCopyWithImpl(this._self, this._then);

  final _WorkspaceStatus _self;
  final $Res Function(_WorkspaceStatus) _then;

  /// Create a copy of WorkspaceStatus
  /// with the given fields replaced by the non-null parameter values.
  @override
  @pragma('vm:prefer-inline')
  $Res call({
    Object? workspaceId = null,
    Object? status = null,
    Object? updatedAt = null,
    Object? lastActiveAt = freezed,
  }) {
    return _then(_WorkspaceStatus(
      workspaceId: null == workspaceId
          ? _self.workspaceId
          : workspaceId // ignore: cast_nullable_to_non_nullable
              as String,
      status: null == status
          ? _self.status
          : status // ignore: cast_nullable_to_non_nullable
              as WorkspaceLifecycleState,
      updatedAt: null == updatedAt
          ? _self.updatedAt
          : updatedAt // ignore: cast_nullable_to_non_nullable
              as DateTime,
      lastActiveAt: freezed == lastActiveAt
          ? _self.lastActiveAt
          : lastActiveAt // ignore: cast_nullable_to_non_nullable
              as DateTime?,
    ));
  }
}

// dart format on
