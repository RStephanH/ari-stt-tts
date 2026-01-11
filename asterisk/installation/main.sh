#!/usr/bin/env bash
set -euo pipefail

# Asterisk Installation Main Entry Script with Enhanced Error Tracking
# Usage: ./main.sh [options]

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MODULES_DIR="$SCRIPT_DIR/modules"
LOG_FILE="/tmp/asterisk-install-$(date +%Y%m%d-%H%M%S).log"

# Global error tracking
CURRENT_MODULE=""
CURRENT_STEP=""
ERROR_OCCURRED=false

# Default options
SKIP_DEPS=false
SKIP_ASTERISK=false
SKIP_VOSK=true
VOSK_MODEL="alphacep/kaldi-en:latest"
VOSK_PORT="2700"

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

# Enhanced error handler
error_handler() {
  local exit_code=$?
  local line_number=$1

  log_critical "Script failed with exit code $exit_code at line $line_number"

  if [[ -n "$CURRENT_MODULE" ]]; then
    log_critical "Failed Module: $CURRENT_MODULE"
  fi

  if [[ -n "$CURRENT_STEP" ]]; then
    log_critical "Failed Step: $CURRENT_STEP"
  fi

  echo ""
  echo "========================================"
  echo "💥 INSTALLATION FAILED!"
  echo "========================================"
  echo -e "${RED}Error Details:${NC}"
  echo "  📍 Module: ${CURRENT_MODULE:-'Main Script'}"
  echo "  🔧 Step: ${CURRENT_STEP:-'Unknown'}"
  echo "  📊 Exit Code: $exit_code"
  echo "  📝 Line: $line_number"
  echo "  📋 Log File: $LOG_FILE"
  echo ""
  echo -e "${YELLOW}Troubleshooting:${NC}"
  echo "  1. Check the log file for detailed error messages"
  echo "  2. Ensure you have sudo privileges"
  echo "  3. Check internet connectivity"
  echo "  4. Verify system requirements (Ubuntu/Debian)"
  echo "  5. Try running individual modules separately"
  echo ""
  echo -e "${CYAN}To retry:${NC}"
  echo "  • Fix the issue and re-run: $0 $*"
  echo "  • Skip completed modules with --skip-* flags"
  echo ""

  # Show last few log entries
  echo -e "${BLUE}Last 10 log entries:${NC}"
  tail -n 10 "$LOG_FILE" 2>/dev/null || echo "Log file not available"

  exit $exit_code
}

# Set up error trapping
trap 'error_handler $LINENO' ERR
trap 'log_warning "Installation interrupted by user"; exit 130' INT TERM

parse_args() {
  while [[ $# -gt 0 ]]; do
    case $1 in
    --skip-deps)
      SKIP_DEPS=true
      shift
      ;;
    --skip-asterisk)
      SKIP_ASTERISK=true
      shift
      ;;
    --skip-vosk)
      SKIP_VOSK=true
      shift
      ;;
    --vosk-model)
      VOSK_MODEL="$2"
      shift 2
      ;;
    --vosk-port)
      VOSK_PORT="$2"
      shift 2
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

show_help() {
  cat <<EOF
🚀 Asterisk Installation Script with Error Tracking

Usage: $0 [options]

Options:
    --skip-deps         Skip dependency installation (including Docker CE)
    --skip-asterisk     Skip Asterisk installation
    --skip-vosk         Skip VOSK speech recognition installation
    --vosk-model MODEL  VOSK Docker model (default: alphacep/kaldi-en:latest)
    --vosk-port PORT    VOSK port (default: 2700)
    -h, --help          Show this help message

Examples:
    $0                                           # Full installation
    $0 --skip-deps                               # Skip dependencies
    $0 --vosk-model alphacep/kaldi-fr:latest     # Use French model

Note: All activities are logged to /tmp/asterisk-install-TIMESTAMP.log
EOF
}

check_os() {
  # == Detect Linux Distro ==
  distro=""
  if [[ -f /etc/os-release ]]; then
    # source (import) the file so we get variable like ID, NAME, VERSION_ID
    . /etc/os-release
    distro=$ID
  else
    echo "❌Could not detect distro (no /etc/os-release)"
    exit 1
  fi

  case "$distro" in
  arch | manjaro | endeavouros | cachyos)
    packmanager="pacman"
    ;;
  debian | ubuntu)
    packmanager="apt"
    ;;
  *)
    echo "Unsupported distro: $distro"
    ;;
  esac

  echo "$packmanager"

}

check_prerequisites() {
  CURRENT_MODULE="Prerequisites Check"
  CURRENT_STEP="System validation"

  log_step "Checking prerequisites..."

  # Check OS version
  if [[ -f /etc/os-release ]]; then
    . /etc/os-release
    log_info "Detected OS: $PRETTY_NAME"
  fi

  CURRENT_STEP="Module scripts validation"

  # Check if modules directory exists
  if [[ ! -d "$MODULES_DIR" ]]; then
    log_error "Modules directory not found: $MODULES_DIR"
    exit 1
  fi

  # Check if required module scripts exist
  local required_scripts=("00-dependencies.sh" "10-install.sh")
  for script in "${required_scripts[@]}"; do
    if [[ ! -f "$MODULES_DIR/$script" ]]; then
      log_error "Required script not found: $MODULES_DIR/$script"
      exit 1
    fi
    if [[ ! -x "$MODULES_DIR/$script" ]]; then
      log_info "Making $script executable..."
      chmod +x "$MODULES_DIR/$script"
    fi
  done

  log_success "Prerequisites check passed"
  CURRENT_STEP=""
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

  log_success "$module_name completed successfully!"
  CURRENT_STEP=""
}

