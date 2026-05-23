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
