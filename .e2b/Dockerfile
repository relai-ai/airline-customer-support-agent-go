FROM golang:1.24.5-bookworm

ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates curl git nodejs npm python3 python3-venv sudo && rm -rf /var/lib/apt/lists/*
RUN install -m 0755 -d /etc/apt/keyrings && curl -fsSL https://download.docker.com/linux/debian/gpg -o /etc/apt/keyrings/docker.asc && chmod a+r /etc/apt/keyrings/docker.asc && echo 'deb [arch=amd64 signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/debian bookworm stable' > /etc/apt/sources.list.d/docker.list && apt-get update && apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin && rm -rf /var/lib/apt/lists/*
RUN (id user >/dev/null 2>&1 || useradd --create-home --shell /bin/bash user) && usermod -aG docker user && echo 'user ALL=(ALL) NOPASSWD:ALL' > /etc/sudoers.d/playground
RUN ln -sf /usr/local/go/bin/go /usr/local/bin/go && ln -sf /usr/local/go/bin/gofmt /usr/local/bin/gofmt
ENV PATH=/home/user/.local/bin:/usr/local/go/bin:/usr/local/bin:/usr/bin:/bin
WORKDIR /home/user/airline

# Explicit allowlist: never copy .env, logs, .git, OAuth state or local RELAI config.
COPY go.mod go.sum ./
COPY airline/ ./airline/
COPY cmd/ ./cmd/
COPY benchmarks/ ./benchmarks/
COPY demo/checkpoint/ ./demo/checkpoint/
COPY start.sh README.md DEMO.md ./
COPY .devcontainer/setup.sh ./.devcontainer/setup.sh
COPY .e2b/start.sh /usr/local/bin/start-playground
RUN chmod 755 /usr/local/bin/start-playground && chown -R user:user /home/user/airline
USER user
RUN bash .devcontainer/setup.sh && go test ./... && go vet ./...
