#!/usr/bin/env bash
set -e

REPO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
INSTALL_DIR="$HOME/.local/bin"
CONFIG_DIR="$HOME/.ask"
CONFIG_FILE="$CONFIG_DIR/config"
LAST_OUTPUT="$CONFIG_DIR/last_output"

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
DIM='\033[2m'
RESET='\033[0m'

echo -e "${CYAN}ask-llm installer${RESET}"
echo "---"

# ── 1. Check Go ───────────────────────────────────────────────────────────────
if ! command -v go &>/dev/null; then
  echo -e "${YELLOW}Go not found. Please install Go 1.21+ from https://go.dev/dl/${RESET}"
  exit 1
fi

# ── 2. Command name ───────────────────────────────────────────────────────────
cmd_name="ask"

# ── 3. Build binary ───────────────────────────────────────────────────────────
echo ""
echo "Building client binary..."
cd "$REPO_DIR/client"
go build -ldflags="-s -w" -o "$cmd_name" .
mkdir -p "$INSTALL_DIR"
cp "$cmd_name" "$INSTALL_DIR/$cmd_name"
rm "$cmd_name"
ln -sf "$INSTALL_DIR/$cmd_name" "$INSTALL_DIR/fix"
echo -e "${GREEN}✓ Installed: $INSTALL_DIR/$cmd_name${RESET}"
echo -e "${GREEN}✓ Installed: $INSTALL_DIR/fix (symlink)${RESET}"

# ── 4. Config ─────────────────────────────────────────────────────────────────
mkdir -p "$CONFIG_DIR"

