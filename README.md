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

Pick based on how much RAM you can spare. The model needs to fit entirely in RAM — if it doesn't, it will swap to disk and become unusably slow.

| Model | RAM needed | Speed (CPU) | Quality | Best for |
|---|---|---|---|---|
| `qwen2.5:0.5b` | ~0.5 GB | very fast | basic | Very low-end hardware, simple Q&A |
| `qwen2.5:1.5b` | ~1.1 GB | fast | good | **Pi 4 / 4 GB RAM — recommended default** |
| `qwen2.5:3b` | ~2.0 GB | moderate | better | Pi 4 8 GB, or a laptop where speed matters |
| `qwen2.5:7b` | ~4.5 GB | slow* | best | 8+ GB free RAM, ideally with a GPU |
| `qwen2.5-coder:1.5b` | ~1.1 GB | fast | good (code) | Code-heavy workflows on constrained hardware |
| `qwen2.5-coder:7b` | ~4.5 GB | slow* | great (code) | Serious coding assistant, 8+ GB free |
| `phi3:mini` | ~2.3 GB | moderate | good | Alternative if qwen2.5 doesn't suit you |
| `gemma2:2b` | ~1.7 GB | moderate | good | Alternative mid-tier option |

> *"slow" on CPU-only. With an Nvidia GPU, Ollama auto-detects CUDA and 7B runs fast.
>
> Full model library: https://ollama.com/library

The `qwen2.5` series is recommended — it punches above its weight on code and technical Q&A compared to models of similar size.

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
2. Ask which model you want (or use the default `qwen2.5:1.5b`)
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
docker exec ask-ollama ollama pull qwen2.5:3b

# List what's pulled
docker exec ask-ollama ollama list

# Switch the active model — edit ~/.ask/config
MODEL=qwen2.5:3b

# Or use an env var for a one-off
ASK_MODEL=qwen2.5:3b ask explain what a mutex is
```

### Without Docker (native Ollama install)

```bash
# macOS / Linux
curl -fsSL https://ollama.com/install.sh | sh

ollama pull qwen2.5:1.5b
# Ollama runs as a background service automatically

# Then install just the client
bash install.sh
# choose "local" and skip the Docker start prompt
```

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
ASK_MODEL=qwen2.5:1.5b
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

Minimum specs for `qwen2.5:1.5b`: 2 vCPU, 4 GB RAM. Use Tailscale to avoid exposing port 11434 publicly.

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

With a GPU you can comfortably run `qwen2.5:7b` with fast responses.

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
MODEL=qwen2.5:1.5b
```

Environment variables override the config file for one-off use:

```bash
OLLAMA_HOST=http://100.x.x.x:11434 ask what is a goroutine
ASK_MODEL=qwen2.5:7b ask explain transformer attention in detail
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
| Local laptop, qwen2.5:1.5b, CPU | ~1–2s | ~5–7s |
| Pi 4 4GB, qwen2.5:1.5b, CPU | ~1–2s | ~5–7s |
| Pi 4 4GB, qwen2.5:3b, CPU | ~2–3s | ~10–15s |
| Cloud VM, qwen2.5:7b, GPU | ~0.5s | ~2–3s |

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
docker exec ask-ollama ollama pull qwen2.5:1.5b
```

**`fix` says no output captured**

Reload your shell after install: `source ~/.zshrc`. The hook captures stderr — for programs that write errors to stdout, pipe manually:
```bash
your-command 2>&1 | tee ~/.ask/last_output; fix
```

**Fish shell**

Add manually to `~/.config/fish/config.fish`:

```fish
function __ask_preexec --on-event fish_preexec
    exec 3>&2 2>"$HOME/.ask/.cmd_buf"
end
function __ask_precmd --on-event fish_postexec
    exec 2>&3 3>&- 2>/dev/null
    if test -s "$HOME/.ask/.cmd_buf"
        mv "$HOME/.ask/.cmd_buf" "$HOME/.ask/last_output"
    end
end
```
