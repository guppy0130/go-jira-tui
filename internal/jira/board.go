package jira

import (
	"cmp"
	"slices"

	"github.com/andygrunwald/go-jira"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/evertras/bubble-table/table"
)

type BoardView struct {
	jiraData JiraData
	board    jira.Board
	issues   []jira.Issue
	width    int
}

func (b BoardView) Init() tea.Cmd {
	return tea.Batch(
		func() tea.Msg {
			b.issues = b.jiraData.GetIssuesForBoard(b.board)
			return nil
		},
	)
}

func (b BoardView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		b.width = msg.Width
	}
	return b, nil
}

func (b BoardView) View() string {
	columns := make([]table.Column, 0)

	issueWithLongestID := slices.MaxFunc(b.issues, func(a jira.Issue, b jira.Issue) int {
		return cmp.Compare(len(a.ID), len(b.ID))
	})
	columns = append(columns, table.NewColumn(columnKeyID, "ID", len(issueWithLongestID.ID)))
	columns = append(columns, table.NewFlexColumn(columnKeyName, "Name", 1))

	rows := make([]table.Row, 0)
	for _, issue := range b.issues {
		rows = append(rows, IssueToTableRow(issue))
	}

	return table.New(columns).WithRows(rows).Filtered(true).Focused(true).WithTargetWidth(b.width).View()
}

func IssueToTableRow(issue jira.Issue) table.Row {
	return table.NewRow(table.RowData{
		columnKeyID:   issue.ID,
		columnKeyName: issue.Key,
	})
}
