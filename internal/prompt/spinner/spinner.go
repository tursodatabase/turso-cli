package spinner

import (
	"fmt"
	"os"

	spn "github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

type Spinner struct {
	spinner   spn.Model
	prefix    string
	suffix    string
	quitting  bool
	cancelled bool
	done      chan bool
}

func newSpinner(prefix, suffix string) *Spinner {
	s := spn.New()
	s.Spinner = spn.Dot
	return &Spinner{spinner: s, prefix: prefix, suffix: suffix}
}

func (m *Spinner) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m *Spinner) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.quitting {
		return m, tea.Quit
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			m.quitting, m.cancelled = true, true
			return m, tea.Quit
		}
		return m, nil
	case error:
		return m, nil
	default:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
}

func (m *Spinner) View() string {
	if m.quitting {
		return ""
	}
	return fmt.Sprintf("%s%s %s", m.prefix, m.spinner.View(), m.suffix)
}

func (m *Spinner) Stop() {
	m.quitting = true
	if m.done != nil {
		<-m.done
	}
}

func (m *Spinner) Text(t string) {
	m.suffix = t
}

func (m *Spinner) Start() {
	ch := make(chan bool)
	m.done = ch
	m.quitting = false
	m.cancelled = false
	go func() {
		defer close(ch)
		tea.NewProgram(m).Run()
		if m.cancelled {
			os.Exit(130)
		}
	}()
}

func New(text string) *Spinner {
	spinner := newSpinner("", text)
	return spinner
}

func Start(text string) *Spinner {
	spinner := New(text)
	spinner.Start()
	return spinner
}