if [[ ! -f "$CONFIG_FILE" ]]; then
  echo ""
  echo "How are you running Ollama?"
  echo -e "  ${CYAN}1${RESET}) Locally on this machine ${DIM}(Ollama installed here or via Docker)${RESET}"
  echo -e "  ${CYAN}2${RESET}) Remote server ${DIM}(Raspberry Pi, cloud VM, another machine)${RESET}"
  echo -n "> "
  read -r deploy_mode
  deploy_mode="${deploy_mode:-1}"

  if [[ "$deploy_mode" == "2" ]]; then
    echo ""
    echo "Enter your server's address (e.g. http://100.x.x.x:11434 for Tailscale):"
    echo -n "> "
    read -r host_input
    host_input="${host_input:-http://localhost:11434}"
    # Ensure scheme is present
    if [[ "$host_input" != http://* && "$host_input" != https://* ]]; then
      host_input="http://${host_input}"
    fi
    # Ensure port is present for http:// only (https reverse proxies use their own port)
    if [[ "$host_input" == http://* ]]; then
      host_rest="${host_input#http://}"
      host_authority="${host_rest%%/*}"
      if [[ "$host_authority" != *:* ]]; then
        host_path="${host_rest#$host_authority}"
        host_input="http://${host_authority}:11434${host_path}"
      fi
    fi
  else
    host_input="http://localhost:11434"
    if command -v docker &>/dev/null && [[ -f "$REPO_DIR/docker-compose.yml" ]]; then
      echo ""
      echo -e "Docker detected. Start Ollama via Docker Compose now? ${DIM}(recommended)${RESET} [Y/n]"
      echo -n "> "
      read -r start_docker
      if [[ "${start_docker:-Y}" =~ ^[Yy]$ ]]; then
        echo ""
        echo "Which model? (press Enter for default)"
        echo -e "${DIM}  Pi 4 (4 GB) — CPU inference, ~3–8 tok/s for 1B models:${RESET}"
        echo -e "${DIM}  ~1.2 GB free → gemma3:1b              (recommended — best quality/speed at 1B)${RESET}"
        echo -e "${DIM}  ~1.4 GB free → qwen3.5:0.8b           (alternative; supports thinking mode)${RESET}"
        echo -e "${DIM}  ~0.9 GB free → llama3.2:1b            (strong instruction following)${RESET}"
        echo -e "${DIM}${RESET}"
        echo -e "${DIM}  More RAM / faster hardware:${RESET}"
        echo -e "${DIM}  ~3 GB free   → qwen3.5:2b-q4_K_M     (better quality, still fits 4 GB Pi)${RESET}"
        echo -e "${DIM}  ~3 GB free   → llama3.2:3b            (instruct, strong general knowledge)${RESET}"
        echo -e "${DIM}  ~4 GB free   → qwen3.5:4b${RESET}"
        echo -e "${DIM}  ~8 GB free   → qwen3.5:9b             (GPU recommended)${RESET}"
        echo -e "${DIM}  See .env.example for full table with Pi 4 tok/s estimates${RESET}"
        echo -n "> "
        read -r model_input
        model_input="${model_input:-gemma3:1b}"

        cp "$REPO_DIR/.env.example" "$REPO_DIR/.env" 2>/dev/null || true
        sed -i "s|^ASK_MODEL=.*|ASK_MODEL=${model_input}|" "$REPO_DIR/.env"
        sed -i "s|^OLLAMA_BIND=.*|OLLAMA_BIND=127.0.0.1|" "$REPO_DIR/.env"

        echo ""
        echo "Starting Ollama..."
        cd "$REPO_DIR"
        docker compose --env-file .env up -d
        echo -e "${GREEN}✓ Ollama started. Run: docker logs -f ask-ollama${RESET}"
        cd "$REPO_DIR/client"
      fi
    else
      echo ""
      echo -e "${DIM}No Docker found. Make sure Ollama is running: https://ollama.com/download${RESET}"
    fi
  fi

  host_input="${host_input%/}"

  echo ""
  echo "Which model should the client use? (press Enter for default: gemma3:1b)"
  echo -e "${DIM}  Must match a model pulled on your Ollama server${RESET}"
  echo -n "> "
  read -r client_model
  client_model="${client_model:-gemma3:1b}"

  cat > "$CONFIG_FILE" <<EOF
# ask-llm configuration
# Edit this file to change your Ollama server or model.
#
# OLLAMA_HOST examples:
#   http://localhost:11434          local machine
#   http://100.x.x.x:11434         Raspberry Pi or cloud VM over Tailscale
#
# MODEL: must match a model pulled on your Ollama server.
# Run 'ollama list' on the server to see what's available.
# See .env.example in the repo for the full table with Pi 4 tok/s estimates.
#
# Pi 4 recommended: gemma3:1b or llama3.2:1b (~4–6 tok/s, good quality)
# qwen3.5:0.8b is an alternative if you want thinking mode support.
#
# THINK=false  # set to true to enable reasoning trace on thinking models (qwen3.5 etc.)
#
# NUM_CTX=1024  # context window size; 1024 is the default and halves bandwidth
#               # vs Ollama's 2048 default — meaningful speed gain on Pi 4 CPU

OLLAMA_HOST=${host_input}
MODEL=${client_model}
CMD_NAME=${cmd_name}
EOF
  echo -e "${GREEN}✓ Config written to $CONFIG_FILE${RESET}"
else
  echo -e "${GREEN}✓ Config already exists at $CONFIG_FILE${RESET}"
fi

# ── 5. Shell hooks ────────────────────────────────────────────────────────────
HOOK_BASH='
# ask-llm: record last command metadata for `fix`
__ask_preexec() {
  [[ -d "$HOME/.ask" ]] || return
  [[ "$BASH_COMMAND" == __ask_precmd* ]] && return
  [[ "$BASH_COMMAND" == fix || "$BASH_COMMAND" == "fix "* ]] && return
  [[ "$BASH_COMMAND" == ask || "$BASH_COMMAND" == "ask "* ]] && return
  [[ -n "$__ask_seen" ]] && return
  __ask_seen=1
  echo "$BASH_COMMAND" > "$HOME/.ask/last_command"
  echo "$PWD"          > "$HOME/.ask/last_cwd"
}
__ask_precmd() {
  local _ec=$?
  [[ -d "$HOME/.ask" ]] || return
  [[ -n "$__ask_seen" ]] && echo "$_ec" > "$HOME/.ask/last_exit"
  unset __ask_seen
}
trap __ask_preexec DEBUG
PROMPT_COMMAND="__ask_precmd${PROMPT_COMMAND:+;$PROMPT_COMMAND}"
'

HOOK_ZSH='
# ask-llm: record last command metadata for `fix`
__ask_preexec() {
  [[ -d "$HOME/.ask" ]] || return
  [[ "$1" == fix || "$1" == "fix "* ]] && return
  [[ "$1" == ask || "$1" == "ask "* ]] && return
  __ask_seen=1
  print -r -- "$1"   > "$HOME/.ask/last_command"
  print -r -- "$PWD" > "$HOME/.ask/last_cwd"
}
__ask_precmd() {
  local _ec=$?
  [[ -d "$HOME/.ask" ]] || return
  [[ -n "$__ask_seen" ]] && print -r -- "$_ec" > "$HOME/.ask/last_exit"
  unset __ask_seen
}
autoload -Uz add-zsh-hook
add-zsh-hook preexec __ask_preexec
add-zsh-hook precmd  __ask_precmd
'

# If the command is named 'ask', append an unalias so it wins over the
# oh-my-zsh web-search plugin alias (alias ask='web_search ask').
# This is safe — the managed block is stripped+rewritten on each reinstall.
if [[ "$cmd_name" == "ask" ]]; then
  HOOK_ZSH+='
# clear oh-my-zsh web-search alias that would shadow this command
unalias ask 2>/dev/null || true'
fi

PATH_LINE='export PATH="$HOME/.local/bin:$PATH"'
MARKER='# ask-llm hooks'
END_MARKER='# end ask-llm hooks'

install_hooks() {
  local rc_file="$1"
  local hook="$2"
  local replaced=0

  if [[ ! -f "$rc_file" ]]; then
    return
  fi

  # Strip existing managed block so reinstall always upgrades to the latest hooks.
  if grep -q "$MARKER" "$rc_file" 2>/dev/null; then
    local tmp
    tmp="$(mktemp)"
    awk -v marker="$MARKER" -v end_marker="$END_MARKER" -v path_line="$PATH_LINE" '
      /^[[:space:]]*$/ && found { next }
      $0 == marker { found=1; next }
      found && $0 == path_line { found=0; next }
      $0 == end_marker { found=0; next }
      found { next }
      { print }
    ' "$rc_file" > "$tmp"
    mv "$tmp" "$rc_file"
    replaced=1
  fi

  {
    echo "$MARKER"
    echo ""
    echo "$hook"
    echo "$PATH_LINE"
    echo "$END_MARKER"
  } >> "$rc_file"
  if (( replaced )); then
    echo -e "${GREEN}✓ Hooks updated in $rc_file${RESET}"
  else
    echo -e "${GREEN}✓ Hooks added to $rc_file${RESET}"
  fi
}

install_hooks "$HOME/.bashrc" "$HOOK_BASH"
install_hooks "$HOME/.zshrc" "$HOOK_ZSH"

# ── 6. Done ───────────────────────────────────────────────────────────────────
echo ""
echo -e "${GREEN}All done!${RESET}"
echo ""
echo "Reload your shell:"
echo -e "  ${CYAN}source ~/.zshrc${RESET}   (or ~/.bashrc)"
echo ""
echo "Then try:"
echo -e "  ${CYAN}${cmd_name} how to undo the last git commit${RESET}"
echo -e "  ${CYAN}${cmd_name}${RESET}   (interactive mode)"
echo -e "  ${CYAN}fix${RESET}   (after a command fails)"