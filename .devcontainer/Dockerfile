FROM mcr.microsoft.com/devcontainers/go:1-1.24-bookworm

# Yarn is unused here. Its expired repository key otherwise prevents Features
# from running apt-get update. Keep signature verification enabled everywhere.
RUN rm -f /etc/apt/sources.list.d/yarn.list
