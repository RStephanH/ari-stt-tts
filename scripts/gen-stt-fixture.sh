#!/usr/bin/env bash
set -euo pipefail

FIXTURE="internal/stt/testdata/sample.wav"
mkdir -p "$(dirname "$FIXTURE")"
espeak-ng "Hello this is a test of the speech to text pipeline" -w "$FIXTURE"
echo "Fixture written to $FIXTURE — commit it to git."
