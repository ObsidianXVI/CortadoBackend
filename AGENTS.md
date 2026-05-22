# Cortado Project Context

## Project Description
Cortado is a Flutter/Dart SaaS package providing a cloud IDE backend. Dart/Go orchestrate workspaces across users and provide lightweight workspace agents to manage intra-workspace features such as PTY terminals, LSPs, billing, file changes, etc. On the frontend, a Flutter package is installed by the end-developer who wants to provide a cloud-based backend for their IDE application, and they will simply call APIs to liaise with this abckend.
It is NOT a standalone app — it's a pub.dev package that IDE developers embed.

## Repo layout
```
cortado/
├── agent/            Go 1.23 — workspace agent (PTY, gRPC, filesystem, LSP mgmt)
├── control-plane/    Go 1.23 — HTTP gateway, workspace orchestration, WebSocket mux
├── flutter/          Dart — the pub.dev package consumers embed
├── proto/            .proto files; run `buf generate` after any change
├── terraform/        All GCP infrastructure; never use gcloud for resource creation
│   ├── envs/dev/
│   └── envs/prod/
└── scripts/          Utility scripts (not infra creation)
```


## Workflows and Memory

The project is set up with a few Markdown files for long-horizon planning, situation tracking, efficient context management and surgical-precision code changes. Adhere to the following workflows, conventions, and routines.

### Context Layers

The overall picture is that state/memory is managed in three layers:

```
Layer 1 — Always loaded (cheap)
  AGENTS.md             project brief, hard rules, layout
  CURRENT_TASK.md      exactly where we are right now with the active milestone and task

Layer 2 — Loaded for the active feature (~3-5K tokens)
  _dev/features/feature-code.md   focused spec for the current/specific work unit
  DECISIONS.md         settled architectural decisions

Layer 3 — Referenced on demand, never auto-injected (extremely large documents)
  _dev/docs/release_timeline.md     the full, detailed release timeline
  _dev/docs/technical_report.md       the full, detailed blueprint for all parts of the working system
```

### Workflow Each Turn

1. Read CURRENT_TASK.md first. This tells you exactly what to do.
2. Read DECISIONS.md if you hit an architecture question — it may already be answered. Likewise, if a decision has been answered by the user, document it properly in DECISIONS.md for future refrence.
3. Work through the task. When done:
- Verify every checkbox in the 'Definition of done' section of CURRENT_TASK.md
- Update CURRENT_TASK.md: mark done items, write what you did, set next task
- Append a one-paragraph summary to `_dev/session_log.md` with timestamp in the format:
```
DD/MM/YY HH:mm [FEAT/FIX] (<short commit hash if any commits made>) `<agent name or "dev-pro-large" if own self>` Concise but all-encompassing description/summary
... a list of core file changes (not tests or other minor stuff) in the form:
- <"A"/"M"/"D"> <filepath>
where "A" is for addition, "M" is for modification and "D" is for deletion
```
- Update _dev/test_status.md if any test status changed
- Commit working code only. Do not commit if build or tests fail.
4. If you need more context, refer to the `docs` folder for the technical blueprint of the system. Or search for documentation online. Never use third-party APIs in the codebase without grounding evidence of its actual existence. Prompt the user for guidance when not confident. Provide enough context in the prompt to the user so the user does not have to search through the logs, feature file, diffs, etc to answer.
5. If you cannot complete the task despite user prompts for guidance, write why in CURRENT_TASK.md and stop cleanly, with enough context to pick up later on without having to re-analyse for context gathering later.

Do not invent architecture. If you hit a decision the spec doesn't cover, write it to DECISIONS_NEEDED.md and continue with unblocked work.


### Sub-Agents

