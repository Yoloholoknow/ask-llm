package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

const fixSystem = `You are a terminal error diagnostician. You will receive structured context about a failed command.
Use the command name and exit code as your primary signal. Diagnose the root cause in one sentence.
Then give the exact fix — a command or code snippet they can run immediately.
Be extremely concise. No preamble. Format: problem on one line, then the fix in a code block.
Assume the shell and OS stated in the context.`

const lastOutputFile = ".ask/last_output"
const maxFixLines = 50

var ansiEscapeRe = regexp.MustCompile(`\x1b(?:\[[0-9;?]*[A-Za-z]|\][^\x07]*\x07|[^[\]])|\r`)

// readSidecar reads ~/.ask/<name>, returning empty string if the file is missing.
func readSidecar(home, name string) string {
	data, err := os.ReadFile(filepath.Join(home, ".ask", name))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func runFix(cfg Config) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("could not find home directory: %w", err)
	}

	// captured output is optional — hooks now record metadata only
	var content string
	if data, err := os.ReadFile(filepath.Join(home, lastOutputFile)); err == nil {
		content = ansiEscapeRe.ReplaceAllString(strings.TrimSpace(string(data)), "")
		// trim to last N lines to keep the prompt focused
		if lines := strings.Split(content, "\n"); len(lines) > maxFixLines {
			content = strings.Join(lines[len(lines)-maxFixLines:], "\n")
		}
	}

	// sidecar context — command/exit/cwd written by hooks on every command
	lastCmd := readSidecar(home, "last_command")
	lastExit := readSidecar(home, "last_exit")
	lastCwd := readSidecar(home, "last_cwd")

	if lastCmd == "" && content == "" {
		return fmt.Errorf("no command captured yet — run a command first\n" +
			"  (make sure you've run install.sh to set up shell hooks)")
	}

	shell := filepath.Base(os.Getenv("SHELL"))
	if shell == "" || shell == "." {
		shell = "unknown"
	}
	goos := runtime.GOOS

	// banner
	if content != "" {
		fmt.Fprintf(os.Stderr, "%s─── captured output ───%s\n%s\n%s──────────────────────%s\n\n",
			ansiDim, ansiReset, content, ansiDim, ansiReset)
	}
	bannerTarget := "last command"
	if lastCmd != "" {
		fmt.Fprintf(os.Stderr, "%sCommand: %s%s\n", ansiDim, lastCmd, ansiReset)
		bannerTarget = lastCmd
	}
	if lastExit != "" {
		fmt.Fprintf(os.Stderr, "%sExit: %s%s\n", ansiDim, lastExit, ansiReset)
	}
	fmt.Fprintf(os.Stderr, "%sDiagnosing %s...%s\n\n", ansiDim, bannerTarget, ansiReset)

	// build structured user message
	var sb strings.Builder
	if lastCmd != "" {
		fmt.Fprintf(&sb, "Command: %s\n", lastCmd)
	}
	if lastExit != "" {
		fmt.Fprintf(&sb, "Exit code: %s\n", lastExit)
	}
	fmt.Fprintf(&sb, "Shell: %s on %s\n", shell, goos)
	if lastCwd != "" {
		fmt.Fprintf(&sb, "Working directory: %s\n", lastCwd)
	}
	if content != "" {
		fmt.Fprintf(&sb, "Output (stdout+stderr, last %d lines):\n%s", maxFixLines, content)
	}

	return stream(cfg, fixSystem, sb.String())
}
