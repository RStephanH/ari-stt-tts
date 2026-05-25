#!/usr/bin/env bash
set -euo pipefail

################################################################################
# Asterisk Configuration Script (Vagrant Provisioning)
################################################################################
#
# Purpose:
#   Generate minimal working Asterisk configuration with ARI, HTTP, PJSIP, and
#   a simple dialplan for testing.
#
# Environment variables:
#   - LOG_FILE:          Path to provisioning log file (optional)
#   - ARI_USERNAME:      ARI user (default: ariuser)
#   - ARI_PASSWORD:      ARI password (default: aripass)
#   - ARI_APPLICATION:   ARI Stasis app name (default: ari-stt-tts)
#   - HTTP_BIND_ADDR:    HTTP bind address (default: 0.0.0.0)
#   - HTTP_BIND_PORT:    HTTP bind port (default: 8088)
#   - PJSIP_ENDPOINT_ID: SIP endpoint ID (default: 1001)
#   - PJSIP_PASSWORD:    SIP endpoint password (default: 1001pass)
#   - COPY_ASSETS:       Copy WAV assets to Asterisk sounds (default: true)
#   - ASSETS_DIR:        Assets directory (default: /vagrant/assets)
#
################################################################################

# ============================================================================
# Logging
# ============================================================================

LOG_FILE="${LOG_FILE:-/tmp/asterisk-provision-$(date +%Y%m%d-%H%M%S).log}"

log_info() {
  local msg="[ASTERISK-CONFIG] $1"
  echo -e "\033[0;34m$msg\033[0m" >&2
  echo "$(date '+%Y-%m-%d %H:%M:%S') $msg" >>"$LOG_FILE"
}

log_success() {
  local msg="[ASTERISK-CONFIG-SUCCESS] $1"
  echo -e "\033[0;32m$msg\033[0m" >&2
  echo "$(date '+%Y-%m-%d %H:%M:%S') $msg" >>"$LOG_FILE"
}

log_warning() {
  local msg="[ASTERISK-CONFIG-WARNING] $1"
  echo -e "\033[1;33m$msg\033[0m" >&2
  echo "$(date '+%Y-%m-%d %H:%M:%S') $msg" >>"$LOG_FILE"
}

log_error() {
  local msg="[ASTERISK-CONFIG-ERROR] $1"
  echo -e "\033[0;31m$msg\033[0m" >&2
  echo "$(date '+%Y-%m-%d %H:%M:%S') $msg" >>"$LOG_FILE"
}

error_handler() {
  local exit_code=$?
  local line_number=$1
  log_error "Asterisk configuration failed at line $line_number with exit code $exit_code"
  exit $exit_code
}

trap 'error_handler $LINENO' ERR
trap 'log_warning "Asterisk configuration interrupted"; exit 130' INT TERM

# ============================================================================
# Config Values
# ============================================================================

ASTERISK_ETC_DIR="/etc/asterisk"
ASTERISK_USER="${ASTERISK_USER:-asterisk}"
ASTERISK_GROUP="${ASTERISK_GROUP:-asterisk}"

ARI_USERNAME="${ARI_USERNAME:-ariuser}"
ARI_PASSWORD="${ARI_PASSWORD:-aripass}"
ARI_APPLICATION="${ARI_APPLICATION:-ari-stt-tts}"

HTTP_BIND_ADDR="${HTTP_BIND_ADDR:-0.0.0.0}"
HTTP_BIND_PORT="${HTTP_BIND_PORT:-8088}"

PJSIP_ENDPOINT_ID="${PJSIP_ENDPOINT_ID:-1001}"
PJSIP_PASSWORD="${PJSIP_PASSWORD:-1001pass}"

COPY_ASSETS="${COPY_ASSETS:-true}"
ASSETS_DIR="${ASSETS_DIR:-/vagrant/assets}"

# ============================================================================
# Helpers
# ============================================================================

backup_file() {
  local file="$1"
  if [[ -f "$file" ]]; then
    local timestamp
    timestamp=$(date +%Y%m%d-%H%M%S)
    cp "$file" "${file}.bak-${timestamp}"
    log_info "Backed up $(basename "$file") to $(basename "$file").bak-${timestamp}"
  fi
}

write_config() {
  local file="$1"
  local content="$2"

  backup_file "$file"
  printf "%s\n" "$content" >"$file"
  chown "$ASTERISK_USER:$ASTERISK_GROUP" "$file"
  chmod 640 "$file"
  log_success "Wrote $(basename "$file")"
}

