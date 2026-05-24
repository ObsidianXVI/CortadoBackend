# Feature: Repo & Dev Environment Bootstrap (Tasks 1.1.1 – 1.1.2)

## Scope
Create the monorepo skeleton and devcontainer from scratch.
There are no existing files — this is the first commit.

## Files to create
.devcontainer/Dockerfile
.devcontainer/devcontainer.json
.gitignore
proto/buf.yaml
proto/buf.gen.yaml
proto/agent/v1/agent.proto   (empty service skeleton only)
agent/go.mod
control-plane/go.mod
flutter/pubspec.yaml          (package skeleton, not a full app)
terraform/envs/dev/.gitkeep
terraform/envs/prod/.gitkeep
terraform/modules/.gitkeep
scripts/work.sh               (copy from repo root scripts/work.sh)
README.md                     (one paragraph, placeholder)

## Pinned versions (use exactly these)
GO_VERSION=1.23.4
FLUTTER_VERSION=3.27.0
TERRAFORM_VERSION=1.9.8
BUF_VERSION=1.47.2
NODE_MAJOR=22

## devcontainer.json requirements
- Docker-outside-of-Docker feature (not DinD)
- postCreateCommand: "cd proto && buf generate && cd ../flutter && flutter pub get"
- remoteEnv: CORTADO_ENV=development
- forwardPorts: [8080, 9090, 9731, 3001]
- Extensions: golang.go, dart-code.dart-code, dart-code.flutter, hashicorp.terraform

## buf.gen.yaml requirements
- Go stubs → agent/gen/ (paths=source_relative)
- Go gRPC stubs → agent/gen/ (require_unimplemented_servers=false)
- Dart stubs → flutter/lib/src/gen/

## .gitignore must include
_dev/
**/gen/
flutter/lib/src/gen/
**/.dart_tool/
**/build/
node_modules/
.terraform/
*.tfstate
*.tfstate.backup
.env
*.pem
*.key

## agent.proto skeleton (empty, just enough for buf lint to pass)
syntax = "proto3";
package agent.v1;
option go_package = "github.com/your-org/cortado/agent/gen/agent/v1";
service WorkspaceAgentService {}

## Definition of done
- [ ] buf lint passes with zero errors
- [ ] buf generate runs without errors (stubs created in agent/gen/ and flutter/lib/src/gen/)
- [ ] go build ./... passes in agent/ (empty module, no errors)
- [ ] flutter pub get succeeds in flutter/
- [ ] .gitignore includes _dev/, gen/, .env
- [ ] Initial commit: "chore: monorepo scaffold and devcontainer"