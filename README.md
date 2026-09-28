# 📞 **ari-stt-tts**

A complete IVR (Interactive Voice Response) workflow built with **Go**, **Asterisk ARI**, **Deepgram (STT + TTS)**, and **Google Gemini (LLM)**.
This project provides a fully automated conversational IVR system capable of:

* Recording the caller's request
* Transcribing speech → text
* Processing intent with Gemini
* Generating a spoken response via Deepgram TTS
* Playing the response back to the caller

This repository contains the first working **MVP based on WAV file TTS output**, with future support for **RTP TTS streaming** currently under development. It also ships a **Vagrant + Tailscale** setup that provisions the PBX VM for you.

---

## 🚀 **Features**

### ✔ Fully automated IVR workflow

* Incoming call enters a Stasis app
* System plays a welcome prompt
* User records a request
* The recording is transcribed using **Deepgram STT**
* The text is processed by **Google Gemini** (LLM)
* The LLM output is converted to audio via **Deepgram TTS**
* Asterisk plays the generated WAV file

---

### ✔ WAV-based TTS MVP (stable)

This version uses **file-based TTS** instead of RTP streaming.

* Deepgram generates a **Linear16 WAV file** with an **8000 Hz sample rate**
* The file is saved in a shared directory
* Asterisk retrieves and plays the file
* Ensures stability and avoids ARI ExternalMedia issues

---

### ✔ Recording + TTS files stored in the same directory

Both:

* the **caller recording**, and
* the **TTS response**

are stored in the **same folder**, which is mounted as a **Docker volume** so both Asterisk and the Go app can access it.

Example (docker-compose):

```
/var/spool/asterisk/recording:/mnt/tts
```

---

### ✔ Reproducible PBX VM (Vagrant + Tailscale)

* Ubuntu 22.04 + Asterisk 22 built from source, with ARI and a basic PJSIP endpoint
* The VM joins your **Tailscale** tailnet, so SIP softphones and debugging tools can reach it from any device on the tailnet
* Docker + Docker Compose installed, repository root mounted at `/vagrant`

---

### ✔ Docker Compose environment

The stack includes:

* Go application (runs in a container on the VM, next to Asterisk)
* Shared mounted directory for recordings and TTS files
* Environment variable injection via `.env`

---

### ✔ Future RTP version planned

This MVP is based on WAV playback.
A more advanced version using **RTP streaming through ARI ExternalMedia** is being developed on a separate branch.

---

# 🏗 **Architecture Overview**

```
Caller (SIP softphone, e.g. over the tailnet)
   ↓
Asterisk (native on the VM, Stasis app)
   ↓ ARI events + recording
Go IVR app (Docker container on the same VM)
   ↓ send audio → Deepgram STT
   ↓ text → Gemini LLM
   ↓ LLM output → Deepgram TTS (WAV file)
   ↓ saved to shared volume
Asterisk plays WAV file
```

Shared directory example:

```
/var/spool/asterisk/recording
   ├─ msg_<channelID>_<timestamp>.wav       (caller recording)
   ├─ msg_<channelID>_<timestamp>_tts.wav   (TTS response)
```

**Code design notes**

* Each incoming call is handled by a `CallSession` (`internal/ivr`) that carries the per-call state and runs the steps in order: transcribe → generate reply → synthesize → play.
* `internal/stt` exposes its own `TranscriptionResult` / `Transcribe()`, so the rest of the app does not depend on Deepgram's SDK types.
* `internal/ariutil` builds and validates the ARI connection from a `Config` struct, failing fast when a required variable is missing.

---

# 📦 **Requirements**

