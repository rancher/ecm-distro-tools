package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/go-github/v39/github"
	"golang.org/x/oauth2"
)

const (
	releaseNoteSection = "```release-note"

	httpTimeout = time.Second * 10
)

// repoToOrg associates repo to org.
var repoToOrg = map[string]string{
	"rke2": "rancher",
	"k3s":  "k3s-io",
}

// TokenSource
type TokenSource struct {
	AccessToken string
}

// Token
func (t *TokenSource) Token() (*oauth2.Token, error) {
	token := &oauth2.Token{
		AccessToken: t.AccessToken,
	}
	return token, nil
}

// NewGithub creates a value of type github.Client pointer
// with the given context and Github token.
func NewGithub(ctx context.Context, token string) *github.Client {
	ts := TokenSource{
		AccessToken: token,
	}
	oauthClient := oauth2.NewClient(ctx, &ts)
	oauthClient.Timeout = httpTimeout
	return github.NewClient(oauthClient)
}

// OrgFromRepo gets the Github organization that the
// given repository is in or returns an error if
// it is not found.
func OrgFromRepo(repo string) (string, error) {
	if repo, ok := repoToOrg[repo]; ok {
		return repo, nil
	}

	return "", errors.New("repo not found: " + repo)
}

// IsValidRepo determines if the given
// repository is valid for this program
// to operate against.
func IsValidRepo(repo string) bool {
	for r := range repoToOrg {
		if repo == r {
			return true
		}
	}

	return false
}

// RetrieveOriginalIssue
func RetrieveOriginalIssue(ctx context.Context, client *github.Client, repo string, issueID uint) (*github.Issue, error) {
	org, err := OrgFromRepo(repo)
	if err != nil {
		return nil, err
	}

	issue, _, err := client.Issues.Get(ctx, org, repo, int(issueID))
	if err != nil {
		return nil, err
	}

	return issue, nil
}

type Issue struct {
	ID    uint
	Title string
	Body  string
}

// ChangeLog contains the found changes
// for the given release, to be used in
// to populate the template.
type ChangeLog struct {
	Title  string
	Number int
	URL    string
}

// CreateBackportIssues
func CreateBackportIssues(ctx context.Context, client *github.Client, origIssue *github.Issue, repo, branch string, i *Issue) (*github.Issue, error) {
	org, err := OrgFromRepo(repo)
	if err != nil {
		return nil, err
	}

	title := fmt.Sprintf(i.Title, strings.Title(branch), origIssue.GetTitle())
	body := fmt.Sprintf(i.Body, origIssue.GetTitle(), *origIssue.Number)

	issue, _, err := client.Issues.Create(ctx, org, repo, &github.IssueRequest{
		Title:    github.String(title),
		Body:     github.String(body),
		Assignee: origIssue.GetAssignee().Login,
	})
	if err != nil {
		return nil, err
	}

	return issue, nil
}

// RetrieveChangeLogContents gets the relevant changes
// for the given release, formats, and returns them.
func RetrieveChangeLogContents(ctx context.Context, client *github.Client, repo, prevMilestone, milestone string) ([]ChangeLog, error) {
	org, err := OrgFromRepo(repo)
	if err != nil {
		return nil, err
	}

	comp, _, err := client.Repositories.CompareCommits(ctx, org, repo, prevMilestone, milestone, &github.ListOptions{})
	if err != nil {
		return nil, err
	}

	var found []ChangeLog

	for _, commit := range comp.Commits {
		sha := commit.GetSHA()
		if sha == "" {
			continue
		}

		prs, _, err := client.PullRequests.ListPullRequestsWithCommit(ctx, org, repo, sha, &github.PullRequestListOptions{})
		if err != nil {
			return nil, err
		}
		if len(prs) == 1 {
			body := prs[0].GetBody()

			var releaseNote string
			if strings.Contains(body, releaseNoteSection) {
				lines := strings.Split(body, "\n")
				for i, line := range lines {
					if strings.Contains(line, releaseNoteSection) {
						if lines[i+1] == "```" || lines[i+1] == "" {
							continue
						}
						releaseNote += lines[i+1]
					}
				}
				releaseNote = strings.TrimSpace(releaseNote)
			} else {
				releaseNote = prs[0].GetTitle()
			}

			found = append(found, ChangeLog{
				Title:  releaseNote,
				Number: prs[0].GetNumber(),
				URL:    prs[0].GetHTMLURL(),
			})
		}
	}

	return found, nil
}

