# File Mapping & Migration Tracker

## Complete File Migration Map

### Infrastructure Files (infra/)

#### 1. Vagrantfile
**Status:** Not yet created  
**Location:** `infra/vagrant/Vagrantfile`  
**Source:** New file  
**Purpose:** Define and provision Ubuntu 22.04 VM  
**Configuration:**
- Box: ubuntu/jammy64 (Ubuntu 22.04 LTS)
- Memory: 2GB (configurable)
- CPUs: 2 (configurable)
- Synced folder: current dir → /vagrant
- Port forwarding: 8088 (Asterisk ARI), 4002 (Go app)
- Provisioning script: provisioning/bootstrap.sh

#### 2. Provisioning Bootstrap Script
**Status:** Not yet created  
**Location:** `infra/vagrant/provisioning/bootstrap.sh`  
**Source:** New file (derived from asterisk/installation/main.sh logic)  
**Purpose:** Main entry point for Vagrant provisioning  
**Responsibilities:**
- Check Vagrant environment
- Set up global logging (/tmp/asterisk-provision-*.log)
- Source other provisioning scripts in order
- Handle overall provisioning flow
- Global error handling and reporting
**Key Features:**
- Logging with timestamps
- Color-coded output
- Error tracking
- Progress reporting

#### 3. System Dependencies Script
**Status:** Not yet created  
**Location:** `infra/vagrant/provisioning/dependencies.sh`  
**Source:** Convert from `asterisk/installation/modules/00-dependencies.sh`  
**Purpose:** Install system packages and Docker  
**Responsibilities:**
- Detect OS (Ubuntu/Debian)
- Update package manager
- Install base packages (build-essential, git, curl, etc.)
- Install Docker CE
- Install Docker Compose
- Add vagrant user to docker group
- Verify installations
**Key Changes from Original:**
- Remove CLI argument parsing (not needed)
- Simplify for Vagrant context
- Make idempotent
- Update logging path

#### 4. Asterisk Installation Script
**Status:** Not yet created  
**Location:** `infra/vagrant/provisioning/asterisk/install.sh`  
**Source:** Convert from `asterisk/installation/modules/10-install.sh`  
**Purpose:** Install Asterisk from source  
**Responsibilities:**
- Download Asterisk source code
- Run menuselect for module selection
- Compile Asterisk
- Install compiled binaries
- Set up systemd service
- Verify installation
**Key Changes from Original:**
- **REMOVE:** Vosk installation logic (SKIP_VOSK=true)
- **REMOVE:** CLI argument parsing
- **SIMPLIFY:** Remove optional module flags
- **PRESERVE:** Build error handling
- **PRESERVE:** Module verification

#### 5. Asterisk Configuration Script
**Status:** Not yet created  
**Location:** `infra/vagrant/provisioning/asterisk/configure.sh`  
**Source:** Consolidate from `asterisk/configuration/main.sh` + 7 modules  
**Purpose:** Configure Asterisk  
**Responsibilities:**
- Load Asterisk configuration files
- Configure PJSIP (SIP protocol)
- Set up extensions (dial plans)
- Enable ARI (Asterisk REST Interface)
- Configure HTTP server
- Set up logging
- Configure shared recording directory
- Download and setup assets
**Key Changes from Original:**
- **CONSOLIDATE:** All 7 modules into one script
- **PRESERVE:** Module functionality (can be grouped by section)
- **ADD:** Clear section comments
- **ENSURE:** Idempotency (safe to re-run)
**Modules Consolidated:**
- 00-assets.sh → Assets section
- ari.sh → ARI configuration
- extensions.sh → Dialplan configuration
- http.sh → HTTP server configuration
- log.sh → Logging configuration
- pjsip.sh → PJSIP configuration
- assets.sh → Asset management

#### 6. Docker Files
**Status:** Not yet created  
**Locations:**
- `infra/docker/Dockerfile.app`
- `infra/docker/docker-compose.yaml`

**Source:** Move from root level  
**Dockerfile.app Purpose:** Build Go IVR application  
**Changes:**
- Rename: Dockerfile → Dockerfile.app
- Preserve: All stages and commands
- Update: Copy paths if needed

**docker-compose.yaml Purpose:** Development environment stack  
**Changes:**
- Move from root to infra/docker/
- Update build context: `.` → `.` (same from new location)
- Verify volume mounts
- Update image references if needed

### Application Files (app/)

#### 1. Main Application
**Status:** Not yet created  
**Location:** `app/cmd/ivr-server/main.go`  
**Source:** Move from `main.go` (root)  
**Purpose:** IVR server entry point  
**Changes:** None - code remains the same

#### 2. Go Modules
**Status:** Not yet created  
**Locations:**
- `app/go.mod`
- `app/go.sum`

**Source:** Move from root  
**Changes:** Update paths in go.mod if any reference root directory

#### 3. Internal Packages
**Status:** Not yet created  
**Location:** `app/internal/`  
**Source:** Move from `internal/` (root)  
**Structure Preserved:**
```
app/internal/
├── ai/              (Gemini LLM integration)
├── ariutil/         (ARI WebSocket client)
├── externalmedia/   (RTP streaming - future)
├── ivr/             (IVR handler)
├── stt/             (Deepgram STT)
└── tts/             (Deepgram TTS)
```
**Changes:** None - structure remains the same

#### 4. Assets
**Status:** Not yet created  
**Location:** `app/assets/`  
**Source:** Move from `assets/` (root)  
**Purpose:** Deepgram asset generation scripts  
**Changes:** None - scripts remain the same

### Documentation Files

#### 1. Plan Document
**Status:** ✅ Created  
**Location:** `docs/PLAN.md`  
**Purpose:** High-level restructuring plan

