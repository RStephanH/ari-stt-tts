#!/usr/bin/env bash
set -euo pipefail

# Load ARI_* (and DEEPGRAM_API_KEY, etc.) from .env, without printing it.
if [ -f .env ]; then
	set -a
	source .env
	set +a
fi

if [ -z "${ARI_URL:-}" ]; then
	echo "ARI_URL not set — copy env.example to .env and fill it in." >&2
	exit 1
fi

go test ./internal/ariutil/... -run TestNewARIClient_Integration -v
