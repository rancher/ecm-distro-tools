package release

import (
	"bufio"
	"context"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/google/go-github/v39/github"
	"github.com/rancher/ecm-distro-tools/repository"
	"github.com/sirupsen/logrus"
	"golang.org/x/mod/modfile"
)

func GenReleaseNotes(ctx context.Context, repo, milestone, prevMilestone, ghToken string) error {
	const templateName = "release-notes"

	var tmpl *template.Template
	switch repo {
	case "rke2":
		tmpl = template.Must(template.New(templateName).Parse(repository.RKE2ReleaseNoteTemplate))
	case "k3s":
		tmpl = template.Must(template.New(templateName).Parse(repository.K3sReleaseNoteTemplate))
	}

	client := repository.NewGithub(ctx, ghToken)

	content, err := repository.RetrieveChangeLogContents(ctx, client, repo, prevMilestone, milestone)
	if err != nil {
		return err
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
	calicoVersion := imageTagVersion("calico-node", repo, milestone)
	calicoVersionTrimmed := strings.Replace(calicoVersion, ".", "", -1)
	calicoVersionMajMin := calicoVersion[:strings.LastIndex(calicoVersion, ".")]
	sqliteVersionK3S := goModLibVersion("go-sqlite3", repo, milestone)
	sqliteVersionBinding := sqliteVersionBinding(sqliteVersionK3S)

	if err := tmpl.Execute(os.Stdout, map[string]interface{}{
		"milestone":                   milestone,
		"prevMilestone":               prevMilestone,
		"changeLogSince":              changeLogSince,
		"content":                     content,
		"k8sVersion":                  k8sVersion,
		"changeLogVersion":            markdownVersion,
		"majorMinor":                  majorMinor,
		"EtcdVersion":                 buildScriptVersion("ETCD_VERSION", repo, milestone),
		"ContainerdVersion":           goModLibVersion("containerd", repo, milestone),
		"RuncVersion":                 goModLibVersion("runc", repo, milestone),
		"CNIPluginsVersion":           imageTagVersion("cni-plugins", repo, milestone),
		"MetricsServerVersion":        imageTagVersion("metrics-server", repo, milestone),
		"TraefikVersion":              imageTagVersion("traefik", repo, milestone),
		"CoreDNSVersion":              imageTagVersion("coredns", repo, milestone),
		"IngressNginxVersion":         dockerfileVersion("rke2-ingress-nginx", repo, milestone),
		"HelmControllerVersion":       goModLibVersion("helm-controller", repo, milestone),
		"FlannelVersionRKE2":          imageTagVersion("flannel", repo, milestone),
		"FlannelVersionK3S":           goModLibVersion("flannel", repo, milestone),
		"CalicoVersion":               calicoVersion,
		"CalicoVersionMajMin":         calicoVersionMajMin,
		"CalicoVersionTrimmed":        calicoVersionTrimmed,
		"CiliumVersion":               imageTagVersion("cilium-cilium", repo, milestone),
		"MultusVersion":               imageTagVersion("multus-cni", repo, milestone),
		"KineVersion":                 goModLibVersion("kine", repo, milestone),
		"SQLiteVersion":               sqliteVersionBinding,
		"SQLiteVersionReplaced":       strings.ReplaceAll(sqliteVersionBinding, ".", "_"),
		"LocalPathProvisionerVersion": imageTagVersion("local-path-provisioner", repo, milestone),
	}); err != nil {
		return err
	}

	return nil
}

// CheckUpstreamRelease takes the given org, repo, and tags and checks
// for the tags' existence.
func CheckUpstreamRelease(ctx context.Context, client *github.Client, org, repo string, tags []string) ([]*github.RepositoryRelease, error) {
	releases := make([]*github.RepositoryRelease, len(tags))

	for _, tag := range tags {
		release, _, err := client.Repositories.GetReleaseByTag(ctx, org, repo, tag)
		if err != nil {
			return nil, err
		}
		releases = append(releases, release)
	}

	return releases, nil
}

func goModLibVersion(libraryName, repo, branchVersion string) string {
	repoName := "k3s-io/k3s"
	if repo == "rke2" {
		repoName = "rancher/rke2"
	}

	goModURL := "https://raw.githubusercontent.com/" + repoName + "/" + branchVersion + "/go.mod"

	resp, err := http.Get(goModURL)
	if err != nil {
		logrus.Debugf("failed to fetch url %s: %v", goModURL, err)
		return ""
	}
	if resp.StatusCode != http.StatusOK {
		logrus.Debugf("status error: %v when fetching %s", resp.StatusCode, goModURL)
		return ""
	}

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logrus.Debugf("read body error: %v", err)
		return ""
	}

	modFile, err := modfile.Parse("go.mod", b, nil)
	if err != nil {
		logrus.Debugf("failed to parse go.mod file: %v", err)
		return ""
	}

	// use replace section if found
	for _, replace := range modFile.Replace {
		if strings.Contains(replace.Old.Path, libraryName) {
			return replace.New.Version
		}
	}

	// if replace not found search in require
	for _, require := range modFile.Require {
		if strings.Contains(require.Mod.Path, libraryName) {
			return require.Mod.Version
		}
	}
	logrus.Debugf("library %s not found", libraryName)

	return ""
}

