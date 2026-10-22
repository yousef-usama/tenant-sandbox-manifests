#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

echo "Generating architecture diagram with Python Diagrams & Graphviz..."

if command -v docker >/dev/null 2>&1 && docker info >/dev/null 2>&1; then
  # Build builder image if not present
  if ! docker image inspect diagrams:local >/dev/null 2>&1; then
    echo "Building local diagrams image..."
    docker build -t diagrams:local -f - "$REPO_ROOT" <<'EOF'
FROM python:3.11-alpine
RUN apk add --no-cache graphviz ttf-freefont
RUN pip install --no-cache-dir diagrams
WORKDIR /work
EOF
  fi
  docker run --rm -v "$REPO_ROOT":/work diagrams:local python /work/scripts/generate_architecture_diagram.py
else
  # Native fallback if python and graphviz are installed locally
  python3 "$REPO_ROOT/scripts/generate_architecture_diagram.py"
fi

echo "Diagram successfully generated at: $REPO_ROOT/docs/architecture-overview.png"
