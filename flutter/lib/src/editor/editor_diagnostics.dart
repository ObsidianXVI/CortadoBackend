import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../filesystem/vfs_notifier.dart';
import 'cortado_lsp_client.dart';

enum CortadoFileDiagnosticStatus {
  none,
  warning,
  error,
}

final cortadoWorkspaceDiagnosticStatusProvider =
    StateProvider<Map<String, CortadoFileDiagnosticStatus>>(
  (ref) => const <String, CortadoFileDiagnosticStatus>{},
);

Map<String, CortadoFileDiagnosticStatus> summarizeWorkspaceDiagnosticStatuses(
  CortadoLSPDiagnosticsByUri diagnosticsByUri, {
  String workspaceRoot = '/workspace',
}) {
  final statuses = <String, CortadoFileDiagnosticStatus>{};
  for (final entry in diagnosticsByUri.entries) {
    final path = workspacePathFromDocumentUri(
      entry.key,
      workspaceRoot: workspaceRoot,
    );
    if (path == null) {
      continue;
    }

    final status = summarizeDiagnosticStatus(entry.value);
    if (status == CortadoFileDiagnosticStatus.none) {
      continue;
    }
    statuses[path] = status;
  }
  return Map<String, CortadoFileDiagnosticStatus>.unmodifiable(statuses);
}

CortadoFileDiagnosticStatus summarizeDiagnosticStatus(
  List<CortadoLSPDiagnostic> diagnostics,
) {
  var hasWarning = false;
  for (final diagnostic in diagnostics) {
    final severity = (diagnostic['severity'] as num?)?.toInt();
    if (severity == 1) {
      return CortadoFileDiagnosticStatus.error;
    }
    if (severity == 2) {
      hasWarning = true;
    }
  }
  return hasWarning
      ? CortadoFileDiagnosticStatus.warning
      : CortadoFileDiagnosticStatus.none;
}

String workspaceDocumentUriForPath(
  String path, {
  String workspaceRoot = '/workspace',
}) {
  return Uri(
    scheme: 'file',
    path: '${normalizeVfsPath(workspaceRoot)}${normalizeVfsPath(path)}',
  ).toString();
}

String? workspacePathFromDocumentUri(
  String uri, {
  String workspaceRoot = '/workspace',
}) {
  final parsed = Uri.tryParse(uri);
  if (parsed == null || parsed.scheme != 'file') {
    return null;
  }

  final normalizedRoot = normalizeVfsPath(workspaceRoot);
  final documentPath = normalizeVfsPath(parsed.path);
  final inWorkspace = documentPath == normalizedRoot ||
      documentPath.startsWith('$normalizedRoot/');
  if (!inWorkspace) {
    return null;
  }

  final relativePath = documentPath.substring(normalizedRoot.length);
  return relativePath.isEmpty ? vfsRootPath : normalizeVfsPath(relativePath);
}
