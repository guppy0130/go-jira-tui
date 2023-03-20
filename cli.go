package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/andygrunwald/go-jira"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/evertras/bubble-table/table"
	"github.com/guppy0130/j2m"
	"github.com/knipferrc/teacup/statusbar"
	"github.com/spf13/viper"
)

const (
	columnKeyID        = "id"
	columnKeyName      = "name"
	columnKeyStartDate = "start_date" // sprint start date
	columnKeyEndDate   = "end_date"   // sprint end date
	columnKeyIssueKey  = "issue_key"  // issue key
	columnKeySummary   = "summary"    // issue summary
	columnKeyBack      = "back"       // some arbitrary int to go back to the higher level

	keywordBoards  = "Boards"
	keywordSprints = "Sprints"
	keywordIssues  = "Issues"

	default_view = keywordBoards
	accentColor  = lipgloss.Color("57")
	glamourTheme = "dark"
)

var (
	sbAccent = statusbar.ColorConfig{
		Foreground: lipgloss.AdaptiveColor{Dark: "FG", Light: "BG"},
		Background: lipgloss.AdaptiveColor{Dark: string(accentColor), Light: string(accentColor)},
	}
	sbStandard = statusbar.ColorConfig{
		Foreground: lipgloss.AdaptiveColor{Dark: "FG", Light: "BG"},
		Background: lipgloss.AdaptiveColor{Dark: "BG", Light: "FG"},
	}
	border      = lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder()).BorderForeground(accentColor)
	lightBorder = lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("250"))
)

type jiraData struct {
	client jira.Client
	user   jira.User
}

// the breadcrumb describes how to get back/what path we've taken
type breadcrumb struct {
	t     string // one of the keywords*
	value string // value for lookup (id or key)
}

type model struct {
	globalHeight int              // usable height
	globalWidth  int              // usable width
	viewState    string           // board, issue, sprint, etc.
	table        table.Model      // some table that will get assigned
	viewport     viewport.Model   // some issue content??
	statusBar    statusbar.Bubble // statusbar
	jiraData     jiraData         // jira data
	breadcrumbs  []breadcrumb     // supports going back with esc
}

type updateViewState string
type updateViewport string
type updateTable table.Model

func tableGenerator(columns []table.Column, rows []table.Row) updateTable {
	// the primary table
	t := table.New(columns).
		WithRows(rows).
		Filtered(true).
		Focused(true).
		BorderRounded().
		HighlightStyle(
			lipgloss.NewStyle().Background(accentColor),
		)

	return updateTable(t)
}

func updateStatusBar(state string) tea.Cmd {
	return func() tea.Msg {
		return updateViewState(state)
	}
}

func getBoards(jiraClient jira.Client) tea.Cmd {
	return tea.Batch(
		updateStatusBar(keywordBoards),
		func() tea.Msg {
			// get boards available to the user
			boards, _, err := jiraClient.Board.GetAllBoards(&jira.BoardListOptions{})
			if err != nil {
				panic(err)
			}
			// table the boards
			columns := []table.Column{
				table.NewColumn(columnKeyID, "ID", 4),
				table.NewFlexColumn(columnKeyName, "Name", 1).WithFiltered(true),
			}
			rows := []table.Row{}
			for _, board := range boards.Values {
				rows = append(rows, table.NewRow(
					table.RowData{
						columnKeyID:   board.ID,
						columnKeyName: board.Name,
					},
				))
			}
			return tableGenerator(columns, rows)
		},
	)
}

func getActiveSprintsInBoard(jiraClient jira.Client, boardId int) tea.Cmd {
	return tea.Batch(
		updateStatusBar(keywordSprints),
		func() tea.Msg {
			sprints, _, err := jiraClient.Board.GetAllSprintsWithOptions(boardId, &jira.GetAllSprintsOptions{State: "active"})
			if err != nil {
				panic(err)
			}
			columns := []table.Column{
				table.NewColumn(columnKeyID, "ID", 4),
				table.NewFlexColumn(columnKeyName, "Name", 1).WithFiltered(true),
				table.NewColumn(columnKeyStartDate, "Start Date", 30),
				table.NewColumn(columnKeyEndDate, "End Date", 30),
			}
			rows := []table.Row{}
			for _, sprint := range sprints.Values {
				rows = append(rows, table.NewRow(
					table.RowData{
						columnKeyID:        sprint.ID,
						columnKeyName:      sprint.Name,
						columnKeyStartDate: sprint.StartDate.Local().Format(time.RFC1123),
						columnKeyEndDate:   sprint.EndDate.Local().Format(time.RFC1123),
						columnKeyBack:      sprint.OriginBoardID,
					},
				))
			}

			return tableGenerator(columns, rows)
		},
	)
}

