# Terraform Bootstrap

Terraform state is stored in a GCS bucket. That bucket must be created once by hand before the first `terraform init` because Terraform cannot manage its own backend bucket without a bootstrap step.

## One-time bootstrap

Run the bootstrap script once to create the dev state bucket:

```bash
./scripts/bootstrap.sh
```

That script runs:

```bash
gcloud storage buckets create gs://cortado-tf-state-dev \
  --location=us-central1 \
  --uniform-bucket-level-access
```

After the bucket exists, initialize Terraform from the matching environment directory as usual.

## Kubernetes credentials note

The `null_resource` pattern used for early Kubernetes bootstrap runs `gcloud container clusters get-credentials` before `kubectl apply`. When Terraform runs inside the devcontainer, that updates the devcontainer's `~/.kube/config`, not the host machine's kubeconfig. If `kubectl` works in the devcontainer but not on the host, refresh credentials in the environment where you plan to run the command.
