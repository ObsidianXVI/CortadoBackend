# Decisions Needed

- Should Cortado allow read-only file reads for absolute paths under `/usr/local/dart-sdk` so go-to-definition can load real Dart SDK source into read-only tabs?
  Rationale: Task 4.2.4 now opens SDK definition targets as read-only tabs, but the current file API intentionally confines reads to the workspace root. Supporting real SDK source would require a deliberate read-only whitelist in the agent plus a client path that preserves absolute SDK paths.
