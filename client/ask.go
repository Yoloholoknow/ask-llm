package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

const askSystem = `You are a concise CLI assistant. Answer directly and briefly.
Prefer practical examples over lengthy explanation. Use code blocks for commands and code.
Never add unnecessary preamble like "Sure!" or "Great question!".`

func runAsk(cfg Config, args []string) error {
	if len(args) > 0 {
		prompt := strings.Join(args, " ")
		return stream(cfg, askSystem, prompt)
	}
	return runInteractive(cfg)
}

func runInteractive(cfg Config) error {
	fmt.Fprintf(os.Stderr, "%s(model: %s — type your question, Ctrl+C to exit)%s\n",
		ansiDim, cfg.Model, ansiReset)

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print(ansiBold + "> " + ansiReset)
		if !scanner.Scan() {
			fmt.Println()
			break
		}
		prompt := strings.TrimSpace(scanner.Text())
		if prompt == "" {
			continue
		}
		if prompt == "exit" || prompt == "quit" {
			break
		}
		fmt.Println()
		if err := stream(cfg, askSystem, prompt); err != nil {
			fmt.Fprintf(os.Stderr, "\033[31mError: %v\033[0m\n", err)
		}
		fmt.Println()
	}
	return nil
}