func getIssuesInSprint(jiraClient jira.Client, sprintId int) tea.Cmd {
	return tea.Batch(
		updateStatusBar(keywordIssues),
		func() tea.Msg {
			issues, _, err := jiraClient.Sprint.GetIssuesForSprint(sprintId)
			if err != nil {
				panic(err)
			}
			columns := []table.Column{
				// table.NewColumn(columnKeyID, "ID", 4),
				table.NewColumn(columnKeyIssueKey, "Key", 16).WithFiltered(true),
				table.NewFlexColumn(columnKeySummary, "Summary", 1).WithFiltered(true),
			}
			rows := []table.Row{}
			for _, issue := range issues {
				rows = append(rows, table.NewRow(
					table.RowData{
						columnKeyID:       issue.ID,
						columnKeyIssueKey: issue.Key,
						columnKeySummary:  issue.Fields.Summary,
						columnKeyBack:     issue.Fields.Sprint.ID,
					},
				))
			}
			return tableGenerator(columns, rows)
		},
	)
}

/*
 * summary
 * desc       | assignee
 * comments?  | reporter
 */

func renderGlamourJira(jiraContent string) string {
	content, err := glamour.Render(j2m.JiraToMD(jiraContent), glamourTheme)
	if err != nil {
		panic(err)
	}
	return content
}

func getIssue(jiraClient jira.Client, issueId string, issueKey string) tea.Cmd {
	// serialize the issue into glamour
	return tea.Batch(
		updateStatusBar(issueKey),
		func() tea.Msg {
			issue, _, err := jiraClient.Issue.Get(issueId, nil)
			if err != nil {
				panic(err)
			}
			// handle rendering the summary
			summary, err := glamour.Render(fmt.Sprintf("# %s", issue.Fields.Summary), glamourTheme)
			if err != nil {
				summary = issue.Fields.Summary
			}

			// left half is
			// description and comments
			description := renderGlamourJira(issue.Fields.Description)

			comments := strings.Builder{}
			for _, comment := range issue.Fields.Comments.Comments {
				s := strings.Builder{}
				// handle author rendering
				s.WriteString(fmt.Sprintf("%s, at %s", comment.Author.DisplayName, comment.Created))
				// and if there's an update, indicate changes
				if comment.Created != comment.Updated {
					s.WriteString(fmt.Sprintf("\n(Last updated by %s at %s)", comment.UpdateAuthor.DisplayName, comment.Updated))
				}
				// write the body of the comment
				s.WriteString("\n")
				s.WriteString(renderGlamourJira(comment.Body))
				comments.WriteString(lightBorder.Render(s.String()))
				comments.WriteString("\n")
			}
			comment_header, err := glamour.Render("## Comments", glamourTheme)
			if err != nil {
				comment_header = "Comments"
			}
			left_half := lipgloss.JoinVertical(
				lipgloss.Top,
				border.Render(description),
				border.Render(fmt.Sprintf("%s%s", comment_header, comments.String())),
			)

			// right half is
			// author, assignee
			rhs_content := strings.Builder{}
			details_header, err := glamour.Render("## Details", glamourTheme)
			if err != nil {
				details_header = "Details"
			}
			rhs_content.WriteString(details_header)
			rhs_content.WriteString(fmt.Sprintf("Assignee: %s", issue.Fields.Assignee.DisplayName))
			rhs_content.WriteString("\n")
			rhs_content.WriteString(fmt.Sprintf("Reporter: %s", issue.Fields.Reporter.DisplayName))
			right_half := lipgloss.JoinVertical(
				lipgloss.Top,
				border.Render(rhs_content.String()),
			)

			// render the left and right halves
			content := lipgloss.JoinHorizontal(lipgloss.Top, left_half, right_half)
			// add the summary to the top
			content = lipgloss.JoinVertical(lipgloss.Top, summary, content)
			return updateViewport(content)
		},
	)
}

