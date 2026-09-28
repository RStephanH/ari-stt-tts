#!/usr/bin/env bash
set -euo pipefail

################################################################################
# Tailscale Network Join Script (Vagrant Provisioning)
################################################################################
#
# Purpose:
#   Install Tailscale and join this VM to the tailnet, so it becomes
#   reachable by hostname/IP from any device on the same tailnet (SSH, ARI
#   debugging, and SIP softphones for demo purposes) without relying on
#   VirtualBox port forwarding.
#
# Environment variables:
#   - LOG_FILE:            Path to provisioning log file (optional)
#   - TAILSCALE_AUTHKEY:   Auth key for unattended `tailscale up` (required)
#   - TAILSCALE_HOSTNAME:  Hostname to advertise on the tailnet (default: pbx-server)
#
################################################################################

LOG_FILE="${LOG_FILE:-/tmp/asterisk-provision-$(date +%Y%m%d-%H%M%S).log}"

log_info() {
  local msg="[TAILSCALE] $1"
  echo -e "\033[0;34m$msg\033[0m" >&2
  echo "$(date '+%Y-%m-%d %H:%M:%S') $msg" >>"$LOG_FILE"
}

log_success() {
  local msg="[TAILSCALE-SUCCESS] $1"
  echo -e "\033[0;32m$msg\033[0m" >&2
  echo "$(date '+%Y-%m-%d %H:%M:%S') $msg" >>"$LOG_FILE"
}

log_warning() {
  local msg="[TAILSCALE-WARNING] $1"
  echo -e "\033[1;33m$msg\033[0m" >&2
  echo "$(date '+%Y-%m-%d %H:%M:%S') $msg" >>"$LOG_FILE"
}

log_error() {
  local msg="[TAILSCALE-ERROR] $1"
  echo -e "\033[0;31m$msg\033[0m" >&2
  echo "$(date '+%Y-%m-%d %H:%M:%S') $msg" >>"$LOG_FILE"
}

error_handler() {
  local exit_code=$?
  local line_number=$1
  log_error "Tailscale setup failed at line $line_number with exit code $exit_code"
  exit $exit_code
}

trap 'error_handler $LINENO' ERR
trap 'log_warning "Tailscale setup interrupted"; exit 130' INT TERM

TAILSCALE_HOSTNAME="${TAILSCALE_HOSTNAME:-pbx-server}"

install_tailscale() {
  if command -v tailscale &>/dev/null; then
    log_info "Tailscale already installed: $(tailscale version | head -n1)"
    return 0
  fi

  log_info "Installing Tailscale..."
  if ! curl -fsSL https://tailscale.com/install.sh | sh; then
    log_error "Failed to install Tailscale"
    exit 1
  fi
  log_success "Tailscale installed: $(tailscale version | head -n1)"
}

start_tailscaled() {
  log_info "Enabling and starting tailscaled service..."
  systemctl enable --now tailscaled
}

join_tailnet() {
  if [[ -z "${TAILSCALE_AUTHKEY:-}" ]]; then
    log_error "TAILSCALE_AUTHKEY is not set — cannot join the tailnet unattended"
    log_error "Generate a reusable auth key at https://login.tailscale.com/admin/settings/keys"
    exit 1
  fi

  if tailscale status --json 2>/dev/null | grep -q '"BackendState":"Running"'; then
    log_info "Already joined the tailnet"
    return 0
  fi

  log_info "Joining tailnet as '${TAILSCALE_HOSTNAME}'..."
  if ! tailscale up --authkey="${TAILSCALE_AUTHKEY}" --hostname="${TAILSCALE_HOSTNAME}" --ssh; then
    log_error "Failed to join the tailnet"
    exit 1
  fi

  log_success "Joined tailnet successfully"
}

print_tailscale_ip() {
  local ip
  ip=$(tailscale ip -4 2>/dev/null || true)
  if [[ -n "$ip" ]]; then
    log_success "Tailscale IPv4: $ip"
  else
    log_warning "Could not determine Tailscale IPv4 address"
  fi
}

main() {
  log_info "=================================="
  log_info "Tailscale Network Join"
  log_info "=================================="

  install_tailscale
  start_tailscaled
  join_tailnet
  print_tailscale_ip

  log_success "Tailscale network join completed!"
}

main "$@"
