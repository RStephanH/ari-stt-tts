# Project Restructuring Plan: ari-stt-tts

## Executive Summary

This project is an IVR (Interactive Voice Response) system with Go application and Asterisk infrastructure. Currently, the project structure mixes infrastructure code (Asterisk setup scripts) with application code. The goal is to reorganize into clear separation of concerns:

- **`infra/`** → Infrastructure (Vagrant, Asterisk setup)
- **`app/`** → Application code (Go IVR application)

## Current State Analysis

### Current Structure
```
ari-stt-tts/
├── assets/              (Deepgram API calls to generate WAV files)
├── asterisk/            (Asterisk installation & configuration scripts)
│   ├── installation/
│   │   ├── main.sh
│   │   └── modules/
│   │       ├── 00-dependencies.sh
│   │       ├── 10-install.sh
│   │       └── 20-vosk.sh (marked for removal)
│   └── configuration/
│       ├── main.sh
│       └── modules/
│           ├── 00-assets.sh
│           ├── ari.sh
│           ├── extensions.sh
│           ├── http.sh
│           ├── log.sh
│           └── pjsip.sh
├── internal/            (Go application code)
├── Dockerfile           (Go app containerization)
├── docker-compose.yaml  (Development environment)
├── main.go             (Go app entry point)
├── go.mod, go.sum      (Go dependencies)
└── README.md
```

### Key Findings

1. **Infrastructure components:**
   - Bash scripts for Asterisk installation and configuration
   - Docker/Compose setup for development
   - Asset generation scripts
   - No Vagrant setup (pure Docker Compose)

2. **Application components:**
   - Go IVR application (`main.go`, `internal/`)
   - Docker image build for Go app
   - ARI, STT, TTS, AI integration modules

3. **Current setup uses:**
   - Docker Compose for development (not Vagrant)
   - Separate installation and configuration scripts (good!)
   - Vosk module support (to be removed per requirements)

## Proposed Infrastructure Restructuring

### Target Folder Structure
```
infra/
├── vagrant/
│   ├── Vagrantfile              (VM definition)
│   ├── provisioning/
│   │   ├── bootstrap.sh         (Entry point, environment setup)
│   │   ├── asterisk/
│   │   │   ├── install.sh       (Asterisk installation)
│   │   │   └── configure.sh     (Asterisk configuration)
│   │   └── dependencies.sh      (System dependencies)
│   └── config/
│       └── ...                  (Vagrant-specific configs if needed)
├── docker/
│   ├── Dockerfile.app          (Go app container, moved from root)
│   └── docker-compose.yaml     (Moved from root)
└── README.md                   (Infrastructure setup guide)
```

### Proposed Application Restructuring

```
app/
├── cmd/
│   └── ivr-server/
│       └── main.go              (moved from root)
├── internal/                    (unchanged)
├── assets/                      (Deepgram asset generation)
├── go.mod, go.sum              (moved from root)
└── README.md                   (Application guide)
```

## Detailed Changes

### Phase 1: Infrastructure (`infra/`)

#### 1.1 Create Vagrantfile
- Define Ubuntu 22.04 VM
- Configure shared folder for code
- Set up provisioning hook to call bootstrap script
- Configure port forwarding for:
  - Asterisk ARI: 8088
  - Go app: 4002
  - Vosk (removed): -

#### 1.2 Provisioning Scripts

**`provisioning/bootstrap.sh`**
- Entry point for Vagrant provisioning
- Install base dependencies
- Set environment variables
- Call sub-scripts in sequence

**`provisioning/dependencies.sh`**
- Converted from: `asterisk/installation/modules/00-dependencies.sh`
- Install system packages
- Install Docker CE

**`provisioning/asterisk/install.sh`**
- Converted from: `asterisk/installation/modules/10-install.sh`
- Install Asterisk from source
- Remove Vosk installation references

**`provisioning/asterisk/configure.sh`**
- Converted from: `asterisk/configuration/main.sh` + all modules
- Configure Asterisk services
- Set up ARI, PJSIP, extensions
- Asset management

#### 1.3 Removed Files
- `asterisk/installation/modules/20-vosk.sh` → DELETE
- `asterisk/installation/main.sh` → ARCHIVE (logic moved to provisioning scripts)
- `asterisk/configuration/main.sh` → ARCHIVE (logic moved to provisioning scripts)

#### 1.4 Docker Setup
- Keep docker-compose.yaml in infra/docker/
- Rename Dockerfile → Dockerfile.app
- Maintain same functionality for development

### Phase 2: Application (`app/`)

- Move `main.go` to `app/cmd/ivr-server/`
- Move `go.mod`, `go.sum` to `app/`
- Keep `internal/` as-is
- Keep `assets/` for asset generation scripts
- Create app-specific README

### Phase 3: Root Level

Keep minimal files at root:
- `.git*` files
- `LICENSE.md`
- `README.md` (high-level overview)
- `.env.example` (or move to app/)
- Top-level Makefile (optional, for convenience)

## Benefits

1. **Clear Separation:** Infrastructure and application are clearly separated
2. **Reusability:** Infrastructure can be used independently
3. **Scalability:** Easy to add more components (API server, workers, etc.) in `app/`
4. **Maintainability:** Each folder has its own documentation and purpose
5. **Modern Setup:** Vagrant for VM management is more flexible than pure Docker
6. **Cleaner Scripts:** Separated install and configure phases for Asterisk

## Considerations

1. **Vagrant vs Docker Compose:**
   - Vagrant provides full VM isolation
   - Better for understanding system-level requirements
   - Keep docker-compose.yaml as legacy/fallback option

2. **Vosk Removal:**
   - Clean break from legacy speech recognition
   - Deepgram is the standard now

3. **File Permissions:**
   - Ensure scripts remain executable through restructuring
   - Git should preserve executable bit

4. **Testing:**
   - Verify Vagrant provisioning works end-to-end
   - Test Docker Compose still works (if kept)

## Implementation Order

1. ✅ Create analysis and plan (this document)
2. 📋 Create infra folder structure with Vagrantfile
3. 📋 Convert Bash scripts to provisioning scripts
4. 📋 Move application code to app/
5. 📋 Update docker-compose.yaml path
6. 📋 Update documentation (README files)
7. 📋 Test Vagrant provisioning
8. 📋 Clean up old files
9. 📋 Update CI/CD if applicable

## Notes for Implementation

- Keep detailed logging in provisioning scripts
- Preserve error handling from original scripts
- Add comments to explain module relationships
- Ensure idempotency where possible (can re-run scripts)
- Test provisioning on clean Ubuntu 22.04 VM
