// GENERATED CODE - DO NOT MODIFY BY HAND
// coverage:ignore-file
// ignore_for_file: type=lint
// ignore_for_file: unused_element, deprecated_member_use, deprecated_member_use_from_same_package, use_function_type_syntax_for_parameters, unnecessary_const, avoid_init_to_null, invalid_override_different_default_values_named, prefer_expression_function_bodies, annotate_overrides, invalid_annotation_target, unnecessary_question_mark, unreachable_switch_case

part of 'editor_models.dart';

// **************************************************************************
// FreezedGenerator
// **************************************************************************

// dart format off
T _$identity<T>(T value) => value;

/// @nodoc
mixin _$OpenTab {
  String get path;
  String get title;
  String get languageId;
  String get content;
  String get savedHash;
  String get currentHash;
  bool get isLoading;
  bool get isSaving;
  bool get loaded;
  bool get readOnly;
  String? get errorMessage;

  /// Create a copy of OpenTab
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @pragma('vm:prefer-inline')
  $OpenTabCopyWith<OpenTab> get copyWith =>
      _$OpenTabCopyWithImpl<OpenTab>(this as OpenTab, _$identity);

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is OpenTab &&
            (identical(other.path, path) || other.path == path) &&
            (identical(other.title, title) || other.title == title) &&
            (identical(other.languageId, languageId) ||
                other.languageId == languageId) &&
            (identical(other.content, content) || other.content == content) &&
            (identical(other.savedHash, savedHash) ||
                other.savedHash == savedHash) &&
            (identical(other.currentHash, currentHash) ||
                other.currentHash == currentHash) &&
            (identical(other.isLoading, isLoading) ||
                other.isLoading == isLoading) &&
            (identical(other.isSaving, isSaving) ||
                other.isSaving == isSaving) &&
            (identical(other.loaded, loaded) || other.loaded == loaded) &&
            (identical(other.readOnly, readOnly) ||
                other.readOnly == readOnly) &&
            (identical(other.errorMessage, errorMessage) ||
                other.errorMessage == errorMessage));
  }

  @override
  int get hashCode => Object.hash(
      runtimeType,
      path,
      title,
      languageId,
      content,
      savedHash,
      currentHash,
      isLoading,
      isSaving,
      loaded,
      readOnly,
      errorMessage);

  @override
  String toString() {
    return 'OpenTab(path: $path, title: $title, languageId: $languageId, content: $content, savedHash: $savedHash, currentHash: $currentHash, isLoading: $isLoading, isSaving: $isSaving, loaded: $loaded, readOnly: $readOnly, errorMessage: $errorMessage)';
  }
}

/// @nodoc
abstract mixin class $OpenTabCopyWith<$Res> {
  factory $OpenTabCopyWith(OpenTab value, $Res Function(OpenTab) _then) =
      _$OpenTabCopyWithImpl;
  @useResult
  $Res call(
      {String path,
      String title,
      String languageId,
      String content,
      String savedHash,
      String currentHash,
      bool isLoading,
      bool isSaving,
      bool loaded,
      bool readOnly,
      String? errorMessage});
}

/// @nodoc
class _$OpenTabCopyWithImpl<$Res> implements $OpenTabCopyWith<$Res> {
  _$OpenTabCopyWithImpl(this._self, this._then);

  final OpenTab _self;
  final $Res Function(OpenTab) _then;

  /// Create a copy of OpenTab
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? path = null,
    Object? title = null,
    Object? languageId = null,
    Object? content = null,
    Object? savedHash = null,
    Object? currentHash = null,
    Object? isLoading = null,
    Object? isSaving = null,
    Object? loaded = null,
    Object? readOnly = null,
    Object? errorMessage = freezed,
  }) {
    return _then(_self.copyWith(
      path: null == path
          ? _self.path
          : path // ignore: cast_nullable_to_non_nullable
              as String,
      title: null == title
          ? _self.title
          : title // ignore: cast_nullable_to_non_nullable
              as String,
      languageId: null == languageId
          ? _self.languageId
          : languageId // ignore: cast_nullable_to_non_nullable
              as String,
      content: null == content
          ? _self.content
          : content // ignore: cast_nullable_to_non_nullable
              as String,
      savedHash: null == savedHash
          ? _self.savedHash
          : savedHash // ignore: cast_nullable_to_non_nullable
              as String,
      currentHash: null == currentHash
          ? _self.currentHash
          : currentHash // ignore: cast_nullable_to_non_nullable
              as String,
      isLoading: null == isLoading
          ? _self.isLoading
          : isLoading // ignore: cast_nullable_to_non_nullable
              as bool,
      isSaving: null == isSaving
          ? _self.isSaving
          : isSaving // ignore: cast_nullable_to_non_nullable
              as bool,
      loaded: null == loaded
          ? _self.loaded
          : loaded // ignore: cast_nullable_to_non_nullable
              as bool,
      readOnly: null == readOnly
          ? _self.readOnly
          : readOnly // ignore: cast_nullable_to_non_nullable
              as bool,
      errorMessage: freezed == errorMessage
          ? _self.errorMessage
          : errorMessage // ignore: cast_nullable_to_non_nullable
              as String?,
    ));
  }
}

