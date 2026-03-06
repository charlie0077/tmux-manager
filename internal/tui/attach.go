package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charlie0077/tmux-manager/internal/config"
)

func (m *Model) persistState() {
	config.SaveState(m.configPath, config.State{
		Server:  m.lastServer,
		Session: m.lastSession,
	})
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case sessionActionDoneMsg:
		// Route to the right view handler
		if m.view == viewSessionDetail {
			result, cmd := m.sessionDetailUpdate(msg)
			return result, cmd
		}
		// For server list, just refresh
		if msg.err != nil {
			m.errMsg = msg.err.Error()
		}
		m.servers[msg.serverIdx].loading = true
		return m, m.fetchSessions(msg.serverIdx)
	}

	switch m.view {
	case viewServerList:
		result, cmd := m.serverListUpdate(msg)
		return result, cmd
	case viewSessionDetail:
		result, cmd := m.sessionDetailUpdate(msg)
		return result, cmd
	}

	return m, nil
}

func (m Model) View() string {
	switch m.view {
	case viewSessionDetail:
		return m.sessionDetailView()
	default:
		return m.serverListView()
	}
}
