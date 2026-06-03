# ask-llm

CLI tool: pipe text to a local LLM (Ollama) from the terminal. Two commands — `ask` (query) and `fix` (diagnoses last failed command).

## Stack
- Backend: Go 1.26 (`client/` — single binary)
- Inference: Ollama (Docker, default model `gemma3:1b`)
- No frontend, no database

## Commands
- Install: `bash install.sh` (builds binary + symlinks, installs shell hooks; command name hardcoded to `ask`)
- Build: `cd client && go build -o ask .`
- Run Ollama: `docker compose up -d`
- No test suite yet

## Conventions
- Config stored in `~/.ask/config` (key=value)
- Binary installed to `~/.local/bin/` via symlink
- `ask` subcommand default; `fix` diagnoses last failed command (reads sidecar files `~/.ask/last_command`, `~/.ask/last_exit`, `~/.ask/last_cwd`)

## Layout
- `client/` — Go CLI source (ask.go, fix.go, config.go, stream.go, main.go)
- `scripts/entrypoint.sh` — Docker Ollama startup + model warmup
- `docker-compose.yml` — Ollama service definition
- `install.sh` — installer (non-interactive; hardcodes command name `ask`)
