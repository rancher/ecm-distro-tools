package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/rancher/ecm-distro-tools/release"
	"github.com/rancher/ecm-distro-tools/repository"
)

var (
	name    string
	version string
	gitSHA  string
)

const usage = `version: %s
Usage: %[2]s [-b branches]
Env Variables:
    GITHUB_TOKEN         user token for posting issues
Options:
    -h                   help
    -v                   show version and exit
    -b branch(es)        branches to retrieve Go version from (comma separated)

Examples: 
    # print out Go version for the given branches
    %[2]s -b "release-1.25,release-1.26" 
`

var (
	vers     bool
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
	flag.StringVar(&branches, "b", "", "")
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

	ctx := context.Background()

	client := repository.NewGithub(ctx, ghToken)

	upstreamBranches := strings.Split(branches, ",")
	for _, branch := range upstreamBranches {
		version, err := release.KubernetesGoVersion(ctx, client, branch)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		fmt.Println(branch + ": " + version)
	}

	os.Exit(0)
}
