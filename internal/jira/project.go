package jira

import (
	"reflect"

	jira "github.com/andygrunwald/go-jira/v2/cloud"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type JiraProjectListTableImpl struct {
	tview.TableContentReadOnly

	// each project is a row in the table
	projects []jira.Project

	// each column in the table; is also a project property
	columns []string
}

func NewJiraProjectListTableImpl(projects []jira.Project) JiraProjectListTableImpl {
	return JiraProjectListTableImpl{
		projects: projects,
		columns:  []string{"Key", "Name", "Description", "URL"},
	}
}

func (j JiraProjectListTableImpl) GetCell(row, column int) *tview.TableCell {
	// if you click out of bounds on a table it'll give nonsense values, so guard
	// against that by returning `nil`
	if row < 0 || row > j.GetRowCount() || column < 0 || column > j.GetColumnCount() {
		return nil
	}

	// Header row
	if row == 0 {
		cell := tview.NewTableCell(j.columns[column])
		cell.SetAttributes(tcell.AttrBold)
		cell.SetTextColor(tview.Styles.InverseTextColor)
		cell.SetSelectable(false)
		cell.SetAlign(tview.AlignCenter)
		return cell
	}

	// row 1 == project 0 (offset due to headers)
	project := j.projects[row-1]
	s := reflect.ValueOf(project)
	v := s.FieldByName(j.columns[column])
	value := v.String()
	cell := tview.NewTableCell(value)
	cell.SetReference(project)
	if column > 0 {
		cell.SetExpansion(1)
		cell.SetAlign(tview.AlignCenter)
	}
	return cell
}

func (j JiraProjectListTableImpl) GetRowCount() int {
	// the +1 is the header row
	return len(j.projects) + 1
}

func (j JiraProjectListTableImpl) GetColumnCount() int {
	return len(j.columns)
}