func buildScriptVersion(varName, repo, branchVersion string) string {
	repoName := "k3s-io/k3s"

	if repo == "rke2" {
		repoName = "rancher/rke2"
	}

	buildScriptURL := "https://raw.githubusercontent.com/" + repoName + "/" + branchVersion + "/scripts/version.sh"

	const regex = `(?P<version>v[\d\.]+)(-k3s.\w*)?`
	submatch := findInURL(buildScriptURL, regex, varName)

	if len(submatch) > 1 {
		return submatch[1]
	}

	return ""
}

func dockerfileVersion(chartName, repo, branchVersion string) string {
	if strings.Contains(repo, "k3s") {
		return ""
	}

	const (
		repoName = "rancher/rke2"
		regex    = `CHART_VERSION=\"(?P<version>.*?)([0-9][0-9])?(-build.*)?\"`
	)

	dockerfileURL := "https://raw.githubusercontent.com/" + repoName + "/" + branchVersion + "/Dockerfile"

	submatch := findInURL(dockerfileURL, regex, chartName)
	if len(submatch) > 1 {
		return submatch[1]
	}

	return ""
}

func imageTagVersion(ImageName, repo, branchVersion string) string {
	repoName := "k3s-io/k3s"

	imageListURL := "https://raw.githubusercontent.com/" + repoName + "/" + branchVersion + "/scripts/airgap/image-list.txt"
	if repo == "rke2" {
		repoName = "rancher/rke2"
		imageListURL = "https://raw.githubusercontent.com/" + repoName + "/" + branchVersion + "/scripts/build-images"
	}

	const regex = `:(.*)(-build.*)?`
	submatch := findInURL(imageListURL, regex, ImageName)

	if len(submatch) > 1 {
		if strings.Contains(submatch[1], "-build") {
			versionSplit := strings.Split(submatch[1], "-")
			return versionSplit[0]
		}
		return submatch[1]
	}

	return ""
}

func sqliteVersionBinding(sqliteVersion string) string {
	sqliteBindingURL := "https://raw.githubusercontent.com/mattn/go-sqlite3/" + sqliteVersion + "/sqlite3-binding.h"
	const (
		regex = `\"(.*)\"`
		word  = "SQLITE_VERSION"
	)

	submatch := findInURL(sqliteBindingURL, regex, word)
	if len(submatch) > 1 {
		return submatch[1]
	}

	return ""
}

// findInURL will get and scan a url to find a slice submatch for all the words that matches a regex
// if the regex is empty then it will return the lines in a file that matches the str
func findInURL(url, regex, str string) []string {
	var submatch []string

	resp, err := http.Get(url)
	if err != nil {
		logrus.Debugf("failed to fetch url %s: %v", url, err)
		return nil
	}
	if resp.StatusCode != http.StatusOK {
		logrus.Debugf("status error: %v when fetching %s", resp.StatusCode, url)
		return nil
	}

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logrus.Debugf("read body error: %v", err)
		return nil
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(strings.NewReader(string(b)))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, str) {
			if regex == "" {
				submatch = append(submatch, line)
			} else {
				re := regexp.MustCompile(regex)
				submatch = re.FindStringSubmatch(line)
				if len(submatch) > 1 {
					return submatch
				}
			}
		}
	}

	return submatch
}
