# syntax=docker/dockerfile:1
# renovate: datasource=docker depName=alpine versioning=semver
FROM alpine:3.23

# renovate: datasource=golang-version depName=golang packageName=golang
ARG GO_VERSION=1.26.0
# renovate: datasource=github-releases depName=opentofu/opentofu
ARG TOFU_VERSION=1.12.1
# renovate: datasource=github-releases depName=go-task/task
ARG TASK_VERSION=3.43.3
# renovate: datasource=github-releases depName=vmware/govmomi
ARG GOVC_VERSION=0.46.3
# renovate: datasource=github-releases depName=terraform-linters/tflint
ARG TFLINT_VERSION=0.62.1
# renovate: datasource=github-releases depName=golangci/golangci-lint
ARG GOLANGCI_LINT_VERSION=2.12.2
# renovate: datasource=github-releases depName=hadolint/hadolint
ARG HADOLINT_VERSION=2.13.1
# renovate: datasource=pypi depName=pre-commit
ARG PRE_COMMIT_VERSION=4.6.0

ARG DEVCONTAINER_USER=vscode
ARG DEVCONTAINER_UID=1000
ARG DEVCONTAINER_GID=1000

SHELL ["/bin/ash", "-o", "pipefail", "-c"]

RUN apk add --no-cache \
	bash \
	ca-certificates \
	curl \
	git \
	github-cli \
	gcompat \
	libc6-compat \
	openssh-client \
	python3 \
	py3-pip \
	shadow \
	sudo \
	wireguard-tools \
	&& pip install --break-system-packages --no-cache-dir "pre-commit==${PRE_COMMIT_VERSION}"

RUN set -eux; \
	arch="$(uname -m)"; \
	case "$arch" in \
	x86_64) goarch="amd64"; govc_arch="x86_64"; gcloud_arch="x86_64"; hadolint_arch="x86_64" ;; \
	aarch64) goarch="arm64"; govc_arch="arm64"; gcloud_arch="arm"; hadolint_arch="arm64" ;; \
	*) echo "unsupported architecture: $arch" >&2; exit 1 ;; \
	esac; \
	curl -fsSL "https://go.dev/dl/go${GO_VERSION}.linux-${goarch}.tar.gz" \
	| tar -C /usr/local -xz; \
	curl -fsSL "https://github.com/opentofu/opentofu/releases/download/v${TOFU_VERSION}/tofu_${TOFU_VERSION}_linux_${goarch}.tar.gz" \
	| tar -xz -C /usr/local/bin tofu; \
	curl -fsSL "https://github.com/go-task/task/releases/download/v${TASK_VERSION}/task_linux_${goarch}.tar.gz" \
	| tar -xz -C /usr/local/bin task; \
	curl -fsSL "https://github.com/vmware/govmomi/releases/download/v${GOVC_VERSION}/govc_Linux_${govc_arch}.tar.gz" \
	| tar -xz -C /usr/local/bin govc; \
	export TFLINT_VERSION="v${TFLINT_VERSION}"; \
	curl -fsSL https://raw.githubusercontent.com/terraform-linters/tflint/master/install_linux.sh | bash; \
	curl -fsSL https://golangci-lint.run/install.sh \
	| sh -s -- -b /usr/local/bin "v${GOLANGCI_LINT_VERSION}"; \
	curl -fsSL "https://github.com/hadolint/hadolint/releases/download/v${HADOLINT_VERSION}/hadolint-Linux-${hadolint_arch}" \
	-o /usr/local/bin/hadolint; \
	chmod +x /usr/local/bin/hadolint; \
	tmpdir="$(mktemp -d)"; \
	curl -fsSL "https://dl.google.com/dl/cloudsdk/channels/rapid/downloads/google-cloud-cli-linux-${gcloud_arch}.tar.gz" \
	| tar -xz -C "$tmpdir"; \
	mv "$tmpdir/google-cloud-sdk" /opt/google-cloud-sdk; \
	/opt/google-cloud-sdk/install.sh --quiet --usage-reporting false --command-completion false --path-update false; \
	ln -sf /opt/google-cloud-sdk/bin/gcloud /usr/local/bin/gcloud; \
	ln -sf /opt/google-cloud-sdk/bin/gsutil /usr/local/bin/gsutil; \
	ln -sf /opt/google-cloud-sdk/bin/bq /usr/local/bin/bq; \
	rm -rf "$tmpdir"

ENV PATH="/usr/local/go/bin:/usr/local/bin:${PATH}"

RUN addgroup -g "${DEVCONTAINER_GID}" "${DEVCONTAINER_USER}" \
	&& adduser -D -u "${DEVCONTAINER_UID}" -G "${DEVCONTAINER_USER}" -s /bin/bash "${DEVCONTAINER_USER}" \
	&& echo "${DEVCONTAINER_USER} ALL=(ALL) NOPASSWD:ALL" > "/etc/sudoers.d/${DEVCONTAINER_USER}" \
	&& chmod 0440 "/etc/sudoers.d/${DEVCONTAINER_USER}"

USER ${DEVCONTAINER_USER}
WORKDIR /home/${DEVCONTAINER_USER}
