#!/usr/bin/env bash
set -euo pipefail

# Load DEEPGRAM_API_KEY (and anything else) from .env, without printing it.
if [ -f .env ]; then
	set -a
	source .env
	set +a
fi

if [ -z "${DEEPGRAM_API_KEY:-}" ]; then
	echo "DEEPGRAM_API_KEY not set — copy env.example to .env and fill it in." >&2
	exit 1
fi

go test ./internal/stt/... -run TestDgSendPreRecorded_Integration -v