* Docker & Docker Compose (installed on the VM by the provisioning scripts)
* Asterisk 22+ with ARI enabled (provisioned by the Vagrant setup below)
* Vagrant + VirtualBox on the host (for the PBX VM)
* A Tailscale account and a reusable auth key
* Deepgram API key
* Google Gemini API key
* `.env` file configured (see below)
* [`mise`](https://mise.jdx.dev/) (optional task runner used for the test tasks)

---

# 🧱 **Infrastructure (Vagrant + Tailscale)**

**Guide:** `infra/README.md`

The Vagrant setup provisions the PBX VM in this order: system dependencies (+ Docker) → Tailscale join → Asterisk build/install → Asterisk configuration (ARI, HTTP, PJSIP, dialplan, sounds).

Quick start, from the repository root:

```bash
cp env.example .env                 # then fill in the values
set -a && source .env && set +a     # Vagrant reads variables from your shell, not from .env
cd infra/vagrant
vagrant up
```

Notes:

* `set -a` must be active **while** sourcing `.env`, otherwise the variables are not exported to the `vagrant` process and provisioning fails on missing variables.
* Generate a reusable Tailscale auth key at <https://login.tailscale.com/admin/settings/keys> (reusable is handy if you destroy and recreate the VM).
* Ports 8088 (ARI), 4002 and 5060/udp are also forwarded to `localhost`.
* The prerecorded prompts in `infra/vagrant/assets/` are copied into Asterisk's sounds directory during provisioning.

### Testing with a SIP softphone

Register a softphone (Zoiper, Linphone, …) against the VM's Tailscale IP (`tailscale ip -4` inside the VM), port 5060/UDP. The softphone device must be on the same tailnet.

* Default dev credentials: `1001` / `1001pass` — override them with `PJSIP_ENDPOINT_ID` and `PJSIP_PASSWORD` before provisioning.
* Dial the endpoint ID itself (`1001` by default): plays a test prompt, useful to check SIP and audio.
* Dial `6001`: enters the Stasis app and starts the IVR.

---

# ⚙️ **Environment Variables**

Create a `.env` file in the project root (start from `env.example`):

```
# ------------------------------
# ARI
# ------------------------------
ARI_URL=http://localhost:8088/ari
ARI_WS_URL=ws://localhost:8088/ari/events
ARI_USERNAME=your_username
ARI_PASSWORD=your_password
ARI_APPLICATION_NAME=app_name_stasis

# ------------------------------
# DEEPGRAM
# ------------------------------
DEEPGRAM_API_KEY=your_deepgram_api_key

# ------------------------------
# GEMINI
# ------------------------------
GEMINI_API_KEY=your_gemini_api_key

# ------------------------------
# TAILSCALE (VM provisioning)
# ------------------------------
TAILSCALE_AUTHKEY=your_tailscale_authkey
TAILSCALE_HOSTNAME=pbx-server
```

⚠ **Notes:**

* `ARI_USERNAME`, `ARI_PASSWORD` and `ARI_APPLICATION_NAME` are used both by the provisioning scripts (written into Asterisk's `ari.conf` and dialplan) and by the Go app, so they must be the same on both sides.
* `ARI_URL` / `ARI_WS_URL` keep `localhost` in `.env` (host-side runs and integration tests). Inside the container, `docker-compose.yaml` overrides them with `host.docker.internal`, which resolves to the VM through `extra_hosts`.
* `.env` is git-ignored — never commit it.

---

# 🐳 **Running with Docker Compose**

Asterisk runs natively on the VM; the Go app runs in a container next to it. From the host:

```bash
cd infra/vagrant
vagrant ssh
```

Then inside the VM:

```bash
cd /vagrant
docker compose up --build
```

The Go app then:

* connects to Asterisk's ARI (`host.docker.internal:8088`)
* waits for Stasis events
* processes audio through STT → LLM → TTS
* writes WAV files to the shared folder (`/var/spool/asterisk/recording`, mounted as `/mnt/tts`)

---

# ▶️ **Usage Flow**

1. Caller enters the Stasis app (dial `6001`)
2. System plays the welcome message — `1` record a request, `0` hang up
3. After recording — `1` re-record, `2` listen back, `3` send the request, `0` hang up
4. The Go app fetches the recording through ARI, Deepgram transcribes it, Gemini generates a response, Deepgram creates a WAV file (a waiting sound plays meanwhile)
5. Asterisk plays the TTS WAV back to the caller
6. Afterwards — `1` new recording, `2` listen to the request, `3` send, `4` replay the response, `0` hang up

---

# 🧪 **Testing**

Tasks are defined in `mise.toml`:

```bash
mise run test-unit      # fast unit tests, no network (go test -short ./...)
mise run test-ariutil   # ARI integration test — needs the VM running
mise run test-stt       # Deepgram integration test — needs DEEPGRAM_API_KEY, consumes API quota
mise run stt-fixture    # regenerate the STT test audio (needs espeak-ng); only when the test phrase changes
```

Integration tests are skipped in `-short` mode, and the integration scripts load `.env` automatically.

---

# 📁 **Project Structure**

```
ari-stt-tts/
│
├── docs/                      <-- planning and implementation notes
│
├── infra/
│   ├── README.md              <-- infrastructure guide
│   └── vagrant/
│       ├── Vagrantfile
│       ├── assets/            <-- prerecorded prompts, copied into Asterisk's sounds directory (/var/lib/asterisk/sounds/en) during provisioning
│       └── provisioning/
│           ├── bootstrap.sh
│           ├── dependencies.sh
│           ├── network/       <-- tailscale.sh
│           └── asterisk/      <-- install.sh, configure.sh
│
├── internal/
│   ├── ai/                    <-- gemini
│   ├── ariutil/               <-- ARI client (config + connection)
│   ├── externalmedia/         <-- about rtp (still in development)
│   ├── ivr/                   <-- call handling (handler, session, dtmf, sound, record)
│   ├── stt/                   <-- deepgram STT
│   └── tts/                   <-- deepgram TTS
│
├── scripts/                   <-- helper scripts used by the mise test tasks
│
├── Dockerfile
├── docker-compose.yaml
├── env.example                <-- example env file (copy to .env)
├── go.mod
├── go.sum
├── main.go
├── mise.toml
└── README.md
```

---

# 🧪 **Current Limitations**

* STT uses Deepgram's REST pre-recorded API (the request is transcribed once the recording ends) — no live streaming yet
* RTP streaming not yet implemented (separate branch)
* No retry mechanism for ARI reconnect
* No multi-language support (English only for now)
* The `ai` and `tts` packages are still being reworked

---

# 🗺 **Roadmap**

### v1.0.0 — MVP (WAV TTS)

✔ STT → Gemini → TTS WAV
✔ ARI event handling
✔ Docker compose integration
✔ Shared file-based workflow

---

# 🤝 **Contributions**

Pull Requests are welcome!
Please branch from `rtp`.

---

# 📄 License

This project is licensed under the **MIT License**.
You are free to use, modify, distribute, and integrate this project into commercial or private software.

See the full license in the [`LICENSE`](./LICENSE.md) file.
