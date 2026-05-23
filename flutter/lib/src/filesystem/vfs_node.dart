import 'package:freezed_annotation/freezed_annotation.dart';

part 'vfs_node.freezed.dart';

@freezed
class VfsNode with _$VfsNode {
  const VfsNode._();

  const factory VfsNode.file({
    required String path,
    required String name,
    required int size,
    required DateTime modTime,
  }) = VfsFile;

  const factory VfsNode.directory({
    required String path,
    required String name,
    required List<String> childPaths,
    @Default(false) bool expanded,
    @Default(false) bool loaded,
  }) = VfsDir;
}