#### 2. Implementation Guide
**Status:** ✅ Created  
**Location:** `docs/IMPLEMENTATION_GUIDE.md`  
**Purpose:** Step-by-step implementation tracking

#### 3. File Mapping (this document)
**Status:** ✅ Created  
**Location:** `docs/FILE_MAPPING.md`  
**Purpose:** Detailed file migration reference

#### 4. Infrastructure README
**Status:** Not yet created  
**Location:** `infra/README.md`  
**Purpose:** Infrastructure setup instructions

#### 5. Application README
**Status:** Not yet created  
**Location:** `app/README.md`  
**Purpose:** Application setup instructions

#### 6. Root README
**Status:** Needs update  
**Location:** `README.md` (root)  
**Changes:**
- Add project overview
- Link to infra/ and app/
- Update file structure diagram
- Update "Getting Started" section

### Root Level Changes

#### Files to Keep
- `LICENSE.md`
- `README.md` (updated)
- `.git*` (git files)
- `docs/` (new)

#### Files to Keep (Optional)
- `.env.example` - Can stay at root OR move to app/

#### Files to Remove
- ❌ `Dockerfile` → moved to infra/docker/Dockerfile.app
- ❌ `docker-compose.yaml` → moved to infra/docker/
- ❌ `asterisk/` → logic extracted to infra/vagrant/provisioning/
- ❌ `main.go` → moved to app/cmd/ivr-server/
- ❌ `go.mod` → moved to app/
- ❌ `go.sum` → moved to app/
- ❌ `internal/` → moved to app/
- ❌ `assets/` → moved to app/

## Directory Structure - Before & After

### BEFORE (Current)
```
ari-stt-tts/
├── asterisk/
│   ├── installation/
│   │   ├── main.sh
│   │   └── modules/
│   │       ├── 00-dependencies.sh
│   │       ├── 10-install.sh
│   │       └── 20-vosk.sh (deprecated)
│   └── configuration/
│       ├── main.sh
│       └── modules/
│           ├── 00-assets.sh
│           ├── ari.sh
│           ├── assets.sh
│           ├── extensions.sh
│           ├── http.sh
│           ├── log.sh
│           └── pjsip.sh
├── assets/
│   └── get_assets.sh
├── internal/
│   ├── ai/
│   ├── ariutil/
│   ├── externalmedia/
│   ├── ivr/
│   ├── stt/
│   └── tts/
├── .env
├── .gitignore
├── Dockerfile
├── docker-compose.yaml
├── go.mod
├── go.sum
├── main.go
├── README.md
└── LICENSE.md
```

### AFTER (Target)
```
ari-stt-tts/
├── docs/
│   ├── PLAN.md
│   ├── IMPLEMENTATION_GUIDE.md
│   └── FILE_MAPPING.md
├── infra/
│   ├── vagrant/
│   │   ├── Vagrantfile
│   │   └── provisioning/
│   │       ├── bootstrap.sh
│   │       ├── dependencies.sh
│   │       └── asterisk/
│   │           ├── install.sh
│   │           └── configure.sh
│   ├── docker/
│   │   ├── Dockerfile.app
│   │   └── docker-compose.yaml
│   └── README.md
├── app/
│   ├── cmd/
│   │   └── ivr-server/
│   │       └── main.go
│   ├── internal/
│   │   ├── ai/
│   │   ├── ariutil/
│   │   ├── externalmedia/
│   │   ├── ivr/
│   │   ├── stt/
│   │   └── tts/
│   ├── assets/
│   │   └── get_assets.sh
│   ├── go.mod
│   ├── go.sum
│   └── README.md
├── .env
├── .gitignore
├── README.md
└── LICENSE.md
```

## Migration Checklist

### Phase 1: Documentation Setup
- [x] Create docs/ directory
- [x] Move plan.md to docs/PLAN.md
- [x] Create IMPLEMENTATION_GUIDE.md
- [x] Create FILE_MAPPING.md (this document)

### Phase 2: Infrastructure (infra/)
- [ ] Create infra/vagrant/ directory structure
- [ ] Create infra/vagrant/provisioning/ directory structure
- [ ] Create Vagrantfile
- [ ] Create provisioning/bootstrap.sh
- [ ] Create provisioning/dependencies.sh
- [ ] Create provisioning/asterisk/install.sh
- [ ] Create provisioning/asterisk/configure.sh
- [ ] Create infra/docker/ directory structure
- [ ] Move Dockerfile to infra/docker/Dockerfile.app
- [ ] Move docker-compose.yaml to infra/docker/
- [ ] Create infra/README.md

### Phase 3: Application (app/)
- [ ] Create app/cmd/ivr-server/ directory structure
- [ ] Move main.go to app/cmd/ivr-server/
- [ ] Move go.mod to app/
- [ ] Move go.sum to app/
- [ ] Move internal/ to app/
- [ ] Move assets/ to app/
- [ ] Create app/README.md

### Phase 4: Cleanup & Documentation
- [ ] Update root README.md
- [ ] Remove old asterisk/ folder
- [ ] Remove Dockerfile from root
- [ ] Remove docker-compose.yaml from root
- [ ] Remove main.go from root
- [ ] Remove go.mod from root
- [ ] Remove go.sum from root
- [ ] Remove internal/ from root
- [ ] Remove assets/ from root

### Phase 5: Validation
- [ ] Verify git status clean
- [ ] Test Vagrant provisioning
- [ ] Verify paths in all scripts
- [ ] Check documentation completeness
- [ ] Final validation pass

## Notes

- Git history is preserved during moves
- Scripts should maintain executable permissions
- Error handling preserved from originals
- Logging paths updated for new locations
- Idempotency considered where possible
