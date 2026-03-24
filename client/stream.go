package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const retryDelay = 2 * time.Second

// ANSI codes
const (
	ansiReset     = "\033[0m"
	ansiBold      = "\033[1m"
	ansiDim       = "\033[2m"
	ansiCodeBg    = "\033[48;5;236m"
	ansiCodeFg    = "\033[38;5;114m"
	ansiLangColor = "\033[38;5;244m"
)

type ollamaRequest struct {
	Model    string    `json:"model"`
	Prompt   string    `json:"prompt"`
	System   string    `json:"system,omitempty"`
	Stream   bool      `json:"stream"`
	Messages []message `json:"messages,omitempty"`
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaResponse struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

func stream(cfg Config, system, prompt string) error {
	payload := ollamaRequest{
		Model:  cfg.Model,
		Prompt: prompt,
		System: system,
		Stream: true,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	url := strings.TrimRight(cfg.OllamaHost, "/") + "/api/generate"

	var resp *http.Response
	client := &http.Client{Timeout: 0}

	resp, err = client.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		fmt.Fprintf(colorStderr(), "%s\n", unreachableMsg(cfg.OllamaHost, err))
		fmt.Fprintf(colorStderr(), "Retrying in 2s...\n")
		time.Sleep(retryDelay)
		resp, err = client.Post(url, "application/json", bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("unreachable at %s — is Tailscale connected?\n  %w", cfg.OllamaHost, err)
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ollama returned %d — is the model pulled?", resp.StatusCode)
	}

	printer := newStreamPrinter()
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		var chunk ollamaResponse
		if err := json.Unmarshal(scanner.Bytes(), &chunk); err != nil {
			continue
		}
		printer.write(chunk.Response)
		if chunk.Done {
			break
		}
	}
	printer.flush()
	fmt.Println()
	return scanner.Err()
}

func unreachableMsg(host string, err error) string {
	return fmt.Sprintf("\033[33mCould not reach %s: %v\033[0m", host, err)
}

// streamPrinter handles inline code block detection and highlighting as tokens stream in
type streamPrinter struct {
	buf       strings.Builder
	inCode    bool
	langBuf   strings.Builder
	gettingLang bool
	backtickCount int
}

func newStreamPrinter() *streamPrinter {
	return &streamPrinter{}
}

func (p *streamPrinter) write(token string) {
	for _, ch := range token {
		p.processRune(ch)
	}
}

func (p *streamPrinter) processRune(ch rune) {
	if ch == '`' {
		p.backtickCount++
		if p.backtickCount == 3 {
			p.backtickCount = 0
			if !p.inCode {
				p.inCode = true
				p.gettingLang = true
				p.langBuf.Reset()
				fmt.Print("\n" + ansiCodeBg)
			} else {
				p.inCode = false
				p.gettingLang = false
				fmt.Print(ansiReset + "\n")
			}
		}
		return
	}

	// flush any pending backticks that didn't form a triple
	if p.backtickCount > 0 {
		ticks := strings.Repeat("`", p.backtickCount)
		p.backtickCount = 0
		if p.inCode {
			fmt.Print(ansiCodeFg + ticks + ansiReset + ansiCodeBg)
		} else {
			fmt.Print(ticks)
		}
	}

	if p.inCode && p.gettingLang {
		if ch == '\n' {
			p.gettingLang = false
			lang := p.langBuf.String()
			if lang != "" {
				fmt.Print(ansiLangColor + lang + ansiReset + ansiCodeBg + "\n")
			} else {
				fmt.Print("\n")
			}
		} else {
			p.langBuf.WriteRune(ch)
		}
		return
	}

	if p.inCode {
		fmt.Print(ansiCodeFg + string(ch) + ansiReset + ansiCodeBg)
	} else {
		fmt.Print(string(ch))
	}
}

func (p *streamPrinter) flush() {
	if p.inCode {
		fmt.Print(ansiReset)
	}
}
