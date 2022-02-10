package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/rancher/ecm-distro-tools/repository"
)

var (
	name    string
	version string
	gitSHA  string
)

const usage = `version: %s
Usage: %[2]s [-t token] [-r repo] [-b branches] [-i issue]
Options:
    -h                   help
    -v                   show version and exit
    -t                   github token (optional)
    -r repo              repository that should be used
    -i issue id          original issue id
    -c commit            commit id that is being backported
    -b branch(es)        branches issue is being backported to
Examples: 
	# generate release notes for RKE2 for milestone v1.21.5
    %[2]s -t <TOKEN> -r k3s -b "release-1.21,release-1.22" -i 1234 -c 1
`

const (
	issueTitle = "[%s] - %s"
	issueBody  = "Backport fix for %s\n\n* #%d"
)

var (
	vers     bool
	ghToken  string
	repo     string
	commitID string
	issueID  uint
	branches string
)

func main() {
	flag.Usage = func() {
		w := os.Stderr
		for _, arg := range os.Args {
			if arg == "-h" {
				w = os.Stdout
				break
			}
		}
		fmt.Fprintf(w, usage, version, name)
	}

	flag.BoolVar(&vers, "v", false, "")
	flag.StringVar(&ghToken, "t", "", "")
	flag.StringVar(&repo, "r", "", "")
	flag.StringVar(&commitID, "c", "", "")
	flag.UintVar(&issueID, "i", 0, "")
	flag.StringVar(&branches, "b", "", "")
	flag.Parse()

	if vers {
		fmt.Fprintf(os.Stdout, "version: %s - git sha: %s\n", version, gitSHA)
		return
	}

	if !repository.IsValidRepo(repo) {
		fmt.Println("error: please provide a valid repository")
		os.Exit(1)
	}

	if ghToken == "" {
		fmt.Println("error: please provide a token")
		os.Exit(1)
	}

	if commitID == "" {
		fmt.Println("error: please provide a commit id")
		os.Exit(1)
	}

	if issueID == 0 {
		fmt.Println("error: please provide a valid issue id")
		os.Exit(1)
	}

	backportBranches := strings.Split(branches, ",")
	if len(backportBranches) < 1 || backportBranches[0] == "" {
		fmt.Println("error: please provide at least one branch to perform the backport")
		os.Exit(1)
	}

	ctx := context.Background()

	client := repository.NewGithub(ctx, ghToken)

	origIssue, err := repository.RetrieveOriginalIssue(ctx, client, repo, issueID)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	issue := repository.Issue{
		Title: issueTitle,
		Body:  issueBody,
	}

	for _, branch := range backportBranches {
		ni, err := repository.CreateBackportIssues(ctx, client, origIssue, repo, branch, &issue)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		fmt.Println("Backport issue created: " + ni.GetHTMLURL())
	}

	os.Exit(0)
}
