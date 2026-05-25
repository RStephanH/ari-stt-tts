#!/usr/bin/env bash
set -euo pipefail

################################################################################
# Asterisk Installation Script (Vagrant Provisioning)
################################################################################
#
# Purpose:
#   Build and install Asterisk from source for the Vagrant VM.
#
# Environment variables:
#   - LOG_FILE:               Path to provisioning log file (optional)
#   - ASTERISK_VERSION:       Major version to install (default: 22)
#   - FORCE_ASTERISK_INSTALL: Set to "true" to rebuild even if installed
#
################################################################################

# ============================================================================
# Logging (aligned with provisioning scripts)
# ============================================================================

LOG_FILE="${LOG_FILE:-/tmp/asterisk-provision-$(date +%Y%m%d-%H%M%S).log}"

log_info() {
  local msg="[ASTERISK] $1"
  echo -e "\033[0;34m$msg\033[0m" >&2
  echo "$(date '+%Y-%m-%d %H:%M:%S') $msg" >>"$LOG_FILE"
}

log_success() {
  local msg="[ASTERISK-SUCCESS] $1"
  echo -e "\033[0;32m$msg\033[0m" >&2
  echo "$(date '+%Y-%m-%d %H:%M:%S') $msg" >>"$LOG_FILE"
}

log_warning() {
  local msg="[ASTERISK-WARNING] $1"
  echo -e "\033[1;33m$msg\033[0m" >&2
  echo "$(date '+%Y-%m-%d %H:%M:%S') $msg" >>"$LOG_FILE"
}

log_error() {
  local msg="[ASTERISK-ERROR] $1"
  echo -e "\033[0;31m$msg\033[0m" >&2
  echo "$(date '+%Y-%m-%d %H:%M:%S') $msg" >>"$LOG_FILE"
}

error_handler() {
  local exit_code=$?
  local line_number=$1
  log_error "Asterisk installation failed at line $line_number with exit code $exit_code"
  log_error "Working directory: $(pwd)"
  log_error "Disk usage: $(df -h . | tail -1)"
  exit $exit_code
}

trap 'error_handler $LINENO' ERR
trap 'log_warning "Asterisk installation interrupted"; exit 130' INT TERM

# ============================================================================
# OS Detection
# ============================================================================

detect_os() {
  if [[ ! -f /etc/os-release ]]; then
    log_error "Cannot detect OS: /etc/os-release not found"
    exit 1
  fi

  . /etc/os-release

  case "$ID" in
  ubuntu | debian)
    echo "apt"
    ;;
  *)
    log_error "Unsupported OS: $PRETTY_NAME"
    exit 1
    ;;
  esac
}

# ============================================================================
# Installation
# ============================================================================

ensure_asterisk_user() {
  local user="$1"
  if ! id "$user" >/dev/null 2>&1; then
    log_info "Creating Asterisk user: $user"
    useradd -r -d /var/lib/asterisk -s /bin/bash "$user"
  fi
}

install_prerequisites() {
  log_info "Installing Asterisk build prerequisites..."

  if [[ ! -x "./contrib/scripts/install_prereq" ]]; then
    log_error "Missing install_prereq script (expected in Asterisk source)"
    exit 1
  fi

  ./contrib/scripts/install_prereq install

  log_info "Installing MP3 support..."
  if ! ./contrib/scripts/get_mp3_source.sh; then
    log_error "Failed to install MP3 support"
    exit 1
  fi
}

