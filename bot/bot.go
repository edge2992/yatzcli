package bot

import (
	"fmt"
	"os"
	"os/exec"
)

type Bot struct {
	addr     string
	name     string
	strategy string
}

func New(addr, name, strategy string) *Bot {
	return &Bot{
		addr:     addr,
		name:     name,
		strategy: strategy,
	}
}

func (b *Bot) Run() error {
	claudePath, err := exec.LookPath("claude")
	if err != nil {
		return fmt.Errorf("claude CLI is required. Install it first: %w", err)
	}

	selfPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve executable path: %w", err)
	}

	configFile, err := os.CreateTemp("", "yatz-mcp-*.json")
	if err != nil {
		return fmt.Errorf("create temp config: %w", err)
	}
	defer os.Remove(configFile.Name())

	if _, err := configFile.WriteString(BuildMCPConfig(selfPath)); err != nil {
		configFile.Close()
		return fmt.Errorf("write MCP config: %w", err)
	}
	configFile.Close()

	prompt := BuildPrompt(b.addr, b.name, b.strategy)
	systemPrompt := BuildSystemPrompt(b.strategy)

	cmd := exec.Command(claudePath, "-p",
		"--mcp-config", configFile.Name(),
		"--allowedTools", "mcp__yatzcli__*",
		"--system-prompt", systemPrompt,
		prompt,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("claude exited with error: %w", err)
	}
	return nil
}
