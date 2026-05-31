#!/usr/bin/env bash
set -euo pipefail

arch="$(uname -m)"
case "$arch" in
x86_64) gcloud_arch="x86_64" ;;
aarch64 | arm64) gcloud_arch="arm" ;;
*)
    echo "Unsupported architecture: $arch" >&2
    exit 1
    ;;
esac

tmpdir="$(mktemp -d)"
curl -fsSL "https://dl.google.com/dl/cloudsdk/channels/rapid/downloads/google-cloud-cli-linux-${gcloud_arch}.tar.gz" \
    | tar -xz -C "$tmpdir"
mv "$tmpdir/google-cloud-sdk" /usr/local/
/usr/local/google-cloud-sdk/install.sh --quiet --usage-reporting false --command-completion false --path-update false
for bin in gcloud gsutil bq; do
    ln -sf "/usr/local/google-cloud-sdk/bin/$bin" "/usr/local/bin/$bin"
done
rm -rf "$tmpdir"
