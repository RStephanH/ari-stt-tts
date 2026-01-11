#!/usr/bin/env bash

source ./log.sh
SCRIPT_RUN_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd ../../../
if [[ ! -d "assets" ]]; then
  log_error "Bad directory, assets directory not found"
  exit 1
fi
log_info "Moving into the assets directory"
cd assets
log_info "Copy the assets ..."
if ! sudo cp *.wav "/var/lib/asterisk/sounds/en/"; then
  log_error "Failed to copy the assets!"
  exit 1
fi
log_success "Assets copied successfully"
