package prompt

import (
	"fmt"

	ta "github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
)

type textarea struct {
	textarea ta.Model
	err      error
	prompt   string
	done     bool
}

func newTextArea(prompt, placeholder, value string) textarea {
	ti := ta.New()
	ti.SetWidth(80)
	ti.Placeholder = placeholder
	ti.Focus()

	return textarea{
		textarea: ti,
		err:      nil,
		prompt:   prompt,
	}
}

func (m textarea) Init() tea.Cmd {
	return ta.Blink
}

func (m textarea) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEsc:
			if !m.textarea.Focused() {
				m.done = true
				return m, tea.Quit
			}
			m.textarea.Blur()

		case tea.KeyCtrlC:
			m.done = true
			m.err = fmt.Errorf("cancelled by user")
			return m, tea.Quit

		default:
			if !m.textarea.Focused() {
				cmd = m.textarea.Focus()
				cmds = append(cmds, cmd)
			}
		}

	case error:
		m.err = msg
		return m, nil
	}

	m.textarea, cmd = m.textarea.Update(msg)
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

func (m textarea) View() string {
	if m.done {
		return ""
	}

	return fmt.Sprintf(
		"%s\n\n%s\n\n%s\n\n",
		m.prompt,
		m.textarea.View(),
		"(press <esc> twice to submit)",
	)
}

func TextArea(prompt, placeholder, value string) (string, error) {
	p := tea.NewProgram(newTextArea(prompt, placeholder, value))
	m, err := p.Run()
	if err != nil {
		return "", err
	}

	model, ok := m.(textarea)
	if !ok {
		return "", err
	}

	return model.textarea.Value(), model.err
}
