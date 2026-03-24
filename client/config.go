package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const defaultHost = "http://localhost:11434"
const defaultModel = "qwen2.5:1.5b"

type Config struct {
	OllamaHost string
	Model      string
}

func loadConfig() Config {
	cfg := Config{
		OllamaHost: defaultHost,
		Model:      defaultModel,
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
		}
	}

	// env vars override config file
	if v := os.Getenv("OLLAMA_HOST"); v != "" {
		cfg.OllamaHost = v
	}
	if v := os.Getenv("ASK_MODEL"); v != "" {
		cfg.Model = v
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
		content := fmt.Sprintf("# ask-llm configuration\n# Set OLLAMA_HOST to your Pi's Tailscale IP if running remotely\n# e.g. OLLAMA_HOST=http://100.x.x.x:11434\nOLLAMA_HOST=%s\nMODEL=%s\n", defaultHost, defaultModel)
		return os.WriteFile(cfgPath, []byte(content), 0644)
	}
	return nil
}
