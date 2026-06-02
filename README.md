# ask-llm

A fast local CLI assistant powered by a local LLM. No cloud, no API keys, no latency from a browser tab.

```bash
ask how to undo the last git commit
ask                          # interactive mode
fix                          # diagnose the last command that failed
```

---

## How it works

- **Ollama** runs the model as a local HTTP server (port 11434)
- A small **Go binary** (`ask` / `fix`) talks to it and streams responses to your terminal
- The model stays **loaded in memory** between calls — no cold start on every `ask`

You can run everything on your laptop, or offload Ollama to a Raspberry Pi or cloud VM and connect from anywhere over Tailscale.

---

## Choosing a model

Pick based on how much RAM you can spare. The model needs to fit entirely in RAM — if it doesn't, it will swap to disk and become unusably slow. Runtime RAM ≈ download size × 1.3.

| Model | Family / gen | Download | Runtime RAM | Type | Best for |
|---|---|---|---|---|---|
| `gemma3:1b` | Gemma 3 | 815 MB | ~1.2 GB | instruct | Weakest hardware / max speed |
| `qwen3.5:0.8b` | Qwen 3.5 ✦ | 1.0 GB | ~1.4 GB | think→off | **Pi 4 / 4 GB RAM — default** |
| `qwen3.5:2b-q4_K_M` | Qwen 3.5 ✦ | 1.9 GB | ~2.6 GB | think→off | Quality lean, still fits 4 GB |
| `llama3.2:3b` | Llama 3.2 | 2.0 GB | ~2.7 GB | instruct | General knowledge, laptop / 8 GB Pi |
| `granite4.1:3b` | Granite 4.1 | 2.1 GB | ~2.8 GB | instruct | Tools / RAG / code, non-Qwen |
| `qwen3.5:4b` | Qwen 3.5 ✦ | 3.4 GB | ~4.0 GB | think→off | 6 GB+ free RAM |
| `qwen3.5:9b` | Qwen 3.5 ✦ | 6.6 GB | ~8.0 GB | think→off | 8 GB+ / GPU — best quality |

> ✦ Qwen 3.5 supports a reasoning/thinking mode. The client disables it by default for snappy answers (see **Thinking mode** below). `gemma3`, `llama3.2`, and `granite4.1` are instruct models with no reasoning overhead at all.
>
> ⚠ Use `qwen3.5:2b-q4_K_M` (not plain `qwen3.5:2b`) on a 4 GB Pi. The untagged `:2b` defaults to Q8 (2.7 GB) and leaves too little headroom.
>
> *"slow" tiers are fast with a GPU. Ollama auto-detects CUDA/Metal.
>
> Full model library: https://ollama.com/library

### Thinking mode

`qwen3.5` (and other reasoning models like `qwen3`) emit a thinking/reasoning trace before answering. Out of the box `ask` **disables thinking** — the client sends `"think": false` to Ollama and strips any stray `<think>` blocks from the stream, so you get fast, clean output.

To re-enable the reasoning trace (useful for hard problems):

```bash
# One-off
ASK_THINK=true ask explain why merge sort is O(n log n)

# Persistent — add to ~/.ask/config
THINK=true
```

### Throughput

On a Pi 4 the biggest wins, already wired into `docker-compose.yml` by default:

| Lever | Default | Effect |
|---|---|---|
| Think disabled | `ASK_THINK=false` | Skips generating hundreds of reasoning tokens |
| Flash attention | `OLLAMA_FLASH_ATTENTION=1` | Faster attention, lower KV-cache memory |
| Quantized KV cache | `OLLAMA_KV_CACHE_TYPE=q8_0` | Halves context memory cost |
| Single model loaded | `OLLAMA_MAX_LOADED_MODELS=1` | Keeps full RAM for the active model |

**Speculative decoding** (draft-model inference) is **not used** — Ollama doesn't support it yet ([issue #5800](https://github.com/ollama/ollama/issues/5800)). Even if it did, the technique only meaningfully speeds up large models (30B+) on GPU; for a ~1B model on a Pi CPU it adds a draft model that competes for the same scarce RAM.

The very newest model families at the time of writing (`qwen3.6`, `gemma4`) are large-only (27B+) with no small variants, so they're not listed above.

---

## Setup A — Local (Ollama on your laptop/desktop)

This is the simplest setup. Everything runs on your machine.

### Prerequisites

