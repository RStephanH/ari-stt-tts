#!/usr/bin/env bash

LOG_FILE="/tmp/asterisk-install-$(date +%Y%m%d-%H%M%S).log"
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
