package jira

import (
	"fmt"
	"log/slog"

	"github.com/andygrunwald/go-jira"
)

// container for client + user
type JiraData struct {
	Client jira.Client
	User   *jira.User
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

	return JiraData{Client: *jiraClient, User: jiraUser}
}

// list of all the projects
func (j JiraData) GetProjects() []jira.Project {
	projectEntries, _, err := j.Client.Project.GetList()
	if err != nil {
		panic(err)
	}

	projects := make([]jira.Project, 0)
	for _, projectEntry := range *projectEntries {
		proj, _, err := j.Client.Project.Get(projectEntry.ID)
		if err != nil {
			slog.Error("failed to fetch project", "project ID", projectEntry.ID, "error", err)
			continue
		}
		projects = append(projects, *proj)
	}

	return projects
}

// issues in a particular project
func (j JiraData) GetIssuesForProject(project jira.Project) []jira.Issue {
	jql := fmt.Sprintf("project = %s", project.Key)
	slog.Debug("fetching issues", "JQL", jql, "project", project)
	issues, _, err := j.Client.Issue.Search(jql, &jira.SearchOptions{})
	if err != nil {
		slog.Error("failed to get issues for project", "project", project, "error", err)
		panic(err)
	}

	return issues
}

func (j JiraData) GetIssue(issue jira.Issue) jira.Issue {
	i, _, err := j.Client.Issue.Get(issue.ID, &jira.GetQueryOptions{FieldsByKeys: true})
	if err != nil {
		slog.Error("failed to get issue", "issue", issue)
		panic(err)
	}
	return *i
}
