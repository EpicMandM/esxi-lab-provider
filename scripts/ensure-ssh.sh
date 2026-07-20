#!/usr/bin/env bash
set -euo pipefail

SSH_KEY="${SSH_KEY:-$HOME/.ssh/id_rsa_esxi_lab}"
USER="${DEPLOY_USER:-zhukov}"
HOST="${DEPLOY_HOST:-172.17.17.8}"
PORT="${DEPLOY_PORT:-22}"
SSH_BASE=(ssh -i "$SSH_KEY" -o StrictHostKeyChecking=no -o BatchMode=yes -o ConnectTimeout=10 -p "$PORT")

mkdir -p "$(dirname "$SSH_KEY")"
chmod 700 "$(dirname "$SSH_KEY")" 2>/dev/null || true

if [[ ! -f "$SSH_KEY" ]]; then
	echo "Generating SSH key at $SSH_KEY ..."
	ssh-keygen -t rsa -b 4096 -f "$SSH_KEY" -N "" -C "esxi-lab-deployment"
fi

if "${SSH_BASE[@]}" "${USER}@${HOST}" "true" 2>/dev/null; then
	echo "SSH access OK (${USER}@${HOST}:${PORT})"
	exit 0
fi

if [[ "${CI:-}" == "true" || -n "${GITHUB_ACTIONS:-}" || -n "${GITLAB_CI:-}" ]]; then
	echo "SSH to ${USER}@${HOST}:${PORT} failed in CI (BatchMode). Install the deploy public key on the host." >&2
	exit 1
fi

echo "SSH key not authorized on ${USER}@${HOST} — running ssh-copy-id (password may be required)..."
ssh-copy-id -i "${SSH_KEY}.pub" -o StrictHostKeyChecking=no -p "$PORT" "${USER}@${HOST}"

if ! "${SSH_BASE[@]}" "${USER}@${HOST}" "true" 2>/dev/null; then
	echo "SSH still failing after ssh-copy-id" >&2
	exit 1
fi

echo "SSH access OK (${USER}@${HOST}:${PORT})"