create_backup() {
  CURRENT_MODULE="Backup Creation"
  CURRENT_STEP="Configuration backup"

  local backup_dir
  backup_dir="/tmp/asterisk-backup-$(date +%Y%m%d-%H%M%S)"

  if [[ -d "/etc/asterisk" ]]; then
    log_info "Creating backup of existing Asterisk configuration..."
    sudo mkdir -p "$backup_dir"
    sudo cp -r /etc/asterisk "$backup_dir/" 2>/dev/null || true
    log_info "Backup created: $backup_dir"
  fi

  CURRENT_STEP=""
}

post_install_checks() {
  CURRENT_MODULE="Post-Install Verification"
  CURRENT_STEP="System verification"

  echo ""
  echo "========================================"
  log_step "Performing post-installation checks"
  echo "========================================"

  # Check if Asterisk binary exists
  if command -v asterisk >/dev/null 2>&1; then
    log_success "Asterisk binary found: $(which asterisk)"
    asterisk_version=$(asterisk -V 2>/dev/null | head -n1 || echo "Unknown")
    log_info "Version: $asterisk_version"
  else
    log_warning "Asterisk binary not found in PATH"
  fi

  CURRENT_STEP="Module verification"

  # Check if modules directory exists
  if [[ -d "/usr/lib/asterisk/modules" ]]; then
    module_count=$(find /usr/lib/asterisk/modules -name "*.so" | wc -l)
    log_info "Found $module_count Asterisk modules"
  fi

  # Check if VOSK module exists
  if [[ -f "/usr/lib/asterisk/modules/res_speech_vosk.so" ]]; then
    log_success "VOSK module installed"
  fi

  CURRENT_STEP="Docker verification"

  # Check Docker CE
  if command -v docker >/dev/null 2>&1; then
    docker_version=$(docker --version 2>/dev/null || echo "Unknown")
    log_success "Docker CE: $docker_version"
  fi

  log_warning "Please check the systemd files in /etc/systemd/system/"
  CURRENT_STEP=""
  log_info "Reloading the daemon"
  sudo systemctl daemon-reload
}

print_summary() {
  echo ""
  echo "========================================"
  echo "🎉 INSTALLATION COMPLETE!"
  echo "========================================"

  if [[ "$SKIP_DEPS" == false ]]; then
    echo "✅ Dependencies installed (including Docker CE)"
  fi

  if [[ "$SKIP_ASTERISK" == false ]]; then
    echo "✅ Asterisk installed with interactive menuselect"
  fi

  if [[ "$SKIP_VOSK" == false ]]; then
    echo "✅ VOSK speech recognition module installed"
    echo "   📦 Model: $VOSK_MODEL"
    echo "   🌐 Port: $VOSK_PORT"
    echo "   🔗 WebSocket URL: ws://127.0.0.1:$VOSK_PORT"
  fi

  echo ""
  echo "📋 Installation Log: $LOG_FILE"
  echo ""
  echo "🚀 NEXT STEPS:"
  echo "1. 🔄 Log out and back in (for Docker group membership)"
  echo "2. 🏃 Start Asterisk: sudo systemctl start asterisk"
  echo "3. ⚡ Enable on boot: sudo systemctl enable asterisk"
  echo "4. 📊 Check status: sudo systemctl status asterisk"
  echo "5. 💬 Access CLI: sudo asterisk -r"
}

main() {
  # Initialize logging
  echo "$(date '+%Y-%m-%d %H:%M:%S') [START] Asterisk Installation Started" >"$LOG_FILE"
  log_info "Installation log: $LOG_FILE"

  parse_args "$@"

  echo "🚀 Starting Asterisk Installation Process"
  echo "========================================"
  echo "Configuration:"
  echo "  Skip Dependencies: $SKIP_DEPS"
  echo "  Skip Asterisk: $SKIP_ASTERISK"
  echo "  Skip VOSK: $SKIP_VOSK"
  echo "  VOSK Model: $VOSK_MODEL"
  echo "  VOSK Port: $VOSK_PORT"
  echo "  Log File: $LOG_FILE"

  local packman
  packman=$(check_os)
  export packman

  check_prerequisites
  create_backup

  # Install dependencies (including Docker CE)
  if [[ "$SKIP_DEPS" == false ]]; then
    run_module "Dependencies & Docker CE Installation" "$MODULES_DIR/00-dependencies.sh"
  else
    log_info "Skipping dependency installation"
  fi

  # Install Asterisk (with interactive menuselect)
  if [[ "$SKIP_ASTERISK" == false ]]; then
    run_module "Asterisk Installation" "$MODULES_DIR/10-install.sh"
  else
    log_info "Skipping Asterisk installation"
  fi

  # Install VOSK
  if [[ "$SKIP_VOSK" == false ]]; then
    run_module "VOSK Speech Recognition" "$MODULES_DIR/20-vosk.sh" "$VOSK_MODEL" "$VOSK_PORT"
  else
    log_info "Skipping VOSK installation"
  fi

  post_install_checks
  print_summary

  log_success "All installations completed successfully! 🎉"
  echo "$(date '+%Y-%m-%d %H:%M:%S') [END] Installation completed successfully" >>"$LOG_FILE"
}

main "$@"
