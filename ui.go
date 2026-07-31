package main

import (
	"errors"
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// errAborted reports that the user dismissed an interactive prompt.
var errAborted = errors.New("aborted")

type item struct {
	title string
	desc  string
}

func (i item) FilterValue() string { return i.title }
func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.desc }

type model struct {
	list     list.Model
	choice   string
	quitting bool
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetSize(msg.Width, msg.Height)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.quitting = true
			return m, tea.Quit

		case "enter":
			if i, ok := m.list.SelectedItem().(item); ok {
				m.choice = i.title
			}
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m model) View() string {
	if m.choice != "" {
		return ""
	}
	if m.quitting {
		return "Bye!\n"
	}
	return "\n" + m.list.View()
}

// selectVersion shows a picker for the locally downloaded versions. It
// returns errAborted if the user quits without choosing.
func selectVersion(versions []string) (string, error) {
	items := make([]list.Item, len(versions))
	for i, version := range versions {
		items[i] = item{title: version, desc: fmt.Sprintf("Paper %s", version)}
	}

	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Select Minecraft Version"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = lipgloss.NewStyle().MarginLeft(2)

	finalModel, err := tea.NewProgram(model{list: l}).Run()
	if err != nil {
		return "", err
	}

	m, ok := finalModel.(model)
	if !ok || m.quitting || m.choice == "" {
		return "", errAborted
	}
	return m.choice, nil
}
