package github

import (
	"context"
	"errors"

	"github.com/google/go-github/v39/github"
	"github.com/rancher/ecm-distro-tools/repository"
)

func NewGithubClient(ctx context.Context, token string) (*github.Client, error) {
	if token == "" {
		return nil, errors.New("error: github token required")
	}

	return repository.NewGithub(ctx, token), nil
}
