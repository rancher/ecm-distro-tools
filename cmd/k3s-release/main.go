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
    -h                    help
    -v                    show version and exit
    -w  workspace         path to workspace dir
	-o  oldk8sver         old kubernetes version
	-n  newk8sver		  new kubernetes version
	-ok oldk3sver         old k3s version
	-nk newk3sver         new k3s version
	-oc oldclient         old kubernetes client
	-nc newclient         new kubernetes client
	-b  branch            release branch
	-g  github            github handler
	-e  email             email
    -d                    enable debug logs
`

var (
	vers          bool
	debug         bool
	workspace     string
	newK8SClient  string
	oldK8SClient  string
	newK8SVersion string
	oldK8SVersion string
	newK3SVersion string
	oldK3SVersion string
	releaseBranch string
	githubHandler string
	email         string
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
	flag.StringVar(&releaseBranch, "b", "", "")
	flag.StringVar(&oldK3SVersion, "ok", "", "")
	flag.StringVar(&newK3SVersion, "nk", "", "")
	flag.StringVar(&oldK8SVersion, "o", "", "")
	flag.StringVar(&newK8SVersion, "n", "", "")
	flag.StringVar(&oldK8SClient, "oc", "", "")
	flag.StringVar(&newK8SClient, "nc", "", "")
	flag.StringVar(&email, "e", "", "")
	flag.StringVar(&githubHandler, "g", "", "")
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
		// NewK8SVersion: "v1.22.10-rc.0",
		// OldK8SVersion: "v1.22.9",
		// NewK8SClient:  "0.22.10",
		// OldK8SClient:  "0.22.9",
		// NewK3SVersion: "k3s1",
		// OldK3SVersion: "k3s1",
		// ReleaseBranch: "release-1.22",
		NewK8SVersion: newK8SVersion,
		OldK8SVersion: oldK8SVersion,
		NewK8SClient:  newK8SClient,
		OldK8SClient:  oldK8SClient,
		NewK3SVersion: newK3SVersion,
		OldK3SVersion: oldK3SVersion,
		ReleaseBranch: releaseBranch,
	}
	tags, err := release.RebaseAndTag(ctx, client, githubHandler, email, workspace, options)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if err := release.PushTags(ctx, tags, client, workspace, email, githubHandler, githubHandler); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if err := release.ModifyAndPush(ctx, options, workspace, githubHandler, email); err != nil {
		fmt.Println(err)
	}
	if err := release.CreatePRFromK3S(ctx, client, workspace, githubHandler, options); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
