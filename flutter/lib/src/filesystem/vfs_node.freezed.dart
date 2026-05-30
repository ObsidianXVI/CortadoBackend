// GENERATED CODE - DO NOT MODIFY BY HAND
// coverage:ignore-file
// ignore_for_file: type=lint
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark, unreachable_switch_case

part of 'vfs_node.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

// dart format off
T _$identity<T>(T value) => value;

/// @nodoc
mixin _$VfsNode {
  String get path;
  String get name;
  VfsNodeSyncState get syncState;
  String? get syncMessage;

  /// Create a copy of VfsNode
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @pragma('vm:prefer-inline')
  $VfsNodeCopyWith<VfsNode> get copyWith =>
      _$VfsNodeCopyWithImpl<VfsNode>(this as VfsNode, _$identity);

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is VfsNode &&
            (identical(other.path, path) || other.path == path) &&
            (identical(other.name, name) || other.name == name) &&
            (identical(other.syncState, syncState) ||
                other.syncState == syncState) &&
            (identical(other.syncMessage, syncMessage) ||
                other.syncMessage == syncMessage));
  }

  @override
  int get hashCode =>
      Object.hash(runtimeType, path, name, syncState, syncMessage);

  @override
  String toString() {
    return 'VfsNode(path: $path, name: $name, syncState: $syncState, syncMessage: $syncMessage)';
  }
}

/// @nodoc
abstract mixin class $VfsNodeCopyWith<$Res> {
  factory $VfsNodeCopyWith(VfsNode value, $Res Function(VfsNode) _then) =
      _$VfsNodeCopyWithImpl;
  @useResult
  $Res call(
      {String path,
      String name,
      VfsNodeSyncState syncState,
      String? syncMessage});
}

/// @nodoc
class _$VfsNodeCopyWithImpl<$Res> implements $VfsNodeCopyWith<$Res> {
  _$VfsNodeCopyWithImpl(this._self, this._then);

  final VfsNode _self;
  final $Res Function(VfsNode) _then;

  /// Create a copy of VfsNode
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? path = null,
    Object? name = null,
    Object? syncState = null,
    Object? syncMessage = freezed,
  }) {
    return _then(_self.copyWith(
      path: null == path
          ? _self.path
          : path // ignore: cast_nullable_to_non_nullable
              as String,
      name: null == name
          ? _self.name
          : name // ignore: cast_nullable_to_non_nullable
              as String,
      syncState: null == syncState
          ? _self.syncState
          : syncState // ignore: cast_nullable_to_non_nullable
              as VfsNodeSyncState,
      syncMessage: freezed == syncMessage
          ? _self.syncMessage
          : syncMessage // ignore: cast_nullable_to_non_nullable
              as String?,
    ));
  }
}

