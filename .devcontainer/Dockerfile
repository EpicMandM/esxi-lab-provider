# syntax=docker/dockerfile:1
# renovate: datasource=docker depName=debian versioning=semver
FROM debian:stable-slim

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
# renovate: datasource=github-releases depName=cli/cli
ARG GH_VERSION=2.93.0
# renovate: datasource=pypi depName=pre-commit
ARG PRE_COMMIT_VERSION=4.6.0

ARG DEVCONTAINER_USER=vscode
ARG DEVCONTAINER_UID=1000
ARG DEVCONTAINER_GID=1000

SHELL ["/bin/bash", "-o", "pipefail", "-c"]

# hadolint ignore=DL3008
RUN set -eux; \
	apt-get update; \
	DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
		ca-certificates \
		curl \
		git \
		iproute2 \
		libstdc++6 \
		openssh-client \
		python3 \
		python3-pip \
		socat \
		sudo \
		wireguard-tools \
	; \
	rm -rf /var/lib/apt/lists/*; \
	pip3 install --break-system-packages --no-cache-dir "pre-commit==${PRE_COMMIT_VERSION}";

RUN set -eux; \
	curl -fsSL "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz" \
	| tar -C /usr/local -xz; \
	curl -fsSL "https://github.com/opentofu/opentofu/releases/download/v${TOFU_VERSION}/tofu_${TOFU_VERSION}_linux_amd64.tar.gz" \
	| tar -xz -C /usr/local/bin tofu; \
	curl -fsSL "https://github.com/go-task/task/releases/download/v${TASK_VERSION}/task_linux_amd64.tar.gz" \
	| tar -xz -C /usr/local/bin task; \
	curl -fsSL "https://github.com/vmware/govmomi/releases/download/v${GOVC_VERSION}/govc_Linux_x86_64.tar.gz" \
	| tar -xz -C /usr/local/bin govc; \
	curl -fsSL "https://github.com/golangci/golangci-lint/releases/download/v${GOLANGCI_LINT_VERSION}/golangci-lint-${GOLANGCI_LINT_VERSION}-linux-amd64.tar.gz" \
	| tar -xz -C /usr/local/bin --strip-components=1 "golangci-lint-${GOLANGCI_LINT_VERSION}-linux-amd64/golangci-lint"; \
	curl -fsSL "https://github.com/hadolint/hadolint/releases/download/v${HADOLINT_VERSION}/hadolint-Linux-x86_64" \
	-o /usr/local/bin/hadolint; \
	chmod +x /usr/local/bin/hadolint; \
	curl -fsSL "https://github.com/cli/cli/releases/download/v${GH_VERSION}/gh_${GH_VERSION}_linux_amd64.tar.gz" \
	| tar -xz -C /usr/local/bin --strip-components=2 "gh_${GH_VERSION}_linux_amd64/bin/gh"; \
	tmpdir="$(mktemp -d)"; \
	curl -fsSL "https://github.com/terraform-linters/tflint/releases/download/v${TFLINT_VERSION}/tflint_linux_amd64.zip" \
		-o "${tmpdir}/tflint.zip"; \
	python3 -m zipfile -e "${tmpdir}/tflint.zip" "${tmpdir}"; \
	install -m 0755 "${tmpdir}/tflint" /usr/local/bin/tflint; \
	curl -fsSL "https://dl.google.com/dl/cloudsdk/channels/rapid/downloads/google-cloud-cli-linux-x86_64.tar.gz" \
	| tar -xz -C "${tmpdir}"; \
	mv "${tmpdir}/google-cloud-sdk" /opt/google-cloud-sdk; \
	/opt/google-cloud-sdk/install.sh --quiet --usage-reporting false --command-completion false --path-update false; \
	ln -sf /opt/google-cloud-sdk/bin/gcloud /usr/local/bin/gcloud; \
	ln -sf /opt/google-cloud-sdk/bin/gsutil /usr/local/bin/gsutil; \
	ln -sf /opt/google-cloud-sdk/bin/bq /usr/local/bin/bq; \
	rm -rf "${tmpdir}";

ENV PATH="/usr/local/go/bin:/usr/local/bin:${PATH}"

RUN groupadd -g "${DEVCONTAINER_GID}" "${DEVCONTAINER_USER}" \
	&& useradd -l -m -u "${DEVCONTAINER_UID}" -g "${DEVCONTAINER_GID}" -s /bin/bash "${DEVCONTAINER_USER}" \
	&& echo "${DEVCONTAINER_USER} ALL=(ALL) NOPASSWD:ALL" > "/etc/sudoers.d/${DEVCONTAINER_USER}" \
	&& chmod 0440 "/etc/sudoers.d/${DEVCONTAINER_USER}"

USER ${DEVCONTAINER_USER}
WORKDIR /home/${DEVCONTAINER_USER}
