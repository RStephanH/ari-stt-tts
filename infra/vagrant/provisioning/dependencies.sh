#!/usr/bin/env bash

################################################################################
# System Dependencies Installation Script
################################################################################
#
# Purpose:
#   Install system packages and Docker CE required for Asterisk and the IVR
#   application to run.
#
# This script:
#   - Detects OS (Ubuntu/Debian)
#   - Updates package manager
#   - Installs Docker CE and Docker Compose (plugin)
#   - Configures user permissions for Docker
#   - Verifies installations
#
# Called by: provisioning/bootstrap.sh
# Exit code: 0 on success, 1 on failure
#
# Environment variables (from bootstrap.sh):
#   - LOG_FILE: Path to provisioning log file
#   - VAGRANT_ENVIRONMENT: Whether running under Vagrant
#
################################################################################

set -euo pipefail

# ============================================================================
# Logging (inherited from bootstrap.sh, with fallback)
# ============================================================================

log_info() {
  local msg="[DEPS] $1"
  echo -e "\033[0;34m$msg\033[0m" >&2
  [[ -n "${LOG_FILE:-}" ]] && echo "$(date '+%Y-%m-%d %H:%M:%S') $msg" >>"$LOG_FILE"
}

log_success() {
  local msg="[DEPS-SUCCESS] $1"
  echo -e "\033[0;32m$msg\033[0m" >&2
  [[ -n "${LOG_FILE:-}" ]] && echo "$(date '+%Y-%m-%d %H:%M:%S') $msg" >>"$LOG_FILE"
}

log_warning() {
  local msg="[DEPS-WARNING] $1"
  echo -e "\033[1;33m$msg\033[0m" >&2
  [[ -n "${LOG_FILE:-}" ]] && echo "$(date '+%Y-%m-%d %H:%M:%S') $msg" >>"$LOG_FILE"
}

log_error() {
  local msg="[DEPS-ERROR] $1"
  echo -e "\033[0;31m$msg\033[0m" >&2
  [[ -n "${LOG_FILE:-}" ]] && echo "$(date '+%Y-%m-%d %H:%M:%S') $msg" >>"$LOG_FILE"
}

# Error handler
error_handler() {
  local exit_code=$?
  local line_number=$1
  log_error "Dependencies installation failed at line $line_number with exit code $exit_code"
  exit $exit_code
}

trap 'error_handler $LINENO' ERR
trap 'log_warning "Dependencies installation interrupted"; exit 130' INT TERM

# ============================================================================
# OS Detection
# ============================================================================

detect_os() {
  log_info "Detecting operating system..."

  if [[ ! -f /etc/os-release ]]; then
    log_error "Cannot detect OS: /etc/os-release not found"
    exit 1
  fi

  # Source OS information
  . /etc/os-release

  case "$ID" in
  ubuntu | debian)
    log_info "Detected: $PRETTY_NAME"
    echo "apt"
    ;;
  *)
    log_error "Unsupported OS: $PRETTY_NAME"
    exit 1
    ;;
  esac
}

# ============================================================================
# Package Management Functions
# ============================================================================

update_package_lists() {
  log_info "Updating package lists..."

  if ! apt update; then
    log_error "Failed to update package lists"
    exit 1
  fi

  log_success "Package lists updated"
}

install_core_dependencies() {
  log_info "Installing core dependencies for Asterisk..."

  # Core build tools
  local core_deps=(
    build-essential git subversion wget curl autoconf automake libtool
    pkg-config cmake sqlite3 ca-certificates gnupg lsb-release
    apt-transport-https
  )

  # Asterisk-specific libraries
  local asterisk_deps=(
    libjansson-dev libxml2-dev libncurses5-dev libssl-dev libedit-dev
    uuid-dev libxslt1-dev libsqlite3-dev libsrtp2-dev libspandsp-dev
    libgsm1-dev libnewt-dev libvorbis-dev libcurl4-openssl-dev
    libical-dev libneon27-dev libgmime-3.0-dev liblua5.2-dev
    libunbound-dev libsystemd-dev
  )

  # Audio and media processing
  local media_deps=(sox mpg123 ffmpeg alsa-utils pulseaudio-utils)

  # Python environment
  local python_deps=(python3 python3-pip python3-dev python3-venv)

  # All packages to install
  local all_packages=(
    "${core_deps[@]}" "${asterisk_deps[@]}" "${media_deps[@]}"
    "${python_deps[@]}"
  )

  if ! apt install -y "${all_packages[@]}"; then
    log_error "Failed to install core dependencies"
    exit 1
  fi

  log_success "Core dependencies installed"
}