/// Adds pattern-matching-related methods to [VfsNode].
extension VfsNodePatterns on VfsNode {
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
  TResult maybeMap<TResult extends Object?>({
    TResult Function(VfsFile value)? file,
    TResult Function(VfsDir value)? directory,
    required TResult orElse(),
  }) {
    final _that = this;
    switch (_that) {
      case VfsFile() when file != null:
        return file(_that);
      case VfsDir() when directory != null:
        return directory(_that);
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
  TResult map<TResult extends Object?>({
    required TResult Function(VfsFile value) file,
    required TResult Function(VfsDir value) directory,
  }) {
    final _that = this;
    switch (_that) {
      case VfsFile():
        return file(_that);
      case VfsDir():
        return directory(_that);
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
  TResult? mapOrNull<TResult extends Object?>({
    TResult? Function(VfsFile value)? file,
    TResult? Function(VfsDir value)? directory,
  }) {
    final _that = this;
    switch (_that) {
      case VfsFile() when file != null:
        return file(_that);
      case VfsDir() when directory != null:
        return directory(_that);
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
  TResult maybeWhen<TResult extends Object?>({
    TResult Function(String path, String name, int size, DateTime modTime,
            VfsNodeSyncState syncState, String? syncMessage)?
        file,
    TResult Function(
            String path,
            String name,
            List<String> childPaths,
            bool expanded,
            bool loaded,
            VfsNodeSyncState syncState,
            String? syncMessage)?
        directory,
    required TResult orElse(),
  }) {
    final _that = this;
    switch (_that) {
      case VfsFile() when file != null:
        return file(_that.path, _that.name, _that.size, _that.modTime,
            _that.syncState, _that.syncMessage);
      case VfsDir() when directory != null:
        return directory(_that.path, _that.name, _that.childPaths,
            _that.expanded, _that.loaded, _that.syncState, _that.syncMessage);
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
  TResult when<TResult extends Object?>({
    required TResult Function(String path, String name, int size,
            DateTime modTime, VfsNodeSyncState syncState, String? syncMessage)
        file,
    required TResult Function(
            String path,
            String name,
            List<String> childPaths,
            bool expanded,
            bool loaded,
            VfsNodeSyncState syncState,
            String? syncMessage)
        directory,
  }) {
    final _that = this;
    switch (_that) {
      case VfsFile():
        return file(_that.path, _that.name, _that.size, _that.modTime,
            _that.syncState, _that.syncMessage);
      case VfsDir():
        return directory(_that.path, _that.name, _that.childPaths,
            _that.expanded, _that.loaded, _that.syncState, _that.syncMessage);
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
  TResult? whenOrNull<TResult extends Object?>({
    TResult? Function(String path, String name, int size, DateTime modTime,
            VfsNodeSyncState syncState, String? syncMessage)?
        file,
    TResult? Function(
            String path,
            String name,
            List<String> childPaths,
            bool expanded,
            bool loaded,
            VfsNodeSyncState syncState,
            String? syncMessage)?
        directory,
  }) {
    final _that = this;
    switch (_that) {
      case VfsFile() when file != null:
        return file(_that.path, _that.name, _that.size, _that.modTime,
            _that.syncState, _that.syncMessage);
      case VfsDir() when directory != null:
        return directory(_that.path, _that.name, _that.childPaths,
            _that.expanded, _that.loaded, _that.syncState, _that.syncMessage);
      case _:
        return null;
    }
  }
}

/// @nodoc

class VfsFile extends VfsNode {
  const VfsFile(
      {required this.path,
      required this.name,
      required this.size,
      required this.modTime,
      this.syncState = VfsNodeSyncState.idle,
      this.syncMessage})
      : super._();

  @override
  final String path;
  @override
  final String name;
  final int size;
  final DateTime modTime;
  @override
  @JsonKey()
  final VfsNodeSyncState syncState;
  @override
  final String? syncMessage;

  /// Create a copy of VfsNode
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  @pragma('vm:prefer-inline')
  $VfsFileCopyWith<VfsFile> get copyWith =>
      _$VfsFileCopyWithImpl<VfsFile>(this, _$identity);

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is VfsFile &&
            (identical(other.path, path) || other.path == path) &&
            (identical(other.name, name) || other.name == name) &&
            (identical(other.size, size) || other.size == size) &&
            (identical(other.modTime, modTime) || other.modTime == modTime) &&
            (identical(other.syncState, syncState) ||
                other.syncState == syncState) &&
            (identical(other.syncMessage, syncMessage) ||
                other.syncMessage == syncMessage));
  }

  @override
  int get hashCode => Object.hash(
      runtimeType, path, name, size, modTime, syncState, syncMessage);

  @override
  String toString() {
    return 'VfsNode.file(path: $path, name: $name, size: $size, modTime: $modTime, syncState: $syncState, syncMessage: $syncMessage)';
  }
}

/// @nodoc
abstract mixin class $VfsFileCopyWith<$Res> implements $VfsNodeCopyWith<$Res> {
  factory $VfsFileCopyWith(VfsFile value, $Res Function(VfsFile) _then) =
      _$VfsFileCopyWithImpl;
  @override
  @useResult
  $Res call(
      {String path,
      String name,
      int size,
      DateTime modTime,
      VfsNodeSyncState syncState,
      String? syncMessage});
}

/// @nodoc
class _$VfsFileCopyWithImpl<$Res> implements $VfsFileCopyWith<$Res> {
  _$VfsFileCopyWithImpl(this._self, this._then);

  final VfsFile _self;
  final $Res Function(VfsFile) _then;

  /// Create a copy of VfsNode
  /// with the given fields replaced by the non-null parameter values.
  @override
  @pragma('vm:prefer-inline')
  $Res call({
    Object? path = null,
    Object? name = null,
    Object? size = null,
    Object? modTime = null,
    Object? syncState = null,
    Object? syncMessage = freezed,
  }) {
    return _then(VfsFile(
      path: null == path
          ? _self.path
          : path // ignore: cast_nullable_to_non_nullable
              as String,
      name: null == name
          ? _self.name
          : name // ignore: cast_nullable_to_non_nullable
              as String,
      size: null == size
          ? _self.size
          : size // ignore: cast_nullable_to_non_nullable
              as int,
      modTime: null == modTime
          ? _self.modTime
          : modTime // ignore: cast_nullable_to_non_nullable
              as DateTime,
      syncState: null == syncState
          ? _self.syncState
          : syncState // ignore: cast_nullable_to_non_nullable
              as VfsNodeSyncState,
      syncMessage: freezed == syncMessage
          ? _self.syncMessage
          : syncMessage // ignore: cast_nullable_to_non_nullable
              as String?,
    ));
  }
}

/// @nodoc

class VfsDir extends VfsNode {
  const VfsDir(
      {required this.path,
      required this.name,
      required final List<String> childPaths,
      this.expanded = false,
      this.loaded = false,
      this.syncState = VfsNodeSyncState.idle,
      this.syncMessage})
      : _childPaths = childPaths,
        super._();

  @override
  final String path;
  @override
  final String name;
  final List<String> _childPaths;
  List<String> get childPaths {
    if (_childPaths is EqualUnmodifiableListView) return _childPaths;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(_childPaths);
  }

  @JsonKey()
  final bool expanded;
  @JsonKey()
  final bool loaded;
  @override
  @JsonKey()
  final VfsNodeSyncState syncState;
  @override
  final String? syncMessage;

  /// Create a copy of VfsNode
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  @pragma('vm:prefer-inline')
  $VfsDirCopyWith<VfsDir> get copyWith =>
      _$VfsDirCopyWithImpl<VfsDir>(this, _$identity);

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is VfsDir &&
            (identical(other.path, path) || other.path == path) &&
            (identical(other.name, name) || other.name == name) &&
            const DeepCollectionEquality()
                .equals(other._childPaths, _childPaths) &&
            (identical(other.expanded, expanded) ||
                other.expanded == expanded) &&
            (identical(other.loaded, loaded) || other.loaded == loaded) &&
            (identical(other.syncState, syncState) ||
                other.syncState == syncState) &&
            (identical(other.syncMessage, syncMessage) ||
                other.syncMessage == syncMessage));
  }

  @override
  int get hashCode => Object.hash(
      runtimeType,
      path,
      name,
      const DeepCollectionEquality().hash(_childPaths),
      expanded,
      loaded,
      syncState,
      syncMessage);

  @override
  String toString() {
    return 'VfsNode.directory(path: $path, name: $name, childPaths: $childPaths, expanded: $expanded, loaded: $loaded, syncState: $syncState, syncMessage: $syncMessage)';
  }
}

/// @nodoc
abstract mixin class $VfsDirCopyWith<$Res> implements $VfsNodeCopyWith<$Res> {
  factory $VfsDirCopyWith(VfsDir value, $Res Function(VfsDir) _then) =
      _$VfsDirCopyWithImpl;
  @override
  @useResult
  $Res call(
      {String path,
      String name,
      List<String> childPaths,
      bool expanded,
      bool loaded,
      VfsNodeSyncState syncState,
      String? syncMessage});
}

/// @nodoc
class _$VfsDirCopyWithImpl<$Res> implements $VfsDirCopyWith<$Res> {
  _$VfsDirCopyWithImpl(this._self, this._then);

  final VfsDir _self;
  final $Res Function(VfsDir) _then;

  /// Create a copy of VfsNode
  /// with the given fields replaced by the non-null parameter values.
  @override
  @pragma('vm:prefer-inline')
  $Res call({
    Object? path = null,
    Object? name = null,
    Object? childPaths = null,
    Object? expanded = null,
    Object? loaded = null,
    Object? syncState = null,
    Object? syncMessage = freezed,
  }) {
    return _then(VfsDir(
      path: null == path
          ? _self.path
          : path // ignore: cast_nullable_to_non_nullable
              as String,
      name: null == name
          ? _self.name
          : name // ignore: cast_nullable_to_non_nullable
              as String,
      childPaths: null == childPaths
          ? _self._childPaths
          : childPaths // ignore: cast_nullable_to_non_nullable
              as List<String>,
      expanded: null == expanded
          ? _self.expanded
          : expanded // ignore: cast_nullable_to_non_nullable
              as bool,
      loaded: null == loaded
          ? _self.loaded
          : loaded // ignore: cast_nullable_to_non_nullable
              as bool,
      syncState: null == syncState
          ? _self.syncState
          : syncState // ignore: cast_nullable_to_non_nullable
              as VfsNodeSyncState,
      syncMessage: freezed == syncMessage
          ? _self.syncMessage
          : syncMessage // ignore: cast_nullable_to_non_nullable
              as String?,
    ));
  }
}

// dart format on
