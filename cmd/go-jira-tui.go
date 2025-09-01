package main

import (
	"fmt"
	"log/slog"
	"os"

	"codeberg.org/tslocum/cview"
	gojira "github.com/andygrunwald/go-jira"
	"github.com/gdamore/tcell/v2"
	"github.com/guppy0130/go-jira-tui/internal/config"
	"github.com/guppy0130/go-jira-tui/internal/jira"
	"github.com/guppy0130/j2m"
)

// generate the issue flex. return the number of items in the flex.
func generateIssueFlex(issue gojira.Issue) (*cview.Flex, int) {
	itemCounter := 0
	// issue fields/text here vertically stacked I guess
	issueFlex := cview.NewFlex()
	issueFlex.SetDirection(cview.FlexRow)

	// TODO: maybe glow to render here?
	if description := j2m.JiraToMD(issue.Fields.Description); len(description) > 0 {
		issueDescriptionText := cview.NewTextView()
		issueDescriptionText.SetTitle("Description")
		issueDescriptionText.SetBorder(true)
		issueDescriptionText.SetText(description)
		// default focus to the desc if it exists
		issueFlex.AddItem(issueDescriptionText, 0, 1, true)
		itemCounter += 1
	}

	if c := issue.Fields.Comments; c != nil && len(c.Comments) > 0 {
		issueComments := cview.NewFlex()
		issueComments.SetTitle("Comments")
		issueComments.SetBorder(true)
		issueComments.SetDirection(cview.FlexRow)

		slog.Debug("fetched comments", "comments", c)
		// TODO: maybe this is a list of all the comments instead, or not individual
		// text boxes
		for _, comment := range c.Comments {
			commentBox := cview.NewTextView()
			commentBox.SetTitle(fmt.Sprintf("%s @ %s", comment.Author.DisplayName, comment.Updated))
			commentBox.SetBorder(true)
			commentBox.SetText(comment.Body)
			issueComments.AddItem(commentBox, 0, 1, false)
		}
		issueFlex.AddItem(issueComments, 0, 1, false)
		itemCounter += 1
	}

	issueFlex.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCancel:
			// this is stupid. how do you pass this back to the app?
			return nil
		}
		// probably hand it to the issueFlex for handling?
		return event
	})

	return issueFlex, itemCounter
}

func setupApp(jiraData jira.JiraData) *cview.Application {
	app := cview.NewApplication()
	flex := cview.NewFlex()

	projectList := cview.NewList()
	projectList.SetBorder(true)
	projectList.SetTitle("Projects")
	projectList.SetIndicators(">>", "", "", "")

	for _, project := range jiraData.GetProjects() {
		li := cview.NewListItem(project.Name)
		li.SetReference(project)
		projectList.AddItem(li)
	}

	issueTable := cview.NewTable()
	issueTable.SetBorder(true)
	issueTable.SetTitle("Issues")
	issueTable.SetSelectable(true, false)

	projectList.SetSelectedFunc(func(i int, li *cview.ListItem) {
		project := li.GetReference()
		slog.Debug("selected project", "project", project)

		if project, ok := project.(gojira.Project); ok {
			issueTable.Clear()
			// update issues table
			for idx, issue := range jiraData.GetIssuesForProject(project) {
				tcKey := cview.NewTableCell(issue.Key)
				tcKey.SetReference(issue)
				tcSummary := cview.NewTableCell(issue.Fields.Summary)

				// TODO: maybe not hard code these positions?
				issueTable.SetCell(idx, 0, tcKey)
				issueTable.SetCell(idx, 1, tcSummary)
			}
			// since you've selected a project, shift focus to the issues
			app.SetFocus(issueTable)
		} else {
			slog.Warn("selected project list item didn't have a ref to a project?")
		}
	})

	// a dummy value; we need to track the last issueFlex we created because we
	// need to remove it when we re-draw the issue
	lastIssueFlex := cview.NewFlex()
	issueTable.SetSelectedFunc(func(row, column int) {
		// fetch the ref
		issue := issueTable.GetCell(row, 0).GetReference()

		// type checker could be a little more sane here
		if issue, ok := issue.(gojira.Issue); ok {
			// resolve issue to more data, then generate issueFlex from more data
			issueFlex, issueFlexItemCounter := generateIssueFlex(jiraData.GetIssue(issue))

			// remove the last issueFlex if it's present? this won't explode if it's
			// already removed, so this should be fine to do
			flex.RemoveItem(lastIssueFlex)

			// if there's actually issue content to display, display it
			if issueFlexItemCounter > 0 {
				flex.AddItem(issueFlex, 0, 2, false)

				// since you've selected an issue, move focus
				app.SetFocus(issueFlex)
				// update lastIssueFlex to hold a ref; we'll need to clear it next time we
				// re-draw, because we don't have access to the flex's items.
				lastIssueFlex = issueFlex
			}
		} else {
			slog.Warn("selected cell didn't have a ref to an issue?")
			return
		}
	})

	flex.AddItem(projectList, 0, 1, false)
	flex.AddItem(issueTable, 0, 1, false)

	// the frame has a footer telling us who we are and what instance we're
	// operating on
	frame := cview.NewFrame(flex)
	frame.AddText(
		fmt.Sprintf("%s @ %s", jiraData.User.DisplayName, jiraData.Client.GetBaseURL().Host),
		false,
		cview.AlignRight,
		cview.Styles.ContrastSecondaryTextColor,
	)

	app.SetRoot(frame, true)
	app.SetFocus(projectList)

	return app
}

func main() {
	// handle config
	config := config.LoadViper()

	// generate client
	jiraData := jira.NewJiraData(config.Email, config.Token, config.Url)

	// setup logging
	f, err := os.OpenFile("go-jira-tui.debug.log", os.O_WRONLY|os.O_TRUNC|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	slog.SetDefault(slog.New(slog.NewTextHandler(f, &slog.HandlerOptions{AddSource: true, Level: slog.LevelDebug})))

	// setup + run app
	app := setupApp(jiraData)
	if err := app.Run(); err != nil {
		panic(err)
	}
}
