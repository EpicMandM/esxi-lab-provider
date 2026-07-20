#!/usr/bin/env bash
set -euo pipefail

SSH_KEY="${SSH_KEY:-$HOME/.ssh/id_rsa_esxi_lab}"
USER="${DEPLOY_USER:-zhukov}"
HOST="${DEPLOY_HOST:-172.17.17.8}"
PORT="${DEPLOY_PORT:-22}"
SSH=(ssh -i "$SSH_KEY" -o StrictHostKeyChecking=no -o BatchMode=yes -o ConnectTimeout=10 -p "$PORT" "${USER}@${HOST}")
SSH_TTY=(ssh -i "$SSH_KEY" -o StrictHostKeyChecking=no -p "$PORT" -t "${USER}@${HOST}")

if "${SSH[@]}" "sudo -n true" 2>/dev/null; then
	echo "Passwordless sudo OK (${USER}@${HOST})"
	exit 0
fi

if [[ "${CI:-}" == "true" || -n "${GITHUB_ACTIONS:-}" || -n "${GITLAB_CI:-}" ]]; then
	echo "Passwordless sudo not configured on ${USER}@${HOST} (required in CI)" >&2
	exit 1
fi

echo "Configuring passwordless sudo for ${USER} (sudo password may be required)..."
"${SSH_TTY[@]}" "echo '${USER} ALL=(ALL) NOPASSWD: ALL' | sudo tee /etc/sudoers.d/${USER} > /dev/null && sudo chmod 0440 /etc/sudoers.d/${USER}"

if ! "${SSH[@]}" "sudo -n true" 2>/dev/null; then
	echo "Passwordless sudo still failing after setup" >&2
	exit 1
fi

echo "Passwordless sudo OK (${USER}@${HOST})"
