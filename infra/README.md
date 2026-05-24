## Infrastructure (Vagrant + Asterisk)

This folder contains the Infrastructure-as-Code setup for the PBX server.
It provisions an Ubuntu 22.04 VM with Asterisk 22, ARI enabled, and a minimal
PJSIP + dialplan for testing.

### Prerequisites

- Vagrant
- VirtualBox
- Enough RAM/CPU for compilation (2 CPU / 2GB RAM minimum)

### Quick start

From the repository root:

```bash
cd infra/vagrant
vagrant up
```

This will:
- Install system dependencies and Docker
- Build and install Asterisk 22
- Generate Asterisk config files (ARI/HTTP/PJSIP/extensions/logger)
- Restart Asterisk

### Interactive menuselect

Asterisk compilation runs `make menuselect`. This is interactive. If Vagrant
hangs, it is waiting for your input. Use the on-screen menu instructions and
press `q` to save/exit when finished.

### Default configuration values

You can override these by exporting environment variables before `vagrant up`
or `vagrant provision`:

| Variable | Default | Purpose |
| --- | --- | --- |
| `ARI_USERNAME` | `ariuser` | ARI HTTP username |
| `ARI_PASSWORD` | `aripass` | ARI HTTP password |
| `ARI_APPLICATION` | `ari-stt-tts` | Stasis app name |
| `HTTP_BIND_ADDR` | `0.0.0.0` | Asterisk HTTP bind address |
| `HTTP_BIND_PORT` | `8088` | Asterisk HTTP port |
| `PJSIP_ENDPOINT_ID` | `1001` | Example SIP endpoint ID |
| `PJSIP_PASSWORD` | `1001pass` | Example SIP endpoint password |
| `COPY_ASSETS` | `true` | Copy WAV assets to Asterisk sounds |
| `ASSETS_DIR` | `/vagrant/assets` | WAV assets directory |

Example:

```bash
export ARI_USERNAME=admin
export ARI_PASSWORD=secret
export PJSIP_ENDPOINT_ID=2001
export PJSIP_PASSWORD=2001pass
cd infra/vagrant
vagrant provision
```

### Port mapping

- `8088` -> ARI HTTP (`http://localhost:8088/ari`)
- `4002` -> Go IVR app (if running on the VM)
- `5060/udp` -> SIP signaling (PJSIP)

### Useful commands inside the VM

```bash
vagrant ssh
sudo asterisk -r
systemctl status asterisk
tail -f /var/log/asterisk/messages
```

### Troubleshooting

- **Asterisk does not start**: check `/var/log/asterisk/messages`.
- **Provisioning fails**: check `/tmp/asterisk-provision-*.log`.
- **Missing WAV prompts**: ensure `assets/*.wav` exists in the repo and re-run
  `vagrant provision` (or set `COPY_ASSETS=true`).
