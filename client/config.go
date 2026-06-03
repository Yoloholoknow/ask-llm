package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const defaultHost = "http://localhost:11434"
const defaultModel = "gemma3:1b"

type Config struct {
	OllamaHost string
	Model      string
	Think      bool
	NumCtx     int
}

func loadConfig() Config {
	cfg := Config{
		OllamaHost: defaultHost,
		Model:      defaultModel,
		NumCtx:     1024,
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return cfg
	}

	path := filepath.Join(home, ".ask", "config")
	f, err := os.Open(path)
	if err != nil {
		return cfg
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		switch key {
		case "OLLAMA_HOST":
			cfg.OllamaHost = val
		case "MODEL":
			cfg.Model = val
		case "THINK":
			cfg.Think = val == "true"
		case "NUM_CTX":
			if n, err := strconv.Atoi(val); err == nil && n > 0 {
				cfg.NumCtx = n
			}
		}
	}

	// env vars override config file
	if v := os.Getenv("OLLAMA_HOST"); v != "" {
		cfg.OllamaHost = v
	}
	if v := os.Getenv("ASK_MODEL"); v != "" {
		cfg.Model = v
	}
	if v := os.Getenv("ASK_THINK"); v != "" {
		cfg.Think = v == "true"
	}
	if v := os.Getenv("ASK_NUM_CTX"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.NumCtx = n
		}
	}

	return cfg
}

func ensureConfigDir() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	dir := filepath.Join(home, ".ask")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	cfgPath := filepath.Join(dir, "config")
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		content := fmt.Sprintf("# ask-llm configuration\n# Set OLLAMA_HOST to your Pi's Tailscale IP if running remotely\n# e.g. OLLAMA_HOST=http://100.x.x.x:11434\nOLLAMA_HOST=%s\nMODEL=%s\n# THINK=false    # set to true to enable the reasoning trace on thinking models (qwen3.5 etc.)\n# NUM_CTX=1024   # context window; 1024 default halves bandwidth vs Ollama's 2048\n", defaultHost, defaultModel)
		return os.WriteFile(cfgPath, []byte(content), 0644)
	}
	return nil
}
