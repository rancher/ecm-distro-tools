package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strings"
	"text/template"

	"github.com/rancher/ecm-distro-tools/repository"
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
	calicoVersion := getImageTagVersion("calico-node", repo, milestone)
	calicoVersionTrimmed := strings.Replace(calicoVersion, ".", "", -1)
	sqliteVersionK3S := getGoModVersion("go-sqlite3", repo, milestone)
	sqliteVersionBinding := getSqliteVersionBinding(sqliteVersionK3S)

	if err := tmpl.Execute(os.Stdout, map[string]interface{}{
		"milestone":                   milestone,
		"prevMilestone":               prevMilestone,
		"changeLogSince":              changeLogSince,
		"content":                     content,
		"k8sVersion":                  k8sVersion,
		"changeLogVersion":            markdownVersion,
		"majorMinor":                  majorMinor,
		"EtcdVersion":                 getGoModVersion("etcd", repo, milestone),
		"ContainerdVersion":           getGoModVersion("containerd", repo, milestone),
		"RuncVersion":                 getGoModVersion("runc", repo, milestone),
		"CNIPluginsVersion":           getImageTagVersion("cni-plugins", repo, milestone),
		"MetricsServerVersion":        getImageTagVersion("metrics-server", repo, milestone),
		"TraefikVersion":              getImageTagVersion("traefik", repo, milestone),
		"CoreDNSVersion":              getImageTagVersion("coredns", repo, milestone),
		"IngressNginxVersion":         getDockerfileVersion("rke2-ingress-nginx", repo, milestone),
		"HelmControllerVersion":       getGoModVersion("helm-controller", repo, prevMilestone),
		"FlannelVersionRKE2":          getImageTagVersion("flannel", repo, milestone),
		"FlannelVersionK3S":           getGoModVersion("flannel", repo, milestone),
		"CalicoVersion":               calicoVersion,
		"CalicoVersionTrimmed":        calicoVersionTrimmed,
		"CiliumVersion":               getImageTagVersion("cilium-cilium", repo, milestone),
		"MultusVersion":               getImageTagVersion("multus-cni", repo, milestone),
		"KineVersion":                 getGoModVersion("kine", repo, milestone),
		"SQLiteVersion":               sqliteVersionBinding,
		"SQLiteVersionReplaced":       strings.ReplaceAll(sqliteVersionBinding, ".", "_"),
		"LocalPathProvisionerVersion": getImageTagVersion("local-path-provisioner", repo, milestone),
	}); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	os.Exit(0)
}

func getGoModVersion(libraryName, repo, branchVersion string) string {
	repoName := "k3s-io/k3s"
	if repo == "rke2" {
		repoName = "rancher/rke2"
	}
	goModURL := "https://raw.githubusercontent.com/" + repoName + "/" + branchVersion + "/go.mod"
	resp, err := http.Get(goModURL)
	if err != nil {
		fmt.Printf("failed to fetch go.mod file from %s: %v\n", goModURL, err)
		os.Exit(1)
	}
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("status error: %v\n", resp.StatusCode)
		os.Exit(1)
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("read body error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	return findLibraryVersion(string(data), libraryName)
}

func findLibraryVersion(goModStr, libraryName string) string {
	scanner := bufio.NewScanner(strings.NewReader(goModStr))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, libraryName) {
			trimmedLine := strings.TrimSpace(line)
			// use replace section if found
			if strings.Contains(trimmedLine, "=>") {
				libVersionLine := strings.Split(trimmedLine, " ")
				return libVersionLine[3]
			} else {
				libVersionLine := strings.Split(trimmedLine, " ")
				return libVersionLine[1]
			}
		}
	}
	return ""
}

func getDockerfileVersion(chartName, repo, branchVersion string) string {
	if strings.Contains(repo, "k3s") {
		return ""
	}
	repoName := "rancher/rke2"
	DockerfileURL := "https://raw.githubusercontent.com/" + repoName + "/" + branchVersion + "/Dockerfile"
	resp, err := http.Get(DockerfileURL)
	if err != nil {
		fmt.Printf("failed to fetch dockerfile from %s: %v\n", DockerfileURL, err)
		os.Exit(1)
	}
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("status error: %v\n", resp.StatusCode)
		os.Exit(1)
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("read body error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	return findChartVersion(string(data), chartName)
}

func findChartVersion(dockerfileStr, chartName string) string {
	scanner := bufio.NewScanner(strings.NewReader(dockerfileStr))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, chartName) {
			re := regexp.MustCompile(`CHART_VERSION=\"(?P<version>.*?)([0-9][0-9])?(-build.*)?\"`)
			chartVersion := re.FindStringSubmatch(line)
			if len(chartVersion) > 1 {
				return chartVersion[1]
			}

		}
	}
	return ""
}

func getImageTagVersion(ImageName, repo, branchVersion string) string {
	repoName := "k3s-io/k3s"
	imageListURL := "https://raw.githubusercontent.com/" + repoName + "/" + branchVersion + "/scripts/airgap/image-list.txt"
	if repo == "rke2" {
		repoName = "rancher/rke2"
		imageListURL = "https://raw.githubusercontent.com/" + repoName + "/" + branchVersion + "/scripts/build-images"
	}
	resp, err := http.Get(imageListURL)
	if err != nil {
		return ""
	}
	if resp.StatusCode != http.StatusOK {
		return ""
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, ImageName) {
			re := regexp.MustCompile(`:(.*)(-build.*)?`)
			chartVersion := re.FindStringSubmatch(line)
			if len(chartVersion) > 1 {
				if strings.Contains(chartVersion[1], "-build") {
					versionSplit := strings.Split(chartVersion[1], "-")
					return versionSplit[0]
				}
				return chartVersion[1]
			}
		}
	}
	return ""
}

func getSqliteVersionBinding(sqliteVersion string) string {
	sqliteBindingURL := "https://raw.githubusercontent.com/mattn/go-sqlite3/" + sqliteVersion + "/sqlite3-binding.h"
	resp, err := http.Get(sqliteBindingURL)
	if err != nil {
		return ""
	}
	if resp.StatusCode != http.StatusOK {
		return ""
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "SQLITE_VERSION") {
			re := regexp.MustCompile(`\"(.*)\"`)
			chartVersion := re.FindStringSubmatch(line)
			if len(chartVersion) > 1 {
				return chartVersion[1]
			}
		}
	}
	return ""
}
