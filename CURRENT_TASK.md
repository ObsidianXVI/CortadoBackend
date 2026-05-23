# CURRENT TASK


## Release · Feature · Task
v0.1 → Feature 1.2 (Workspace Agent — PTY Core) → Task 1.2.4

## Status
DONE

## What was done last session
Completed Task 1.2.3 by implementing the first runnable workspace-agent gRPC server with PTY streaming, a `cmd/agent` entrypoint, and bufconn tests.

## What was done this session
Added `agent/Dockerfile` as a multi-stage build that produces a statically linked `cortado-agent` binary from `./cmd/agent` and packages it into an Ubuntu 22.04 runtime image with the shell/tooling needed for workspace sessions. Added `.github/workflows/build-agent.yml` to build and push the agent image to `us-central1-docker.pkg.dev/${{ vars.GCP_PROJECT }}/cortado-dev/cortado-workspace:${GITHUB_SHA}`. Verified locally with `docker build -t cortado-agent:test agent` and `docker run --rm --entrypoint file cortado-agent:test /usr/local/bin/cortado-agent`, which reports the binary as `statically linked`.

## Remaining work this session
None.

## Definition of done
- [x] `agent/Dockerfile` builds the workspace agent with `CGO_ENABLED=0`
- [x] Runtime image includes shell/tooling needed for workspace sessions
- [x] `.github/workflows/build-agent.yml` builds and pushes the agent image to Artifact Registry
- [x] Local `docker build` succeeds for `agent/`
- [x] `file /usr/local/bin/cortado-agent` inside the container reports `statically linked`
- [x] CURRENT_RELEASE.md points at the next task after this one

## Next task after this one
Task 1.2.5 — Terraform: Kubernetes manifests for workspace pod
See _dev/docs/release_timeline.md §Feature 1.2 Task 1.2.5 for full spec

## Blocked on / decisions needed
None.
