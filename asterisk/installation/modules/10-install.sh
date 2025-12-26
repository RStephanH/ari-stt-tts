#!/usr/bin/env bash
set -euo pipefail

# Import logging from main script
log_info() {
  local msg="[ASTERISK] $1"
  echo -e "\033[0;34m$msg\033[0m"
  [[ -n "${LOG_FILE:-}" ]] && echo "$(date '+%Y-%m-%d %H:%M:%S') $msg" >>"$LOG_FILE"
}

log_error() {
  local msg="[ASTERISK-ERROR] $1"
  echo -e "\033[0;31m$msg\033[0m"
  [[ -n "${LOG_FILE:-}" ]] && echo "$(date '+%Y-%m-%d %H:%M:%S') $msg" >>"$LOG_FILE"
}

log_success() {
  local msg="[ASTERISK-SUCCESS] $1"
  echo -e "\033[0;32m$msg\033[0m"
  [[ -n "${LOG_FILE:-}" ]] && echo "$(date '+%Y-%m-%d %H:%M:%S') $msg" >>"$LOG_FILE"
}

# Error handler for this module
error_handler() {
  local exit_code=$?
  local line_number=$1
  log_error "Asterisk installation failed at line $line_number with exit code $exit_code"
  log_error "Current working directory: $(pwd)"
  log_error "Available space: $(df -h . | tail -1)"
  exit $exit_code
}

trap 'error_handler $LINENO' ERR

asterisk_dependencies() {

  packman=$1
  echo "$packman"
  case "$packman" in
  pacman)
    log_info "Detected Arch based system"

    # Basic build system:
    PACKAGES_ARCH="make gcc pkg-config autoconf-archive"
    # Asterisk: basic requirements:
    PACKAGES_ARCH="$PACKAGES_ARCH libedit jansson libutil-linux libxml2 sqlite"
    # Asterisk: for addons:
    PACKAGES_ARCH="$PACKAGES_ARCH speex speexdsp libogg libvorbis portaudio curl xmlstarlet bison flex"
    PACKAGES_ARCH="$PACKAGES_ARCH postgresql-libs unixodbc neon gmime3 lua uriparser libxslt openssl"
    PACKAGES_ARCH="$PACKAGES_ARCH libmariadbclient bluez-libs radcli freetds bash libcap"
    PACKAGES_ARCH="$PACKAGES_ARCH net-snmp libnewt popt libical spandsp"
    PACKAGES_ARCH="$PACKAGES_ARCH c-client binutils libsrtp gsm doxygen graphviz zlib-ng-compat libldap"
    PACKAGES_ARCH="$PACKAGES_ARCH fftw libsndfile unbound"
    # Asterisk: for the unpackaged below:
    PACKAGES_ARCH="$PACKAGES_ARCH wget subversion"
    # Asterisk: for ./configure --with-pjproject-bundled:
    PACKAGES_ARCH="$PACKAGES_ARCH bzip2 patch"

    if ! paru -S $PACKAGES_ARCH --needed --noconfirm; then
      log_error "Failed to install Docker "
      exit 1
    fi
    ;;
  *)
    # Install optional dependencies
    log_info "Installing optional dependencies..."
    if ! sudo ./contrib/scripts/install_prereq install; then
      log_error "Failed to install Asterisk prerequisites"
      exit 1
    fi

    # Get MP3 support
    log_info "Installing MP3 support..."
    if ! sudo contrib/scripts/get_mp3_source.sh; then
      log_error "Failed to install MP3 support"
      exit 1
    fi
    ;;
  esac

}

