package github

import (
	"context"

	"github.com/google/go-github/v39/github"
	"golang.org/x/oauth2"
)

func newClient(ctx context.Context, token string) *github.Client {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)

	return github.NewClient(tc)
}