/// Adds pattern-matching-related methods to [OpenTab].
extension OpenTabPatterns on OpenTab {
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
    TResult Function(_OpenTab value)? $default, {
    required TResult orElse(),
  }) {
    final _that = this;
    switch (_that) {
      case _OpenTab() when $default != null:
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
    TResult Function(_OpenTab value) $default,
  ) {
    final _that = this;
    switch (_that) {
      case _OpenTab():
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
    TResult? Function(_OpenTab value)? $default,
  ) {
    final _that = this;
    switch (_that) {
      case _OpenTab() when $default != null:
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
            String path,
            String title,
            String languageId,
            String content,
            String savedHash,
            String currentHash,
            bool isLoading,
            bool isSaving,
            bool loaded,
            bool readOnly,
            String? errorMessage)?
        $default, {
    required TResult orElse(),
  }) {
    final _that = this;
    switch (_that) {
      case _OpenTab() when $default != null:
        return $default(
            _that.path,
            _that.title,
            _that.languageId,
            _that.content,
            _that.savedHash,
            _that.currentHash,
            _that.isLoading,
            _that.isSaving,
            _that.loaded,
            _that.readOnly,
            _that.errorMessage);
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
            String path,
            String title,
            String languageId,
            String content,
            String savedHash,
            String currentHash,
            bool isLoading,
            bool isSaving,
            bool loaded,
            bool readOnly,
            String? errorMessage)
        $default,
  ) {
    final _that = this;
    switch (_that) {
      case _OpenTab():
        return $default(
            _that.path,
            _that.title,
            _that.languageId,
            _that.content,
            _that.savedHash,
            _that.currentHash,
            _that.isLoading,
            _that.isSaving,
            _that.loaded,
            _that.readOnly,
            _that.errorMessage);
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
            String path,
            String title,
            String languageId,
            String content,
            String savedHash,
            String currentHash,
            bool isLoading,
            bool isSaving,
            bool loaded,
            bool readOnly,
            String? errorMessage)?
        $default,
  ) {
    final _that = this;
    switch (_that) {
      case _OpenTab() when $default != null:
        return $default(
            _that.path,
            _that.title,
            _that.languageId,
            _that.content,
            _that.savedHash,
            _that.currentHash,
            _that.isLoading,
            _that.isSaving,
            _that.loaded,
            _that.readOnly,
            _that.errorMessage);
      case _:
        return null;
    }
  }
}

/// @nodoc

class _OpenTab extends OpenTab {
  const _OpenTab(
      {required this.path,
      required this.title,
      required this.languageId,
      this.content = '',
      this.savedHash = '',
      this.currentHash = '',
      this.isLoading = false,
      this.isSaving = false,
      this.loaded = false,
      this.readOnly = false,
      this.errorMessage})
      : super._();

  @override
  final String path;
  @override
  final String title;
  @override
  final String languageId;
  @override
  @JsonKey()
  final String content;
  @override
  @JsonKey()
  final String savedHash;
  @override
  @JsonKey()
  final String currentHash;
  @override
  @JsonKey()
  final bool isLoading;
  @override
  @JsonKey()
  final bool isSaving;
  @override
  @JsonKey()
  final bool loaded;
  @override
  @JsonKey()
  final bool readOnly;
  @override
  final String? errorMessage;

  /// Create a copy of OpenTab
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  @pragma('vm:prefer-inline')
  _$OpenTabCopyWith<_OpenTab> get copyWith =>
      __$OpenTabCopyWithImpl<_OpenTab>(this, _$identity);

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _OpenTab &&
            (identical(other.path, path) || other.path == path) &&
            (identical(other.title, title) || other.title == title) &&
            (identical(other.languageId, languageId) ||
                other.languageId == languageId) &&
            (identical(other.content, content) || other.content == content) &&
            (identical(other.savedHash, savedHash) ||
                other.savedHash == savedHash) &&
            (identical(other.currentHash, currentHash) ||
                other.currentHash == currentHash) &&
            (identical(other.isLoading, isLoading) ||
                other.isLoading == isLoading) &&
            (identical(other.isSaving, isSaving) ||
                other.isSaving == isSaving) &&
            (identical(other.loaded, loaded) || other.loaded == loaded) &&
            (identical(other.readOnly, readOnly) ||
                other.readOnly == readOnly) &&
            (identical(other.errorMessage, errorMessage) ||
                other.errorMessage == errorMessage));
  }

  @override
  int get hashCode => Object.hash(
      runtimeType,
      path,
      title,
      languageId,
      content,
      savedHash,
      currentHash,
      isLoading,
      isSaving,
      loaded,
      readOnly,
      errorMessage);

  @override
  String toString() {
    return 'OpenTab(path: $path, title: $title, languageId: $languageId, content: $content, savedHash: $savedHash, currentHash: $currentHash, isLoading: $isLoading, isSaving: $isSaving, loaded: $loaded, readOnly: $readOnly, errorMessage: $errorMessage)';
  }
}

