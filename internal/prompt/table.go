package prompt

import (
	"os"

	tbl "github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var baseStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("240"))

type table struct {
	table    tbl.Model
	quitting bool
	choice   string
}

func (m *table) Init() tea.Cmd {
	return nil
}

func (m *table) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.quitting {
		return m, tea.Quit
	}

	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "enter":
			m.quitting = true
			m.choice = m.table.SelectedRow()[0]
			return m, tea.Quit
		default:
			m.table, cmd = m.table.Update(msg)
			return m, cmd
		}
	}
	return m, nil
}

func (m *table) View() string {
	if m.quitting {
		return ""
	}
	return baseStyle.Render(m.table.View()) + "\n"
}

func newTable(columns []tbl.Column, rows []tbl.Row, initPos int) *table {
	t := tbl.New(
		tbl.WithColumns(columns),
		tbl.WithRows(rows),
		tbl.WithFocused(true),
		tbl.WithHeight(len(rows)),
	)
	t.SetCursor(initPos)

	s := tbl.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(s)

	return &table{table: t}
}

func (m *table) Start() {
	m.quitting = false
	if _, err := tea.NewProgram(m).Run(); err != nil {
		os.Exit(130)
	}
}

func Table(columns []tbl.Column, rows []tbl.Row, initPos int) string {
	table := newTable(columns, rows, initPos)
	table.Start()
	return table.choice
}
