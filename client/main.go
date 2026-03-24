package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	if err := ensureConfigDir(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not create config dir: %v\n", err)
	}

	cfg := loadConfig()

	// determine command from binary name or first arg
	cmd := filepath.Base(os.Args[0])
	args := os.Args[1:]

	// allow explicit subcommand: ask fix ...
	if len(args) > 0 && (args[0] == "fix" || args[0] == "ask") {
		cmd = args[0]
		args = args[1:]
	}

	var err error
	switch cmd {
	case "fix":
		err = runFix(cfg)
	default:
		err = runAsk(cfg, args)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "\033[31m%v\033[0m\n", err)
		os.Exit(1)
	}
}

// colorStderr returns stderr (used for status messages)
func colorStderr() *os.File {
	return os.Stderr
}
