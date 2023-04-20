package prompt

import (
	"fmt"

	ti "github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type textinput struct {
	textInput ti.Model
	err       error
	done      bool
	prompt    string
}

func newTextinput(prompt, placeholder, value string) textinput {
	ti := ti.New()
	ti.SetValue(value)
	ti.Placeholder = placeholder
	ti.Focus()
	ti.CharLimit = 80
	ti.Width = 40

	return textinput{
		textInput: ti,
		err:       nil,
		prompt:    prompt,
	}
}

func (m textinput) Init() tea.Cmd {
	return ti.Blink
}

func (m textinput) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			m.err = fmt.Errorf("cancelled by user")
			fallthrough
		case tea.KeyEnter, tea.KeyEsc:
			m.done = true
			return m, tea.Quit
		}

	// We handle errors just like any other message
	case error:
		m.err = msg
		return m, nil
	}

	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m textinput) View() string {
	if m.done {
		return ""
	}

	return fmt.Sprintf(
		"%s\n\n%s\n\n%s\n",
		m.prompt,
		m.textInput.View(),
		"(press <enter> to submit)",
	)
}

func TextInput(prompt, placeholder, value string) (string, error) {
	p := tea.NewProgram(newTextinput(prompt, placeholder, value))
	m, err := p.Run()
	if err != nil {
		return "", err
	}

	model, ok := m.(textinput)
	if !ok {
		return "", err
	}

	return model.textInput.Value(), model.err
}