In order to prevent the clouding of your own context window, you will be the main orchestrator planning the feature/fix, and commanding sub-agents to perform/work through the nitty-gritty of the actual implementation, so that token costs are lower too. You will settle only key/complex details and let the sub-agents handle the rest. The sub-agents are based on increasing orders of intelligence:
- Use the `dev-light` agent For extremely brain-dead, labour-intensive work or tasks with back-and-forths or trial-and-errors. Cheap tokens.
- Use the `dev-moderate` agent For simple-to-moderate difficulty tasks or those which require average SWE skill and reasoning.
- Use the `dev-high` agent For complex, critical, or wide-scoped work which require lots of reasoning and have small room for error.
- Use the `dev-pro` For the highest level of critical thinking power, reasoning, and accuracy. For truly large-scale, complex and wide-spanning problems. Use sparingly.

Ensure the sub-agents are given enough context about the plans and approach to implement such that they don't go around in circles. Also ensure they follow all the context- and project-tracking guidelines/workflows and technical guidelines outlined in this document.

# Development/Technical Guidelines

## Hard rules — always follow these
1. `buf lint` must pass before any proto commit. Run: `cd proto && buf lint`
2. `go build ./...` must pass in both agent/ and control-plane/ before committing Go code.
3. `flutter analyze` must return zero warnings before committing Dart code.
4. `terraform validate` must pass before committing any .tf file.
5. Never use `gcloud ... create` or `gcloud ... delete` for GCP resources — use Terraform. Only if not possible, only then create a shell script for those commands, never inline them.
6. Never add API keys or secrets to any file. They live in the `.env` and GCP Secret Manager (in prod).
7. Generated files (gen/, lib/src/gen/) and build files are in .gitignore — never commit them.
8. CORTADO_ENV=development in the control plane enables dev-bypass auth.
   Do NOT add real JWT auth to any endpoint before Task 2.4.x.
9. EIO from a PTY master fd read = clean termination (shell exited). Not an error.
10. All Terraform resources must have tags: { env = var.env, project = "cortado" }

## Language notes
- agent/ and control-plane/: CGO_ENABLED=0 always (static binaries)
  Exception: cortado-daemon uses modernc.org/sqlite (pure Go, still CGO_ENABLED=0)
- Dart: use freezed for all data classes, Riverpod for state management. Use MCP for static analysis
- Go errors: wrap with fmt.Errorf("context: %w", err), never log.Fatal in library code
- Terraform provider: hashicorp/google ~> 6.0

## WebSocket mux channel ranges
- 0x0001–0x00FF  Terminal sessions
- 0x0100–0x01FF  LSP instances
- 0x0200         File sync
- 0x0300         Metrics/heartbeat
- 0x0400         Port forward tunnels
- 0x0500         System events (workspace status, cold start progress)
- 0x0600         Conflict notices (file sync)

## GCP project
- Project ID: cortado-ide
- Default zone: us-central1-a
- Default region: us-central1
- Docker Hub image: obsidianxvi/cortado:v1 (version tags run upwards with each new image)
- GKE cluster: cortado-dev (Autopilot, us-central1)
- Workspace pod namespace: cortado-workspaces
- Control plane namespace: cortado-system

## Dev bypass auth (v0.1 through early v0.2)
All endpoints accept `X-Cortado-Dev-Token: dev-bypass` header when
CORTADO_ENV=development. Pass `?dev_token=dev-bypass` for WebSocket
upgrade requests (browsers can't set headers on WS connections).
Real JWT auth is added in Task 2.4.x.

## Before finishing any task
1. Run the relevant build/lint check (see rule #1–4 above)
2. Write or update the test for the code you changed
3. `git add -p` to review your own diff before committing
4. Commit with message format: `feat(component): description` or `fix(component): description`
5. Update CURRENT_RELEASE.md with the next task to work on
6. Log into `_dev/session_log.md`

## If you hit an architecture decision the task spec doesn't cover
Stop, write the question to DECISIONS_NEEDED.md, and continue with
the next unblocked task. Do not invent architecture — flag it. At the end, prompt the user with the particular decisions needed.
