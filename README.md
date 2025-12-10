# 📞 **ari-stt-tts**

A complete IVR (Interactive Voice Response) workflow built with **Go**, **Asterisk ARI**, **Deepgram (STT + TTS)**, and **Google Gemini (LLM)**.
This project provides a fully automated conversational IVR system capable of:

* Recording the caller’s request
* Transcribing speech → text
* Processing intent with Gemini
* Generating a spoken response via Deepgram TTS
* Playing the response back to the caller

This repository contains the first working **MVP based on WAV file TTS output**, with future support for **RTP TTS streaming** currently under development.

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

* Deepgram generates a **Linear16 WAV file** with **8000 Hz simple rate**
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
/var/spool/asterisk/recordings:/mnt/tts
```

---

### ✔ Docker Compose development environment

The stack includes:

* Go application
* Shared mounted directory
* Environment variable injection via `.env`
* Logs and recordings persisted on the host machine

---

### ✔ Future RTP version planned

This MVP is based on WAV playback.
A more advanced version using **RTP streaming through ARI ExternalMedia** is being developed on a separate branch.

Some `.env` variables are already prepared for this but **not yet used**.

---

# 🏗 **Architecture Overview**

```
Caller
   ↓
Asterisk (Stasis App)
   ↓ recording
Go IVR App
   ↓ send audio → Deepgram STT
   ↓ text → Gemini LLM
   ↓ LLM output → Deepgram TTS (WAV file)
   ↓ saved to shared volume
Asterisk plays WAV file
```

Shared directory example:

```
/var/spool/asterisk/recordings
   ├─ request.wav
   ├─ request_tts.wav(response of the request)
```

---

# 📦 **Requirements**

* Docker & Docker Compose
* Asterisk 22+ (with ARI enabled)
* Deepgram API key
* Google Gemini API key
* `.env` file configured (see below)

---

# ⚙️ **Environment Variables**

Create a `.env` file in the project root:

```
# ------------------------------
# GENERAL
# ------------------------------
ARI_URL=http://localhost:8088/ari
ARI_WS_URL=ws://localhost:8088/ari/events
ARI_IP=localhost
ARI_USERNAME=your_username
ARI_PASSWORD=your_password
ARI_APPLICATION_NAME=app_name_stasis

# ------------------------------
# DEEPGRAM
# ------------------------------
DEEPGRAM_API_KEY=your_key_here

# ------------------------------
# GEMINI
# ------------------------------
GEMINI_API_KEY=your_key_here

# ------------------------------
# RTP MODE (not used in MVP)
# ------------------------------
EXTERNAL_HOST_IP=localhost
EXTERNAL_MEDIA_PORT=4002
ARI_EXTERNAL_MEDIA_BASE_URL=http://localhost:8088
```

⚠ **Note:**
Some variables (EXTERNAL_HOST_IP,…) are not used in the MVP because the RTP version is still under development.

---

# 🐳 **Running with Docker Compose**

### 1. Build & start the stack

```
docker compose up --build
```

### 2. Asterisk automatically

* exposes ARI
* loads your Stasis application
* interacts with the Go container

### 3. Go app automatically

* waits for ARI events
* processes audio through STT–LLM–TTS
* writes WAV files to the shared folder

---

# ▶️ **Usage Flow**

1. Caller enters the Stasis app
2. System plays the welcome WAV message
3. Caller records a request
4. The Go app fetches the recording through ARI
5. Deepgram transcribes the audio
6. Gemini generates a response
7. Deepgram creates a WAV file
8. Asterisk plays the TTS WAV back to the caller
9. Caller can continue or end the call

---

# 📁 **Project Structure**

```
ari-stt-tts/
│
├── assets/       <-- prerecorded audio message for welcoming (all the audio files in this directory not the directory need to be copied into /var/lib/asterisk/sounds/en of the asterisk server)
│
├── asterisk/ <--- scripts for the asterisk server
│   └── installation/
│                  ├─modules/
                   └──main.sh
│
│
├── internal/
│   ├── ai/ <-- gemini
│   ├── ariutil/ <-- client web socket of ARI
│   ├── externalmedia/ <-- about rpt (still in development)
│   ├── ivr/ <-- ivr handler (call handler, playing sound,etc)
│   ├── stt/ <-- deepgram STT
│   └── tts/ <-- deepgram TTS
│
├── Dockerfile
│
├── go.mod
│
├── go.sum
│
├── main.go
│
├── docker-compose.yaml
│
├── .env <--- example of env file
│
└── README.md
```

---

# 🧪 **Current Limitations**

* RTP streaming not yet implemented (separate branch)
* No retry mechanism for ARI reconnect
* No multi-language support (English only for now)

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
