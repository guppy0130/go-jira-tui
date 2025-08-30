package jira

import (
	"fmt"

	"github.com/andygrunwald/go-jira"
)

// container for client + user
type JiraData struct {
	client jira.Client
	user   *jira.User
}

// get a client + user object
func NewJiraData(email string, token string, url string) JiraData {
	jiraAuthBasic := jira.BasicAuthTransport{
		Username: email,
		Password: token,
	}
	jiraClient, err := jira.NewClient(jiraAuthBasic.Client(), url)
	if err != nil {
		panic(err)
	}
	jiraUser, _, err := jiraClient.User.GetSelf()
	if err != nil {
		panic(err)
	}

	return JiraData{client: *jiraClient, user: jiraUser}
}

// list of all the boards
func (j JiraData) GetBoards() *jira.BoardsList {
	boards, _, err := j.client.Board.GetAllBoards(&jira.BoardListOptions{})
	if err != nil {
		panic(err)
	}
	return boards
}

// issues in a particular board
func (j JiraData) GetIssuesForBoard(board jira.Board) []jira.Issue {
	issues, _, err := j.client.Issue.Search(fmt.Sprintf("project = %s", board.Name), &jira.SearchOptions{})
	if err != nil {
		panic(err)
	}

	return issues
}
