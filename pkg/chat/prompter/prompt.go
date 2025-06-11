package prompter

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

func saveHistory(history []string) error {
	dir, err := os.UserConfigDir()
	if err != nil {
		return err
	}

	historyFile := filepath.Join(dir, "nanobot/prompter_history.json")
	if err := os.MkdirAll(filepath.Dir(historyFile), 0o700); err != nil {
		return err
	}

	data, err := json.Marshal(history)
	if err != nil {
		return err
	}

	return os.WriteFile(historyFile, data, 0600)
}

func readHistory() (result []string) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return nil
	}

	historyFile := filepath.Join(dir, "nanobot/prompter_history.json")
	data, err := os.ReadFile(historyFile)
	if err != nil {
		return nil
	}

	_ = json.Unmarshal(data, &result)
	return
}

func ReadInput() (string, error) {
	p := tea.NewProgram(initialModel())
	m, err := p.Run()
	if err != nil {
		return "", err
	}

	ret := m.(model)
	if ret.err != nil {
		return "", ret.err
	}

	value := ret.textInput.Value()
	if ret.multiline {
		value = ret.textArea.Value()
	}

	ret.history[len(ret.history)-1] = value
	_ = saveHistory(ret.history)
	return value, nil
}

type (
	errMsg error
)

type model struct {
	multiline    bool
	textInput    textinput.Model
	textArea     textarea.Model
	historyIndex int
	history      []string
	err          error
}

func initialModel() model {
	ti := textinput.New()
	ti.Placeholder = "Enter your prompt here. (ctrl+e to switch to multiline)"
	ti.Focus()

	ta := textarea.New()
	ta.Placeholder = "Enter your prompt here. (ctrl+enter to submit)"

	return model{
		textInput: ti,
		textArea:  ta,
		history:   append(readHistory(), ""),
		err:       nil,
	}
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m *model) upHistory() {
	newIndex := m.historyIndex + 1
	if newIndex >= len(m.history) {
		return
	}

	if m.historyIndex == 0 {
		m.history[len(m.history)-1] = m.textInput.Value()
	}

	m.textInput.SetValue(m.history[len(m.history)-1-newIndex])
	m.historyIndex = newIndex
}

func (m *model) downHistory() {
	newIndex := m.historyIndex - 1
	if newIndex < 0 {
		return
	}

	m.textInput.SetValue(m.history[len(m.history)-1-newIndex])
	m.historyIndex = newIndex
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmds []tea.Cmd
		cmd  tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.textInput.Width = msg.Width
		m.textArea.SetWidth(msg.Width)
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyUp:
			if !m.multiline {
				m.upHistory()
			}
		case tea.KeyDown:
			if !m.multiline {
				m.downHistory()
			}
		case tea.KeyEnter:
			if msg.Alt == m.multiline {
				cmds = append(cmds, tea.Quit)
			}
		case tea.KeyCtrlS, tea.KeyCtrlJ:
			if m.multiline {
				cmds = append(cmds, tea.Quit)
			}
		case tea.KeyCtrlE:
			m.textInput.Blur()
			m.textArea.SetValue(m.textInput.Value())
			m.multiline = true
			cmds = append(cmds, m.textArea.Focus())
		case tea.KeyCtrlC, tea.KeyCtrlD:
			m.err = io.EOF
			cmds = append(cmds, tea.Quit)
		}
	// We handle errors just like any other message
	case errMsg:
		m.err = msg
		cmds = append(cmds, tea.Quit)
	}

	m.textInput, cmd = m.textInput.Update(msg)
	cmds = append(cmds, cmd)
	m.textArea, cmd = m.textArea.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	if m.multiline {
		return "\n" + m.textArea.View()
	}
	return "\n" + m.textInput.View()
}
