package main

import (
	"cmp"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"runtime/debug"
	"slices"

	gojira "github.com/andygrunwald/go-jira/v2/cloud"
	"github.com/charmbracelet/glamour"
	"github.com/gdamore/tcell/v2"
	"github.com/guppy0130/go-jira-tui/internal/config"
	"github.com/guppy0130/go-jira-tui/internal/jira"
	"github.com/guppy0130/j2m"
	"github.com/rivo/tview"
)

const (
	PageProject     = "Projects"
	PageIssues      = "Issues"
	PageSingleIssue = "Single Issue"
)

func linkify(url, text string) string {
	return fmt.Sprintf("[::u:%s]%s[::-:-]", url, text)
}

// generate a TextView with given title + text. borders automatically enabled
// (to render title).
func newTextView(title string, text string) *tview.TextView {
	textView := tview.NewTextView()
	textView.SetText(text)
	textView.SetTitle(title)
	textView.SetBorder(true)
	return textView
}

// generate the issue flex.
func generateIssueFlex(issue gojira.Issue, jiraData jira.JiraData) *tview.Flex {
	rootFlex := tview.NewFlex()
	rootFlex.SetTitle(fmt.Sprintf("Issue %s", issue.Key))
	rootFlex.SetBorder(true)

	// issue description and comments
	descAndCommentsFlex := tview.NewFlex()
	descAndCommentsFlex.SetDirection(tview.FlexRow)

	if description := j2m.JiraToMD(issue.Fields.Description); len(description) > 0 {
		issueDescriptionText := tview.NewTextView()
		issueDescriptionText.SetTitle("Description")
		issueDescriptionText.SetBorder(true)

		// if we *can* augment it with glamour let's do so
		if renderedDesc, err := glamour.Render(description, "dark"); err == nil {
			description = tview.TranslateANSI(renderedDesc)
			issueDescriptionText.SetDynamicColors(true)
		} else {
			slog.Error("failed to glamour description", "issue", issue.Key, "description", description, "error", err)
		}
		issueDescriptionText.SetText(description)
		descAndCommentsFlex.AddItem(issueDescriptionText, 0, 1, true)
		// TODO: shell out to editor to edit this maybe?
	}

	// if there's comments to render, we'll go ahead and show them in a list, but
	// without shortcuts (no guarantee we have < len(runes) # of comments?)
	if c := issue.Fields.Comments; c != nil && len(c.Comments) > 0 {
		issueList := tview.NewList()
		// for some reason they don't support vim keybinds here?
		// TODO: handle `gg` to go to top (requires storing prev key(s))
		issueList.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			switch event.Rune() {
			case 'j':
				issueList.SetCurrentItem(issueList.GetCurrentItem() + 1)
				return nil
			case 'k':
				issueList.SetCurrentItem(issueList.GetCurrentItem() - 1)
				return nil
			case 'G':
				issueList.SetCurrentItem(-1)
				return nil
			}

			return event
		})

		issueList.SetTitle("Comments")
		issueList.SetBorder(true)

		for _, comment := range c.Comments {
			issueList.AddItem(
				comment.Body,
				fmt.Sprintf("%s @ %s", comment.Author.DisplayName, comment.Updated),
				0, // set to 0 for no binding
				nil,
			)
		}
		descAndCommentsFlex.AddItem(issueList, 0, 1, false)
	}

	rootFlex.AddItem(descAndCommentsFlex, 0, 2, true)

	// state/assignee/other details
	detailsFlex := tview.NewFlex()
	detailsFlex.SetTitle("Details")
	detailsFlex.SetBorder(true)
	detailsFlex.SetDirection(tview.FlexRow)

	// in progress, done, cancelled, etc.
	statusDropDown := tview.NewDropDown()
	statusDropDown.SetLabel("Status")
	// TODO: use RuneCountInString or GraphemeCountInString instead of `len`
	statusDropDown.SetLabelWidth(len(statusDropDown.GetLabel()) + 1)
	slices.SortFunc(issue.Transitions, func(a, b gojira.Transition) int {
		return cmp.Compare(a.ID, b.ID)
	})
	for idx, transition := range issue.Transitions {
		slog.Debug("adding issue transition", "transition", transition, "issue", issue)
		statusDropDown.AddOption(transition.To.Name, func() {
			// TODO: figure out if we should just send a message to the caller
			jiraData.TransitionIssue(issue, transition)
			// they might use some stupid names like `blue-gray` which we don't
			// understand, but `GetColor` will return the default color in that case
			// which should be ok?
			statusDropDown.SetFieldBackgroundColor(
				tcell.GetColor(transition.To.StatusCategory.ColorName),
			)
		})
		// we're in the transition that we're adding, so update the field
		if transition.To.ID == issue.Fields.Status.ID {
			statusDropDown.SetCurrentOption(idx)
			statusDropDown.SetFieldBackgroundColor(
				tcell.GetColor(transition.To.StatusCategory.ColorName),
			)
		}
	}

	// statusText := newTextView("Status", issue.Fields.Status.Name)
	detailsFlex.AddItem(statusDropDown, 3, 0, false)

	// reporter + assignee if assigned
	reporterAssigneeFlex := tview.NewFlex()
	// TODO: find a case where reporter is empty (then who filed the ticket?)
	reporterText := newTextView("Reporter", issue.Fields.Reporter.DisplayName)
	reporterAssigneeFlex.AddItem(reporterText, 0, 1, false)

	var assigneeText *tview.TextView
	if issue.Fields.Assignee != nil {
		assigneeText = newTextView("Assignee", issue.Fields.Assignee.DisplayName)
	} else {
		assigneeText = newTextView("Assignee", "unassigned")
	}
	reporterAssigneeFlex.AddItem(assigneeText, 0, 1, false)
	detailsFlex.AddItem(reporterAssigneeFlex, 3, 0, false)

	// created/updated
	createdUpdatedFlex := tview.NewFlex()
	createdUpdatedFlex.SetDirection(tview.FlexColumn)
	if createdBytes, err := issue.Fields.Created.MarshalJSON(); err == nil {
		createdUpdatedFlex.AddItem(
			newTextView("Created", string(createdBytes)[1:len(createdBytes)-1]), 0, 1, false,
		)
	}
	if updatedBytes, err := issue.Fields.Updated.MarshalJSON(); err == nil {
		createdUpdatedFlex.AddItem(
			newTextView("Updated", string(updatedBytes)[1:len(updatedBytes)-1]), 0, 1, false,
		)
	}
	detailsFlex.AddItem(createdUpdatedFlex, 3, 0, false)

	// issue url for browser
	if u, err := url.JoinPath(jiraData.URL, "/browse/"); err == nil {
		if u2, err := url.JoinPath(u, issue.Key); err == nil {
			urlTextView := newTextView("Issue URL", linkify(u2, u2))
			urlTextView.SetDynamicColors(true)
			detailsFlex.AddItem(urlTextView, 3, 0, false)
		}
		if issue.Fields.Parent != nil {
			if u2, err := url.JoinPath(u, issue.Fields.Parent.Key); err == nil {
				urlTextView := newTextView("Parent URL", linkify(u2, u2))
				urlTextView.SetDynamicColors(true)
				detailsFlex.AddItem(urlTextView, 3, 0, false)
			}
		}
	}

	rootFlex.AddItem(detailsFlex, 0, 1, false)

	return rootFlex
}

