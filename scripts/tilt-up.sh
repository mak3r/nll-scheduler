#!/usr/bin/env bash
# tilt-up.sh — macOS wrapper for 'tilt up' when using Podman
#
# Sets up an SSH tunnel from a local Unix socket to the Podman machine's
# Docker-compatible socket, then launches Tilt. The tunnel is torn down
# when Tilt exits.
#
# Usage: ./scripts/tilt-up.sh [tilt args...]

set -euo pipefail

SOCKET=/tmp/podman.sock
IDENTITY=/Users/$(whoami)/.local/share/containers/podman/machine/machine

# Ensure the Podman machine is running
if ! podman machine list --format '{{.Running}}' | grep -q true; then
  echo "Starting Podman machine..."
  podman machine start
fi

PORT=$(podman machine inspect --format '{{.SSHConfig.Port}}')

# Kill any existing tunnel using this socket
rm -f "$SOCKET"

# Start SSH tunnel in background; suppress known_hosts to avoid stale entries
ssh -fNT \
  -i "$IDENTITY" \
  -L "$SOCKET":/run/user/501/podman/podman.sock \
  -o StrictHostKeyChecking=no \
  -o UserKnownHostsFile=/dev/null \
  -o LogLevel=ERROR \
  -p "$PORT" \
  core@127.0.0.1

export DOCKER_HOST="unix://$SOCKET"
export DOCKER_BUILDKIT=0  # Podman does not expose a BuildKit gRPC endpoint

echo "Podman tunnel established on $SOCKET"

# Run Tilt, passing through any extra arguments
tilt up "$@"

# Tear down the tunnel when Tilt exits
SSH_PID=$(lsof -t -c ssh -a -U "$SOCKET" 2>/dev/null || true)
if [ -n "$SSH_PID" ]; then
  kill "$SSH_PID" 2>/dev/null || true
fi
rm -f "$SOCKET"
