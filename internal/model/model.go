package model

import (
	"fmt"
	"log/slog"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/guppy0130/go-jira-tui/internal/jira"
	"github.com/guppy0130/go-jira-tui/internal/keymap"
	"github.com/mistakenelf/teacup/statusbar"
)

// board, issue, etc.
type ViewState string

const (
	ViewStateBoards      ViewState = "boards"
	ViewStateIssues      ViewState = "issues"
	ViewStateSingleIssue ViewState = "issue"
)

type Model struct {
	globalHeight int       // usable height
	globalWidth  int       // usable width
	viewState    ViewState // board, issue, sprint, etc.

	boardsView jira.BoardsView
	boardView  jira.BoardView

	// tables       map[ViewState]table.Model // some table that will get assigned
	// viewport  viewport.Model  // some issue content??

	statusBar statusbar.Model // statusbar
	JiraData  jira.JiraData   // jira data

	AccentColor lipgloss.Color
	// breadcrumbs  []breadcrumb    // supports going back with esc
}

func NewModel(jiraData jira.JiraData, accentColor lipgloss.Color) Model {
	m := Model{
		JiraData:    jiraData,
		AccentColor: accentColor,
		viewState:   ViewStateBoards,
	}
	sbAccent := statusbar.ColorConfig{
		Foreground: lipgloss.AdaptiveColor{Dark: "FG", Light: "BG"},
		Background: lipgloss.AdaptiveColor{Dark: string(m.AccentColor), Light: string(m.AccentColor)},
	}
	sbStandard := statusbar.ColorConfig{
		Foreground: lipgloss.AdaptiveColor{Dark: "FG", Light: "BG"},
		Background: lipgloss.AdaptiveColor{Dark: "BG", Light: "FG"},
	}
	m.statusBar = statusbar.New(sbAccent, sbStandard, sbAccent, sbAccent)
	return m
}

// func (m *Model) updateBoards() tea.Msg {
// 	m.boardsView = jira.NewBoardsView(m.JiraData)
// 	return m.boardView
// }

func (m Model) Init() tea.Cmd {
	return func() tea.Msg {
		m.boardsView = jira.NewBoardsView(m.JiraData, m.globalWidth)
		return m.boardView.Init
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := make([]tea.Cmd, 0)

	slog.Debug("update", "msg", msg)

	switch msg := msg.(type) {

	// handle resize
	case tea.WindowSizeMsg:
		m.globalHeight = msg.Height
		m.globalWidth = msg.Width
		m.statusBar.SetSize(msg.Width)

	// handle keystrokes
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keymap.DefaultKeyMap.Quit):
			return m, tea.Quit
		}

	default:
		slog.Warn("unknown update type", "type", msg)
	}

	// hand update message to child views in case they need it for something
	// model, newCmd := m.boardsView.Update(msg)
	// m.boardsView = model.(jira.BoardsView)
	// cmds = append(cmds, newCmd)

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	strings := make([]string, 0)
	// the header is what page we're on?
	strings = append(strings, string(m.viewState))

	// some body
	body := ""
	slog.Debug("rendering main app", "viewstate", m.viewState)
	switch m.viewState {
	case ViewStateBoards:
		body = m.boardsView.View()
	case ViewStateIssues:
		body = m.boardView.View()
	default:
		panic(fmt.Errorf("unable to handle viewState %+v", m.viewState))
	}
	strings = append(
		strings,
		lipgloss.
			NewStyle().
			Height(m.globalHeight-m.statusBar.Height-3).
			MaxHeight(m.globalHeight-m.statusBar.Height-3).Render(body),
	)

	// the statusbar
	strings = append(strings, m.statusBar.View())

	return lipgloss.JoinVertical(lipgloss.Top, strings...)
}
