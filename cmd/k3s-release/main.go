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
    -w workspace         path to workspace dir
    -d                   enable debug logs
`

var (
	vers      bool
	debug     bool
	workspace string
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
	flag.StringVar(&workspace, "w", "", "")
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

	if workspace == "" {
		fmt.Println("error: workspace can not be empty")
		os.Exit(1)
	}

	ctx := context.Background()
	client := repository.NewGithub(ctx, ghToken)

	err := release.SetupK8sRemotes(ctx, client, "galal-hussein", workspace)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	options := release.K8STagsOptions{
		NewK8SVersion:  "v1.22.10-rc.0",
		OldK8SVersion:  "v1.22.9",
		NewK8SClient:   "0.22.10",
		OldK8SClient:   "0.22.9",
		NewK3SVersion:  "k3s1",
		OldK3SVersion:  "k3s1",
		ReleaseBranche: "release-1.22",
	}
	_, err = release.RebaseAndTag(ctx, client, "galal-hussein", "hussein.galal.ahmed.11@gmail.com", workspace, options)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
