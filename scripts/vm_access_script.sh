#!/usr/bin/env bash
# Helper commands for accessing the Cortado VM.
# Use the GCE path for the first login, then switch to Tailscale SSH.

set -euo pipefail

INSTANCE_NAME="cortado-dev-vm1"
ZONE="us-central1-a"
PROJECT_ID="cortado-ide"
USER="OBSiDIAN"

cat <<EOF
Initial GCE access:
  gcloud compute ssh ${INSTANCE_NAME} --zone=${ZONE} --project=${PROJECT_ID}

After Tailscale is connected on the VM:
  ssh ${USER}@<vm-tailscale-hostname-or-ip>

If MagicDNS is enabled, the hostname is usually the VM name:
  ssh ${USER}@${INSTANCE_NAME}

To print the VM's Tailscale IP from inside the VM:
  tailscale ip -4
EOF
# gcloud compute instances start cortado-dev-vm1
# gcloud compute instances stop cortado-dev-vm1
# gcloud compute ssh cortado-dev-vm1 --zone=us-central1-a --project=cortado-ide