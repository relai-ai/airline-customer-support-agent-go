FROM golang:1.24.5-bookworm
ARG TARGETARCH

ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates curl git nodejs npm python3 python3-venv sudo && rm -rf /var/lib/apt/lists/*
RUN install -m 0755 -d /etc/apt/keyrings && curl -fsSL https://download.docker.com/linux/debian/gpg -o /etc/apt/keyrings/docker.asc && chmod a+r /etc/apt/keyrings/docker.asc && echo "deb [arch=${TARGETARCH} signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/debian bookworm stable" > /etc/apt/sources.list.d/docker.list && apt-get update && apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin && rm -rf /var/lib/apt/lists/*
RUN (id user >/dev/null 2>&1 || useradd --create-home --shell /bin/bash user) && usermod -aG docker user && echo 'user ALL=(ALL) NOPASSWD:ALL' > /etc/sudoers.d/playground
RUN ln -sf /usr/local/go/bin/go /usr/local/bin/go && ln -sf /usr/local/go/bin/gofmt /usr/local/bin/gofmt
ENV PATH=/home/user/.local/bin:/usr/local/go/bin:/usr/local/bin:/usr/bin:/bin
ARG CODE_SERVER_VERSION=4.141.0
RUN case "${TARGETARCH}:${CODE_SERVER_VERSION}" in \
      amd64:4.141.0) checksum=65bcd20b524d03eb1b5adb7f360244599ef0e5b74089a46a982d32c074b97a29 ;; \
      arm64:4.141.0) checksum=e5cc78da1d631c6cf6162f9b18eeaa8ee9d3239e1a88bb99d0602cdc5fa1852d ;; \
      *) echo 'Unsupported code-server version or architecture' >&2; exit 1 ;; \
    esac && curl -fsSL "https://github.com/coder/code-server/releases/download/v${CODE_SERVER_VERSION}/code-server-${CODE_SERVER_VERSION}-linux-${TARGETARCH}.tar.gz" -o /tmp/code-server.tgz && echo "${checksum}  /tmp/code-server.tgz" | sha256sum -c - && mkdir -p /opt/code-server && tar -xzf /tmp/code-server.tgz --strip-components=1 -C /opt/code-server && ln -s /opt/code-server/bin/code-server /usr/local/bin/code-server && rm /tmp/code-server.tgz
COPY .e2b/ide/start-ide.py /opt/playground/start-ide.py
COPY .e2b/ide/extension/ /opt/playground/extensions/relai.relai-playground-0.1.0/
RUN chmod -R a+rX /opt/playground && chown -R user:user /opt/playground/extensions
WORKDIR /home/user/airline

# Explicit allowlist: never copy .env, logs, .git, OAuth state or local RELAI config.
COPY go.mod go.sum ./
COPY airline/ ./airline/
COPY cmd/ ./cmd/
COPY benchmarks/ ./benchmarks/
COPY demo/checkpoint/ ./demo/checkpoint/
COPY demo/challenges/ ./demo/challenges/
# Public RELAI test definitions only. Never include local auth/config or run state.
COPY .relai/harbor/ ./.relai/harbor/
COPY .relai/evals-wiki/ ./.relai/evals-wiki/
COPY .relai/runtime-manifest.json ./.relai/runtime-manifest.json
COPY .relai/.internal/records/ ./.relai/.internal/records/
COPY .relai/.internal/layout.json ./.relai/.internal/layout.json
COPY .gitignore start.sh README.md DEMO.md ./
COPY .devcontainer/setup.sh ./.devcontainer/setup.sh
COPY .e2b/start.sh /usr/local/bin/start-playground
RUN chmod 755 /usr/local/bin/start-playground && chown -R user:user /home/user/airline
USER user
RUN bash .devcontainer/setup.sh && go test ./... && go vet ./...
