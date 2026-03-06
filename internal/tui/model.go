package tui

import (
	"context"
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charlie0077/tmux-manager/internal/config"
	"github.com/charlie0077/tmux-manager/internal/tmux"
)

const autoRefreshInterval = 10 * time.Second

type tickMsg time.Time

type viewState int

const (
	viewServerList viewState = iota
	viewSessionDetail
)

type serverInfo struct {
	config   config.Server
	sessions []tmux.Session
	err      error
	loading  bool
}

type addServerStep int

const (
	addServerName addServerStep = iota
	addServerHost
	addServerKey
)

type Model struct {
	servers      []serverInfo
	view         viewState
	serverTable  table.Model
	sessionTable table.Model
	selectedIdx  int
	width        int
	height       int
	configPath   string

	// filter
	filtering   bool
	filterInput textinput.Model

	// new session
	naming    bool
	nameInput textinput.Model

	// add server
	addingServer   bool
	addServerStep  addServerStep
	addNameInput   textinput.Model
	addHostInput   textinput.Model
	addKeyInput    textinput.Model

	// delete server confirm
	deletingServer    bool
	deleteServerTarget string

	// kill confirm
	confirming    bool
	confirmTarget string

	// help overlay
	showHelp bool

	// error display
	errMsg string

	// persist last selection
	lastServer  string
	lastSession string

	ctx context.Context
}

// Messages
type sessionsLoadedMsg struct {
	index    int
	sessions []tmux.Session
	err      error
}

type sessionActionDoneMsg struct {
	serverIdx int
	err       error
}

type configSavedMsg struct {
	err error
}

func (m *Model) saveConfig() tea.Cmd {
	cfg := &config.Config{}
	for _, s := range m.servers {
		cfg.Servers = append(cfg.Servers, s.config)
	}
	path := m.configPath
	return func() tea.Msg {
		return configSavedMsg{err: config.Save(cfg, path)}
	}
}

func NewModel(cfg *config.Config, configPath string) Model {
	servers := make([]serverInfo, len(cfg.Servers))
	for i, s := range cfg.Servers {
		servers[i] = serverInfo{config: s, loading: true}
	}

	filterInput := textinput.New()
	filterInput.Placeholder = "filter servers..."
	filterInput.CharLimit = 64

	nameInput := textinput.New()
	nameInput.Placeholder = "session name"
	nameInput.CharLimit = 64

	addNameInput := textinput.New()
	addNameInput.Placeholder = "e.g. prod-api"
	addNameInput.CharLimit = 64
	addNameInput.Width = 40

	addHostInput := textinput.New()
	addHostInput.Placeholder = "e.g. user@hostname"
	addHostInput.CharLimit = 128
	addHostInput.Width = 40

	addKeyInput := textinput.New()
	addKeyInput.Placeholder = "e.g. ~/.ssh/id_rsa (leave empty to skip)"
	addKeyInput.CharLimit = 256
	addKeyInput.Width = 50

	m := Model{
		servers:      servers,
		view:         viewServerList,
		configPath:   configPath,
		filterInput:  filterInput,
		nameInput:    nameInput,
		addNameInput: addNameInput,
		addHostInput: addHostInput,
		addKeyInput:  addKeyInput,
		ctx:          context.Background(),
	}

	m.serverTable = m.buildServerTable()

	// Restore last selected server
	state := config.LoadState(configPath)
	if state.Server != "" {
		m.lastServer = state.Server
		m.lastSession = state.Session
		for i, s := range m.servers {
			if s.config.Name == state.Server {
				m.serverTable.SetCursor(i)
				break
			}
		}
	}

	return m
}

func tickCmd() tea.Cmd {
	return tea.Tick(autoRefreshInterval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{tickCmd()}
	for i := range m.servers {
		cmds = append(cmds, m.fetchSessions(i))
	}
	return tea.Batch(cmds...)
}