install_asterisk() {
  local package_manager
  package_manager=$(detect_os)

  local asterisk_version="${ASTERISK_VERSION:-22}"
  local build_dir="/usr/src"
  local asterisk_user="${ASTERISK_USER:-asterisk}"
  local asterisk_url="https://downloads.asterisk.org/pub/telephony/asterisk/asterisk-${asterisk_version}-current.tar.gz"

  if command -v asterisk >/dev/null 2>&1; then
    local installed_version
    installed_version=$(asterisk -V | awk '{print $2}')

    if [[ "${FORCE_ASTERISK_INSTALL:-false}" != "true" && "$installed_version" == "$asterisk_version"* ]]; then
      log_success "Asterisk $installed_version already installed; skipping rebuild"
      log_info "Set FORCE_ASTERISK_INSTALL=true to rebuild"
      return 0
    fi

    log_warning "Asterisk detected ($installed_version); rebuilding due to FORCE_ASTERISK_INSTALL=true"
  fi

  if [[ "$package_manager" != "apt" ]]; then
    log_error "Unsupported package manager: $package_manager"
    exit 1
  fi

  ensure_asterisk_user "$asterisk_user"

  mkdir -p "$build_dir"
  cd "$build_dir"

  log_info "Cleaning up any previous Asterisk build directories..."
  rm -rf "$build_dir"/asterisk-* || true

  log_info "Downloading Asterisk ${asterisk_version}..."
  if ! wget -O "asterisk-${asterisk_version}.tar.gz" "$asterisk_url"; then
    log_error "Failed to download Asterisk from $asterisk_url"
    exit 1
  fi

  log_info "Extracting Asterisk archive..."
  if ! tar -xzf "asterisk-${asterisk_version}.tar.gz"; then
    log_error "Failed to extract Asterisk archive"
    exit 1
  fi

  local asterisk_dir
  asterisk_dir=$(find "$build_dir" -maxdepth 1 -type d -name "asterisk-*" | head -n1)
  if [[ -z "$asterisk_dir" ]]; then
    log_error "Could not find extracted Asterisk source directory"
    exit 1
  fi

  log_info "Using Asterisk source directory: $asterisk_dir"
  cd "$asterisk_dir"

  install_prerequisites

  log_info "Configuring Asterisk build..."
  if ! ./configure \
    --prefix=/usr \
    --sysconfdir=/etc \
    --localstatedir=/var \
    --with-jansson-bundled \
    --enable-shared \
    --disable-video \
    --disable-opencore-amr \
    --with-pjproject-bundled \
    --with-libcurl \
    --with-ssl \
    CFLAGS='-O2 -DNDEBUG'; then
    log_error "Failed to configure Asterisk build"
    exit 1
  fi

  # if [[ "${SKIP_MENUSELECT:-false}" != "true" ]]; then
  #   echo ""
  #   echo "📋 MENUSELECT INSTRUCTIONS:"
  #   echo "   • Use arrow keys to navigate"
  #   echo "   • Press ENTER to enter a category"
  #   echo "   • Press SPACE to enable/disable modules"
  #   echo "   • Press 'x' to exit a category"
  #   echo "   • Press 'q' to quit and save"
  #   echo ""
  #   if ! make menuselect; then
  #     log_error "Menuselect failed or was cancelled"
  #     exit 1
  #   fi
  # else
  #   log_warning "Skipping menuselect (SKIP_MENUSELECT=true)"
  # fi

  log_info "Asterisk Module Selection (Automated)"

  # 1. Generate the initial makeopts file
  log_info "Initializing default module selection..."
  make menuselect.makeopts

  # 2. Enable critical modules for ARI, PJSIP and STT/TTS
  log_info "Enabling required modules for IVR/ARI infrastructure..."

  # ARI & JSON Core (Essential for Go application communication)
  ./menuselect/menuselect --enable res_ari --enable res_ari_model --enable res_ari_events menuselect.makeopts
  ./menuselect/menuselect --enable res_http_websocket menuselect.makeopts

  # PJSIP Stack (Modern SIP protocol)
  ./menuselect/menuselect --enable chan_pjsip --enable res_pjsip --enable res_pjsip_session menuselect.makeopts

  # Sound Formats & Codecs
  log_info "Enabling sound formats and codecs..."

  # SLN (all sample rates including sln16) — built into format_sln since Asterisk 10
  # No separate sln16 module exists; format_sln handles 8kHz to 192kHz
  ./menuselect/menuselect --enable format_sln menuselect.makeopts

  # G.722 wideband codec (HD voice, 16kHz) — built-in, no dependency needed
  ./menuselect/menuselect --enable codec_g722 menuselect.makeopts

  # Opus codec — requires libopus-dev installed before ./configure
  # codec_opus is enabled if ./configure detected libopus-dev; otherwise silently skipped
  if ./menuselect/menuselect --enable codec_opus menuselect.makeopts 2>/dev/null; then
    log_info "codec_opus enabled (libopus detected)"
  else
    log_info "codec_opus skipped (libopus-dev not found — install it and re-run ./configure to enable)"
  fi

  # 3. Enable English Core Sounds (WAV format)
  log_info "Enabling English Core Sounds (WAV format)..."
  ./menuselect/menuselect --enable CORE-SOUNDS-EN-WAV menuselect.makeopts
  ./menuselect/menuselect --enable MOH-OPSOUND-WAV menuselect.makeopts

  # 4. Optional: Disable obsolete modules to optimize the build

  log_success "Module selection completed automatically."

  # 5. Compilation
  log_info "Compiling Asterisk using $(nproc) CPU cores..."
  if ! make -j"$(nproc)"; then
    log_error "Failed to compile Asterisk"
    exit 1
  fi

  log_info "Compiling Asterisk using $(nproc) CPU cores..."
  if ! make -j"$(nproc)"; then
    log_error "Failed to compile Asterisk"
    exit 1
  fi

  log_info "Installing Asterisk..."
  if ! make install; then
    log_error "Failed to install Asterisk"
    exit 1
  fi

  log_info "Installing sample configurations..."
  if ! make samples; then
    log_error "Failed to install sample configurations"
    exit 1
  fi

  log_info "Installing systemd service files..."
  if ! make config; then
    log_error "Failed to install systemd service files"
    exit 1
  fi

  log_info "Reloading systemd daemon..."
  systemctl daemon-reload

  log_info "Setting Asterisk file ownership..."
  chown -R "$asterisk_user:$asterisk_user" /var/lib/asterisk
  chown -R "$asterisk_user:$asterisk_user" /var/log/asterisk
  chown -R "$asterisk_user:$asterisk_user" /var/spool/asterisk
  chown -R "$asterisk_user:$asterisk_user" /etc/asterisk

  log_info "Updating shared libraries..."
  ldconfig

  log_info "Cleaning up build artifacts..."
  rm -rf "$build_dir/asterisk-${asterisk_version}.tar.gz"

  log_success "Asterisk installation completed!"
}

install_asterisk "$@"
