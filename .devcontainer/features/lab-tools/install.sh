#!/usr/bin/env bash
set -euo pipefail

export DEBIAN_FRONTEND=noninteractive

apt-get update
apt-get install -y --no-install-recommends wireguard-tools curl
rm -rf /var/lib/apt/lists/*

GOVC_VERSION="0.46.3"
curl -fsSL "https://github.com/vmware/govmomi/releases/download/v${GOVC_VERSION}/govc_Linux_x86_64.tar.gz" \
	| tar -xz -C /usr/local/bin govc

curl -fsSL https://raw.githubusercontent.com/terraform-linters/tflint/master/install_linux.sh | bash
