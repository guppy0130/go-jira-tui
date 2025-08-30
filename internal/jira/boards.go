package jira

import (
	"log/slog"

	"github.com/andygrunwald/go-jira"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/evertras/bubble-table/table"
)

const (
	columnKeyID   = "id"
	columnKeyName = "name"
)

type BoardsView struct {
	jiraData JiraData
	boards   []jira.Board
	width    int
	table    table.Model
}

type updatedBoardsEvent int

func NewBoardsView(jiraData JiraData, width int) BoardsView {
	columns := make([]table.Column, 0)
	maxColumnKeyIDWidth := 4

	columns = append(columns, table.NewColumn(columnKeyID, "ID", maxColumnKeyIDWidth+1))
	columns = append(columns, table.NewFlexColumn(columnKeyName, "Name", 1))

	table := table.New(columns).WithTargetWidth(width)

	return BoardsView{
		jiraData: jiraData,
		boards:   make([]jira.Board, 0),
		width:    width,
		table:    table,
	}
}

func (b BoardsView) Init() tea.Cmd {
	return func() tea.Msg {
		b.boards = append(b.boards, b.jiraData.GetBoards().Values...)
		slog.Debug("retrieving boards", "boards", b.boards)
		return updatedBoardsEvent(len(b.boards))
	}
}

func (b BoardsView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	slog.Debug("update", "msg", msg)

	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		b.width = msg.Width

	case updatedBoardsEvent:
		slog.Debug("updated boards", "count", msg)

	default:
		slog.Warn("unknown update type", "msg", msg)
	}
	return b, nil
}

func (b BoardsView) View() string {

	// TODO: figure out if we want to keep this
	// if len(b.boards) > 0 {
	// 	// compute the board with the longest ID. this used to be hardcoded to 4,
	// 	// but if your company has >9999 boards maybe you should re-evaluate why...
	// 	boardWithLongestID := slices.MaxFunc(b.boards, func(a jira.Board, b jira.Board) int {
	// 		return cmp.Compare(len(strconv.Itoa(a.ID)), len(strconv.Itoa(a.ID)))
	// 	})
	// 	maxColumnKeyIDWidth = len(strconv.Itoa(boardWithLongestID.ID))
	// }

	rows := make([]table.Row, 0)
	for _, board := range b.boards {
		rows = append(rows, boardToTableRow(board))
	}
	return b.table.WithRows(rows).Filtered(true).Focused(true).WithTargetWidth(b.width).View()
	// return table.New(columns).WithRows(rows).Filtered(true).Focused(true).WithTargetWidth(b.width).View()
}

func boardToTableRow(board jira.Board) table.Row {
	return table.NewRow(table.RowData{
		columnKeyID:   board.ID,
		columnKeyName: board.Name,
	})
}
