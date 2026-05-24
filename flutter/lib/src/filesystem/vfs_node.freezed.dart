// coverage:ignore-file
// GENERATED CODE - DO NOT MODIFY BY HAND
// ignore_for_file: type=lint
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark

part of 'vfs_node.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

T _$identity<T>(T value) => value;

final _privateConstructorUsedError = UnsupportedError(
    'It seems like you constructed your class using `MyClass._()`. This constructor is only meant to be used by freezed and you are not supposed to need it nor use it.\nPlease check the documentation here for more information: https://github.com/rrousselGit/freezed#adding-getters-and-methods-to-our-models');

/// @nodoc
mixin _$VfsNode {
  String get path => throw _privateConstructorUsedError;
  String get name => throw _privateConstructorUsedError;
  VfsNodeSyncState get syncState => throw _privateConstructorUsedError;
  String? get syncMessage => throw _privateConstructorUsedError;
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
  }) =>
      throw _privateConstructorUsedError;
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
  }) =>
      throw _privateConstructorUsedError;
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
  }) =>
      throw _privateConstructorUsedError;
  @optionalTypeArgs
  TResult map<TResult extends Object?>({
    required TResult Function(VfsFile value) file,
    required TResult Function(VfsDir value) directory,
  }) =>
      throw _privateConstructorUsedError;
  @optionalTypeArgs
  TResult? mapOrNull<TResult extends Object?>({
    TResult? Function(VfsFile value)? file,
    TResult? Function(VfsDir value)? directory,
  }) =>
      throw _privateConstructorUsedError;
  @optionalTypeArgs
  TResult maybeMap<TResult extends Object?>({
    TResult Function(VfsFile value)? file,
    TResult Function(VfsDir value)? directory,
    required TResult orElse(),
  }) =>
      throw _privateConstructorUsedError;

  /// Create a copy of VfsNode
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  $VfsNodeCopyWith<VfsNode> get copyWith => throw _privateConstructorUsedError;
}

/// @nodoc
abstract class $VfsNodeCopyWith<$Res> {
  factory $VfsNodeCopyWith(VfsNode value, $Res Function(VfsNode) then) =
      _$VfsNodeCopyWithImpl<$Res, VfsNode>;
  @useResult
  $Res call(
      {String path,
      String name,
      VfsNodeSyncState syncState,
      String? syncMessage});
}

/// @nodoc
class _$VfsNodeCopyWithImpl<$Res, $Val extends VfsNode>
    implements $VfsNodeCopyWith<$Res> {
  _$VfsNodeCopyWithImpl(this._value, this._then);

  // ignore: unused_field
  final $Val _value;
  // ignore: unused_field
  final $Res Function($Val) _then;

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
    return _then(_value.copyWith(
      path: null == path
          ? _value.path
          : path // ignore: cast_nullable_to_non_nullable
              as String,
      name: null == name
          ? _value.name
          : name // ignore: cast_nullable_to_non_nullable
              as String,
      syncState: null == syncState
          ? _value.syncState
          : syncState // ignore: cast_nullable_to_non_nullable
              as VfsNodeSyncState,
      syncMessage: freezed == syncMessage
          ? _value.syncMessage
          : syncMessage // ignore: cast_nullable_to_non_nullable
              as String?,
    ) as $Val);
  }
}

