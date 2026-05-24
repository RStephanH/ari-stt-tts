#!/usr/bin/env bash

################################################################################
# ari-stt-tts Infrastructure Provisioning Bootstrap Script
################################################################################
# 
# Purpose:
#   Main entry point for Vagrant provisioning. Orchestrates the execution of
#   all provisioning scripts in the correct order.
#
# Environment:
#   - Runs as root (privileged provisioning)
#   - Executed by Vagrant during 'vagrant up'
#   - Source scripts in sequence
#
# Exit Codes:
#   0   - Success
#   1   - General error
#   2   - Prerequisites check failed
#   3   - Script execution failed
#
# Usage:
#   Called automatically by Vagrant via Vagrantfile
#   (Manual: bash provisioning/bootstrap.sh)
#
################################################################################

set -euo pipefail

# ============================================================================
# Global Configuration
# ============================================================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
VAGRANT_ENV="${VAGRANT_ENVIRONMENT:-false}"
PROVISIONING_DIR="$SCRIPT_DIR"

if [[ "$VAGRANT_ENV" == "true" ]]; then
  for candidate in /vagrant/provisioning /vagrant/infra/vagrant/provisioning; do
    if [[ -d "$candidate" ]]; then
      PROVISIONING_DIR="$candidate"
      break
    fi
  done
fi

ASTERISK_DIR="$PROVISIONING_DIR/asterisk"

# Logging
LOG_DIR="/tmp"
LOG_FILE="$LOG_DIR/asterisk-provision-$(date +%Y%m%d-%H%M%S).log"

# Colors for output (disabled if not a TTY)
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
MAGENTA='\033[0;35m'
NC='\033[0m' # No Color

# Tracking
SCRIPTS_RUN=()
SCRIPTS_FAILED=()
CURRENT_SCRIPT=""
START_TIME=$(date +%s)

# ============================================================================
# Utility Functions
# ============================================================================

log_info() {
  local msg="[INFO] $1"
  echo -e "${BLUE}$msg${NC}" >&2
  echo "$(date '+%Y-%m-%d %H:%M:%S') $msg" >> "$LOG_FILE"
}

log_success() {
  local msg="[SUCCESS] $1"
  echo -e "${GREEN}$msg${NC}" >&2
  echo "$(date '+%Y-%m-%d %H:%M:%S') $msg" >> "$LOG_FILE"
}

log_warning() {
  local msg="[WARNING] $1"
  echo -e "${YELLOW}$msg${NC}" >&2
  echo "$(date '+%Y-%m-%d %H:%M:%S') $msg" >> "$LOG_FILE"
}

log_error() {
  local msg="[ERROR] $1"
  echo -e "${RED}$msg${NC}" >&2
  echo "$(date '+%Y-%m-%d %H:%M:%S') $msg" >> "$LOG_FILE"
}

log_step() {
  local msg="[STEP] $1"
  echo -e "${CYAN}$msg${NC}" >&2
  echo "$(date '+%Y-%m-%d %H:%M:%S') $msg" >> "$LOG_FILE"
}

log_section() {
  local msg="$1"
  echo "" >&2
  echo "================================================================================" >&2
  echo -e "${MAGENTA}$msg${NC}" >&2
  echo "================================================================================" >&2
  echo "$(date '+%Y-%m-%d %H:%M:%S') === $msg ===" >> "$LOG_FILE"
}

# Error handler
error_handler() {
  local exit_code=$?
  local line_number=$1
  
  log_error "Provisioning failed with exit code $exit_code at line $line_number"
  
  if [[ -n "$CURRENT_SCRIPT" ]]; then
    log_error "Failed script: $CURRENT_SCRIPT"
    SCRIPTS_FAILED+=("$CURRENT_SCRIPT")
  fi
  
  print_summary
  exit $exit_code
}

# Set up error trapping
trap 'error_handler $LINENO' ERR
trap 'log_warning "Provisioning interrupted by user"; exit 130' INT TERM

# ============================================================================
# Validation Functions
# ============================================================================

check_vagrant_environment() {
  log_step "Validating Vagrant environment..."
  
  if [[ "$VAGRANT_ENV" == "true" ]]; then
    log_success "Running in Vagrant environment"
  else
    log_warning "Not running in Vagrant environment (manual run)"
  fi
  
  # Check if running as root
  if [[ $EUID -ne 0 ]]; then
    log_error "This script must be run as root (via sudo or Vagrant)"
    exit 2
  fi
  log_info "Running with root privileges"
}

check_prerequisites() {
  log_step "Checking prerequisites..."
  
  # Check if /vagrant directory exists (synced folder)
  if [[ ! -d "/vagrant" ]]; then
    log_warning "Synced folder /vagrant not found (expected in Vagrant)"
  else
    log_info "Synced folder found: /vagrant"
  fi
  
  # Check if provisioning scripts exist
  local required_scripts=(
    "$PROVISIONING_DIR/dependencies.sh"
    "$ASTERISK_DIR/install.sh"
    "$ASTERISK_DIR/configure.sh"
  )
  
  for script in "${required_scripts[@]}"; do
    if [[ ! -f "$script" ]]; then
      log_error "Required script not found: $script"
      exit 2
    fi
    
    if [[ ! -x "$script" ]]; then
      log_info "Making script executable: $script"
      chmod +x "$script"
    fi
  done
  
  log_success "All required scripts found and executable"
}

