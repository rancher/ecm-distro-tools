package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/rancher/ecm-distro-tools/release"
	"github.com/rancher/ecm-distro-tools/repository"
	"github.com/sirupsen/logrus"
)

var (
	name    string
	version string
	gitSHA  string
)

const usage = `version: %s
Usage: %[2]s [-r repo] [-m milestone] [-p prev milestone]
Options:
    -h                   help
    -v                   show version and exit
    -r repo              repository that should be used
    -m milestone         milestone to be used
	-p prev milestone    previous milestone
Examples:
    # generate 2 backport issues for k3s issue 1234
    %[2]s -r rke2 -m v1.21.5+rke2r1 -p v1.21.4+rke2r1
`

var (
	vers          bool
	repo          string
	milestone     string
	prevMilestone string
	debug         bool
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
	flag.BoolVar(&debug, "d", false, "")
	flag.StringVar(&repo, "r", "", "")
	flag.StringVar(&milestone, "m", "", "")
	flag.StringVar(&prevMilestone, "p", "", "")
	flag.Parse()

	if vers {
		logrus.Infof("version: %s - git sha: %s\n", version, gitSHA)
		os.Exit(0)
	}

	if debug {
		logrus.SetLevel(logrus.DebugLevel)
	}

	ghToken := os.Getenv("GITHUB_TOKEN")
	if ghToken == "" {
		fmt.Println("error: github token required")
		os.Exit(1)
	}

	if !repository.IsValidRepo(repo) {
		fmt.Println("error: please provide a valid repository")
		os.Exit(1)
	}

	if milestone == "" || prevMilestone == "" {
		fmt.Println("error: a valid milestone and prev milestone are required")
		os.Exit(1)
	}

	ctx := context.Background()
	notes, err := release.GenReleaseNotes(ctx, repo, milestone, prevMilestone, ghToken)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Print(notes.String())
}