/// @nodoc
abstract class _$$VfsFileImplCopyWith<$Res> implements $VfsNodeCopyWith<$Res> {
  factory _$$VfsFileImplCopyWith(
          _$VfsFileImpl value, $Res Function(_$VfsFileImpl) then) =
      __$$VfsFileImplCopyWithImpl<$Res>;
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
class __$$VfsFileImplCopyWithImpl<$Res>
    extends _$VfsNodeCopyWithImpl<$Res, _$VfsFileImpl>
    implements _$$VfsFileImplCopyWith<$Res> {
  __$$VfsFileImplCopyWithImpl(
      _$VfsFileImpl _value, $Res Function(_$VfsFileImpl) _then)
      : super(_value, _then);

  /// Create a copy of VfsNode
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? path = null,
    Object? name = null,
    Object? size = null,
    Object? modTime = null,
    Object? syncState = null,
    Object? syncMessage = freezed,
  }) {
    return _then(_$VfsFileImpl(
      path: null == path
          ? _value.path
          : path // ignore: cast_nullable_to_non_nullable
              as String,
      name: null == name
          ? _value.name
          : name // ignore: cast_nullable_to_non_nullable
              as String,
      size: null == size
          ? _value.size
          : size // ignore: cast_nullable_to_non_nullable
              as int,
      modTime: null == modTime
          ? _value.modTime
          : modTime // ignore: cast_nullable_to_non_nullable
              as DateTime,
      syncState: null == syncState
          ? _value.syncState
          : syncState // ignore: cast_nullable_to_non_nullable
              as VfsNodeSyncState,
      syncMessage: freezed == syncMessage
          ? _value.syncMessage
          : syncMessage // ignore: cast_nullable_to_non_nullable
              as String?,
    ));
  }
}

/// @nodoc

class _$VfsFileImpl extends VfsFile {
  const _$VfsFileImpl(
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
  @override
  final int size;
  @override
  final DateTime modTime;
  @override
  @JsonKey()
  final VfsNodeSyncState syncState;
  @override
  final String? syncMessage;

  @override
  String toString() {
    return 'VfsNode.file(path: $path, name: $name, size: $size, modTime: $modTime, syncState: $syncState, syncMessage: $syncMessage)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$VfsFileImpl &&
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

  /// Create a copy of VfsNode
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$VfsFileImplCopyWith<_$VfsFileImpl> get copyWith =>
      __$$VfsFileImplCopyWithImpl<_$VfsFileImpl>(this, _$identity);

  @override
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
    return file(path, name, size, modTime, syncState, syncMessage);
  }

  @override
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
    return file?.call(path, name, size, modTime, syncState, syncMessage);
  }

  @override
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
    if (file != null) {
      return file(path, name, size, modTime, syncState, syncMessage);
    }
    return orElse();
  }

  @override
  @optionalTypeArgs
  TResult map<TResult extends Object?>({
    required TResult Function(VfsFile value) file,
    required TResult Function(VfsDir value) directory,
  }) {
    return file(this);
  }

  @override
  @optionalTypeArgs
  TResult? mapOrNull<TResult extends Object?>({
    TResult? Function(VfsFile value)? file,
    TResult? Function(VfsDir value)? directory,
  }) {
    return file?.call(this);
  }

  @override
  @optionalTypeArgs
  TResult maybeMap<TResult extends Object?>({
    TResult Function(VfsFile value)? file,
    TResult Function(VfsDir value)? directory,
    required TResult orElse(),
  }) {
    if (file != null) {
      return file(this);
    }
    return orElse();
  }
}

abstract class VfsFile extends VfsNode {
  const factory VfsFile(
      {required final String path,
      required final String name,
      required final int size,
      required final DateTime modTime,
      final VfsNodeSyncState syncState,
      final String? syncMessage}) = _$VfsFileImpl;
  const VfsFile._() : super._();

  @override
  String get path;
  @override
  String get name;
  int get size;
  DateTime get modTime;
  @override
  VfsNodeSyncState get syncState;
  @override
  String? get syncMessage;

  /// Create a copy of VfsNode
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$VfsFileImplCopyWith<_$VfsFileImpl> get copyWith =>
      throw _privateConstructorUsedError;
}

