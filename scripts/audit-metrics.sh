#!/usr/bin/env bash
set -euo pipefail
PROJECT="${GCP_PROJECT:-exsi-chat-app-478319}"
TOKEN="$(gcloud auth print-access-token)"
START="$(date -u -d '7 days ago' +%Y-%m-%dT%H:%M:%SZ)"
NOW="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
OUT_DIR="${1:-/tmp/metrics-audit}"
mkdir -p "$OUT_DIR"

gcloud monitoring dashboards describe \
	projects/132386503292/dashboards/2b8968e9-901a-43a2-ba47-451633c68dd2 \
	--project="$PROJECT" --format=json >"$OUT_DIR/dash.json"

python3 - "$OUT_DIR/dash.json" <<'PY'
import json, re, sys
d = json.load(open(sys.argv[1]))
print("displayName:", d.get("displayName"))
tiles = d.get("mosaicLayout", {}).get("tiles", [])
print("tiles:", len(tiles))
for i, t in enumerate(tiles):
    w = t.get("widget", {})
    text = json.dumps(w)
    filters = re.findall(r'metric\.type=\\"([^\\"]+)\\"', text)
    if not filters:
        filters = re.findall(r'metric\.type="([^"]+)"', text)
    aligners = re.findall(r"ALIGN_[A-Z0-9_]+", text)
    reducers = re.findall(r"REDUCE_[A-Z0-9_]+", text)
    print(f"--- tile {i} title={w.get('title')}")
    print("  filters:", filters)
    print("  aligners:", aligners, "reducers:", reducers)
    print("  type:", [k for k in w if k != "title"])
PY

METRICS=(
	lab.run.total
	lab.run.duration
	lab.vm.restore.total
	lab.vm.restore.duration
	lab.calendar.events.active
	lab.calendar.fetch.duration
	lab.email.send.total
	lab.password.rotation.total
	lab.wireguard.key.rotation.total
	lab.wireguard.peer.registration.total
	lab.vm.inventory.total
)

echo "=== TIME SERIES (7d) ==="
for m in "${METRICS[@]}"; do
	curl -s -H "Authorization: Bearer $TOKEN" -H "x-goog-user-project: $PROJECT" \
		--get "https://monitoring.googleapis.com/v3/projects/$PROJECT/timeSeries" \
		--data-urlencode "filter=metric.type=\"workload.googleapis.com/$m\"" \
		--data-urlencode "interval.startTime=$START" \
		--data-urlencode "interval.endTime=$NOW" \
		--data-urlencode "pageSize=3" \
		-o "$OUT_DIR/ts-$m.json"
	python3 - "$OUT_DIR/ts-$m.json" "$m" <<'PY'
import json, sys
d = json.load(open(sys.argv[1]))
m = sys.argv[2]
ts = d.get("timeSeries", [])
err = d.get("error", {})
print(f"{m}: series={len(ts)} err={err.get('message','')[:90]}")
for t in ts:
    labels = t.get("metric", {}).get("labels")
    kind = t.get("metricKind")
    pts = t.get("points", [])
    print(f"  labels={labels} kind={kind}")
    for p in pts[:2]:
        print(f"    {p.get('interval', {}).get('endTime')} {p.get('value')}")
PY
done
