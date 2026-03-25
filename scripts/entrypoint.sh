#!/bin/sh
set -e

MODEL="${ASK_MODEL:-qwen2.5:1.5b}"

ollama serve &
SERVER_PID=$!

echo "[ask-llm] Waiting for Ollama to be ready..."
until ollama list > /dev/null 2>&1; do
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
ollama run "${MODEL}" "hi" > /dev/null 2>&1
echo "[ask-llm] Model warm. Ready."

wait $SERVER_PID