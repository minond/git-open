package main

import (
	"bytes"
	"errors"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"

	gitlab "github.com/xanzy/go-gitlab"
)

func getBranch() (string, error) {
	log.Print("getting current working branch")

	out := &bytes.Buffer{}
	cmd := exec.Command("git", "branch")
	cmd.Dir = "/Users/marcosmindon/code/go/src/gitlab.mx.com/mx/atlas"
	cmd.Stdout = out
	if err := cmd.Run(); err != nil {
		return "", err
	}

	for _, line := range strings.Split(out.String(), "\n") {
		if strings.HasPrefix(line, "*") {
			return strings.TrimSpace(strings.TrimPrefix(line, "*")), nil
		}
	}

	return "", errors.New("unable to get current working branch")
}

func getProjectID(client *gitlab.Client) (string, error) {
	log.Println("getting gitlab project id")

	name, err := getProjectName()
	if err != nil {
		return "", err
	}

	opt := &gitlab.ListProjectsOptions{
		Search: stringptr(name),
	}

	projs, _, err := client.Projects.ListProjects(opt)
	if err != nil {
		return "", err
	} else if len(projs) == 0 {
		return "", errors.New("unable to find project")
	}

	return strconv.Itoa(projs[0].ID), nil
}

func getMRURL(client *gitlab.Client, projID, branch string) (string, error) {
	log.Printf("finding MRs for project id(%s) on branch %s", projID, branch)

	opt := &gitlab.ListProjectMergeRequestsOptions{
		SourceBranch: stringptr(branch),
		State:        stringptr("opened"),
		View:         stringptr("simple"),
	}

	mrs, _, err := client.MergeRequests.ListProjectMergeRequests(projID, opt)
	if err != nil {
		return "", err
	} else if len(mrs) == 0 {
		return "", errors.New("no matching MRs found")
	}

	return mrs[0].WebURL, nil
}

func getProjectName() (string, error) {
	var url string

	out := &bytes.Buffer{}
	cmd := exec.Command("git", "remote", "-v")
	cmd.Dir = "/Users/marcosmindon/code/go/src/gitlab.mx.com/mx/atlas"
	cmd.Stdout = out
	if err := cmd.Run(); err != nil {
		return "", err
	}

	for _, line := range strings.Split(out.String(), "\n") {
		if line == "" {
			continue
		}

		parts := strings.Fields(line)

		if len(parts) < 3 {
			continue
		} else if parts[2] == "(push)" {
			url = parts[1]
			break
		}
	}

	if url == "" {
		return "", errors.New("unable to find remote push url")
	}

	return parseRepoURLProjectName(url), nil
}

func parseRepoURLProjectName(rawurl string) string {
	hostAndProject := strings.SplitN(rawurl, ":", 2)
	orgAndName := strings.SplitN(hostAndProject[1], "/", 2)
	return strings.TrimSuffix(orgAndName[1], ".git")
}

func load(url string) error {
	log.Printf("loading %s", url)

	open := "open"
	if commandExists("xdg-open") {
		open = "xdg-open"
	}
	return exec.Command(open, url).Run()
}

func commandExists(name string) bool {
	cmd := exec.Command("command", "-v", name)
	if err := cmd.Run(); err != nil {
		return false
	}
	return true
}

func stringptr(str string) *string {
	return &str
}

func must(err error) {
	if err != nil {
		log.Fatalf("error: %v\n", err)
	}
}

func stringmust(str string, err error) string {
	must(err)
	return str
}

func main() {
	client := gitlab.NewClient(nil, os.Getenv("GITLAB_API_KEY"))
	if os.Getenv("GITLAB_HOST") != "" {
		client.SetBaseURL(os.Getenv("GITLAB_HOST"))
	}

	branch := stringmust(getBranch())
	projID := stringmust(getProjectID(client))
	mrURL := stringmust(getMRURL(client, projID, branch))
	must(load(mrURL))
}
