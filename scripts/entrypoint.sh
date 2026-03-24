#!/bin/sh
set -e

MODEL="${ASK_MODEL:-qwen2.5:1.5b}"

ollama serve &
SERVER_PID=$!

echo "[ask-llm] Waiting for Ollama to be ready..."
until curl -sf http://localhost:11434/api/tags > /dev/null 2>&1; do
  sleep 1
done

echo "[ask-llm] Ollama ready."

if ! ollama list | grep -q "^${MODEL}"; then
  echo "[ask-llm] Pulling model: ${MODEL} (first run — this may take a few minutes)..."
  ollama pull "${MODEL}"
  echo "[ask-llm] Model pulled."
else
  echo "[ask-llm] Model ${MODEL} already present."
fi

echo "[ask-llm] Warming model into memory..."
echo '{"model":"'"${MODEL}"'","prompt":"hi","stream":false}' \
  | curl -sf -X POST http://localhost:11434/api/generate \
    -H 'Content-Type: application/json' \
    -d @- > /dev/null
echo "[ask-llm] Model warm. Ready."

wait $SERVER_PID