func (m model) Init() tea.Cmd {
	return getBoards(m.jiraData.client)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds = []tea.Cmd{}
	)

	m.table, cmd = m.table.Update(msg)
	cmds = append(cmds, cmd)

	switch msg := msg.(type) {
	case updateTable:
		m.table = table.Model(msg)
		m.table = m.table.WithTargetWidth(m.statusBar.Width)

	case updateViewState:
		m.viewState = string(msg)
		serializedBreadcrumbs := []string{}
		for _, breadcrumb := range m.breadcrumbs {
			serializedBreadcrumbs = append(serializedBreadcrumbs, fmt.Sprintf("%s (%s)", breadcrumb.value, breadcrumb.t))
		}
		m.statusBar.SetContent(
			m.viewState,
			strings.Join(serializedBreadcrumbs, " > "),
			m.jiraData.user.DisplayName,
			m.jiraData.client.GetBaseURL().Host,
		)

	case updateViewport:
		m.viewport = viewport.New(m.globalWidth, m.globalHeight)
		content := string(msg)
		m.viewport.SetContent(content)

	case tea.WindowSizeMsg:
		m.globalHeight = msg.Height
		m.statusBar.SetSize(msg.Width)
		m.table = m.table.WithTargetWidth(msg.Width)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "enter":
			switch m.viewState {
			case keywordBoards: // going to sprints
				row := m.table.HighlightedRow()
				if row.Data != nil {
					m.breadcrumbs = append(m.breadcrumbs, breadcrumb{
						t:     keywordBoards,
						value: fmt.Sprint(row.Data[columnKeyID].(int)),
					})
					return m, getActiveSprintsInBoard(m.jiraData.client, row.Data[columnKeyID].(int))
				}
			case keywordSprints: // going to issues
				row := m.table.HighlightedRow()
				if row.Data != nil {
					m.breadcrumbs = append(m.breadcrumbs, breadcrumb{
						t:     keywordSprints,
						value: fmt.Sprint(row.Data[columnKeyID].(int)),
					})
					return m, getIssuesInSprint(m.jiraData.client, row.Data[columnKeyID].(int))
				}
			case keywordIssues: // reading a single issue
				row := m.table.HighlightedRow()
				if row.Data != nil {
					m.breadcrumbs = append(m.breadcrumbs, breadcrumb{
						t:     keywordIssues,
						value: fmt.Sprint(row.Data[columnKeyID]),
					})
					return m, getIssue(m.jiraData.client, row.Data[columnKeyID].(string), row.Data[columnKeyIssueKey].(string))
				}
			default:
				panic(m.viewState)
			}
		case "esc":
			// no breadcrumbs, nothing to esc here
			if len(m.breadcrumbs) == 0 {
				return m, nil
			}
			// going back. pop the last, because it's where we're at
			m.breadcrumbs = m.breadcrumbs[:len(m.breadcrumbs)-1]
			// if we have nothing left, we should render all boards
			if len(m.breadcrumbs) == 0 {
				return m, getBoards(m.jiraData.client)
			}
			// otherwise, we can go back some more. what's the new last value?
			last := m.breadcrumbs[len(m.breadcrumbs)-1]
			switch last.t {
			case keywordIssues:
				return m, getIssue(m.jiraData.client, last.value, last.value)
			case keywordSprints:
				i, err := strconv.Atoi(last.value)
				if err != nil {
					panic(err)
				}
				return m, getIssuesInSprint(m.jiraData.client, i)
			case keywordBoards:
				i, err := strconv.Atoi(last.value)
				if err != nil {
					panic(err)
				}
				return m, getActiveSprintsInBoard(m.jiraData.client, i)
			}
		}
	default:
		tea.Println(msg)
	}

	// likely to always have updates
	m.statusBar, cmd = m.statusBar.Update(msg)
	cmds = append(cmds, cmd)
	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	strings := []string{
		m.viewState,
	}

	// body is either a viewport of issue stuff or a table
	body := ""
	switch m.viewState {
	case keywordBoards, keywordSprints, keywordIssues:
		body = m.table.View()
	default:
		body = m.viewport.View()
	}
	// render the body
	strings = append(
		strings,
		lipgloss.NewStyle().
			Height(m.globalHeight-statusbar.Height).
			MaxHeight(m.globalHeight-m.statusBar.Height).
			Render(body),
	)
	// and then the status bar goes at the bottom
	strings = append(strings, m.statusBar.View())

	return lipgloss.JoinVertical(
		lipgloss.Top,
		strings...,
	)
}

func main() {
	// handle config
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("fatal error config file: %w", err))
	}

	// generate client
	jiraAuthBasic := jira.BasicAuthTransport{
		Username: viper.GetString("email"),
		Password: viper.GetString("token"),
	}
	jiraClient, err := jira.NewClient(jiraAuthBasic.Client(), viper.GetString("atlassian_root"))
	if err != nil {
		panic(err)
	}
	jiraUser, _, err := jiraClient.User.GetSelf()
	if err != nil {
		panic(err)
	}

	// create the bubble tea model
	m := model{
		jiraData: jiraData{
			client: *jiraClient,
			user:   *jiraUser,
		},
		viewState: default_view,
		statusBar: statusbar.New(sbAccent, sbStandard, sbAccent, sbAccent),
	}

	// run the UI
	if _, err := tea.NewProgram(m).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
