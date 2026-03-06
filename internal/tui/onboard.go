package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charlie0077/tmux-manager/internal/config"
)

type onboardStep int

const (
	stepWelcome onboardStep = iota
	stepName
	stepHost
	stepMore
)

type OnboardModel struct {
	step       onboardStep
	nameInput  textinput.Model
	hostInput  textinput.Model
	servers    []config.Server
	configPath string
	err        error
	width      int
	height     int
}

type onboardDoneMsg struct {
	cfg *config.Config
}

// OnboardResult is returned from Run() so main can extract the config.
type OnboardResult struct {
	Cfg *config.Config
}

func (m OnboardResult) Init() tea.Cmd                           { return nil }
func (m OnboardResult) Update(tea.Msg) (tea.Model, tea.Cmd)     { return m, tea.Quit }
func (m OnboardResult) View() string                            { return "" }

var (
	onboardTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("170")).
			MarginBottom(1)

	onboardSubtle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	onboardAccent = lipgloss.NewStyle().
			Foreground(lipgloss.Color("42"))

	onboardBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("57")).
			Padding(1, 2).
			MarginTop(1).
			MarginBottom(1)

	serverEntry = lipgloss.NewStyle().
			Foreground(lipgloss.Color("229"))
)

func NewOnboardModel(configPath string) OnboardModel {
	nameInput := textinput.New()
	nameInput.Placeholder = "e.g. prod-api"
	nameInput.CharLimit = 64
	nameInput.Width = 40

	hostInput := textinput.New()
	hostInput.Placeholder = "e.g. user@hostname"
	hostInput.CharLimit = 128
	hostInput.Width = 40

	return OnboardModel{
		step:       stepWelcome,
		nameInput:  nameInput,
		hostInput:  hostInput,
		configPath: configPath,
	}
}

func (m OnboardModel) Init() tea.Cmd {
	return nil
}

func (m OnboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case onboardDoneMsg:
		if msg.cfg != nil {
			return OnboardResult{Cfg: msg.cfg}, tea.Quit
		}
		m.err = fmt.Errorf("failed to save config")
		return m, nil
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	}

	switch m.step {
	case stepWelcome:
		return m.updateWelcome(msg)
	case stepName:
		return m.updateName(msg)
	case stepHost:
		return m.updateHost(msg)
	case stepMore:
		return m.updateMore(msg)
	}

	return m, nil
}

func (m OnboardModel) updateWelcome(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.String() {
		case "enter":
			m.step = stepName
			m.nameInput.Reset()
			m.nameInput.Focus()
			return m, textinput.Blink
		case "q":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m OnboardModel) updateName(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.String() {
		case "enter":
			name := strings.TrimSpace(m.nameInput.Value())
			if name == "" {
				return m, nil
			}
			m.step = stepHost
			m.hostInput.Reset()
			m.hostInput.Focus()
			return m, textinput.Blink
		case "esc":
			if len(m.servers) > 0 {
				m.step = stepMore
			} else {
				m.step = stepWelcome
			}
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.nameInput, cmd = m.nameInput.Update(msg)
	return m, cmd
}

func (m OnboardModel) updateHost(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.String() {
		case "enter":
			host := strings.TrimSpace(m.hostInput.Value())
			if host == "" {
				return m, nil
			}
			name := strings.TrimSpace(m.nameInput.Value())
			m.servers = append(m.servers, config.Server{Name: name, Host: host})
			m.step = stepMore
			return m, nil
		case "esc":
			m.step = stepName
			m.nameInput.Focus()
			return m, textinput.Blink
		}
	}

	var cmd tea.Cmd
	m.hostInput, cmd = m.hostInput.Update(msg)
	return m, cmd
}

func (m OnboardModel) updateMore(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.String() {
		case "a":
			m.step = stepName
			m.nameInput.Reset()
			m.nameInput.Focus()
			return m, textinput.Blink
		case "d":
			return m, m.saveAndLaunch()
		case "q":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m OnboardModel) saveAndLaunch() tea.Cmd {
	return func() tea.Msg {
		cfg := &config.Config{Servers: m.servers}
		if err := config.Save(cfg, m.configPath); err != nil {
			return onboardDoneMsg{cfg: nil}
		}
		return onboardDoneMsg{cfg: cfg}
	}
}

func (m OnboardModel) View() string {
	var b strings.Builder

	b.WriteString(onboardTitle.Render("tmux-manager setup"))
	b.WriteString("\n")

	switch m.step {
	case stepWelcome:
		b.WriteString(m.welcomeView())
	case stepName:
		b.WriteString(m.nameView())
	case stepHost:
		b.WriteString(m.hostView())
	case stepMore:
		b.WriteString(m.moreView())
	}

	return b.String()
}

func (m OnboardModel) welcomeView() string {
	var b strings.Builder
	b.WriteString("No config file found.\n")
	b.WriteString(onboardSubtle.Render(fmt.Sprintf("  %s\n\n", m.configPath)))
	b.WriteString("Let's add your remote servers.\n\n")
	b.WriteString(onboardAccent.Render("  enter") + "  Start setup\n")
	b.WriteString(onboardSubtle.Render("  q      Quit"))
	return b.String()
}

func (m OnboardModel) nameView() string {
	var b strings.Builder
	b.WriteString(m.serverListPreview())
	b.WriteString(fmt.Sprintf("Server %d — Name:\n\n", len(m.servers)+1))
	b.WriteString("  " + m.nameInput.View() + "\n\n")
	b.WriteString(onboardSubtle.Render("  enter=next  esc=back"))
	return b.String()
}

func (m OnboardModel) hostView() string {
	var b strings.Builder
	name := strings.TrimSpace(m.nameInput.Value())
	b.WriteString(m.serverListPreview())
	b.WriteString(fmt.Sprintf("Server %d — Host for %q:\n\n", len(m.servers)+1, name))
	b.WriteString("  " + m.hostInput.View() + "\n\n")
	b.WriteString(onboardSubtle.Render("  Uses ~/.ssh/config for auth, keys, ProxyJump\n"))
	b.WriteString(onboardSubtle.Render("  enter=add  esc=back"))
	return b.String()
}

func (m OnboardModel) moreView() string {
	var b strings.Builder
	b.WriteString(m.serverListPreview())
	b.WriteString(fmt.Sprintf("%s added!\n\n", onboardAccent.Render(fmt.Sprintf("%d server(s)", len(m.servers)))))
	b.WriteString(onboardAccent.Render("  a") + "  Add another server\n")
	b.WriteString(onboardAccent.Render("  d") + "  Done — save & launch\n")
	b.WriteString(onboardSubtle.Render("  q  Quit without saving"))
	return b.String()
}

func (m OnboardModel) serverListPreview() string {
	if len(m.servers) == 0 {
		return ""
	}
	var b strings.Builder
	for _, s := range m.servers {
		b.WriteString(serverEntry.Render(fmt.Sprintf("  + %s (%s)\n", s.Name, s.Host)))
	}
	b.WriteString("\n")
	return onboardBox.Render(strings.TrimRight(b.String(), "\n")) + "\n\n"
}
