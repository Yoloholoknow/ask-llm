package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const fixSystem = `You are a terminal error diagnostician. The user will give you the output of a failed command.
Diagnose the root cause in one sentence. Then give the exact fix — a command or code snippet they can run immediately.
Be extremely concise. No preamble. Format: problem on one line, then the fix in a code block.`

const lastOutputFile = ".ask/last_output"
const maxFixLines = 50

func runFix(cfg Config) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("could not find home directory: %w", err)
	}

	outputPath := filepath.Join(home, lastOutputFile)
	data, err := os.ReadFile(outputPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("no command output captured yet — run a command first\n" +
				"  (make sure you've run install.sh to set up shell hooks)")
		}
		return fmt.Errorf("could not read last output: %w", err)
	}

	content := strings.TrimSpace(string(data))
	if content == "" {
		return fmt.Errorf("last command produced no output to diagnose")
	}

	// trim to last N lines to keep the prompt focused
	lines := strings.Split(content, "\n")
	if len(lines) > maxFixLines {
		lines = lines[len(lines)-maxFixLines:]
		content = strings.Join(lines, "\n")
	}

	fmt.Fprintf(os.Stderr, "%sDiagnosing last command output...%s\n\n", ansiDim, ansiReset)
	return stream(cfg, fixSystem, "Here is the terminal output to diagnose:\n\n"+content)
}