/// @nodoc
abstract mixin class _$OpenTabCopyWith<$Res> implements $OpenTabCopyWith<$Res> {
  factory _$OpenTabCopyWith(_OpenTab value, $Res Function(_OpenTab) _then) =
      __$OpenTabCopyWithImpl;
  @override
  @useResult
  $Res call(
      {String path,
      String title,
      String languageId,
      String content,
      String savedHash,
      String currentHash,
      bool isLoading,
      bool isSaving,
      bool loaded,
      bool readOnly,
      String? errorMessage});
}

/// @nodoc
class __$OpenTabCopyWithImpl<$Res> implements _$OpenTabCopyWith<$Res> {
  __$OpenTabCopyWithImpl(this._self, this._then);

  final _OpenTab _self;
  final $Res Function(_OpenTab) _then;

  /// Create a copy of OpenTab
  /// with the given fields replaced by the non-null parameter values.
  @override
  @pragma('vm:prefer-inline')
  $Res call({
    Object? path = null,
    Object? title = null,
    Object? languageId = null,
    Object? content = null,
    Object? savedHash = null,
    Object? currentHash = null,
    Object? isLoading = null,
    Object? isSaving = null,
    Object? loaded = null,
    Object? readOnly = null,
    Object? errorMessage = freezed,
  }) {
    return _then(_OpenTab(
      path: null == path
          ? _self.path
          : path // ignore: cast_nullable_to_non_nullable
              as String,
      title: null == title
          ? _self.title
          : title // ignore: cast_nullable_to_non_nullable
              as String,
      languageId: null == languageId
          ? _self.languageId
          : languageId // ignore: cast_nullable_to_non_nullable
              as String,
      content: null == content
          ? _self.content
          : content // ignore: cast_nullable_to_non_nullable
              as String,
      savedHash: null == savedHash
          ? _self.savedHash
          : savedHash // ignore: cast_nullable_to_non_nullable
              as String,
      currentHash: null == currentHash
          ? _self.currentHash
          : currentHash // ignore: cast_nullable_to_non_nullable
              as String,
      isLoading: null == isLoading
          ? _self.isLoading
          : isLoading // ignore: cast_nullable_to_non_nullable
              as bool,
      isSaving: null == isSaving
          ? _self.isSaving
          : isSaving // ignore: cast_nullable_to_non_nullable
              as bool,
      loaded: null == loaded
          ? _self.loaded
          : loaded // ignore: cast_nullable_to_non_nullable
              as bool,
      readOnly: null == readOnly
          ? _self.readOnly
          : readOnly // ignore: cast_nullable_to_non_nullable
              as bool,
      errorMessage: freezed == errorMessage
          ? _self.errorMessage
          : errorMessage // ignore: cast_nullable_to_non_nullable
              as String?,
    ));
  }
}

/// @nodoc
mixin _$TabsState {
  List<OpenTab> get tabs;
  String? get activePath;

  /// Create a copy of TabsState
  /// with the given fields replaced by the non-null parameter values.
  @JsonKey(includeFromJson: false, includeToJson: false)
  @pragma('vm:prefer-inline')
  $TabsStateCopyWith<TabsState> get copyWith =>
      _$TabsStateCopyWithImpl<TabsState>(this as TabsState, _$identity);

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is TabsState &&
            const DeepCollectionEquality().equals(other.tabs, tabs) &&
            (identical(other.activePath, activePath) ||
                other.activePath == activePath));
  }

  @override
  int get hashCode => Object.hash(
      runtimeType, const DeepCollectionEquality().hash(tabs), activePath);

  @override
  String toString() {
    return 'TabsState(tabs: $tabs, activePath: $activePath)';
  }
}

/// @nodoc
abstract mixin class $TabsStateCopyWith<$Res> {
  factory $TabsStateCopyWith(TabsState value, $Res Function(TabsState) _then) =
      _$TabsStateCopyWithImpl;
  @useResult
  $Res call({List<OpenTab> tabs, String? activePath});
}