ensure_asterisk_dirs() {
  if [[ ! -d "$ASTERISK_ETC_DIR" ]]; then
    log_error "Asterisk config directory not found: $ASTERISK_ETC_DIR"
    exit 1
  fi
}

# ============================================================================
# Configuration Writers
# ============================================================================

write_ari_conf() {
  write_config "$ASTERISK_ETC_DIR/ari.conf" "$(
    cat <<EOF
[general]
enabled = yes
pretty = yes
allowed_origins = *

[${ARI_USERNAME}]
type = user
read_only = no
password = ${ARI_PASSWORD}
EOF
  )"
}

write_http_conf() {
  write_config "$ASTERISK_ETC_DIR/http.conf" "$(
    cat <<EOF
[general]
enabled = yes
bindaddr = ${HTTP_BIND_ADDR}
bindport = ${HTTP_BIND_PORT}
tlsenable = no
EOF
  )"
}

write_pjsip_conf() {
  write_config "$ASTERISK_ETC_DIR/pjsip.conf" "$(
    cat <<EOF
[transport-udp]
type=transport
protocol=udp
bind=0.0.0.0

[${PJSIP_ENDPOINT_ID}]
type=endpoint
context=from-internal
disallow=all
allow=ulaw,alaw
auth=${PJSIP_ENDPOINT_ID}
aors=${PJSIP_ENDPOINT_ID}
direct_media=no
rtp_symmetric=yes
force_rport=yes
rewrite_contact=yes

[${PJSIP_ENDPOINT_ID}]
type=auth
auth_type=userpass
username=${PJSIP_ENDPOINT_ID}
password=${PJSIP_PASSWORD}

[${PJSIP_ENDPOINT_ID}]
type=aor
max_contacts=1
EOF
  )"
}

write_extensions_conf() {
  write_config "$ASTERISK_ETC_DIR/extensions.conf" "$(
    cat <<EOF
[from-internal]
exten => ${PJSIP_ENDPOINT_ID},1,NoOp(Test extension ${PJSIP_ENDPOINT_ID})
 same => n,Answer()
 same => n,Playback(hello-world)
 same => n,Hangup()

exten => 6001,1,NoOp(Enter ARI app ${ARI_APPLICATION})
 same => n,Stasis(${ARI_APPLICATION})
 same => n,Hangup()
EOF
  )"
}

write_logger_conf() {
  write_config "$ASTERISK_ETC_DIR/logger.conf" "$(
    cat <<EOF
[general]
dateformat=%F %T

[logfiles]
console => notice,warning,error
messages => notice,warning,error,verbose
EOF
  )"
}

copy_assets() {
  if [[ "$COPY_ASSETS" != "true" ]]; then
    log_warning "Skipping assets copy (COPY_ASSETS=$COPY_ASSETS)"
    return 0
  fi

  if [[ ! -d "$ASSETS_DIR" ]]; then
    log_warning "Assets directory not found: $ASSETS_DIR"
    return 0
  fi

  shopt -s nullglob
  local assets=("$ASSETS_DIR"/*.wav)
  shopt -u nullglob

  if [[ ${#assets[@]} -eq 0 ]]; then
    log_warning "No WAV assets found in $ASSETS_DIR"
    return 0
  fi

  log_info "Copying ${#assets[@]} asset(s) to /var/lib/asterisk/sounds/en/"
  cp "${assets[@]}" /var/lib/asterisk/sounds/en/

  for asset in "${assets[@]}"; do
    local filename
    filename=$(basename "$asset")
    chown "$ASTERISK_USER:$ASTERISK_GROUP" "/var/lib/asterisk/sounds/en/$filename"
  done
  log_success "Assets copied successfully"
}

restart_asterisk() {
  # Force systemd to look for new service files
  systemctl daemon-reload

  if ! systemctl cat asterisk.service >/dev/null 2>&1; then
    log_error "asterisk.service file not found in /etc/systemd/system/."
    log_info "Did you run 'make config' after compiling the source code?"
    exit 1
  fi
  log_info "Enabling Asterisk service..."
  systemctl enable asterisk

  log_info "Restarting Asterisk service..."
  systemctl restart asterisk

  log_success "Asterisk service restarted"
}

# ============================================================================
# Main
# ============================================================================

main() {
  log_info "Configuring Asterisk (ARI/HTTP/PJSIP/extensions/logger)"
  ensure_asterisk_dirs

  write_ari_conf
  write_http_conf
  write_pjsip_conf
  write_extensions_conf
  write_logger_conf
  copy_assets
  restart_asterisk

  log_success "Asterisk configuration completed"
}

main "$@"
