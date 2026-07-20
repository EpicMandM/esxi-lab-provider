#!/usr/bin/env bash
# Forward a local TCP port to ESXi (or another host) reachable via WireGuard (wg0).
# With devcontainer --network=host, 127.0.0.0.1 in the container is the host loopback.
set -euo pipefail

REMOTE_HOST="${REMOTE_HOST:-172.17.17.29}"
REMOTE_PORT="${REMOTE_PORT:-443}"
LOCAL_BIND="${LOCAL_BIND:-127.0.0.1}"
LOCAL_PORT="${LOCAL_PORT:-10443}"
PIDFILE="/tmp/esxi-lab-tunnel-${LOCAL_PORT}.pid"
WG_CONF="${WG_CONF:-/etc/wireguard/wg0.conf}"

ensure_wg() {
	if ip link show wg0 &>/dev/null; then
		return 0
	fi
	ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
	if [[ -x "$ROOT/scripts/apply-wg0.sh" ]] && WG_CONF="$WG_CONF" "$ROOT/scripts/apply-wg0.sh"; then
		return 0
	fi
	if [[ ! -f "$WG_CONF" ]]; then
		echo "WireGuard config not found: $WG_CONF" >&2
		echo "Run task secrets:push then task wg:apply, or copy your lab .conf to $WG_CONF." >&2
		exit 1
	fi
	echo "Bringing up wg0 from $WG_CONF ..."
	sudo wg-quick up wg0
}

start() {
	ensure_wg
	if [[ -f "$PIDFILE" ]]; then
		pid="$(cat "$PIDFILE")"
		if kill -0 "$pid" 2>/dev/null; then
			echo "Tunnel already running (pid $pid) -> ${REMOTE_HOST}:${REMOTE_PORT} on ${LOCAL_BIND}:${LOCAL_PORT}"
			exit 0
		fi
		rm -f "$PIDFILE"
	fi
	socat "TCP-LISTEN:${LOCAL_PORT},bind=${LOCAL_BIND},fork,reuseaddr" "TCP:${REMOTE_HOST}:${REMOTE_PORT}" &
	echo $! >"$PIDFILE"
	echo "Tunnel started (pid $(cat "$PIDFILE"))"
	echo "  https://${LOCAL_BIND}:${LOCAL_PORT}/  ->  ${REMOTE_HOST}:${REMOTE_PORT}"
	echo "  ESXi SDK: https://${LOCAL_BIND}:${LOCAL_PORT}/sdk"
}

stop() {
	if [[ ! -f "$PIDFILE" ]]; then
		echo "No tunnel pid file ($PIDFILE)"
		exit 0
	fi
	pid="$(cat "$PIDFILE")"
	if kill -0 "$pid" 2>/dev/null; then
		kill "$pid"
		echo "Stopped tunnel (pid $pid)"
	else
		echo "Tunnel process $pid not running"
	fi
	rm -f "$PIDFILE"
}

status() {
	if [[ -f "$PIDFILE" ]] && kill -0 "$(cat "$PIDFILE")" 2>/dev/null; then
		echo "running (pid $(cat "$PIDFILE")) -> ${REMOTE_HOST}:${REMOTE_PORT} on ${LOCAL_BIND}:${LOCAL_PORT}"
		wg show wg0 2>/dev/null | sed -n '1,6p' || true
	else
		echo "not running"
		exit 1
	fi
}

cmd="${1:-start}"
case "$cmd" in
start) start ;;
stop) stop ;;
status) status ;;
*) echo "Usage: $0 {start|stop|status}" >&2; exit 1 ;;
esac
