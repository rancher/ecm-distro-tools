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
Usage: %[2]s [-r repo] [-b branches] [-i issue]

Env Variables:
    GITHUB_TOKEN         user token for posting issues
Options:
    -h                   help
    -v                   show version and exit
	-o owner             owner that should be used
    -r repo              repository that should be used
    -i issue id          original issue id
    -c commits           commits to be backported (comma seperated)
    -b branch(es)        branches issue is being backported to
    -u user              user to assign new issues to (default: user assigned to orig. issue)

Examples: 
    # generate 2 backport issues for k3s issue 1234
    %[2]s -o k3s-io -r k3s -b "release-1.25,release-1.26" -i 1234 -c 1
	%[2]s -o k3s-io -r k3s -b "release-1.26" -i 1234 -c 1,2,3
	%[2]s -o k3s-io -r k3s -b "release-1.26" -i 1234 -c 1,2,3 -u susejsmith

Note: if a commit is provided, %[2]s utility needs to be ran from either
	  the RKE2 or k3s directory.
`

var (
	vers      bool
	owner     string
	repo      string
	commitIDs string
	issueID   uint
	branches  string
	user      string
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
	flag.StringVar(&owner, "o", "", "")
	flag.StringVar(&repo, "r", "", "")
	flag.StringVar(&commitIDs, "c", "", "")
	flag.UintVar(&issueID, "i", 0, "")
	flag.StringVar(&branches, "b", "", "")
	flag.StringVar(&user, "u", "", "")
	flag.Parse()

	if vers {
		fmt.Fprintf(os.Stdout, "version: %s - git sha: %s\n", version, gitSHA)
		return
	}

	ghToken := os.Getenv("GITHUB_TOKEN")
	if ghToken == "" {
		fmt.Println("error: please provide a GITHUB_TOKEN")
		os.Exit(1)
	}

	var commits []string
	if commitIDs != "" {
		commits = strings.Split(commitIDs, ",")
	}

	if issueID == 0 {
		fmt.Println("error: please provide a valid issue id")
		os.Exit(1)
	}

	ctx := context.Background()

	client := repository.NewGithub(ctx, ghToken)

	pbo := repository.PerformBackportOpts{
		Owner:    owner,
		Repo:     repo,
		Commits:  commits,
		IssueID:  issueID,
		Branches: branches,
		User:     user,
	}
	issues, err := repository.PerformBackport(ctx, client, &pbo)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	for _, issue := range issues {
		fmt.Println("Backport issue created: " + issue.GetHTMLURL())
	}

	os.Exit(0)
}