- Go 1.21+ — https://go.dev/dl/
- Docker + Docker Compose — https://docs.docker.com/get-docker/
  *(or install Ollama directly: https://ollama.com/download)*

### Install

```bash
git clone https://github.com/Yoloholoknow/ask-llm
cd ask-llm
bash install.sh
```

The installer will:
1. Build the Go binary and install `ask` + `fix` to `~/.local/bin/`
2. Ask which model you want (or use the default `qwen3.5:0.8b`)
3. Start Ollama via Docker Compose (pulls the model on first run)
4. Write your config to `~/.ask/config`
5. Add shell hooks to `.zshrc` / `.bashrc` for `fix` to work

Then reload your shell:

```bash
source ~/.zshrc   # or ~/.bashrc
```

### Manual model management

```bash
# Pull a different model
docker exec ask-ollama ollama pull qwen3.5:2b-q4_K_M

# List what's pulled
docker exec ask-ollama ollama list

# Switch the active model — edit ~/.ask/config
MODEL=qwen3.5:2b-q4_K_M

# Or use an env var for a one-off
ASK_MODEL=qwen3.5:9b ask explain what a mutex is
```

### Without Docker (native Ollama install)

```bash
# macOS / Linux
curl -fsSL https://ollama.com/install.sh | sh

ollama pull qwen3.5:0.8b
# Ollama runs as a background service automatically

# Then install just the client
bash install.sh
# choose "local" and skip the Docker start prompt
```

### Container hardening

The Docker container runs with a few security and resource constraints out of the box:

- **Memory cap** — `OLLAMA_MEM_LIMIT` (default `3500m`) prevents the container from OOM-killing the host. Set it in `.env` based on your model choice.
- **Single model** — `OLLAMA_MAX_LOADED_MODELS=1` and `OLLAMA_NUM_PARALLEL=1` ensure only one model occupies RAM at a time.
- **Capability drop** — `cap_drop: [ALL]` removes all Linux capabilities from the container.
- **No privilege escalation** — `security_opt: no-new-privileges:true` blocks `setuid` exploits.
- **Pinned image** — the compose file uses a fixed `ollama/ollama:0.24.0` tag rather than `:latest` to prevent unexpected upstream changes.
- **Local-only bind** — port 11434 binds to `127.0.0.1` by default. On a server/Pi, set `OLLAMA_BIND=0.0.0.0` in `.env` and use Tailscale ACLs (see below) rather than exposing the port publicly.

---

## Setup B — Remote server (Raspberry Pi)

Run Ollama on a Pi and use `ask` from any of your machines over Tailscale. The Pi stays on, the model stays warm, and your laptop never spends RAM on it.

### Pi prerequisites

- Raspberry Pi 4, 4 GB RAM minimum (8 GB recommended for larger models)
- Raspberry Pi OS 64-bit (Bookworm)
- Docker + Tailscale

```bash
# Install Docker
curl -fsSL https://get.docker.com | sh
sudo usermod -aG docker $USER
newgrp docker

# Enable cgroups v2 — required for Docker on Pi
# Open /boot/firmware/cmdline.txt (one line, no newlines at all)
sudo nano /boot/firmware/cmdline.txt
# Append at the end of the existing line (space-separated):
#   cgroup_enable=memory cgroup_memory=1
sudo reboot

# Install Tailscale
curl -fsSL https://tailscale.com/install.sh | sh
sudo tailscale up

# Note your Pi's Tailscale IP — you'll need this for the client
tailscale ip -4
```

### Start the server on the Pi

```bash
git clone https://github.com/Yoloholoknow/ask-llm
cd ask-llm
cp .env.example .env
nano .env
```

For a Pi 4 with 4 GB RAM, `.env` should look like:

```bash
ASK_MODEL=qwen3.5:0.8b
OLLAMA_BIND=0.0.0.0        # expose on network so Tailscale can reach it
```

```bash
docker compose --env-file .env up -d

# Watch the first-run model pull
docker logs -f ask-ollama
# Wait for: "[ask-llm] Model warm. Ready."
```

### Install the client on your laptop

```bash
git clone https://github.com/Yoloholoknow/ask-llm
cd ask-llm
bash install.sh
# choose "Remote server"
# enter: http://<pi-tailscale-ip>:11434
```

### Tailscale ACL (recommended)

Restrict port 11434 to your own devices only. In your Tailscale admin panel under Access Controls:

```json
{
  "acls": [
    {
      "action": "accept",
      "src": ["your-laptop-tailscale-ip"],
      "dst": ["pi-tailscale-ip:11434"]
    }
  ]
}
```

---

## Setup C — Remote server (cloud VM)

Same as the Pi setup but on a VPS (Hetzner, DigitalOcean, Linode, etc.). Useful if you want the model available 24/7 without keeping a Pi running, or if you want GPU acceleration.

Minimum specs for `qwen3.5:0.8b`: 2 vCPU, 4 GB RAM. Use Tailscale to avoid exposing port 11434 publicly.

```bash
# On the VM — same steps as Pi, no cgroups step needed
git clone https://github.com/Yoloholoknow/ask-llm
cd ask-llm
cp .env.example .env
# edit .env: OLLAMA_BIND=0.0.0.0, pick your model
docker compose --env-file .env up -d

# Install Tailscale on the VM
curl -fsSL https://tailscale.com/install.sh | sh
sudo tailscale up
```

Client install on your laptop is identical to the Pi path.

### GPU on a cloud VM

If your VM has an Nvidia GPU, first install the Nvidia container toolkit:

```bash
curl -fsSL https://nvidia.github.io/libnvidia-container/gpgkey \
  | sudo gpg --dearmor -o /usr/share/keyrings/nvidia-container-toolkit-keyring.gpg
curl -sL https://nvidia.github.io/libnvidia-container/stable/deb/nvidia-container-toolkit.list \
  | sed 's#deb https://#deb [signed-by=/usr/share/keyrings/nvidia-container-toolkit-keyring.gpg] https://#g' \
  | sudo tee /etc/apt/sources.list.d/nvidia-container-toolkit.list
sudo apt-get update && sudo apt-get install -y nvidia-container-toolkit
sudo nvidia-ctk runtime configure --runtime=docker
sudo systemctl restart docker
```

Then uncomment the GPU block in `docker-compose.yml`:

```yaml
deploy:
  resources:
    reservations:
      devices:
        - driver: nvidia
          count: all
          capabilities: [gpu]
```

With a GPU you can comfortably run `qwen3.5:9b` with fast responses.

---

## Configuration reference

Config lives at `~/.ask/config` on each client machine:

```bash
# ~/.ask/config

# Where Ollama is running
OLLAMA_HOST=http://localhost:11434       # local
# OLLAMA_HOST=http://100.x.x.x:11434    # remote Pi or VM over Tailscale

# Which model to use (must be pulled on the server)
# Run: docker exec ask-ollama ollama list
MODEL=qwen3.5:0.8b
# THINK=false  # set to true to enable the reasoning trace (see Thinking mode above)
```

Environment variables override the config file for one-off use:

```bash
OLLAMA_HOST=http://100.x.x.x:11434 ask what is a goroutine
ASK_MODEL=qwen3.5:9b ask explain transformer attention in detail
ASK_THINK=true ask why is the sky blue
```

---

## Usage

```bash
# Inline — everything after ask is the prompt
ask how to create a new branch in git
ask what does chmod 755 mean
ask explain the difference between tcp and udp

# Interactive mode — run bare, then type freely
ask
> how do I list all docker containers including stopped ones
> what flag shows container sizes
> exit

# Fix — diagnose the last failed command automatically
# No copy-paste needed. fix reads what your terminal just output.
cargo build
fix

npm install
fix
```

---

## Performance expectations

| Setup | First token | Full response (~100 tokens) |
|---|---|---|
| Local laptop, qwen3.5:0.8b, CPU, think off | ~0.5–1s | ~3–5s |
| Pi 4 4GB, qwen3.5:0.8b, CPU, think off | ~1–2s | ~5–8s |
| Pi 4 4GB, qwen3.5:2b-q4_K_M, CPU, think off | ~2–3s | ~10–15s |
| Cloud VM, qwen3.5:9b, GPU | ~0.5s | ~2–3s |

Flash attention and quantized KV cache (both on by default) reduce memory pressure and shorten time-to-first-token, especially noticeable at longer prompts.

The model is kept warm in memory permanently (`OLLAMA_KEEP_ALIVE=-1`). The only slow moment is the very first `docker compose up` on a fresh machine. After that, every `ask` hits a loaded model.

---

## Troubleshooting

**`unreachable — is Tailscale connected?`**
```bash
tailscale status
ping <server-tailscale-ip>
docker ps   # on the server — is ask-ollama running?
```

**Slow responses**
```bash
docker stats ask-ollama   # check RAM — if near the limit, model is swapping
# Fix: switch to a smaller model in ~/.ask/config
```

**Model not found**
```bash
docker exec ask-ollama ollama list
docker exec ask-ollama ollama pull qwen3.5:0.8b
```

**`fix` says no output captured**

Reload your shell after install: `source ~/.zshrc`. The hook captures stderr — for programs that write errors to stdout, pipe manually:
```bash
your-command 2>&1 | tee ~/.ask/last_output; fix
```

**Fish shell**

Fish doesn't support the `exec 2> >(tee ...)` process-substitution pattern needed for
live stderr tee. Auto-capture isn't available. Use manual capture per command instead:

```fish
your-command 2>&1 | tee ~/.ask/last_output; fix
```
