22/05/26 12:41
- PASS `cd proto && buf lint`
- PASS `cd proto && buf generate`
- PASS `cd agent && GOTOOLCHAIN=local go mod tidy`
- PASS `cd agent && CGO_ENABLED=0 GOTOOLCHAIN=local go build ./...`
- PASS `cd control-plane && CGO_ENABLED=0 GOTOOLCHAIN=local go build ./...`
- PASS `cd flutter && flutter pub get`
- PASS `cd flutter && flutter analyze`
22/05/26 13:33
- PASS `terraform -chdir=terraform/envs/dev init -backend=false`
- PASS `terraform -chdir=terraform/envs/prod init -backend=false`
- PASS `terraform -chdir=terraform/envs/dev validate`
- PASS `terraform -chdir=terraform/envs/prod validate`
- PASS `terraform -chdir=terraform/envs/dev init -reconfigure`
- PASS `terraform -chdir=terraform/envs/dev plan -input=false -lock=false -out=/tmp/cortado-dev.tfplan`
23/05/26 04:08
- PASS `kubectl apply -f scripts/k8s/workspace-bootstrap.yaml`
- PASS `kubectl apply -f scripts/k8s/workspace-bootstrap.yaml` (idempotency re-run)
- PASS `kubectl get namespace cortado-workspaces`
- PASS `kubectl get serviceaccount workspace-sa -n cortado-workspaces -o yaml`
23/05/26 05:23
- PASS `cd proto && buf lint`
- PASS `cd proto && buf generate`
- PASS `cd agent && CGO_ENABLED=0 GOTOOLCHAIN=local /usr/local/go/bin/go build ./...`
- PASS `cd flutter && /home/OBSiDIAN/tools/flutter/bin/flutter analyze`
23/05/26 05:44
- PASS `cd agent && GOTOOLCHAIN=local /usr/local/go/bin/go mod tidy`
- PASS `cd agent && GOTOOLCHAIN=local /usr/local/go/bin/go test ./...`
- PASS `cd agent && CGO_ENABLED=0 GOTOOLCHAIN=local /usr/local/go/bin/go build ./...`
23/05/26 05:59
- PASS `cd agent && GOTOOLCHAIN=local /usr/local/go/bin/go mod tidy`
- PASS `cd agent && CGO_ENABLED=0 GOTOOLCHAIN=local /usr/local/go/bin/go test ./...`
- PASS `cd agent && CGO_ENABLED=0 GOTOOLCHAIN=local /usr/local/go/bin/go build ./...`
23/05/26 06:11
- PASS `docker build -t cortado-agent:test agent`
- PASS `docker run --rm --entrypoint file cortado-agent:test /usr/local/bin/cortado-agent`
23/05/26 06:37
- PASS `gcloud auth configure-docker us-central1-docker.pkg.dev --quiet`
- PASS `docker push us-central1-docker.pkg.dev/cortado-ide/cortado-dev/cortado-workspace:781d613`
- PASS `terraform fmt -recursive terraform`
- PASS `terraform -chdir=terraform/envs/dev init -reconfigure`
- PASS `terraform -chdir=terraform/envs/prod init -backend=false`
- PASS `terraform -chdir=terraform/envs/dev validate`
- PASS `terraform -chdir=terraform/envs/prod validate`
- PASS `terraform -chdir=terraform/envs/dev apply -auto-approve`
- PASS `kubectl get serviceaccount workspace-sa -n cortado-workspaces -o yaml`
- PASS `kubectl wait --for=condition=Ready pod/workspace-pod-test -n cortado-workspaces --timeout=300s`
23/05/26 05:47
- PASS `cd control-plane && CGO_ENABLED=0 GOTOOLCHAIN=local /usr/local/go/bin/go test ./...`
- PASS `cd control-plane && CGO_ENABLED=0 GOTOOLCHAIN=local /usr/local/go/bin/go build ./...`
- PASS `terraform fmt -recursive terraform`
- PASS `terraform -chdir=terraform/envs/dev init -backend=false`
- PASS `terraform -chdir=terraform/envs/dev validate`
- PASS `terraform -chdir=terraform/envs/prod init -backend=false`
- PASS `terraform -chdir=terraform/envs/prod validate`
23/05/26 13:43
- PASS `cd control-plane && go test ./...`
- PASS `cd control-plane && CGO_ENABLED=0 go build ./...`
23/05/26 06:03
- PASS `cd control-plane && GOTOOLCHAIN=local /usr/local/go/bin/go mod tidy`
- PASS `cd control-plane && CGO_ENABLED=0 GOTOOLCHAIN=local /usr/local/go/bin/go test ./...`
- PASS `cd control-plane && CGO_ENABLED=0 GOTOOLCHAIN=local /usr/local/go/bin/go build ./...`
- PASS `terraform fmt -recursive terraform`
- PASS `terraform -chdir=terraform/envs/dev init -backend=false`
- PASS `terraform -chdir=terraform/envs/dev validate`
- PASS `terraform -chdir=terraform/envs/prod init -backend=false`
- PASS `terraform -chdir=terraform/envs/prod validate`
23/05/26 06:18
- PASS `docker build -t cortado-workspace:test agent/`
23/05/26 06:19
- PASS `cd control-plane && GOTOOLCHAIN=local /usr/local/go/bin/go mod tidy`
- PASS `cd control-plane && CGO_ENABLED=0 GOTOOLCHAIN=local /usr/local/go/bin/go test ./...`
- PASS `cd control-plane && CGO_ENABLED=0 GOTOOLCHAIN=local /usr/local/go/bin/go build ./...`
23/05/26 06:51
- PASS `cd control-plane && GOTOOLCHAIN=local go mod tidy`
- PASS `cd control-plane && CGO_ENABLED=0 GOTOOLCHAIN=local /usr/local/go/bin/go test ./...`
- PASS `cd control-plane && CGO_ENABLED=0 GOTOOLCHAIN=local /usr/local/go/bin/go build ./...`
23/05/26 07:00
- PASS `cd flutter && /home/OBSiDIAN/tools/flutter/bin/flutter pub get`
- PASS `cd flutter && /home/OBSiDIAN/tools/flutter/bin/flutter test`
- PASS `cd flutter && /home/OBSiDIAN/tools/flutter/bin/flutter analyze`
23/05/26 07:12
- PASS `cd flutter && /home/OBSiDIAN/tools/flutter/bin/dart format lib/src/mux_frame.dart test/mux_frame_test.dart test/cortado_client_test.dart`
- PASS `cd flutter && /home/OBSiDIAN/tools/flutter/bin/flutter test`
- PASS `cd flutter && /home/OBSiDIAN/tools/flutter/bin/flutter analyze`
23/05/26 07:30
- PASS `cd control-plane && gofmt -w internal/gateway/mux.go internal/gateway/bridge.go internal/gateway/mux_test.go internal/gateway/bridge_test.go`
- PASS `cd control-plane && CGO_ENABLED=0 GOTOOLCHAIN=local /usr/local/go/bin/go test ./...`
- PASS `cd control-plane && CGO_ENABLED=0 GOTOOLCHAIN=local /usr/local/go/bin/go build ./...`
- PASS `cd flutter && /home/OBSiDIAN/tools/flutter/bin/dart format lib/src/terminal/cortado_terminal.dart lib/src/terminal/terminal_platform.dart lib/src/terminal/terminal_platform_stub.dart lib/src/terminal/web/terminal_platform_web.dart test/cortado_terminal_test.dart lib/cortado.dart`
- PASS `cd flutter && /home/OBSiDIAN/tools/flutter/bin/flutter pub get`
- PASS `cd flutter && /home/OBSiDIAN/tools/flutter/bin/flutter test`
- PASS `cd flutter && /home/OBSiDIAN/tools/flutter/bin/flutter analyze`
23/05/26 07:31
- PASS `cd flutter && /home/OBSiDIAN/tools/flutter/bin/flutter test`
- PASS `cd flutter && /home/OBSiDIAN/tools/flutter/bin/flutter analyze`
23/05/26 07:44
- PASS `cd demo_app && /home/OBSiDIAN/tools/flutter/bin/flutter pub get`
- PASS `cd demo_app && /home/OBSiDIAN/tools/flutter/bin/flutter analyze`
- PASS `cd demo_app && /home/OBSiDIAN/tools/flutter/bin/flutter test`
- PASS `cd demo_app && /home/OBSiDIAN/tools/flutter/bin/flutter build web`
- PASS `cd flutter && /home/OBSiDIAN/tools/flutter/bin/flutter analyze`
- PASS `cd flutter && /home/OBSiDIAN/tools/flutter/bin/flutter test`
23/05/26 08:00
- PASS `cd control-plane && CGO_ENABLED=0 GOTOOLCHAIN=local /usr/local/go/bin/go test ./...`
- PASS `cd control-plane && CGO_ENABLED=0 GOTOOLCHAIN=local /usr/local/go/bin/go build ./...`
- PASS `docker build -f control-plane/Dockerfile -t cortado-control-plane:test .`
- PASS `terraform -chdir=terraform/envs/dev validate`
- PASS `terraform -chdir=terraform/envs/prod validate`
- PASS `terraform -chdir=terraform/envs/dev apply -auto-approve -target=null_resource.k8s_workspace_test_pod`
- PASS `kubectl -n cortado-workspaces get svc workspace-pod-test -o wide`
- PASS `kubectl wait --for=condition=Ready pod/workspace-pod-test -n cortado-workspaces --timeout=180s`
23/05/26 10:35
- PASS `cd agent && CGO_ENABLED=0 GOTOOLCHAIN=local /usr/local/go/bin/go test ./...`
- PASS `cd agent && CGO_ENABLED=0 GOTOOLCHAIN=local /usr/local/go/bin/go build ./...`
- PASS `docker build -t cortado-workspace:test agent/`
- PASS `cd control-plane && CGO_ENABLED=0 GOTOOLCHAIN=local /usr/local/go/bin/go test ./...`
- PASS `cd control-plane && CGO_ENABLED=0 GOTOOLCHAIN=local /usr/local/go/bin/go build ./...`
- PASS `terraform -chdir=terraform/envs/dev validate`
- PASS `terraform -chdir=terraform/envs/dev output -raw control_plane_service_uri`
- PASS `terraform -chdir=terraform/envs/dev apply -auto-approve -replace=null_resource.k8s_bootstrap -replace=null_resource.k8s_workspace_test_pod[0] -target=null_resource.k8s_bootstrap -target=null_resource.k8s_workspace_test_pod[0]`
- PASS `terraform -chdir=terraform/envs/dev apply -auto-approve -target=null_resource.k8s_workspace_test_pod[0]`
- PASS `terraform -chdir=terraform/envs/dev apply -auto-approve -target=module.cloudrun.google_cloud_run_v2_service.control_plane`
- PASS `kubectl wait --for=condition=Ready pod/workspace-pod-test -n cortado-workspaces --timeout=300s`
- PASS `cd control-plane && GOTOOLCHAIN=local /usr/local/go/bin/go run /tmp/cortado_ws_smoke.go`
23/05/26 11:02
- PASS `terraform -chdir=terraform/envs/dev import google_firestore_database.default 'projects/cortado-ide/databases/(default)'`
- PASS `terraform -chdir=terraform/envs/dev apply -auto-approve`
- PASS `/opt/google/chrome/chrome --version`
- PASS `cd control-plane && CGO_ENABLED=0 GOTOOLCHAIN=local /usr/local/go/bin/go test ./...`
- PASS `cd control-plane && CGO_ENABLED=0 GOTOOLCHAIN=local /usr/local/go/bin/go build ./...`
- PASS `terraform -chdir=terraform/envs/dev validate`
- PASS `docker build -f control-plane/Dockerfile -t us-central1-docker.pkg.dev/cortado-ide/cortado-dev/cortado-control-plane:20260523-110240-keepalivefix .`
- PASS `docker push us-central1-docker.pkg.dev/cortado-ide/cortado-dev/cortado-control-plane:20260523-110240-keepalivefix`
- PASS `terraform -chdir=terraform/envs/dev apply -auto-approve -target=module.cloudrun.google_cloud_run_v2_service.control_plane`
- PASS Chrome `demo_app` smoke against `https://cortado-control-plane-dev-dzozcgk63q-uc.a.run.app`: `echo hello_v0_1`, `python3`, `vim`, browser-driven resize (`tput cols` `100 -> 130`), browser RTT `~342 ms`
23/05/26 12:02
- PASS `cd control-plane && GOTOOLCHAIN=local /usr/local/go/bin/go mod tidy`
- PASS `cd control-plane && CGO_ENABLED=0 GOTOOLCHAIN=local /usr/local/go/bin/go test ./...`
- PASS `cd control-plane && CGO_ENABLED=0 GOTOOLCHAIN=local /usr/local/go/bin/go build ./...`
- PASS `docker build -f control-plane/Dockerfile -t cortado-control-plane:test .`
- PASS `terraform -chdir=terraform/envs/dev validate`
- PASS `terraform -chdir=terraform/envs/prod validate`
23/05/26 12:20
- PASS `cd proto && buf lint`
- PASS `cd proto && buf generate`
- PASS `cd agent && CGO_ENABLED=0 GOTOOLCHAIN=local /usr/local/go/bin/go test ./...`
- PASS `cd agent && CGO_ENABLED=0 GOTOOLCHAIN=local /usr/local/go/bin/go build ./...`
- PASS `cd control-plane && CGO_ENABLED=0 GOTOOLCHAIN=local /usr/local/go/bin/go test ./...`
- PASS `cd control-plane && CGO_ENABLED=0 GOTOOLCHAIN=local /usr/local/go/bin/go build ./...`
- PASS `docker build -t cortado-workspace:test agent/`
- PASS `docker build -f control-plane/Dockerfile -t cortado-control-plane:test .`
23/05/26 12:44
- PASS `cd flutter && flutter pub get`
- PASS `cd flutter && dart run build_runner build --delete-conflicting-outputs`
- PASS `cd flutter && flutter test`
- PASS `cd flutter && flutter analyze`
23/05/26 12:56
- PASS `cd flutter && flutter test`
- PASS `cd flutter && flutter analyze`
23/05/26 13:00
- PASS `terraform fmt -recursive terraform`
- PASS `terraform -chdir=terraform/envs/dev init -backend=false`
- PASS `terraform -chdir=terraform/envs/prod init -backend=false`
- PASS `terraform -chdir=terraform/envs/dev validate`
- PASS `terraform -chdir=terraform/envs/prod validate`
23/05/26 13:16
- PASS `cd proto && buf lint`
- PASS `cd agent && GOTOOLCHAIN=local go test ./...`
- PASS `cd agent && CGO_ENABLED=0 GOTOOLCHAIN=local go build ./...`
- PASS `cd control-plane && GOTOOLCHAIN=local go mod tidy`
- PASS `cd control-plane && GOTOOLCHAIN=local go test ./...`
- PASS `cd control-plane && CGO_ENABLED=0 GOTOOLCHAIN=local go build ./...`
- PASS `terraform -chdir=terraform/envs/dev init -backend=false`
- PASS `terraform -chdir=terraform/envs/dev validate`
- PASS `terraform -chdir=terraform/envs/prod init -backend=false`
- PASS `terraform -chdir=terraform/envs/prod validate`
23/05/26 13:33
- PASS `cd proto && buf lint`
- PASS `cd control-plane && gofmt -w cmd/server/main.go cmd/server/bootstrap.go internal/api/router.go internal/api/sessions.go internal/api/jwks.go internal/api/sessions_test.go internal/auth/model.go internal/auth/service.go internal/auth/service_test.go internal/store/firestore_auth_store.go`
- PASS `cd control-plane && GOTOOLCHAIN=local go mod tidy`
- PASS `cd control-plane && GOTOOLCHAIN=local go test ./...`
- PASS `cd control-plane && CGO_ENABLED=0 GOTOOLCHAIN=local go build ./...`
- PASS `terraform fmt -recursive terraform`
- PASS `terraform -chdir=terraform/envs/dev init -backend=false`
- PASS `terraform -chdir=terraform/envs/dev validate`
- PASS `terraform -chdir=terraform/envs/prod init -backend=false`
- PASS `terraform -chdir=terraform/envs/prod validate`
23/05/26 13:50
- PASS `cd control-plane && go test ./...`
- PASS `cd control-plane && CGO_ENABLED=0 go build ./...`
23/05/26 13:52
- PASS `cd control-plane && go test ./...`
- PASS `cd control-plane && CGO_ENABLED=0 go build ./...`
- PASS `cd flutter && flutter test`
- PASS `cd flutter && flutter analyze`
23/05/26 13:54
- PASS `cd flutter && flutter test test/auth_refresh_smoke_test.dart`
- PASS `cd flutter && flutter test`
- PASS `cd flutter && flutter analyze`
23/05/26 14:58
- PASS `cd flutter && flutter test test/cortado_auth_session_test.dart`
23/05/26 14:12
- PASS `cd proto && buf lint`
- PASS `cd proto && buf generate`
- PASS `cd agent && GOTOOLCHAIN=local go mod tidy`
- PASS `cd agent && GOTOOLCHAIN=local go test ./...`
- PASS `cd agent && CGO_ENABLED=0 GOTOOLCHAIN=local go build ./...`
23/05/26 14:19
- PASS `cd control-plane && GOTOOLCHAIN=local go test ./...`
- PASS `cd control-plane && CGO_ENABLED=0 GOTOOLCHAIN=local go build ./...`
23/05/26 14:33
- PASS `cd flutter && flutter pub get`
- PASS `cd flutter && dart run build_runner build --delete-conflicting-outputs`
- PASS `cd flutter && flutter test`
- PASS `cd flutter && flutter analyze`
23/05/26 14:38
- PASS `cd flutter && flutter test`
- PASS `cd flutter && flutter analyze`
23/05/26 14:47
- PASS `cd proto && buf lint`
- PASS `cd proto && buf generate`
- PASS `cd agent && GOTOOLCHAIN=local go test ./...`
- PASS `cd agent && CGO_ENABLED=0 GOTOOLCHAIN=local go build ./...`
- PASS `cd control-plane && GOTOOLCHAIN=local go test ./...`
- PASS `cd control-plane && CGO_ENABLED=0 GOTOOLCHAIN=local go build ./...`
23/05/26 15:02
- PASS `cd flutter && flutter test test/cortado_auth_session_test.dart test/workspace_manager_test.dart test/cortado_file_tree_test.dart`
- PASS `cd flutter && flutter test test/cortado_client_test.dart`
- PASS `cd flutter && flutter test`
- PASS `cd flutter && flutter analyze`
23/05/26 23:44
- PASS `cd demo_app/web && npm run build`
- PASS `cd flutter && dart run build_runner build --delete-conflicting-outputs`
- PASS `cd flutter && flutter test`
- PASS `cd flutter && flutter analyze`
- PASS `cd demo_app && /home/OBSiDIAN/tools/flutter/bin/flutter build web`
24/05/26 00:02
- PASS `cd control-plane && GOTOOLCHAIN=local go test ./internal/workspace`
- PASS `cd control-plane && GOTOOLCHAIN=local go test ./...`
24/05/26 05:03
- PASS `cd control-plane && go test ./...`
- PASS `cd control-plane && CGO_ENABLED=0 go build ./...`
- PASS `terraform -chdir=terraform/envs/dev init -backend=false`
- PASS `terraform -chdir=terraform/envs/prod init -backend=false`
- PASS `terraform -chdir=terraform/envs/dev validate`
- PASS `terraform -chdir=terraform/envs/prod validate`
24/05/26 05:30
- PASS `bash -n scripts/dev_portforward_deploy.sh scripts/dev_workspace.sh scripts/dev_portforward_probe.sh scripts/lib/cortado_dev_smoke.sh`
- PASS `./scripts/dev_portforward_deploy.sh --help`
- PASS `./scripts/dev_workspace.sh --help`
- PASS `./scripts/dev_portforward_probe.sh --help`
24/05/26 04:37
- PASS `cd daemon && go test ./...`
- PASS `cd daemon && CGO_ENABLED=0 go build ./...`
- PASS `cd flutter && dart run build_runner build --delete-conflicting-outputs`
- PASS `cd flutter && flutter test`
- PASS `cd flutter && flutter analyze`
24/05/26 05:00
- PASS `cd proto && buf lint`
25/05/26 06:39
- PASS `cd agent && CGO_ENABLED=0 go build ./...`
- PASS `cd control-plane && go test ./...`
- PASS `cd control-plane && CGO_ENABLED=0 go build ./...`
- PASS `cd proto && buf generate`
- PASS `cd agent && go test ./...`
- PASS `cd agent && CGO_ENABLED=0 go build ./...`
- PASS `cd flutter && flutter analyze`
24/05/26 01:45
- PASS `cd demo_app/web && npm test`
- PASS `cd demo_app/web && npm run build`
- PASS `cd flutter && dart run build_runner build --delete-conflicting-outputs`
- PASS `cd flutter && flutter test test/cortado_code_editor_test.dart`
- PASS `cd flutter && flutter analyze`
24/05/26 02:00
- PASS `bash -n scripts/rewrite_git_authorship.sh`
- PASS `scripts/rewrite_git_authorship.sh` (dry run only; no refs rewritten)
24/05/26 02:02
- PASS `cd indexer && PYTHONPATH=src python3 -m unittest discover -s tests`
- PASS `cd indexer && PYTHONPATH=src python3 -m cortado_indexer --help`
- PASS `docker build -t cortado-indexer:test indexer`
24/05/26 02:23
- PASS `cd indexer && PYTHONPATH=src python3 -m unittest discover -s tests`
- PASS `cd indexer && PYTHONPATH=src python3 -m cortado_indexer --help`
- PASS `docker build -t cortado-indexer:test indexer`
- PASS `docker run --rm -v "$PWD/indexer:/app" -w /app python:3.11-slim bash -lc "pip install --quiet . && python -m unittest discover -s tests"`
24/05/26 03:01
- PASS `cd indexer && PYTHONPATH=src python3 -m unittest discover -s tests`
- PASS `cd indexer && PYTHONPATH=src python3 -m cortado_indexer --help`
- PASS `docker run --rm -u $(id -u):$(id -g) -v "$PWD/indexer:/app" -w /app python:3.11-slim bash -lc "python -m venv /tmp/cortado-venv && . /tmp/cortado-venv/bin/activate && pip install --quiet . && python -m unittest discover -s tests"`
- PASS `docker build -t cortado-indexer:test indexer`
- PASS `cd control-plane && GOTOOLCHAIN=local go test ./internal/workspace`
- PASS `cd control-plane && CGO_ENABLED=0 GOTOOLCHAIN=local go build ./...`
24/05/26 01:15
- PASS `cd flutter && flutter test test/cortado_lsp_client_test.dart test/cortado_code_editor_test.dart`
- PASS `cd flutter && flutter analyze`
- PASS `cd flutter && flutter test`
24/05/26 01:22
- PASS `cd demo_app/web && npm test`
- PASS `cd demo_app/web && npm run build`
- PASS `cd flutter && flutter test`
- PASS `cd flutter && flutter analyze`
- PASS `cd control-plane && CGO_ENABLED=0 GOTOOLCHAIN=local go build ./...`
24/05/26 01:32
- PASS `cd demo_app/web && npm test`
- PASS `cd flutter && flutter test test/cortado_lsp_client_test.dart test/cortado_code_editor_test.dart test/cortado_file_tree_test.dart`
- PASS `cd flutter && flutter analyze`
24/05/26 00:11
- PASS `cd proto && buf lint`
- PASS `cd proto && buf generate`
- PASS `cd agent && GOTOOLCHAIN=local go test ./...`
- PASS `cd agent && CGO_ENABLED=0 GOTOOLCHAIN=local go build ./...`
- PASS `cd control-plane && GOTOOLCHAIN=local go test ./...`
- PASS `cd control-plane && CGO_ENABLED=0 GOTOOLCHAIN=local go build ./...`
- PASS `terraform fmt -recursive terraform`
- PASS `terraform -chdir=terraform/envs/dev init -backend=false -input=false`
- PASS `terraform -chdir=terraform/envs/dev validate`
- PASS `terraform -chdir=terraform/envs/prod init -backend=false -input=false`
- PASS `terraform -chdir=terraform/envs/prod validate`
24/05/26 00:42
- PASS `cd proto && buf lint`
- PASS `cd proto && buf generate`
- PASS `docker build --build-arg INCLUDE_DART_SDK=true -t cortado-workspace:test agent`
- PASS `cd agent && GOTOOLCHAIN=local go test ./...`
- PASS `cd agent && CGO_ENABLED=0 GOTOOLCHAIN=local go build ./...`
- PASS `cd control-plane && GOTOOLCHAIN=local go test ./...`
- PASS `cd control-plane && CGO_ENABLED=0 GOTOOLCHAIN=local go build ./...`
- PASS `cd flutter && flutter test test/workspace_manager_test.dart test/cortado_code_editor_test.dart test/cortado_file_tree_test.dart`
- PASS `cd flutter && flutter analyze`
24/05/26 03:06
- PASS `cd indexer && python3 -m unittest discover -s tests`
- PASS `docker build -t cortado-indexer:test indexer`
- PASS `docker run --rm --entrypoint python cortado-indexer:test -c "import cortado_indexer.updater_server, cortado_indexer.updater, cortado_indexer.qdrant; print('ok')"`
- PASS `terraform fmt -recursive terraform`
- PASS `terraform -chdir=terraform/envs/dev init -backend=false`
- PASS `terraform -chdir=terraform/envs/prod init -backend=false`
- PASS `terraform -chdir=terraform/envs/dev validate`
- PASS `terraform -chdir=terraform/envs/prod validate`
24/05/26 03:18
- PASS `cd control-plane && GOTOOLCHAIN=local go test ./...`
- PASS `cd control-plane && CGO_ENABLED=0 GOTOOLCHAIN=local go build ./...`
- PASS `terraform fmt -recursive terraform`
- PASS `terraform -chdir=terraform/envs/dev init -backend=false`
- PASS `terraform -chdir=terraform/envs/prod init -backend=false`
- PASS `terraform -chdir=terraform/envs/dev validate`
- PASS `terraform -chdir=terraform/envs/prod validate`
- PASS `cd flutter && dart format lib/cortado.dart lib/src/ai/cortado_ai_service.dart test/cortado_ai_service_test.dart`
- PASS `cd flutter && flutter test test/cortado_ai_service_test.dart`
- PASS `cd flutter && flutter analyze`
- PASS `cd flutter && flutter test`
24/05/26 03:38
- PASS `cd demo_app/web && npm test`
- PASS `cd demo_app/web && npm run build`
- PASS `cd flutter && flutter test test/cortado_code_editor_test.dart`
- PASS `cd flutter && flutter analyze`
24/05/26 03:49
- PASS `bash -n scripts/install_cortado_daemon.sh`
- PASS `cd daemon && go test ./...`
- PASS `cd daemon && CGO_ENABLED=0 go build ./...`
- PASS `terraform -chdir=terraform/envs/dev validate`
- PASS `terraform -chdir=terraform/envs/prod validate`
24/05/26 03:59
- PASS `cd proto && buf lint`
- PASS `cd proto && buf generate`
- PASS `cd control-plane && go test ./...`
- PASS `cd control-plane && CGO_ENABLED=0 go build ./...`
- PASS `cd agent && CGO_ENABLED=0 go build ./...`
24/05/26 04:10
- PASS `cd daemon && go test ./...`
- PASS `cd daemon && CGO_ENABLED=0 go build ./...`
24/05/26 04:22
- PASS `cd daemon && go test ./...`
- PASS `cd daemon && CGO_ENABLED=0 go build ./...`
25/05/26 05:42
- PASS `cd control-plane && GOTOOLCHAIN=local go mod tidy`
- PASS `cd control-plane && GOTOOLCHAIN=local go test ./internal/auth ./internal/api ./internal/middleware ./internal/store`
- PASS `cd control-plane && CGO_ENABLED=0 GOTOOLCHAIN=local go test ./...`
- PASS `cd control-plane && CGO_ENABLED=0 GOTOOLCHAIN=local go build ./...`
- PASS `cd agent && CGO_ENABLED=0 GOTOOLCHAIN=local go build ./...`
25/05/26 05:55
- PASS `cd demo_app && /home/OBSiDIAN/tools/flutter/bin/flutter pub get`
- PASS `cd demo_app && /home/OBSiDIAN/tools/flutter/bin/flutter analyze`
- PASS `cd demo_app && /home/OBSiDIAN/tools/flutter/bin/flutter test`
25/05/26 06:12
- PASS `cd control-plane && GOTOOLCHAIN=local go test ./internal/auth ./internal/api ./internal/middleware`
- PASS `cd control-plane && CGO_ENABLED=0 GOTOOLCHAIN=local go test ./...`
- PASS `cd control-plane && CGO_ENABLED=0 GOTOOLCHAIN=local go build ./...`
- PASS `cd agent && CGO_ENABLED=0 GOTOOLCHAIN=local go build ./...`
- PASS `cd demo_app && /home/OBSiDIAN/tools/flutter/bin/flutter analyze`
- PASS `cd demo_app && /home/OBSiDIAN/tools/flutter/bin/flutter test`
