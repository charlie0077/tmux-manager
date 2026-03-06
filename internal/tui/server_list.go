package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charlie0077/tmux-manager/internal/config"
	"github.com/charlie0077/tmux-manager/internal/ssh"
	"github.com/charlie0077/tmux-manager/internal/tmux"
)

func (m *Model) buildServerTable() table.Model {
	columns := []table.Column{
		{Title: "Server", Width: 20},
		{Title: "Host", Width: 30},
		{Title: "Sessions", Width: 10},
		{Title: "Status", Width: 10},
	}

	rows := m.serverRows()

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(min(len(rows)+1, 20)),
	)

	t.SetStyles(tableStyles(columnsWidth(columns)))

	return t
}

func (m *Model) serverRows() []table.Row {
	filter := ""
	if m.filtering {
		filter = strings.ToLower(m.filterInput.Value())
	}

	var rows []table.Row
	for _, s := range m.servers {
		if filter != "" && !strings.Contains(strings.ToLower(s.config.Name), filter) {
			continue
		}
		var status, sessionCount string
		switch {
		case s.loading:
			status = "..."
			sessionCount = "..."
		case s.err != nil:
			status = statusErr
			sessionCount = "-"
		case len(s.sessions) > 0:
			status = statusUp
			sessionCount = fmt.Sprintf("%d", len(s.sessions))
		default:
			status = statusNone
			sessionCount = "0"
		}
		rows = append(rows, table.Row{s.config.Name, s.config.Host, sessionCount, status})
	}
	return rows
}

func (m Model) fetchSessions(index int) tea.Cmd {
	srv := m.servers[index].config
	return func() tea.Msg {
		out, err := ssh.ListSessions(m.ctx, srv.Host, srv.KeyFile)
		if err != nil {
			return sessionsLoadedMsg{index: index, err: err}
		}
		sessions := tmux.ParseTmuxLS(out)
		return sessionsLoadedMsg{index: index, sessions: sessions}
	}
}

func (m *Model) serverListUpdate(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Modal input handlers first
		if m.filtering {
			return m.handleFilterKey(msg)
		}
		if m.naming {
			return m.handleNewSessionKey(msg)
		}
		if m.addingServer {
			return m.handleAddServerKey(msg)
		}
		if m.deletingServer {
			return m.handleDeleteServerKey(msg)
		}

		switch msg.String() {
		case "q", "ctrl+c":
			return *m, tea.Quit
		case "enter":
			return m.enterSessionDetail()
		case "a":
			m.addingServer = true
			m.addServerStep = addServerName
			m.addNameInput.Reset()
			m.addNameInput.Focus()
			return *m, textinput.Blink
		case "x":
			return m.startDeleteServer()
		case "r":
			return m.refreshAll()
		case "/":
			m.filtering = true
			m.filterInput.Reset()
			m.filterInput.Focus()
			return *m, textinput.Blink
		case "?":
			m.showHelp = !m.showHelp
			return *m, nil
		}
	case configSavedMsg:
		if msg.err != nil {
			m.errMsg = "Failed to save config: " + msg.err.Error()
		} else {
			m.errMsg = ""
		}
		return *m, nil
	case sessionsLoadedMsg:
		return m.handleSessionsLoaded(msg)
	case tickMsg:
		mm, cmd := m.refreshAll()
		return mm, tea.Batch(cmd, tickCmd())
	}

	var cmd tea.Cmd
	m.serverTable, cmd = m.serverTable.Update(msg)
	if selected := m.serverTable.SelectedRow(); selected != nil {
		m.lastServer = selected[0]
		m.persistState()
	}
	return *m, cmd
}

func (m *Model) handleFilterKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.filtering = false
		m.filterInput.Reset()
		m.serverTable.SetRows(m.serverRows())
		return *m, nil
	case "enter":
		// Keep filter active but stop editing
		m.filtering = false
		return *m, nil
	}
	var cmd tea.Cmd
	m.filterInput, cmd = m.filterInput.Update(msg)
	m.serverTable.SetRows(m.serverRows())
	return *m, cmd
}

func (m *Model) handleNewSessionKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.naming = false
		return *m, nil
	case "enter":
		name := strings.TrimSpace(m.nameInput.Value())
		if name == "" {
			return *m, nil
		}
		m.naming = false
		idx := m.selectedServerIndex()
		if idx < 0 {
			return *m, nil
		}
		srv := m.servers[idx].config
		return *m, func() tea.Msg {
			err := ssh.NewSession(m.ctx, srv.Host, srv.KeyFile, name)
			return sessionActionDoneMsg{serverIdx: idx, err: err}
		}
	}
	var cmd tea.Cmd
	m.nameInput, cmd = m.nameInput.Update(msg)
	return *m, cmd
}

func (m *Model) enterSessionDetail() (Model, tea.Cmd) {
	idx := m.selectedServerIndex()
	if idx < 0 {
		return *m, nil
	}
	s := m.servers[idx]
	if s.loading {
		return *m, nil
	}
	if s.err != nil {
		m.errMsg = fmt.Sprintf("%s: %v", s.config.Name, s.err)
		return *m, nil
	}
	m.selectedIdx = idx
	m.view = viewSessionDetail
	m.errMsg = ""
	m.lastServer = s.config.Name
	m.sessionTable = m.buildSessionTable()

	// Restore last session cursor
	if m.lastSession != "" {
		for i, sess := range s.sessions {
			if sess.Name == m.lastSession {
				m.sessionTable.SetCursor(i)
				break
			}
		}
	}

	return *m, nil
}