const RKE2ReleaseNoteTemplate = `<!-- {{.milestone}} -->

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
| Etcd            | [v3.4.13-k3s1](https://github.com/k3s-io/etcd/releases/tag/v3.4.13-k3s1)                          |
| Containerd      | [v1.4.9-k3s1](https://github.com/k3s-io/containerd/releases/tag/v1.4.9-k3s1)                      |
| Runc            | [v1.0.0](https://github.com/opencontainers/runc/releases/tag/v1.0.0)                              |
| CNI Plugins     | [v0.8.7](https://github.com/containernetworking/plugins/releases/tag/v0.8.7)                      |
| Metrics-server  | [v0.3.6](https://github.com/kubernetes-sigs/metrics-server/releases/tag/v0.3.6)                   |
| CoreDNS         | [v1.8.3](https://github.com/coredns/coredns/releases/tag/v1.8.3)                                  |
| Ingress-Nginx   | [3.34.001](https://github.com/kubernetes/ingress-nginx/releases)                                  |
| Helm-controller | [v0.10.6](https://github.com/k3s-io/helm-controller/releases/tag/v0.10.6)                         |

### Available CNIs
| Component       | Version                                                                                                                                                                             | FIPS Compliant |
| --------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------- |
| Canal (Default) | [Flannel v0.13.0-rancher1](https://github.com/k3s-io/flannel/releases/tag/v0.13.0-rancher1)<br/>[Calico v3.13.3](https://docs.projectcalico.org/archive/v3.13/release-notes/#v3133) | Yes            |
| Calico          | [v3.19.2](https://docs.projectcalico.org/release-notes/#v3192)                                                                                                                      | No             |
| Cilium          | [v1.9.8](https://github.com/cilium/cilium/releases/tag/v1.9.8)                                                                                                                      | No             |
| Multus          | [v3.7.1](https://github.com/k8snetworkplumbingwg/multus-cni/releases/tag/v3.7.1)                                                                                                    | No             |

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

const K3sReleaseNoteTemplate = `<!-- {{.milestone}} -->
This release updates Kubernetes to {{.k8sVersion}}, and fixes a number of issues.

For more details on what's new, see the [Kubernetes release notes](https://github.com/kubernetes/kubernetes/blob/master/CHANGELOG/CHANGELOG-{{.majorMinor}}.md#changelog-since-{{.changeLogSince}}).

## Changes since {{.prevMilestone}}:
{{range .content}}
* {{.Title}} [(#{{.Number}})]({{.URL}}){{end}}

## Embedded Component Versions
| Component | Version |
|---|---|
| Kubernetes | [{{.k8sVersion}}](https://github.com/kubernetes/kubernetes/blob/master/CHANGELOG/CHANGELOG-{{.majorMinor}}.md#{{.changeLogVersion}}) |
| Kine | [v0.6.2](https://github.com/k3s-io/kine/releases/tag/v0.6.2) |
| SQLite | [3.33.0](https://sqlite.org/releaselog/3_33_0.html) |
| Etcd | [v3.4.13-k3s1](https://github.com/k3s-io/etcd/releases/tag/v3.4.13-k3s1) |
| Containerd | [v1.4.9-k3s1](https://github.com/k3s-io/containerd/releases/tag/v1.4.9-k3s1) |
| Runc | [v1.0.2](https://github.com/opencontainers/runc/releases/tag/v1.0.2) |
| Flannel | [v0.14.0](https://github.com/flannel-io/flannel/releases/tag/v0.14.0) | 
| Metrics-server | [v0.3.6](https://github.com/kubernetes-sigs/metrics-server/releases/tag/v0.3.6) |
| Traefik | [v2.4.8](https://github.com/traefik/traefik/releases/tag/v2.4.8) |
| CoreDNS | [v1.8.3](https://github.com/coredns/coredns/releases/tag/v1.8.3) | 
| Helm-controller | [v0.10.5](https://github.com/k3s-io/helm-controller/releases/tag/v0.10.1) |
| Local-path-provisioner | [v0.0.19](https://github.com/rancher/local-path-provisioner/releases/tag/v0.0.19) |

## Helpful Links
As always, we welcome and appreciate feedback from our community of users. Please feel free to:
- [Open issues here](https://github.com/rancher/k3s/issues/new/choose)
- [Join our Slack channel](https://slack.rancher.io/)
- [Check out our documentation](https://rancher.com/docs/k3s/latest/en/) for guidance on how to get started or to dive deep into K3s.
- [Read how you can contribute here](https://github.com/rancher/k3s/blob/master/CONTRIBUTING.md)
`