/// @nodoc
abstract class _$$VfsDirImplCopyWith<$Res> implements $VfsNodeCopyWith<$Res> {
  factory _$$VfsDirImplCopyWith(
          _$VfsDirImpl value, $Res Function(_$VfsDirImpl) then) =
      __$$VfsDirImplCopyWithImpl<$Res>;
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
class __$$VfsDirImplCopyWithImpl<$Res>
    extends _$VfsNodeCopyWithImpl<$Res, _$VfsDirImpl>
    implements _$$VfsDirImplCopyWith<$Res> {
  __$$VfsDirImplCopyWithImpl(
      _$VfsDirImpl _value, $Res Function(_$VfsDirImpl) _then)
      : super(_value, _then);

  /// Create a copy of VfsNode
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? path = null,
    Object? name = null,
    Object? childPaths = null,
    Object? expanded = null,
    Object? loaded = null,
    Object? syncState = null,
    Object? syncMessage = freezed,
  }) {
    return _then(_$VfsDirImpl(
      path: null == path
          ? _value.path
          : path // ignore: cast_nullable_to_non_nullable
              as String,
      name: null == name
          ? _value.name
          : name // ignore: cast_nullable_to_non_nullable
              as String,
      childPaths: null == childPaths
          ? _value._childPaths
          : childPaths // ignore: cast_nullable_to_non_nullable
              as List<String>,
      expanded: null == expanded
          ? _value.expanded
          : expanded // ignore: cast_nullable_to_non_nullable
              as bool,
      loaded: null == loaded
          ? _value.loaded
          : loaded // ignore: cast_nullable_to_non_nullable
              as bool,
      syncState: null == syncState
          ? _value.syncState
          : syncState // ignore: cast_nullable_to_non_nullable
              as VfsNodeSyncState,
      syncMessage: freezed == syncMessage
          ? _value.syncMessage
          : syncMessage // ignore: cast_nullable_to_non_nullable
              as String?,
    ));
  }
}

/// @nodoc

class _$VfsDirImpl extends VfsDir {
  const _$VfsDirImpl(
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
  @override
  List<String> get childPaths {
    if (_childPaths is EqualUnmodifiableListView) return _childPaths;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(_childPaths);
  }

  @override
  @JsonKey()
  final bool expanded;
  @override
  @JsonKey()
  final bool loaded;
  @override
  @JsonKey()
  final VfsNodeSyncState syncState;
  @override
  final String? syncMessage;

  @override
  String toString() {
    return 'VfsNode.directory(path: $path, name: $name, childPaths: $childPaths, expanded: $expanded, loaded: $loaded, syncState: $syncState, syncMessage: $syncMessage)';
  }

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _$VfsDirImpl &&
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

  /// Create a copy of VfsNode
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @override
  @pragma('vm:prefer-inline')
  _$$VfsDirImplCopyWith<_$VfsDirImpl> get copyWith =>
      __$$VfsDirImplCopyWithImpl<_$VfsDirImpl>(this, _$identity);

  @override
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
    return directory(
        path, name, childPaths, expanded, loaded, syncState, syncMessage);
  }

  @override
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
    return directory?.call(
        path, name, childPaths, expanded, loaded, syncState, syncMessage);
  }

  @override
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
    if (directory != null) {
      return directory(
          path, name, childPaths, expanded, loaded, syncState, syncMessage);
    }
    return orElse();
  }

  @override
  @optionalTypeArgs
  TResult map<TResult extends Object?>({
    required TResult Function(VfsFile value) file,
    required TResult Function(VfsDir value) directory,
  }) {
    return directory(this);
  }

  @override
  @optionalTypeArgs
  TResult? mapOrNull<TResult extends Object?>({
    TResult? Function(VfsFile value)? file,
    TResult? Function(VfsDir value)? directory,
  }) {
    return directory?.call(this);
  }

  @override
  @optionalTypeArgs
  TResult maybeMap<TResult extends Object?>({
    TResult Function(VfsFile value)? file,
    TResult Function(VfsDir value)? directory,
    required TResult orElse(),
  }) {
    if (directory != null) {
      return directory(this);
    }
    return orElse();
  }
}

abstract class VfsDir extends VfsNode {
  const factory VfsDir(
      {required final String path,
      required final String name,
      required final List<String> childPaths,
      final bool expanded,
      final bool loaded,
      final VfsNodeSyncState syncState,
      final String? syncMessage}) = _$VfsDirImpl;
  const VfsDir._() : super._();

  @override
  String get path;
  @override
  String get name;
  List<String> get childPaths;
  bool get expanded;
  bool get loaded;
  @override
  VfsNodeSyncState get syncState;
  @override
  String? get syncMessage;

  /// Create a copy of VfsNode
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  _$$VfsDirImplCopyWith<_$VfsDirImpl> get copyWith =>
      throw _privateConstructorUsedError;
}