func (m *Model) handleSessionsLoaded(msg sessionsLoadedMsg) (Model, tea.Cmd) {
	if msg.index < len(m.servers) {
		m.servers[msg.index].loading = false
		m.servers[msg.index].sessions = msg.sessions
		m.servers[msg.index].err = msg.err
		m.serverTable.SetRows(m.serverRows())

	}
	return *m, nil
}

func (m *Model) refreshAll() (Model, tea.Cmd) {
	var cmds []tea.Cmd
	for i := range m.servers {
		if m.servers[i].loading {
			continue
		}
		m.servers[i].loading = true
		m.servers[i].err = nil
		m.servers[i].sessions = nil
		cmds = append(cmds, m.fetchSessions(i))
	}
	m.serverTable.SetRows(m.serverRows())
	return *m, tea.Batch(cmds...)
}

func (m *Model) selectedServerIndex() int {
	selected := m.serverTable.SelectedRow()
	if selected == nil {
		return -1
	}
	name := selected[0]
	for i, s := range m.servers {
		if s.config.Name == name {
			return i
		}
	}
	return -1
}

func (m *Model) handleAddServerKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	if msg.String() == "esc" {
		m.addingServer = false
		return *m, nil
	}

	switch m.addServerStep {
	case addServerName:
		if msg.String() == "enter" {
			if strings.TrimSpace(m.addNameInput.Value()) == "" {
				return *m, nil
			}
			m.addServerStep = addServerHost
			m.addHostInput.Reset()
			m.addHostInput.Focus()
			return *m, textinput.Blink
		}
		var cmd tea.Cmd
		m.addNameInput, cmd = m.addNameInput.Update(msg)
		return *m, cmd

	case addServerHost:
		if msg.String() == "enter" {
			if strings.TrimSpace(m.addHostInput.Value()) == "" {
				return *m, nil
			}
			m.addServerStep = addServerKey
			m.addKeyInput.Reset()
			m.addKeyInput.Focus()
			return *m, textinput.Blink
		}
		var cmd tea.Cmd
		m.addHostInput, cmd = m.addHostInput.Update(msg)
		return *m, cmd

	case addServerKey:
		if msg.String() == "enter" {
			name := strings.TrimSpace(m.addNameInput.Value())
			host := strings.TrimSpace(m.addHostInput.Value())
			keyFile := strings.TrimSpace(m.addKeyInput.Value())
			m.addingServer = false

			newServer := config.Server{Name: name, Host: host, KeyFile: keyFile}
			m.servers = append(m.servers, serverInfo{config: newServer, loading: true})
			m.serverTable = m.buildServerTable()

			idx := len(m.servers) - 1
			return *m, tea.Batch(m.saveConfig(), m.fetchSessions(idx))
		}
		var cmd tea.Cmd
		m.addKeyInput, cmd = m.addKeyInput.Update(msg)
		return *m, cmd
	}

	return *m, nil
}

func (m *Model) startDeleteServer() (Model, tea.Cmd) {
	idx := m.selectedServerIndex()
	if idx < 0 {
		return *m, nil
	}
	m.deletingServer = true
	m.deleteServerTarget = m.servers[idx].config.Name
	return *m, nil
}

func (m *Model) handleDeleteServerKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		m.deletingServer = false
		for i, s := range m.servers {
			if s.config.Name == m.deleteServerTarget {
				m.servers = append(m.servers[:i], m.servers[i+1:]...)
				break
			}
		}
		m.serverTable = m.buildServerTable()
		return *m, m.saveConfig()
	default:
		m.deletingServer = false
		return *m, nil
	}
}

func (m Model) serverListView() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("tmux-manager"))
	b.WriteString("\n\n")
	tableView := postProcessTable(m.serverTable.View(), m.serverTable.Cursor(), 2)
	b.WriteString(tableView)
	b.WriteString("\n")

	if m.filtering {
		b.WriteString("\n  / " + m.filterInput.View())
	}

	if m.addingServer {
		b.WriteString("\n" + confirmStyle.Render("Add server:") + "\n")
		switch m.addServerStep {
		case addServerName:
			b.WriteString("  Name: " + m.addNameInput.View())
		case addServerHost:
			b.WriteString(fmt.Sprintf("  Name: %s\n", m.addNameInput.Value()))
			b.WriteString("  Host: " + m.addHostInput.View())
		case addServerKey:
			b.WriteString(fmt.Sprintf("  Name: %s\n", m.addNameInput.Value()))
			b.WriteString(fmt.Sprintf("  Host: %s\n", m.addHostInput.Value()))
			b.WriteString("  Key:  " + m.addKeyInput.View())
		}
		b.WriteString("\n" + helpStyle.Render("enter=next  esc=cancel"))
	}

	if m.deletingServer {
		b.WriteString("\n" + confirmStyle.Render(
			fmt.Sprintf("Delete server %q from config? (y/n)", m.deleteServerTarget)))
	}

	if m.errMsg != "" {
		b.WriteString("\n" + errorStyle.Render(m.errMsg))
	}

	if m.showHelp {
		b.WriteString("\n")
		b.WriteString(m.helpOverlay())
	} else {
		b.WriteString(helpStyle.Render("enter=view  a=add  x=delete  r=refresh  /=filter  ?=help  q=quit"))
	}

	return b.String()
}

func (m Model) helpOverlay() string {
	help := `Keybindings

Server List:
  enter   View sessions
  a       Add server
  x       Delete server
  r       Refresh all servers
  /       Filter servers by name
  ?       Toggle this help
  q       Quit

Session Detail:
  enter   Attach to session
  n       New session
  x       Kill session (with confirm)
  ←/esc   Back to server list
  ?       Toggle this help

While Attached:
  Ctrl-q  Detach and return to TUI`

	return overlayStyle.Render(help)
}
