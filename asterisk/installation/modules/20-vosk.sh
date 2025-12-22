#!/usr/bin/env bash
set -euo pipefail

# Import logging from main script
log_info() {
  local msg="[VOSK] $1"
  echo -e "\033[0;34m$msg\033[0m"
  [[ -n "${LOG_FILE:-}" ]] && echo "$(date '+%Y-%m-%d %H:%M:%S') $msg" >>"$LOG_FILE"
}

log_error() {
  local msg="[VOSK-ERROR] $1"
  echo -e "\033[0;31m$msg\033[0m"
  [[ -n "${LOG_FILE:-}" ]] && echo "$(date '+%Y-%m-%d %H:%M:%S') $msg" >>"$LOG_FILE"
}

log_success() {
  local msg="[VOSK-SUCCESS] $1"
  echo -e "\033[0;32m$msg\033[0m"
  [[ -n "${LOG_FILE:-}" ]] && echo "$(date '+%Y-%m-%d %H:%M:%S') $msg" >>"$LOG_FILE"
}

# Error handler for this module
error_handler() {
  local exit_code=$?
  local line_number=$1
  log_error "VOSK installation failed at line $line_number with exit code $exit_code"
  log_error "Current working directory: $(pwd)"
  if [[ -d "vosk-asterisk" ]]; then
    log_error "VOSK build directory exists, check build logs"
  fi
  exit $exit_code
}

trap 'error_handler $LINENO' ERR

MODEL_IMAGE="${1:-alphacep/kaldi-en:latest}"
MODEL_PORT="${2:-2700}"

BUILD_DIR="/usr/src"
ASTERISK_DIR=$(sudo find "${BUILD_DIR}" -name "asterisk-*" -type d | head -n1)
if [[ -z "$ASTERISK_DIR" ]]; then
  log_error "Failed to find extracted Asterisk directory during the ASTERISK_DIR definition"
  exit 1
fi
REPO_URL="https://github.com/alphacep/vosk-asterisk.git"
REPO_DIR="vosk-asterisk"
AST_MODULES_DIR="/usr/lib/asterisk/modules"
AST_CONFIG_DIR="/etc/asterisk"
MODULES_CONF="$AST_CONFIG_DIR/modules.conf"

echo "🗣️ Installing VOSK speech recognition module..."
log_info "Model: $MODEL_IMAGE"
log_info "Port: $MODEL_PORT"

check_prereqs() {
  if ! command -v git >/dev/null; then
    log_error "git is required but not found"
    exit 1
  fi
  if ! command -v docker >/dev/null; then
    log_error "docker is required but not found"
    exit 1
  fi
  log_success "Prerequisites check passed"
}

clone_or_update_repo() {
  log_info "Cloning/updating VOSK-Asterisk repository..."
  if [[ -d "$REPO_DIR/.git" ]]; then
    if ! git -C "$REPO_DIR" fetch --all --prune; then
      log_error "Failed to fetch repository updates"
      exit 1
    fi
    if ! git -C "$REPO_DIR" reset --hard origin/master; then
      log_error "Failed to reset repository to latest"
      exit 1
    fi
  else
    if ! git clone "$REPO_URL" "$REPO_DIR"; then
      log_error "Failed to clone repository from $REPO_URL"
      exit 1
    fi
  fi
  log_success "Repository ready"
}

build_and_install() {
  log_info "Building VOSK module..."
  pushd "$REPO_DIR" >/dev/null || {
    log_error "Failed to enter repository directory"
    exit 1
  }

  if [[ -x "./bootstrap" ]]; then
    if ! ./bootstrap; then
      log_error "Bootstrap failed"
      exit 1
    fi
  else
    if ! autoreconf -fi; then
      log_error "autoreconf failed"
      exit 1
    fi
  fi

  if ! ./configure --with-asterisk="${ASTERISK_DIR}" --prefix=/usr; then
    log_error "Configure failed"
    exit 1
  fi

  if ! make -j"$(nproc)"; then
    log_error "Build failed"
    exit 1
  fi

  if ! sudo make install; then
    log_error "Install failed"
    exit 1
  fi

  # Find and install the module
  if [[ -f "./conf/res_speech_vosk.so" ]]; then
    sudo install -m 0644 "./conf/res_speech_vosk.so" "$AST_MODULES_DIR/"
  elif [[ -f "./res_speech_vosk.so" ]]; then
    sudo install -m 0644 "./res_speech_vosk.so" "$AST_MODULES_DIR/"
  else
    MOD_PATH="$(find . -name 'res_speech_vosk.so' -type f | head -n1 || true)"
    if [[ -n "${MOD_PATH:-}" ]]; then
      sudo install -m 0644 "$MOD_PATH" "$AST_MODULES_DIR/"
    else
      log_error "res_speech_vosk.so not found after build"
      exit 1
    fi
  fi

  popd >/dev/null
  log_success "VOSK module built and installed"
}

ensure_modules_conf() {
  log_info "Configuring Asterisk modules..."
  sudo touch "$MODULES_CONF" || {
    log_error "Failed to create/access modules.conf"
    exit 1
  }

  add_line() {
    local line="$1"
    if ! grep -qE "^[[:space:]]*${line//\//\\}[[:space:]]*$" "$MODULES_CONF"; then
      if ! echo "$line" | sudo tee -a "$MODULES_CONF" >/dev/null; then
        log_error "Failed to add line to modules.conf: $line"
        exit 1
      fi
      log_info "Added to modules.conf: $line"
    fi
  }

  add_line "load = res_http_websocket.so"
  add_line "load = res_speech.so"
  add_line "load = res_speech_vosk.so"
  log_success "Modules configuration updated"
}

start_vosk_container() {
  log_info "Starting VOSK Docker container..."
  local name
  name="vosk-model-$(echo "$MODEL_IMAGE" | tr '/:' '__')"
  # Check if container exists and remove it
  if docker ps -a --format '{{.Names}}' | grep -qx "$name"; then
    log_info "Removing existing container: $name"
    docker rm -f "$name" >/dev/null 2>&1 || true
  fi

  if ! docker run -d \
    --name "$name" \
    --restart unless-stopped \
    -p "${MODEL_PORT}:2700" \
    "$MODEL_IMAGE"; then
    log_error "Failed to start VOSK container"
    exit 1
  fi

  log_success "Started VOSK container '$name' on port ${MODEL_PORT}"
}

main() {
  check_prereqs
  clone_or_update_repo
  build_and_install
  ensure_modules_conf
  start_vosk_container
  log_success "VOSK module installation completed!"
}

main "$@"
