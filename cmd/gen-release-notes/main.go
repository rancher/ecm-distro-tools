package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"text/template"

	"github.com/rancher/ecm-distro-tools/pkg/repository"
)

var (
	name    string
	version string
	gitSHA  string
)

const templateName = "release-notes"

const usage = `version: %s
Usage: %[2]s [-r repo] [-m milestone] [-p prev milestone]
Options:
    -h                   help
    -v                   show version and exit
    -t                   github token (optional)
    -r repo              repository that should be used
    -m milestone         milestone to be used
	-p prev milestone    previous milestone
Examples: 
	# generate release notes for RKE2 for milestone v1.21.5
    %[2]s -r k3s -m v1.21.5+k3s1 -p v1.21.4+k3s1 
`

var (
	vers          bool
	ghToken       string
	repo          string
	milestone     string
	prevMilestone string
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
	flag.StringVar(&milestone, "m", "", "")
	flag.StringVar(&prevMilestone, "p", "", "")
	flag.Parse()

	if vers {
		fmt.Fprintf(os.Stdout, "version: %s - git sha: %s\n", version, gitSHA)
		return
	}

	if ghToken == "" {
		fmt.Println("error: please provide a token")
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

	var tmpl *template.Template
	switch repo {
	case "rke2":
		tmpl = template.Must(template.New(templateName).Parse(repository.RKE2ReleaseNoteTemplate))
	case "k3s":
		tmpl = template.Must(template.New(templateName).Parse(repository.K3sReleaseNoteTemplate))
	}

	ctx := context.Background()
	client := repository.NewGithub(ctx, ghToken)

	content, err := repository.RetrieveChangeLogContents(ctx, client, repo, prevMilestone, milestone)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// account for processing against an rc
	milestone = strings.Replace(milestone, "-rc", "", -1)

	idx := strings.Index(milestone, "-rc")
	if idx != -1 {
		tmpMilestone := []rune(milestone)
		tmpMilestone = append(tmpMilestone[0:idx], tmpMilestone[idx+4:]...)
		milestone = string(tmpMilestone)
	}

	k8sVersion := strings.Split(milestone, "+")[0]
	markdownVersion := strings.Replace(k8sVersion, ".", "", -1)
	tmp := strings.Split(strings.Replace(k8sVersion, "v", "", -1), ".")
	majorMinor := tmp[0] + "." + tmp[1]
	changeLogSince := strings.Replace(strings.Split(prevMilestone, "+")[0], ".", "", -1)

	if err := tmpl.Execute(os.Stdout, map[string]interface{}{
		"milestone":        milestone,
		"prevMilestone":    prevMilestone,
		"changeLogSince":   changeLogSince,
		"content":          content,
		"k8sVersion":       k8sVersion,
		"changeLogVersion": markdownVersion,
		"majorMinor":       majorMinor,
	}); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	os.Exit(0)
}
