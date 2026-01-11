#!/usr/bin/env bash

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MODULES_DIR="$SCRIPT_DIR/modules"
LOG_FILE="/tmp/asterisk-configuration-$(date +%Y%m%d-%H%M%S).log"

#Global variable

SKIP_ARI=false
SKIP_HTTP=false

# Global error tracking
CURRENT_MODULE=""
CURRENT_STEP=""
ERROR_OCCURRED=false
# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
MAGENTA='\033[0;35m'
NC='\033[0m' # No Color

log_info() {
  local msg="[INFO] $1"
  echo -e "${BLUE}$msg${NC}"
  echo "$(date '+%Y-%m-%d %H:%M:%S') $msg" >>"$LOG_FILE"
}

log_success() {
  local msg="[SUCCESS] $1"
  echo -e "${GREEN}$msg${NC}"
  echo "$(date '+%Y-%m-%d %H:%M:%S') $msg" >>"$LOG_FILE"
}

log_warning() {
  local msg="[WARNING] $1"
  echo -e "${YELLOW}$msg${NC}"
  echo "$(date '+%Y-%m-%d %H:%M:%S') $msg" >>"$LOG_FILE"
}

log_error() {
  local msg="[ERROR] $1"
  echo -e "${RED}$msg${NC}"
  echo "$(date '+%Y-%m-%d %H:%M:%S') $msg" >>"$LOG_FILE"
  ERROR_OCCURRED=true
}

log_step() {
  local msg="[STEP] $1"
  echo -e "${CYAN}$msg${NC}"
  echo "$(date '+%Y-%m-%d %H:%M:%S') $msg" >>"$LOG_FILE"
}

log_critical() {
  local msg="[CRITICAL] $1"
  echo -e "${MAGENTA}$msg${NC}"
  echo "$(date '+%Y-%m-%d %H:%M:%S') $msg" >>"$LOG_FILE"
  ERROR_OCCURRED=true
}

show_help() {
  cat <<EOF
🚀 Asterisk Configuration Script with Error Tracking

Usage: $0 [options]

Options:

    --no-ari            Skip the Asterisk Rest Insterface configuration
    --no-http           Skip the Asterisk http server configuration
    -h, --help          Show this help message
EOF
}

parse_args() {
  while [[ $# -gt 0 ]]; do
    case $1 in
    --no-ari)
      SKIP_ARI=true
      shift
      ;;
    --no-http)
      SKIP_HTTP=true
      shift
      ;;
    -h | --help)
      show_help
      exit 0
      ;;
    *)
      log_error "Unknown option: $1"
      show_help
      exit 1
      ;;
    esac
  done
}

#Check modules scripts
check_modules() {
  if [[ ! -d "$MODULES_DIR" ]]; then
    log_error "Modules directory not found : $MODULES_DIR"
    exit 1
  fi

  local required_scripts=("00-assets.sh")
  for script in "${required_scripts[@]}"; do
    if [[ ! -f "$MODULES_DIR/$script" ]]; then
      log_error "Required script not found : $MODULES_DIR/$script"
    fi

  done

  if [[ ! -x "$MODULES_DIR/$script" ]]; then
    log_info "Making $script executable ..."
    chmod +x "$MODULES_DIR/$script"
  fi
}

run_module() {
  local module_name="$1"
  local script_path="$2"
  shift 2

  CURRENT_MODULE="$module_name"
  CURRENT_STEP="Module execution"

  echo ""
  echo "========================================"
  log_step "Running $module_name"
  echo "========================================"

  # Export variables so module scripts can use them
  export CURRENT_MODULE
  export LOG_FILE

  if ! "$script_path" "$@"; then
    log_error "$module_name failed!"
    exit 1
  fi

  log_success "$module_name completed successfully1"
  CURRENT_STEP=""
}

#====Main execution ====
parse_args "$@"
check_modules
run_module "Set up assets" "$MODULES_DIR/00-assets.sh"
