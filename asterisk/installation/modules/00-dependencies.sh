#!/usr/bin/env bash
set -euo pipefail

#-----/* Log function section ---*/ --------

# Import logging from main script
log_info() {
  local msg="[DEPS] $1"
  echo -e "\033[0;34m$msg\033[0m"
  [[ -n "${LOG_FILE:-}" ]] && echo "$(date '+%Y-%m-%d %H:%M:%S') $msg" >>"$LOG_FILE"
}

log_error() {
  local msg="[DEPS-ERROR] $1"
  echo -e "\033[0;31m$msg\033[0m"
  [[ -n "${LOG_FILE:-}" ]] && echo "$(date '+%Y-%m-%d %H:%M:%S') $msg" >>"$LOG_FILE"
}

log_success() {
  local msg="[DEPS-SUCCESS] $1"
  echo -e "\033[0;32m$msg\033[0m"
  [[ -n "${LOG_FILE:-}" ]] && echo "$(date '+%Y-%m-%d %H:%M:%S') $msg" >>"$LOG_FILE"
}

# Error handler for this module
error_handler() {
  local exit_code=$?
  local line_number=$1
  log_error "Dependencies installation failed at line $line_number with exit code $exit_code"
  exit $exit_code
}

# Error handler
trap 'error_handler $LINENO' ERR

# /----- function declaration section ---*/ ------------

echo "🛠️ Installing Asterisk prerequisites and Docker CE..."

update_pkg_lists() {
  # Update package lists
  log_info "Updating package lists..."

  if [[ "$packman" == "apt" ]]; then
    sudo apt update || {
      log_error "Failed to update package lists"
      exit 1
    }
  elif [[ "$packman" == "pacman" ]]; then
    sudo pacman -Sy || {
      log_error "Failed to update package lists"
    }
  fi

}

# Core build dependencies
CORE_DEPS=(
  build-essential git subversion wget curl autoconf automake libtool
  pkg-config cmake sqlite3 ca-certificates gnupg lsb-release apt-transport-https
)

# Asterisk-specific libraries
ASTERISK_DEPS=(
  libjansson-dev libxml2-dev libncurses5-dev libssl-dev libedit-dev uuid-dev
  libxslt1-dev libsqlite3-dev libsrtp2-dev libspandsp-dev libgsm1-dev
  libnewt-dev libvorbis-dev libcurl4-openssl-dev libical-dev libneon27-dev
  libgmime-3.0-dev liblua5.2-dev libunbound-dev libsystemd-dev
)

# Audio and media processing
MEDIA_DEPS=(sox mpg123 ffmpeg alsa-utils pulseaudio-utils)

# Python and additional tools
PYTHON_DEPS=(python3 python3-pip python3-dev python3-venv)

install_packages() {
  local category="$1"
  shift
  local packages=("$@")

  log_info "Installing $category..."
  if ! sudo apt install -y "${packages[@]}"; then
    log_error "Failed to install $category"
    exit 1
  fi
  log_success "$category installed successfully"
}

install_docker_ce() {
  log_info "Installing Docker..."

  packman=$1
  echo "$packman"
  case "$packman" in
  apt)
    log_info "Detected Debian/Ubuntu based system"

    # Remove any old Docker packages
    log_info "Removing old Docker packages..."
    sudo apt remove -y docker docker-engine docker.io containerd runc 2>/dev/null || true

    # Add Docker's official GPG key
    log_info "Adding Docker GPG key..."
    sudo mkdir -p /etc/apt/keyrings
    if ! curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg; then
      log_error "Failed to add Docker GPG key"
      exit 1
    fi

    # Set up the Docker repository
    log_info "Setting up Docker repository..."
    echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" | sudo tee /etc/apt/sources.list.d/docker.list >/dev/null

    # Update package index
    log_info "Updating package index for Docker..."
    if ! sudo apt update; then
      log_error "Failed to update package index after adding Docker repository"
      exit 1
    fi

    # Install Docker CE
    log_info "Installing Docker CE packages..."
    if ! sudo apt install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin; then
      log_error "Failed to install Docker CE packages"
      exit 1
    fi
    ;;

  pacman)
    log_info "Detected Arch based system"

    log_info "Installing Docker on Arch based distribution ..."
    if ! sudo pacman -Sy docker --noconfirm; then
      log_error "Failed to install Docker "
      exit 1
    fi
    ;;
  *)
    log_error "Unsupported package manager: $packman"
    exit 1
    ;;
  esac

  # Add current user to docker group
  if ! groups "$USER" | grep -q docker; then
    log_info "Adding user $USER to docker group..."
    sudo usermod -aG docker "$USER"
    log_info "Please log out and back in for Docker group changes to take effect"
  fi

  # Start and enable Docker
  log_info "Starting and enabling Docker service..."
  sudo systemctl start docker || {
    log_error "Failed to start Docker service"
    exit 1
  }
  sudo systemctl enable docker || {
    log_error "Failed to enable Docker service"
    exit 1
  }

  # Test Docker installation
  log_info "Testing Docker installation..."
  if sudo docker run --rm hello-world >/dev/null 2>&1; then
    log_success "Docker CE installed and working correctly!"
  else
    log_error "Docker installation test failed"
    exit 1
  fi
}

dependencies_main() {
  echo "$packman"

  update_pkg_lists
  # Install packages by category
  # TODO: Comments temporarily and leave the dependencies handle by the script provide by Asterisk
  #
  # install_packages "core build tools" "${CORE_DEPS[@]}"
  # install_packages "Asterisk libraries" "${ASTERISK_DEPS[@]}"
  # install_packages "media processing tools" "${MEDIA_DEPS[@]}"
  # install_packages "Python environment" "${PYTHON_DEPS[@]}"

  # Install Docker CE
  #install_docker_ce
  if ! command -v docker >/dev/null; then
    install_docker_ce "$packman"
  fi

  log_success "All dependencies installed successfully!"

}

# function call section
dependencies_main