/// @nodoc
class _$TabsStateCopyWithImpl<$Res> implements $TabsStateCopyWith<$Res> {
  _$TabsStateCopyWithImpl(this._self, this._then);

  final TabsState _self;
  final $Res Function(TabsState) _then;

  /// Create a copy of TabsState
  /// with the given fields replaced by the non-null parameter values.
  @pragma('vm:prefer-inline')
  @override
  $Res call({
    Object? tabs = null,
    Object? activePath = freezed,
  }) {
    return _then(_self.copyWith(
      tabs: null == tabs
          ? _self.tabs
          : tabs // ignore: cast_nullable_to_non_nullable
              as List<OpenTab>,
      activePath: freezed == activePath
          ? _self.activePath
          : activePath // ignore: cast_nullable_to_non_nullable
              as String?,
    ));
  }
}

/// Adds pattern-matching-related methods to [TabsState].
extension TabsStatePatterns on TabsState {
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
    TResult Function(_TabsState value)? $default, {
    required TResult orElse(),
  }) {
    final _that = this;
    switch (_that) {
      case _TabsState() when $default != null:
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
    TResult Function(_TabsState value) $default,
  ) {
    final _that = this;
    switch (_that) {
      case _TabsState():
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
    TResult? Function(_TabsState value)? $default,
  ) {
    final _that = this;
    switch (_that) {
      case _TabsState() when $default != null:
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
    TResult Function(List<OpenTab> tabs, String? activePath)? $default, {
    required TResult orElse(),
  }) {
    final _that = this;
    switch (_that) {
      case _TabsState() when $default != null:
        return $default(_that.tabs, _that.activePath);
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
    TResult Function(List<OpenTab> tabs, String? activePath) $default,
  ) {
    final _that = this;
    switch (_that) {
      case _TabsState():
        return $default(_that.tabs, _that.activePath);
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
    TResult? Function(List<OpenTab> tabs, String? activePath)? $default,
  ) {
    final _that = this;
    switch (_that) {
      case _TabsState() when $default != null:
        return $default(_that.tabs, _that.activePath);
      case _:
        return null;
    }
  }
}

/// @nodoc

class _TabsState extends TabsState {
  const _TabsState(
      {final List<OpenTab> tabs = const <OpenTab>[], this.activePath})
      : _tabs = tabs,
        super._();

  final List<OpenTab> _tabs;
  @override
  @JsonKey()
  List<OpenTab> get tabs {
    if (_tabs is EqualUnmodifiableListView) return _tabs;
    // ignore: implicit_dynamic_type
    return EqualUnmodifiableListView(_tabs);
  }

  @override
  final String? activePath;

  /// Create a copy of TabsState
  /// with the given fields replaced by the non-null parameter values.
  @override
  @JsonKey(includeFromJson: false, includeToJson: false)
  @pragma('vm:prefer-inline')
  _$TabsStateCopyWith<_TabsState> get copyWith =>
      __$TabsStateCopyWithImpl<_TabsState>(this, _$identity);

  @override
  bool operator ==(Object other) {
    return identical(this, other) ||
        (other.runtimeType == runtimeType &&
            other is _TabsState &&
            const DeepCollectionEquality().equals(other._tabs, _tabs) &&
            (identical(other.activePath, activePath) ||
                other.activePath == activePath));
  }

  @override
  int get hashCode => Object.hash(
      runtimeType, const DeepCollectionEquality().hash(_tabs), activePath);

  @override
  String toString() {
    return 'TabsState(tabs: $tabs, activePath: $activePath)';
  }
}

/// @nodoc
abstract mixin class _$TabsStateCopyWith<$Res>
    implements $TabsStateCopyWith<$Res> {
  factory _$TabsStateCopyWith(
          _TabsState value, $Res Function(_TabsState) _then) =
      __$TabsStateCopyWithImpl;
  @override
  @useResult
  $Res call({List<OpenTab> tabs, String? activePath});
}

/// @nodoc
class __$TabsStateCopyWithImpl<$Res> implements _$TabsStateCopyWith<$Res> {
  __$TabsStateCopyWithImpl(this._self, this._then);

  final _TabsState _self;
  final $Res Function(_TabsState) _then;

  /// Create a copy of TabsState
  /// with the given fields replaced by the non-null parameter values.
  @override
  @pragma('vm:prefer-inline')
  $Res call({
    Object? tabs = null,
    Object? activePath = freezed,
  }) {
    return _then(_TabsState(
      tabs: null == tabs
          ? _self._tabs
          : tabs // ignore: cast_nullable_to_non_nullable
              as List<OpenTab>,
      activePath: freezed == activePath
          ? _self.activePath
          : activePath // ignore: cast_nullable_to_non_nullable
              as String?,
    ));
  }
}

// dart format on
