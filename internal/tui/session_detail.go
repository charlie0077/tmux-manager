package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charlie0077/tmux-manager/internal/ssh"
)

func (m *Model) buildSessionTable() table.Model {
	columns := []table.Column{
		{Title: "Session", Width: 30},
		{Title: "Windows", Width: 10},
		{Title: "Created", Width: 20},
		{Title: "State", Width: 12},
	}

	rows := m.sessionRows()

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(min(len(rows)+1, 20)),
	)

	t.SetStyles(tableStyles(columnsWidth(columns)))

	return t
}

func (m *Model) sessionRows() []table.Row {
	srv := m.servers[m.selectedIdx]
	var rows []table.Row
	for _, sess := range srv.sessions {
		state := detachedStyle
		if sess.Attached {
			state = attachedStyle
		}
		rows = append(rows, table.Row{
			sess.Name,
			fmt.Sprintf("%d", sess.Windows),
			sess.Created.Format("2006-01-02 15:04:05"),
			state,
		})
	}
	return rows
}

func (m *Model) sessionDetailUpdate(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.confirming {
			return m.handleConfirmKey(msg)
		}
		if m.naming {
			return m.handleSessionNewKey(msg)
		}

		switch msg.String() {
		case "q", "ctrl+c":
			return *m, tea.Quit
		case "esc", "left":
			m.view = viewServerList
			m.confirming = false
			m.errMsg = ""
			return *m, nil
		case "enter":
			return m.attachToSession()
		case "n":
			m.naming = true
			m.nameInput.Reset()
			m.nameInput.Focus()
			return *m, textinput.Blink
		case "x":
			return m.startKillConfirm()
		case "?":
			m.showHelp = !m.showHelp
			return *m, nil
		}
	case sessionsLoadedMsg:
		return m.handleSessionDetailLoaded(msg)
	case sessionActionDoneMsg:
		if msg.err != nil {
			m.errMsg = msg.err.Error()
		} else {
			m.errMsg = ""
		}
		// Refresh this server
		m.servers[msg.serverIdx].loading = true
		return *m, m.fetchSessions(msg.serverIdx)
	}

	var cmd tea.Cmd
	m.sessionTable, cmd = m.sessionTable.Update(msg)
	m.syncSessionSelection()
	m.persistState()
	return *m, cmd
}

func (m *Model) handleConfirmKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		m.confirming = false
		idx := m.selectedIdx
		srv := m.servers[idx].config
		target := m.confirmTarget
		return *m, func() tea.Msg {
			err := ssh.KillSession(m.ctx, srv.Host, srv.KeyFile, target)
			return sessionActionDoneMsg{serverIdx: idx, err: err}
		}
	default:
		m.confirming = false
		return *m, nil
	}
}

func (m *Model) handleSessionNewKey(msg tea.KeyMsg) (Model, tea.Cmd) {
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
		idx := m.selectedIdx
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

func (m *Model) syncSessionSelection() {
	if selected := m.sessionTable.SelectedRow(); selected != nil {
		m.lastSession = selected[0]
	}
}

func (m *Model) attachToSession() (Model, tea.Cmd) {
	selected := m.sessionTable.SelectedRow()
	if selected == nil {
		return *m, nil
	}
	sessionName := selected[0]
	srv := m.servers[m.selectedIdx].config
	cmd := ssh.AttachCommand(srv.Host, srv.KeyFile, sessionName)
	idx := m.selectedIdx

	return *m, tea.ExecProcess(cmd, func(err error) tea.Msg {
		return sessionActionDoneMsg{serverIdx: idx, err: err}
	})
}

func (m *Model) startKillConfirm() (Model, tea.Cmd) {
	selected := m.sessionTable.SelectedRow()
	if selected == nil {
		return *m, nil
	}
	m.confirming = true
	m.confirmTarget = selected[0]
	return *m, nil
}

func (m *Model) detachSession() (Model, tea.Cmd) {
	selected := m.sessionTable.SelectedRow()
	if selected == nil {
		return *m, nil
	}
	sessionName := selected[0]
	idx := m.selectedIdx
	srv := m.servers[idx].config
	return *m, func() tea.Msg {
		err := ssh.DetachSession(m.ctx, srv.Host, srv.KeyFile, sessionName)
		return sessionActionDoneMsg{serverIdx: idx, err: err}
	}
}

func (m *Model) handleSessionDetailLoaded(msg sessionsLoadedMsg) (Model, tea.Cmd) {
	if msg.index < len(m.servers) {
		m.servers[msg.index].loading = false
		m.servers[msg.index].sessions = msg.sessions
		m.servers[msg.index].err = msg.err
		m.serverTable.SetRows(m.serverRows())

		if msg.index == m.selectedIdx {
			if msg.err != nil || len(msg.sessions) == 0 {
				m.view = viewServerList
			} else {
				cursor := m.sessionTable.Cursor()
				m.sessionTable = m.buildSessionTable()
				if cursor >= len(msg.sessions) {
					cursor = len(msg.sessions) - 1
				}
				m.sessionTable.SetCursor(cursor)
			}
		}
	}
	return *m, nil
}

func (m Model) sessionDetailView() string {
	var b strings.Builder
	srv := m.servers[m.selectedIdx]

	b.WriteString(titleStyle.Render(fmt.Sprintf("tmux-manager > %s", srv.config.Name)))
	b.WriteString("\n\n")
	if len(srv.sessions) == 0 {
		b.WriteString("  No sessions. Press n to create one.\n")
	} else {
		tableView := postProcessTable(m.sessionTable.View(), m.sessionTable.Cursor(), 2)
		b.WriteString(tableView)
		b.WriteString("\n")
	}

	if m.confirming {
		b.WriteString("\n" + confirmStyle.Render(
			fmt.Sprintf("Kill session %q? (y/n)", m.confirmTarget)))
	}

	if m.naming {
		b.WriteString("\n" + confirmStyle.Render("New session name:") + "\n")
		b.WriteString("  " + m.nameInput.View())
	}

	if m.errMsg != "" {
		b.WriteString("\n" + errorStyle.Render(m.errMsg))
	}

	if m.showHelp {
		b.WriteString("\n")
		b.WriteString(m.helpOverlay())
	} else {
		b.WriteString(helpStyle.Render("enter=attach  n=new  x=kill  ←/esc=back  ?=help"))
	}

	return b.String()
}
