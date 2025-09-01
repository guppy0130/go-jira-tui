package jira

import (
	"reflect"

	jira "github.com/andygrunwald/go-jira/v2/cloud"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type JiraIssueListTableImpl struct {
	tview.TableContentReadOnly

	issues  []jira.Issue
	columns []string
}

func NewJiraIssueListTableImpl(issues []jira.Issue) JiraIssueListTableImpl {
	return JiraIssueListTableImpl{
		issues:  issues,
		columns: []string{"Key", "Summary", "Status", "Assignee", "Created", "Updated"},
	}
}

func (j JiraIssueListTableImpl) GetCell(row, column int) *tview.TableCell {
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
		if column > 1 {
			cell.SetAlign(tview.AlignCenter)
		}
		return cell
	}

	// row 1 == issue 0 (offset due to headers)
	issue := j.issues[row-1]
	s := reflect.ValueOf(issue)
	v := s.FieldByName(j.columns[column])

	// if this isn't actually on the root object, try indexing into fields
	if !v.IsValid() {
		// s2 is a *IssueFields, so de-ref via Indirect
		s2 := reflect.ValueOf(issue.Fields)
		v = reflect.Indirect(s2).FieldByName(j.columns[column])
	}

	// v might be some type we know how to serialize into a human-friendly format
	// or it might have some field we want (`DisplayName`, `Name`, etc.), so let's
	// extract those as `value`
	var value string

	// if it's a pointer to something (e.g., `User`), de-ref *again*
	if v.Kind() == reflect.Pointer {
		v = reflect.Indirect(v)
	}

	// we *might* not have a valid value to work with; e.g., we had a pointer to
	// nil that we just deref'd above. guard against that. I hate golang.
	if v.IsValid() {
		switch v.Kind() {
		case reflect.Struct:

			// if we know the struct type we can serialize on our own
			switch v.Type() {
			case reflect.TypeOf(jira.Time{}):
				value = v.
					MethodByName("MarshalJSON").
					Call([]reflect.Value{})[0].
					Convert(reflect.TypeOf("")).
					String()
				// slice first and last chars (double quotes from MarshalJSON)
				value = value[1 : len(value)-1]
			case reflect.TypeOf(jira.User{}):
				value = v.FieldByName("DisplayName").String()
			case reflect.TypeOf(jira.Status{}):
				value = v.FieldByName("Name").String()
			}

			// otherwise, fall back to some of these names?
			for _, fieldName := range []string{"DisplayName", "Name"} {
				if _v := v.FieldByName(fieldName); _v.IsValid() && value != "" {
					value = _v.String()
					break
				}
			}
		}
	}

	// final serialization. writing out `<invalid Value>` for nil values is silly,
	// so just leave it as ``.
	if value == "" {
		if v.IsValid() {
			value = v.String()
		} else {
			value = ""
		}
	}

	cell := tview.NewTableCell(value)
	cell.SetReference(issue)
	if column > 1 {
		cell.SetExpansion(1)
		cell.SetAlign(tview.AlignCenter)
	}
	return cell
}

func (j JiraIssueListTableImpl) GetRowCount() int {
	return len(j.issues) + 1
}

func (j JiraIssueListTableImpl) GetColumnCount() int {
	return len(j.columns)
}
