# Implementation Guide - Project Restructuring

## Overview

This guide tracks the step-by-step implementation of the ari-stt-tts project restructuring. The project is being split into two main directories:

- **`infra/`** → Infrastructure & environment setup (Vagrant, Asterisk, Docker)
- **`app/`** → Application code (Go IVR application)

## Implementation Phases

### Phase 1: Documentation & Setup
- ✅ Create docs directory structure
- ✅ Establish implementation plan
- ⏳ Create comprehensive documentation

### Phase 2: Infrastructure Folder Structure (infra/)
Steps 2-7: Create all infrastructure files and scripts

#### Step 2: Create folder structure
```bash
mkdir -p infra/vagrant/provisioning/asterisk
mkdir -p infra/docker
```

#### Step 3: Create Vagrantfile
- Define Ubuntu 22.04 VM
- Configure port forwarding (8088, 4002)
- Set provisioning hook

#### Step 4: Create bootstrap.sh
- Entry point for provisioning
- Sources dependencies.sh, install.sh, configure.sh
- Global error handling

#### Step 5: Create dependencies.sh
- System package installation
- Docker CE installation
- User group setup

#### Step 6: Create asterisk/install.sh
- Asterisk source download & compilation
- Module building
- Remove Vosk references

#### Step 7: Create asterisk/configure.sh
- Consolidate all configuration modules
- PJSIP, extensions, ARI, HTTP setup
- Logging and assets configuration

### Phase 3: Docker & Cleanup (infra/)
Steps 8: Move Docker files to infra/docker

#### Step 8: Move Docker files
- Move Dockerfile → infra/docker/Dockerfile.app
- Move docker-compose.yaml → infra/docker/
- Update build context paths

### Phase 4: Application Restructuring (app/)
Step 9: Restructure application

#### Step 9: Move app files
- Move main.go → app/cmd/ivr-server/
- Move go.mod, go.sum → app/
- Move internal/ → app/
- Move assets/ → app/

### Phase 5: Documentation
Steps 10: Update all documentation

#### Step 10: Create READMEs
- infra/README.md - Infrastructure setup guide
- app/README.md - Application setup guide
- Root README.md - Project overview

### Phase 6: Cleanup & Validation
Steps 11-12: Remove old files and validate

#### Step 11: Clean up root
- Remove old asterisk/ folder
- Remove Dockerfile from root
- Remove docker-compose.yaml from root

#### Step 12: Validate
- Test Vagrant provisioning
- Verify all paths
- Check documentation

## Key Files During Implementation

### Original → New Locations

| Original | New | Status |
|----------|-----|--------|
| asterisk/installation/modules/00-dependencies.sh | infra/vagrant/provisioning/dependencies.sh | Pending |
| asterisk/installation/modules/10-install.sh | infra/vagrant/provisioning/asterisk/install.sh | Pending |
| asterisk/installation/modules/20-vosk.sh | DELETE | Pending |
| asterisk/configuration/main.sh + modules | infra/vagrant/provisioning/asterisk/configure.sh | Pending |
| Dockerfile | infra/docker/Dockerfile.app | Pending |
| docker-compose.yaml | infra/docker/docker-compose.yaml | Pending |
| main.go | app/cmd/ivr-server/main.go | Pending |
| go.mod, go.sum | app/ | Pending |
| internal/ | app/internal/ | Pending |
| assets/ | app/assets/ | Pending |

## Testing at Each Step

After each major step, verify:
1. Files are in correct location
2. Permissions are preserved
3. No breaking changes introduced
4. Documentation is updated

## Rollback Strategy

Before each step, current working directory is documented. If issues occur:
1. Revert to previous git state
2. Adjust plan based on findings
3. Retry with modifications

## Success Criteria

✅ **Phase Complete when:**
- All files moved to correct locations
- All scripts created and tested
- Documentation is complete
- Root directory is clean
- Vagrant provisioning succeeds
- Application builds and runs

## Notes

- Each step builds on previous steps
- Scripts use error handling & logging
- Preserve original functionality
- Maintain backward compatibility where possible
- Test as we go
