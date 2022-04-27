package release

import (
	"bufio"
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
	"text/template"

	"github.com/google/go-github/v39/github"
	"github.com/rancher/ecm-distro-tools/repository"
	"github.com/sirupsen/logrus"
	"golang.org/x/mod/modfile"
)

// GenReleaseNotes genereates release notes based on the given milestone,
// previous milestone, and repository.
func GenReleaseNotes(ctx context.Context, repo, milestone, prevMilestone, ghToken string) (*bytes.Buffer, error) {
	const templateName = "release-notes"

	var tmpl *template.Template
	switch repo {
	case "rke2":
		tmpl = template.Must(template.New(templateName).Parse(rke2ReleaseNoteTemplate))
	case "k3s":
		tmpl = template.Must(template.New(templateName).Parse(k3sReleaseNoteTemplate))
	}

	client := repository.NewGithub(ctx, ghToken)

	content, err := repository.RetrieveChangeLogContents(ctx, client, repo, prevMilestone, milestone)
	if err != nil {
		return nil, err
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

	buf := bytes.NewBuffer(nil)

	if err := tmpl.Execute(buf, map[string]interface{}{
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
		return nil, err
	}

	return buf, nil
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

const rke2ReleaseNoteTemplate = `<!-- {{.milestone}} -->

This release ... <FILL ME OUT!>

**Important Note**

If your server (control-plane) nodes were not started with the ` + "`--token`" + ` CLI flag or config file key, a randomized token was generated during initial cluster startup. This key is used both for joining new nodes to the cluster, and for encrypting cluster bootstrap data within the datastore. Ensure that you retain a copy of this token, as is required when restoring from backup.

You may retrieve the token value from any server already joined to the cluster:
` + "```bash" + `
cat /var/lib/rancher/rke2/server/token
` + "```" + `

## Changes since {{.prevMilestone}}:
{{range .content}}
* {{.Title}} [(#{{.Number}})]({{.URL}}){{end}}

## Packaged Component Versions
| Component       | Version                                                                                           |
| --------------- | ------------------------------------------------------------------------------------------------- |
| Kubernetes      | [{{.k8sVersion}}](https://github.com/kubernetes/kubernetes/blob/master/CHANGELOG/CHANGELOG-{{.majorMinor}}.md#{{.changeLogVersion}}) |
| Etcd            | [{{.EtcdVersion}}](https://github.com/k3s-io/etcd/releases/tag/{{.EtcdVersion}})                          |
| Containerd      | [{{.ContainerdVersion}}](https://github.com/k3s-io/containerd/releases/tag/{{.ContainerdVersion}})                      |
| Runc            | [{{.RuncVersion}}](https://github.com/opencontainers/runc/releases/tag/{{.RuncVersion}})                              |
| Metrics-server  | [{{.MetricsServerVersion}}](https://github.com/kubernetes-sigs/metrics-server/releases/tag/{{.MetricsServerVersion}})                   |
| CoreDNS         | [{{.CoreDNSVersion}}](https://github.com/coredns/coredns/releases/tag/{{.CoreDNSVersion}})                                  |
| Ingress-Nginx   | [{{.IngressNginxVersion}}](https://github.com/kubernetes/ingress-nginx/releases/tag/helm-chart-{{.IngressNginxVersion}})                                  |
| Helm-controller | [{{.HelmControllerVersion}}](https://github.com/k3s-io/helm-controller/releases/tag/{{.HelmControllerVersion}})                         |

### Available CNIs
| Component       | Version                                                                                                                                                                             | FIPS Compliant |
| --------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------- |
| Canal (Default) | [Flannel {{.FlannelVersionRKE2}}](https://github.com/k3s-io/flannel/releases/tag/{{.FlannelVersionRKE2}})<br/>[Calico {{.CalicoVersion}}](https://projectcalico.docs.tigera.io/archive/{{ .CalicoVersionMajMin }}/release-notes/#{{ .CalicoVersionTrimmed }}) | Yes            |
| Calico          | [{{.CalicoVersion}}](https://projectcalico.docs.tigera.io/archive/{{ .CalicoVersionMajMin }}/release-notes/#{{ .CalicoVersionTrimmed }})                                                                    | No             |
| Cilium          | [{{.CiliumVersion}}](https://github.com/cilium/cilium/releases/tag/{{.CiliumVersion}})                                                                                                                      | No             |
| Multus          | [{{.MultusVersion}}](https://github.com/k8snetworkplumbingwg/multus-cni/releases/tag/{{.MultusVersion}})                                                                                                    | No             |

## Known Issues

- [#1447](https://github.com/rancher/rke2/issues/1447) - When restoring RKE2 from backup to a new node, you should ensure that all pods are stopped following the initial restore:

` + "```" + `bash
curl -sfL https://get.rke2.io | sudo INSTALL_RKE2_VERSION={{.milestone}}
rke2 server \
  --cluster-reset \
  --cluster-reset-restore-path=<PATH-TO-SNAPSHOT> --token <token used in the original cluster>
rke2-killall.sh
systemctl enable rke2-server
systemctl start rke2-server
` + "```" + `

## Helpful Links

As always, we welcome and appreciate feedback from our community of users. Please feel free to:
- [Open issues here](https://github.com/rancher/rke2/issues/new)
- [Join our Slack channel](https://slack.rancher.io/)
- [Check out our documentation](https://docs.rke2.io) for guidance on how to get started.
`

const k3sReleaseNoteTemplate = `<!-- {{.milestone}} -->
This release updates Kubernetes to {{.k8sVersion}}, and fixes a number of issues.

For more details on what's new, see the [Kubernetes release notes](https://github.com/kubernetes/kubernetes/blob/master/CHANGELOG/CHANGELOG-{{.majorMinor}}.md#changelog-since-{{.changeLogSince}}).

## Changes since {{.prevMilestone}}:
{{range .content}}
* {{.Title}} [(#{{.Number}})]({{.URL}}){{end}}

## Embedded Component Versions
| Component | Version |
|---|---|
| Kubernetes | [{{.k8sVersion}}](https://github.com/kubernetes/kubernetes/blob/master/CHANGELOG/CHANGELOG-{{.majorMinor}}.md#{{.changeLogVersion}}) |
| Kine | [{{.KineVersion}}](https://github.com/k3s-io/kine/releases/tag/{{.KineVersion}}) |
| SQLite | [{{.SQLiteVersion}}](https://sqlite.org/releaselog/{{.SQLiteVersionReplaced}}.html) |
| Etcd | [{{.EtcdVersion}}](https://github.com/k3s-io/etcd/releases/tag/{{.EtcdVersion}}) |
| Containerd | [{{.ContainerdVersion}}](https://github.com/k3s-io/containerd/releases/tag/{{.ContainerdVersion}}) |
| Runc | [{{.RuncVersion}}](https://github.com/opencontainers/runc/releases/tag/{{.RuncVersion}}) |
| Flannel | [{{.FlannelVersionK3S}}](https://github.com/flannel-io/flannel/releases/tag/{{.FlannelVersionK3S}}) | 
| Metrics-server | [{{.MetricsServerVersion}}](https://github.com/kubernetes-sigs/metrics-server/releases/tag/{{.MetricsServerVersion}}) |
| Traefik | [v{{.TraefikVersion}}](https://github.com/traefik/traefik/releases/tag/v{{.TraefikVersion}}) |
| CoreDNS | [v{{.CoreDNSVersion}}](https://github.com/coredns/coredns/releases/tag/v{{.CoreDNSVersion}}) | 
| Helm-controller | [{{.HelmControllerVersion}}](https://github.com/k3s-io/helm-controller/releases/tag/{{.HelmControllerVersion}}) |
| Local-path-provisioner | [{{.LocalPathProvisionerVersion}}](https://github.com/rancher/local-path-provisioner/releases/tag/{{.LocalPathProvisionerVersion}}) |

## Helpful Links
As always, we welcome and appreciate feedback from our community of users. Please feel free to:
- [Open issues here](https://github.com/rancher/k3s/issues/new/choose)
- [Join our Slack channel](https://slack.rancher.io/)
- [Check out our documentation](https://rancher.com/docs/k3s/latest/en/) for guidance on how to get started or to dive deep into K3s.
- [Read how you can contribute here](https://github.com/rancher/k3s/blob/master/CONTRIBUTING.md)
`