# ============================================================================
# Docker Installation
# ============================================================================

install_docker_ce() {
  log_info "Installing Docker CE..."

  # Check if Docker is already installed
  if command -v docker &>/dev/null; then
    log_info "Docker is already installed: $(docker --version)"
    return 0
  fi

  # Remove any old Docker packages
  log_info "Removing old Docker packages..."
  apt remove -y docker docker-engine docker.io containerd runc 2>/dev/null || true

  # Add Docker's official GPG key
  log_info "Adding Docker GPG key..."
  mkdir -p /etc/apt/keyrings

  if ! curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg; then
    log_error "Failed to add Docker GPG key"
    exit 1
  fi

  # Set up Docker repository
  log_info "Setting up Docker repository..."
  local distro_codename=$(lsb_release -cs)
  echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu $distro_codename stable" |
    tee /etc/apt/sources.list.d/docker.list >/dev/null

  # Update package lists
  log_info "Updating package lists for Docker..."
  if ! apt update; then
    log_error "Failed to update package lists after adding Docker repository"
    exit 1
  fi

  # Install Docker CE and related tools
  log_info "Installing Docker CE packages..."
  if ! apt install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin; then
    log_error "Failed to install Docker CE packages"
    exit 1
  fi

  log_success "Docker CE installed: $(docker --version)"
}

# ============================================================================
# Docker Configuration
# ============================================================================

configure_docker() {
  log_info "Configuring Docker..."

  # Start Docker service
  log_info "Starting Docker service..."
  if ! systemctl start docker; then
    log_error "Failed to start Docker service"
    exit 1
  fi

  # Enable Docker to start on boot
  log_info "Enabling Docker to start on boot..."
  if ! systemctl enable docker; then
    log_error "Failed to enable Docker service"
    exit 1
  fi

  # Add vagrant user to docker group
  log_info "Adding vagrant user to docker group..."
  if ! groups vagrant | grep -q docker; then
    usermod -aG docker vagrant || {
      log_error "Failed to add vagrant user to docker group"
      exit 1
    }
    log_info "Vagrant user added to docker group (effective after re-login)"
  else
    log_info "Vagrant user is already in docker group"
  fi

  # Test Docker installation
  log_info "Testing Docker installation..."
  if docker run --rm hello-world >/dev/null 2>&1; then
    log_success "Docker test successful"
  else
    log_error "Docker test failed"
    exit 1
  fi
}

# ============================================================================
# Docker Compose (plugin) Verification
# ============================================================================

verify_docker_compose_plugin() {
  log_info "Verifying Docker Compose plugin..."

  if docker compose version &>/dev/null; then
    log_success "Docker Compose plugin available: $(docker compose version | head -n1)"
    return 0
  fi

  log_error "Docker Compose plugin not found. Ensure docker-compose-plugin is installed."
  exit 1
}

# ============================================================================
# Verification
# ============================================================================

verify_installations() {
  log_info "Verifying installations..."

  local success=true

  # Check Docker
  if ! command -v docker &>/dev/null; then
    log_error "Docker not found"
    success=false
  else
    log_info "✓ Docker: $(docker --version)"
  fi

  # Check Docker Compose plugin
  if docker compose version &>/dev/null; then
    log_info "✓ Docker Compose: $(docker compose version | head -n1)"
  else
    log_error "Docker Compose plugin not found (docker compose ...)"
    success=false
  fi

  # Check essential build tools
  for tool in gcc make git curl wget; do
    if ! command -v $tool &>/dev/null; then
      log_error "Required tool not found: $tool"
      success=false
    else
      log_info "✓ $tool: $($tool --version 2>&1 | head -n1)"
    fi
  done

  if [[ "$success" == "true" ]]; then
    log_success "All verifications passed"
    return 0
  else
    log_error "Some verifications failed"
    exit 1
  fi
}

# ============================================================================
# Main Execution
# ============================================================================

main() {
  echo ""
  log_info "=================================="
  log_info "System Dependencies Installation"
  log_info "=================================="

  # Detect OS
  local package_manager
  package_manager=$(detect_os)

  # Update and install
  update_package_lists
  install_core_dependencies
  install_docker_ce
  configure_docker
  verify_docker_compose_plugin

  # Verify
  verify_installations

  log_success "All system dependencies installed successfully!"
  echo ""
}

# Run main
main "$@"
