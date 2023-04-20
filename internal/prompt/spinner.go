package prompt

import (
	"fmt"

	spn "github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

type spinner struct {
	spinner spn.Model
	prefix  string
	suffix  string
	done    bool
}

func newSpinner(prefix, suffix string) *spinner {
	s := spn.New()
	s.Spinner = spn.Dot

	return &spinner{spinner: s, prefix: prefix, suffix: suffix}
}

func (m *spinner) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m *spinner) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.done {
		return m, tea.Quit
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		default:
			return m, nil
		}

	case error:
		return m, nil

	default:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
}

func (m *spinner) View() string {
	str := fmt.Sprintf("%s%s %s", m.prefix, m.spinner.View(), m.suffix)
	if m.done {
		str = ""
	}
	return str
}

func (m *spinner) Stop() {
	m.done = true
}

func (m *spinner) Start() {
	p := tea.NewProgram(m)
	go p.Run()
}

func StoppedSpinner(text string) *spinner {
	spinner := newSpinner("", text)
	return spinner
}

func Spinner(text string) *spinner {
	spinner := StoppedSpinner(text)
	spinner.Start()
	return spinner
}
