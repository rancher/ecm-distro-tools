package main

import "context"

type pullRequest struct {
	id      string
	buildNo int
	title   string
	desc    string
	org     string
	repo    string
}

func (i pullRequest) BuildNo() int {
	return i.buildNo
}
func (i pullRequest) Title() string {
	return i.title
}
func (i pullRequest) Description() string {
	return i.desc
}
func (i pullRequest) FilterValue() string {
	return i.title
}
func (i pullRequest) ID() string {
	return i.id
}
func (i pullRequest) Org() string {
	return i.org
}
func (i pullRequest) Repo() string {
	return i.repo
}
func (i pullRequest) Type() string {
	return "github_pull_request"
}

func (i *pullRequest) Refresh(ctx context.Context) error {
	return nil
}
