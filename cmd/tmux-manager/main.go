package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charlie0077/tmux-manager/internal/config"
	"github.com/charlie0077/tmux-manager/internal/tui"
)

func main() {
	configPath := flag.String("config", config.DefaultConfigPath(), "path to config file")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "tmux-manager — TUI for remote tmux sessions\n\n")
		fmt.Fprintf(os.Stderr, "Usage: tmux-manager [flags]\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nKeybindings:\n")
		fmt.Fprintf(os.Stderr, "  Server List:\n")
		fmt.Fprintf(os.Stderr, "    enter   Expand server sessions\n")
		fmt.Fprintf(os.Stderr, "    n       New session on selected server\n")
		fmt.Fprintf(os.Stderr, "    r       Refresh all servers\n")
		fmt.Fprintf(os.Stderr, "    /       Filter servers by name\n")
		fmt.Fprintf(os.Stderr, "    ?       Toggle help overlay\n")
		fmt.Fprintf(os.Stderr, "    q       Quit\n\n")
		fmt.Fprintf(os.Stderr, "  Session Detail:\n")
		fmt.Fprintf(os.Stderr, "    enter   Attach to session\n")
		fmt.Fprintf(os.Stderr, "    n       New session\n")
		fmt.Fprintf(os.Stderr, "    k       Kill session (with confirm)\n")
		fmt.Fprintf(os.Stderr, "    d       Detach all clients\n")
		fmt.Fprintf(os.Stderr, "    esc     Back to server list\n\n")
		fmt.Fprintf(os.Stderr, "  While Attached:\n")
		fmt.Fprintf(os.Stderr, "    Ctrl-q  Detach and return to TUI\n")
	}
	flag.Parse()

	if !config.Exists(*configPath) {
		cfg := runOnboard(*configPath)
		if cfg == nil {
			return
		}
		runMain(cfg, *configPath)
		return
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	runMain(cfg, *configPath)
}

func runOnboard(configPath string) *config.Config {
	m := tui.NewOnboardModel(configPath)
	p := tea.NewProgram(m, tea.WithAltScreen())

	result, err := p.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if final, ok := result.(tui.OnboardResult); ok && final.Cfg != nil {
		return final.Cfg
	}
	return nil
}

func runMain(cfg *config.Config, configPath string) {
	m := tui.NewModel(cfg, configPath)
	p := tea.NewProgram(m, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