install_main() {

  ASTERISK_VERSION="22"
  ASTERISK_URL="https://downloads.asterisk.org/pub/telephony/asterisk/asterisk-${ASTERISK_VERSION}-current.tar.gz"
  BUILD_DIR="/usr/src"
  ASTERISK_USER="asterisk"

  echo "🚀 Installing Asterisk ${ASTERISK_VERSION}..."

  # Create asterisk user if it doesn't exist
  if ! id "$ASTERISK_USER" >/dev/null 2>&1; then
    log_info "Creating asterisk user..."
    sudo useradd -r -d /var/lib/asterisk -s /bin/bash "$ASTERISK_USER" || {
      log_error "Failed to create asterisk user"
      exit 1
    }
  fi

  cd "$BUILD_DIR" || {
    log_error "Failed to change to build directory: $BUILD_DIR"
    exit 1
  }

  # Clean up any existing asterisk files
  log_info "Cleaning up existing files..."
  sudo rm -rf asterisk-* || true

  # Download Asterisk
  log_info "Downloading Asterisk ${ASTERISK_VERSION}..."
  if ! sudo wget -O "asterisk-${ASTERISK_VERSION}.tar.gz" "$ASTERISK_URL"; then
    log_error "Failed to download Asterisk from $ASTERISK_URL"
    exit 1
  fi

  # Extract
  log_info "Extracting archive..."
  if ! sudo tar -xzf "asterisk-${ASTERISK_VERSION}.tar.gz"; then
    log_error "Failed to extract Asterisk archive"
    exit 1
  fi

  # Find the extracted directory
  ASTERISK_DIR=$(sudo find . -name "asterisk-*" -type d | head -n1)
  if [[ -z "$ASTERISK_DIR" ]]; then
    log_error "Failed to find extracted Asterisk directory"
    exit 1
  fi

  log_info "Found Asterisk directory: $ASTERISK_DIR"
  cd "$ASTERISK_DIR" || {
    log_error "Failed to change to Asterisk directory: $ASTERISK_DIR"
    exit 1
  }

  asterisk_dependencies $packman

  # Configure build
  log_info "Configuring build..."
  if ! sudo ./configure \
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

  echo "📋 Launching menuselect..."
  echo ""
  echo "🔧 MENUSELECT INSTRUCTIONS:"
  echo "   • Use arrow keys to navigate"
  echo "   • Press ENTER to enter a category"
  echo "   • Press SPACE to enable/disable modules"
  echo "   • Press 'x' to exit a category"
  echo "   • Press 'q' to quit and save"
  echo ""

  # Run menuselect interactively
  if ! sudo make menuselect; then
    log_error "Menuselect failed or was cancelled"
    exit 1
  fi

  echo ""
  log_info "Compiling Asterisk (this may take a while)..."
  log_info "Using $(nproc) CPU cores for faster compilation..."

  # Build with all CPU cores
  if ! sudo make -j"$(nproc)"; then
    log_error "Failed to compile Asterisk"
    exit 1
  fi

  log_info "Installing Asterisk..."
  if ! sudo make install; then
    log_error "Failed to install Asterisk"
    exit 1
  fi

  log_info "Installing sample configurations..."
  if ! sudo make samples; then
    log_error "Failed to install sample configurations"
    exit 1
  fi

  case $packman in
  pacman)
    ASTERISK_DIR=$(sudo find /usr/src -name "asterisk-*" -type d | head -n1)
    log_info "Installing init scripts for Arch based and systemd... "
    log_info "ASTERISK_DIR : $ASTERISK_DIR"
    SYSTEMD_CONF_DIR="$ASTERISK_DIR/contrib/systemd"
    log_info "SYSTEMD_CONF_DIR : $SYSTEMD_CONF_DIR"
    if [[ -d "$SYSTEMD_CONF_DIR" ]]; then
      log_success "Systemd directory found"
      cd "$SYSTEMD_CONF_DIR" && ls -A

      if command -v rsync >/dev/null 2>&1; then
        if ! sudo rsync -av \
          --exclude="README.txt" \
          "$SYSTEMD_CONF_DIR"/* \
          /etc/systemd/system/; then
          log_error "Failed to copy systemd files"
          exit 1
        fi
        log_success "Systemd files copied!"
        log_info "Please check them before starting or enabling anydeamon"
        sleep 3
      else
        if ! sudo cp -u "$SYSTEMD_CONF_DIR/*.socket" "$SYSTEMD_CONF_DIR/*.service" "$SYSTEMD_CONF_DIR/*timer" "/etc/systemd/system/"; then
          log_error "Failed to copy systemd files"
          exit 1
        fi
        log_success "Systemd files copied!"
        log_info "Please check them before starting or enabling anydeamon"
        sleep 3
      fi

    else
      log_error "Unable to find the systemd directory"
      exit 1
    fi
    ;;
  *)
    log_info "Installing init scripts..."
    if ! sudo make config; then
      log_error "Failed to install init scripts"
      exit 1
    fi
    ;;
  esac

  # Set proper ownership
  log_info "Setting permissions..."
  sudo chown -R "$ASTERISK_USER:$ASTERISK_USER" /var/lib/asterisk
  sudo chown -R "$ASTERISK_USER:$ASTERISK_USER" /var/log/asterisk
  sudo chown -R "$ASTERISK_USER:$ASTERISK_USER" /var/spool/asterisk
  sudo chown -R "$ASTERISK_USER:$ASTERISK_USER" /etc/asterisk

  # Update shared libraries
  log_info "Updating shared libraries..."
  sudo ldconfig

  # Clean up build files
  log_info "Cleaning up build files..."
  cd /
  sudo rm -rf "$BUILD_DIR/asterisk-${ASTERISK_VERSION}.tar.gz"

  log_success "Asterisk installation completed!"
}

install_main
