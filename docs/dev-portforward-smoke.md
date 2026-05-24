# Dev Portforward Smoke Guide

This is the shortest reliable path to smoke-test the current dev stack after
Task `7.1.2`.

It covers:

- local verification
- deploying the dedicated `cortado-portforward` service to the dev stack
- creating a real workspace through the control-plane API
- opening a terminal into that workspace
- starting an HTTP server inside the workspace
- proving that the new portforward Cloud Run service can reach that server
- cleanup

## Prerequisites

You need these installed and already authenticated where applicable:

- `docker`
- `gcloud`
- `terraform`
- `curl`
- `jq`
- Flutter/Chrome if you want to use the in-repo terminal smoke harness

The scripts below write local helper env files into:

```bash
.tmp/portforward-smoke/
```

## Step 1: Final local checks

Run the exact checks that were used to close the gateway slice:

```bash
cd control-plane && go test ./...
cd control-plane && CGO_ENABLED=0 go build ./...
terraform -chdir=terraform/envs/dev init -backend=false
terraform -chdir=terraform/envs/dev validate
terraform -chdir=terraform/envs/prod init -backend=false
terraform -chdir=terraform/envs/prod validate
```

## Step 2: Deploy the dev portforward service

Build, push, and apply the new Cloud Run service:

```bash
./scripts/dev_portforward_deploy.sh
source ./.tmp/portforward-smoke/dev-env.sh
```

What this does:

- builds `control-plane/Dockerfile.portforward`
- pushes `cortado-portforward:<timestamp-tag>` to Artifact Registry
- runs `terraform apply` in `terraform/envs/dev` with that image tag
- saves:

```bash
.tmp/portforward-smoke/dev-env.sh
```

That file exports:

- `CORTADO_CONTROL_PLANE_URL`
- `CORTADO_PORTFORWARD_URL`
- `CORTADO_DEV_TOKEN`

## Step 3: Create a real workspace

Create a workspace using the current dev workspace image and wait until it is
`RUNNING`:

```bash
./scripts/dev_workspace.sh create
source ./.tmp/portforward-smoke/workspace-env.sh
```

If you need a specific image instead:

```bash
./scripts/dev_workspace.sh create us-central1-docker.pkg.dev/cortado-ide/<repo>/cortado-workspace:<tag>
```

Useful follow-up commands:

```bash
./scripts/dev_workspace.sh status
./scripts/dev_workspace.sh stop
./scripts/dev_workspace.sh start
./scripts/dev_workspace.sh delete
```

## Step 4: Open a terminal into the workspace

There is no plain HTTP command-exec endpoint yet, so this part is still manual.
Use the existing Flutter smoke harness:

```bash
cd demo_app
/home/OBSiDIAN/tools/flutter/bin/flutter run -d chrome
```

In the demo app UI:

1. Set `Base URL` to `$CORTADO_CONTROL_PLANE_URL`
2. Set `Workspace ID` to `$CORTADO_WORKSPACE_ID`
3. Leave the shell as `/bin/bash`
4. Press `Connect`

Quick terminal sanity checks:

```bash
echo hello_terminal
pwd
python3 --version
```

## Step 5: Start a preview server inside the workspace

Paste this into the workspace terminal:

```bash
mkdir -p /workspace/preview
cat >/workspace/preview/index.html <<'EOF'
<!doctype html>
<html>
  <head>
    <meta charset="utf-8" />
    <title>Cortado Portforward Smoke</title>
  </head>
  <body>
    <h1>Cortado portforward smoke</h1>
    <p>If you can read this through Cloud Run, Task 7.1.2 works.</p>
  </body>
</html>
EOF
python3 -m http.server 8080 --directory /workspace/preview
```

Leave that process running.

## Step 6: Probe the portforward gateway

From a second local terminal:

```bash
source ./.tmp/portforward-smoke/dev-env.sh
source ./.tmp/portforward-smoke/workspace-env.sh
./scripts/dev_portforward_probe.sh
```

You should get an HTTP `200` response and the HTML body from the workspace.

Explicit paths also work:

```bash
./scripts/dev_portforward_probe.sh "$CORTADO_WORKSPACE_ID" 8080 /index.html
```

The direct browser URL shape is:

```bash
${CORTADO_PORTFORWARD_URL}/${CORTADO_WORKSPACE_ID}/8080/index.html
```

## Step 7: Negative-path checks

Closed port should fail:

```bash
curl -i -H 'X-Cortado-Dev-Token: dev-bypass' \
  "${CORTADO_PORTFORWARD_URL}/${CORTADO_WORKSPACE_ID}/65530/"
```

Stopped workspace should fail:

```bash
./scripts/dev_workspace.sh stop
curl -i -H 'X-Cortado-Dev-Token: dev-bypass' \
  "${CORTADO_PORTFORWARD_URL}/${CORTADO_WORKSPACE_ID}/8080/"
./scripts/dev_workspace.sh start
./scripts/dev_workspace.sh wait-running
```

## Step 8: Cleanup

Delete the workspace when you are done:

```bash
./scripts/dev_workspace.sh delete
```

## Fastest Repeat Loop

Once the service is already deployed, the shortest repeat path is:

```bash
source ./.tmp/portforward-smoke/dev-env.sh
./scripts/dev_workspace.sh create
source ./.tmp/portforward-smoke/workspace-env.sh
```

Then:

1. connect via `demo_app`
2. run the `python3 -m http.server 8080 --directory /workspace/preview` command
3. run:

```bash
./scripts/dev_portforward_probe.sh
```

4. delete the workspace:

```bash
./scripts/dev_workspace.sh delete
```
