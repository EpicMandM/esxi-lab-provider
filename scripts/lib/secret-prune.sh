# Shared Secret Manager version pruning.
# Disabled versions still count toward the free-tier active-version quota —
# only DESTROYED versions are free. Keep the newest N non-destroyed versions.

# prune_secret_versions SECRET_ID [KEEP]
# Requires: PROJECT set; gcloud authenticated.
prune_secret_versions() {
	local secret_id="$1"
	local keep="${2:-${KEEP_SECRET_VERSIONS:-1}}"
	local versions=()
	local v i

	if [[ -z "${PROJECT:-}" ]]; then
		echo "prune_secret_versions: PROJECT is unset" >&2
		return 1
	fi
	if ! [[ "$keep" =~ ^[0-9]+$ ]] || [[ "$keep" -lt 1 ]]; then
		echo "prune_secret_versions: KEEP must be >= 1 (got: $keep)" >&2
		return 1
	fi

	if ! gcloud secrets describe "$secret_id" --project="$PROJECT" &>/dev/null; then
		echo "Secret $secret_id not found in $PROJECT — skip prune"
		return 0
	fi

	mapfile -t versions < <(
		gcloud secrets versions list "$secret_id" \
			--project="$PROJECT" \
			--filter="state!=DESTROYED" \
			--format="value(name)" \
			--sort-by="~createTime" 2>/dev/null || true
	)

	if [[ "${#versions[@]}" -le "$keep" ]]; then
		echo "Secret $secret_id: ${#versions[@]} version(s) (keep=$keep) — nothing to prune"
		return 0
	fi

	i=0
	for v in "${versions[@]}"; do
		i=$((i + 1))
		if [[ "$i" -le "$keep" ]]; then
			continue
		fi
		echo "Destroying projects/$PROJECT/secrets/$secret_id/versions/$v (keep newest $keep)"
		gcloud secrets versions destroy "$v" \
			--secret="$secret_id" \
			--project="$PROJECT" \
			--quiet
	done
}