func setupApp(jiraData jira.JiraData) *tview.Application {
	app := tview.NewApplication()

	// there are three pages
	pages := tview.NewPages()

	// the project table has the list of projects
	projectTable := tview.NewTable()
	projectTable.SetBorder(true)
	projectTable.SetTitle("Projects")
	projectTable.SetSelectable(true, false)

	projectTableImpl := jira.NewJiraProjectListTableImpl(jiraData.GetProjects())
	projectTable.SetContent(projectTableImpl)
	pages.AddPage(PageProject, projectTable, true, true)

	// selecting a project will show you the issues in that project
	projectTable.SetSelectedFunc(func(row, column int) {
		// retrieve the stored project ref
		ref := projectTable.GetCell(row, column).GetReference()
		// ensure it _is_ a project
		if project, ok := ref.(gojira.Project); ok {
			// fetch issues and update table content
			slog.Debug("selected project", "project", project.Key)

			// the issues table has the list of issues in that project
			issueTable := tview.NewTable()
			issueTable.SetBorder(true)
			issueTable.SetTitle(fmt.Sprintf("Issues in %s", project.Name))
			issueTable.SetSelectable(true, false)

			issueTableImpl := jira.NewJiraIssueListTableImpl(
				jiraData.GetIssuesForProject(project),
			)
			issueTable.SetContent(issueTableImpl)
			// and then switch to it
			pages.AddAndSwitchToPage(PageIssues, issueTable, true)
			pages.SwitchToPage(PageIssues)
			app.SetFocus(issueTable)

			// hitting esc on the issues table should send you back to the projects
			issueTable.SetDoneFunc(func(key tcell.Key) {
				pages.RemovePage(PageIssues)
				app.SetFocus(projectTable)
			})

			// selecting an issue shows issue details
			issueTable.SetSelectedFunc(func(row, column int) {
				issueRef := issueTable.GetCell(row, column).GetReference()
				if issue, ok := issueRef.(gojira.Issue); ok {
					// fully hydrate the issue
					issue = jiraData.GetIssue(issue)
					// then render it
					issueFlex := generateIssueFlex(issue, jiraData)

					// if there's actually anything to update, add it to the page list + show
					pages.AddAndSwitchToPage(PageSingleIssue, issueFlex, true)
					app.SetFocus(issueFlex)

					issueFlex.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
						// handle esc to go back to issue list
						switch event.Key() {
						case tcell.KeyEsc, tcell.KeyCancel:
							pages.RemovePage(PageSingleIssue)
							app.SetFocus(issueTable)
							return nil
						}
						return event
					})
				}
			})

		} else {
			slog.Error(
				"Selection didn't contain a project?", "row", row, "column", column,
			)
		}
	})

	// the frame has a footer telling us who we are and what instance we're
	// operating on
	frame := tview.NewFrame(pages)

	// version
	var version string
	if bi, ok := debug.ReadBuildInfo(); ok {
		version = bi.Main.Version
	} else {
		version = "(devel)"
	}
	frame.AddText(
		fmt.Sprintf("go-jira-tui %s", version),
		false,
		tview.AlignLeft,
		tview.Styles.TertiaryTextColor,
	)

	// user @ jira instance
	frame.AddText(
		fmt.Sprintf("%s @ %s", jiraData.User.DisplayName, jiraData.URL),
		false,
		tview.AlignRight,
		tview.Styles.TertiaryTextColor,
	)

	app.SetRoot(frame, true)
	app.SetFocus(pages)

	return app
}

func main() {
	// handle config
	config := config.LoadViper()

	// generate client
	jiraData := jira.NewJiraData(config.Email, config.Token, config.Url)

	// setup logging
	f, err := os.OpenFile(
		"go-jira-tui.debug.log",
		os.O_WRONLY|os.O_TRUNC|os.O_CREATE|os.O_APPEND,
		0644,
	)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	slog.SetDefault(
		slog.New(
			slog.NewTextHandler(
				f,
				&slog.HandlerOptions{
					AddSource: true,
					Level:     slog.LevelDebug,
				},
			),
		),
	)

	// setup + run app
	app := setupApp(jiraData)
	app.EnableMouse(true)

	if err := app.Run(); err != nil {
		panic(err)
	}
}