setup_logging() {
  log_step "Setting up logging..."
  
  if [[ ! -d "$LOG_DIR" ]]; then
    mkdir -p "$LOG_DIR"
  fi
  
  # Initialize log file
  echo "=================================================================================" > "$LOG_FILE"
  echo "ari-stt-tts Infrastructure Provisioning" >> "$LOG_FILE"
  echo "Started: $(date '+%Y-%m-%d %H:%M:%S')" >> "$LOG_FILE"
  echo "Host: $(hostname)" >> "$LOG_FILE"
  echo "User: $(whoami)" >> "$LOG_FILE"
  echo "================================================================================" >> "$LOG_FILE"
  
  log_success "Logging to: $LOG_FILE"
}

# ============================================================================
# Script Execution Functions
# ============================================================================

run_provisioning_script() {
  local script_name="$1"
  local script_path="$2"
  shift 2
  local args=("$@")
  
  CURRENT_SCRIPT="$script_name"
  
  log_section "Running: $script_name"
  
  # Export variables for sub-scripts
  export VAGRANT_ENVIRONMENT
  export LOG_FILE
  export PROVISIONING_DIR
  
  if [[ ! -f "$script_path" ]]; then
    log_error "Script not found: $script_path"
    return 1
  fi
  
  if [[ ! -x "$script_path" ]]; then
    log_warning "Making script executable: $script_path"
    chmod +x "$script_path"
  fi
  
  # Run the script
  if bash "$script_path" "${args[@]}"; then
    log_success "$script_name completed successfully"
    SCRIPTS_RUN+=("$script_name")
    CURRENT_SCRIPT=""
    return 0
  else
    log_error "$script_name failed!"
    return 1
  fi
}

# ============================================================================
# Summary Functions
# ============================================================================

calculate_duration() {
  local end_time=$(date +%s)
  local duration=$((end_time - START_TIME))
  
  local hours=$((duration / 3600))
  local minutes=$(((duration % 3600) / 60))
  local seconds=$((duration % 60))
  
  if [[ $hours -gt 0 ]]; then
    echo "${hours}h ${minutes}m ${seconds}s"
  elif [[ $minutes -gt 0 ]]; then
    echo "${minutes}m ${seconds}s"
  else
    echo "${seconds}s"
  fi
}

print_summary() {
  echo ""
  log_section "PROVISIONING SUMMARY"
  
  local duration=$(calculate_duration)
  
  echo ""
  echo "Duration: $duration"
  echo ""
  
  if [[ ${#SCRIPTS_RUN[@]} -gt 0 ]]; then
    echo -e "${GREEN}✅ Scripts Completed (${#SCRIPTS_RUN[@]}):${NC}"
    for script in "${SCRIPTS_RUN[@]}"; do
      echo "  ✓ $script"
    done
    echo ""
  fi
  
  if [[ ${#SCRIPTS_FAILED[@]} -gt 0 ]]; then
    echo -e "${RED}❌ Scripts Failed (${#SCRIPTS_FAILED[@]}):${NC}"
    for script in "${SCRIPTS_FAILED[@]}"; do
      echo "  ✗ $script"
    done
    echo ""
  fi
  
  echo "Log file: $LOG_FILE"
  echo "================================================================================"
}

print_next_steps() {
  echo ""
  echo -e "${CYAN}Next Steps:${NC}"
  echo "  1. SSH into VM: vagrant ssh"
  echo "  2. Check Asterisk: systemctl status asterisk"
  echo "  3. Deploy application: cd /vagrant && docker compose up --build"
  echo ""
  echo -e "${CYAN}Useful Commands:${NC}"
  echo "  • Asterisk CLI: sudo asterisk -r"
  echo "  • Check logs: tail -f /var/log/asterisk/messages"
  echo "  • Re-provision: vagrant provision"
  echo ""
}

# ============================================================================
# Main Execution
# ============================================================================

main() {
  log_section "🚀 Starting ari-stt-tts Infrastructure Provisioning"
  
  # Phase 1: Validation
  check_vagrant_environment
  setup_logging
  check_prerequisites
  
  log_info "Provisioning scripts location: $PROVISIONING_DIR"
  log_info "Log file: $LOG_FILE"
  echo ""
  
  # Phase 2: Provisioning Scripts (in order)
  run_provisioning_script \
    "System Dependencies" \
    "$PROVISIONING_DIR/dependencies.sh"
  
  run_provisioning_script \
    "Asterisk Installation" \
    "$ASTERISK_DIR/install.sh"
  
  run_provisioning_script \
    "Asterisk Configuration" \
    "$ASTERISK_DIR/configure.sh"
  
  # Phase 3: Summary and next steps
  if [[ ${#SCRIPTS_FAILED[@]} -eq 0 ]]; then
    log_section "🎉 PROVISIONING COMPLETE!"
    print_summary
    print_next_steps
    log_success "All provisioning scripts executed successfully!"
    echo "$(date '+%Y-%m-%d %H:%M:%S') [COMPLETE] Provisioning finished successfully" >> "$LOG_FILE"
    exit 0
  else
    log_section "💥 PROVISIONING FAILED"
    print_summary
    log_error "Some scripts failed. Check log file: $LOG_FILE"
    exit 3
  fi
}

# Execute main
main "$@"
