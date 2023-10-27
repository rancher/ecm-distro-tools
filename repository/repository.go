package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/google/go-github/v39/github"
	"github.com/rancher/ecm-distro-tools/types"
	"golang.org/x/oauth2"
)

const (
	releaseNoteSection = "```release-note"
	emptyReleaseNote   = "```release-note\r\n\r\n```"
	noneReleaseNote    = "```release-note\r\nNONE\r\n```"
	httpTimeout        = time.Second * 10
	ghContentURL       = "https://raw.githubusercontent.com"
	ghAPIURL           = "https://api.github.com"
)

// repoToOrg associates repo to org.
var repoToOrg = map[string]string{
	"rke2":             "rancher",
	"k3s":              "k3s-io",
	"rancher":          "rancher",
	"image-build-base": "rancher",
}

// stripBackportTag returns a string with a prefix backport tag removed
func stripBackportTag(s string) string {
	if strings.Contains(s, "Release") || strings.Contains(s, "release") && strings.Contains(s, "[") || strings.Contains(s, "]") {
		s = strings.Split(s, "]")[1]
	}
	s = strings.Trim(s, " ")
	return s
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

// CreateReleaseOpts
type CreateReleaseOpts struct {
	Repo         string `json:"repo"`
	Name         string `json:"name"`
	Prerelease   bool   `json:"pre_release"`
	Branch       string `json:"branch"`
	ReleaseNotes string `json:"release_notes"`
	Draft        bool   `json:"draft"`
}

// ListReleases
func ListReleases(ctx context.Context, client *github.Client, repo string) ([]*github.RepositoryRelease, error) {
	org, err := OrgFromRepo(repo)
	if err != nil {
		return nil, err
	}
	releases, _, err := client.Repositories.ListReleases(ctx, org, repo, &github.ListOptions{})
	if err != nil {
		return nil, err
	}
	return releases, nil
}

// CreateRelease
func CreateRelease(ctx context.Context, client *github.Client, cro *CreateReleaseOpts) (*github.RepositoryRelease, error) {
	if cro == nil {
		return nil, errors.New("CreateReleaseOpts cannot be nil")
	}

	org, err := OrgFromRepo(cro.Repo)
	if err != nil {
		return nil, err
	}

	rr := github.RepositoryRelease{
		Name:            &cro.Name,
		TagName:         &cro.Name,
		Prerelease:      &cro.Prerelease,
		TargetCommitish: &cro.Branch,
		Draft:           &cro.Draft,
	}
	if cro.ReleaseNotes != "" {
		genReleaseNotes := true
		rr.Body = &cro.ReleaseNotes
		rr.GenerateReleaseNotes = &genReleaseNotes
	}
	release, _, err := client.Repositories.CreateRelease(ctx, org, cro.Repo, &rr)
	if err != nil {
		return nil, err
	}

	return release, nil
}

// CreateReleaseIssueOpts
type CreateReleaseIssueOpts struct {
	Repo    string
	Release string
	Captain string
}

// CreateReleaseIssue
func CreateReleaseIssue(ctx context.Context, client *github.Client, cri *CreateReleaseIssueOpts) (*github.Issue, error) {
	org, err := OrgFromRepo(cri.Repo)
	if err != nil {
		return nil, err
	}

	body := fmt.Sprintf(cutRKE2ReleaseIssue, cri.Release, cri.Release)
	ir := github.IssueRequest{
		Title:    types.StringPtr("Cut " + cri.Release),
		Body:     types.StringPtr(body),
		Assignee: types.StringPtr(cri.Captain),
		State:    types.StringPtr("open"),
	}
	issue, _, err := client.Issues.Create(ctx, org, cri.Repo, &ir)
	if err != nil {
		return nil, err
	}

	return issue, nil
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
	Note   string
	Number int
	URL    string
}

// CreateBackportIssues
func CreateBackportIssues(ctx context.Context, client *github.Client, origIssue *github.Issue, repo, branch, user string, i *Issue) (*github.Issue, error) {
	org, err := OrgFromRepo(repo)
	if err != nil {
		return nil, err
	}

	title := fmt.Sprintf(i.Title, strings.Title(branch), origIssue.GetTitle())
	body := fmt.Sprintf(i.Body, origIssue.GetTitle(), *origIssue.Number)

	var assignee *string
	if user != "" {
		assignee = types.StringPtr(user)
	} else if origIssue.GetAssignee() != nil {
		assignee = origIssue.GetAssignee().Login
	} else {
		assignee = types.StringPtr("")
	}
	issue, _, err := client.Issues.Create(ctx, org, repo, &github.IssueRequest{
		Title:    github.String(title),
		Body:     github.String(body),
		Assignee: assignee,
	})
	if err != nil {
		return nil, err
	}

	return issue, nil
}

// PerformBackportOpts
type PerformBackportOpts struct {
	Repo     string   `json:"repo"`
	Commits  []string `json:"commits"`
	IssueID  uint     `json:"issue_id"`
	Branches string   `json:"branches"`
	User     string   `json:"user"`
}

// PerformBackport creates backport issues, performs a cherry-pick of the
// given commit if it exists.
func PerformBackport(ctx context.Context, client *github.Client, pbo *PerformBackportOpts) ([]*github.Issue, error) {
	if !IsValidRepo(pbo.Repo) {
		return nil, fmt.Errorf("invalid repo: %s", pbo.Repo)
	}

	const (
		issueTitle = "[%s] - %s"
		issueBody  = "Backport fix for %s\n\n* #%d"
	)

	backportBranches := strings.Split(pbo.Branches, ",")
	if len(backportBranches) < 1 || backportBranches[0] == "" {
		return nil, errors.New("no branches specified")
	}

	origIssue, err := RetrieveOriginalIssue(ctx, client, pbo.Repo, pbo.IssueID)
	if err != nil {
		return nil, err
	}

	issue := Issue{
		Title: issueTitle,
		Body:  issueBody,
	}

	issues := make([]*github.Issue, len(backportBranches))
	for _, branch := range backportBranches {
		newIssue, err := CreateBackportIssues(ctx, client, origIssue, pbo.Repo, branch, pbo.User, &issue)
		if err != nil {
			return nil, err
		}
		issues = append(issues, newIssue)
	}

	// stop here if there are no commits given
	if len(pbo.Commits) == 0 {
		return issues, nil
	}

	// we're assuming this code is called from the repository itself
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	r, err := git.PlainOpen(cwd)
	if err != nil {
		return nil, err
	}

	w, err := r.Worktree()
	if err != nil {
		return nil, err
	}

	for _, branch := range backportBranches {
		coo := git.CheckoutOptions{
			Branch: plumbing.ReferenceName("refs/heads/" + branch),
		}
		if err := w.Checkout(&coo); err != nil {
			return nil, errors.New("failed checkout: " + err.Error())
		}

		newBranchName := fmt.Sprintf("issue-%d_%s", pbo.IssueID, branch)

		headRef, err := r.Head()
		if err != nil {
			return nil, err
		}

		ref := plumbing.NewHashReference(plumbing.NewBranchReferenceName(newBranchName), headRef.Hash())
		if err := r.Storer.SetReference(ref); err != nil {
			return nil, err
		}

		coo = git.CheckoutOptions{
			Branch: plumbing.ReferenceName("refs/heads/" + newBranchName),
		}
		if err := w.Checkout(&coo); err != nil {
			return nil, errors.New("failed checkout: " + err.Error())
		}

		for _, commit := range pbo.Commits {
			cmd := exec.Command("git", "cherry-pick", commit)
			stdoutStderr, err := cmd.CombinedOutput()
			if err != nil {
				return nil, err
			}
			fmt.Printf("%s\n", stdoutStderr)

			cmd = exec.Command("git", "push", "origin", newBranchName)
			stdoutStderr, err = cmd.CombinedOutput()
			if err != nil {
				return nil, err
			}
			fmt.Printf("%s\n", stdoutStderr)
		}
	}

	return issues, nil
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
	addedPRs := make(map[int]bool)
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
			if exists := addedPRs[prs[0].GetNumber()]; exists {
				continue
			}

			title := stripBackportTag(strings.TrimSpace(prs[0].GetTitle()))
			body := prs[0].GetBody()

			var releaseNote string
			var inNote bool
			if strings.Contains(body, releaseNoteSection) && !strings.Contains(body, emptyReleaseNote) && !strings.Contains(body, noneReleaseNote) {
				lines := strings.Split(body, "\n")
				for _, line := range lines {
					if strings.Contains(line, releaseNoteSection) {
						inNote = true
						continue
					}
					if strings.Contains(line, "```") {
						inNote = false
					}
					if inNote && line != "" {
						line = strings.TrimPrefix(line, "* ")
						releaseNote += line
					}
				}
				releaseNote = strings.TrimSpace(releaseNote)
				releaseNote = strings.ReplaceAll(releaseNote, "\r", "\n")
			}

			found = append(found, ChangeLog{
				Title:  title,
				Note:   releaseNote,
				Number: prs[0].GetNumber(),
				URL:    prs[0].GetHTMLURL(),
			})
			addedPRs[prs[0].GetNumber()] = true
		}
	}

	return found, nil
}

const cutRKE2ReleaseIssue = `**Summary:**
Task covering patch release work.
Dev Complete: 1/12 (Typically ~1 week prior to upstream release date)
**List of required releases:**
_To release as soon as able for QA:_
-  %s
_To release once have approval from QA:_
-  %[1]s (Never release on a Friday unless specified otherwise)
**Prep work:**
- [x] PJM: Dev and QA team to be notified of the incoming releases - add event to team calendar
- [ ] PJM: Dev and QA team to be notified of the date we will mark the latest release as stable - add event to team calendar [ONLY APPLICABLE FOR LATEST MINOR RELEASE]
- [ ] PJM: Sync with Rancher PJM to identify applicable Rancer release date
  - [x] Create tracking issues in rancher/rancher for each Rancher line that the RKE2 release is going into. Assign to release captain. Link to this issue. Ensure it's in the proper milestone by aligning with Rancher PJM.
 - <UPDATE WITH RANCHER ISSUE>
  - [ ] Track RKE2 release against the Rancher release date and vice versa. Communicate any changes to Rancher PJM and RKE2 team.
- [ ] QA: Review changes and understand testing efforts
- [ ] Release Captain: Prepare release notes in our private [release-notes repo](https://github.com/rancherlabs/release-notes) (submit PR for changes taking care to carefully check links and the components, once merged, create the release in GitHub and mark as a draft and check the pre-release box, fill in title, set target release branch, leave tag version blank for now until we are ready to release)
- [ ] QA: Validate and close out all issues in the release milestone.
**Vendor and release work:**
To find more information on specific steps, please see documentation [here](https://github.com/rancher/rke2/blob/master/developer-docs/upgrading_kubernetes.md)
- [ ] Release Captain: Tag new Hardened Kubernetes release
- [ ] Release Captain: Update Helm chart versions
- [ ] Release Captain: Update RKE2
- [ ] Release Captain: Tag new RKE2 RC
- [ ] Release Captain: Tag new RKE2 packaging RC "testing"
- [ ] Release Captain: Prepare PRs as needed to update [KDM](https://github.com/rancher/kontainer-driver-metadata/) in the appropriate dev branches using an RC.  For more information on the structure of the PR, see the [docs](https://github.com/rancher/rke2/blob/master/developer-docs/upgrading_kubernetes.md#update-rancher-kdm)
  - [ ] If server args, agent args, or charts are changed, link relevant rancher/rancher issue or create new rancher/rancher issue
  - [ ] If any new issues are created, escalated to Rancher PJM so they know and can plan for it 
- [ ] EM: Review and merge above PR
- [ ] QA: Post merge, run rancher with KDM pointed at the dev branch (where the PR in the previous step was merged) and test import, upgrade, and provisioning against those RCs. This work may be split between Rancher and RKE2 QAs.
- [ ] Release Captain: Tag the RKE2 release
- [ ] Release Captain: Add release notes to release
- [ ] Release Captain: Tag RKE2 packaging release "testing"
- [ ] Release Captain: Tag RKE2 packaging release "latest"
**Post-Release work:**
- [ ] Release Captain: Once release is fully complete (CI is all green and all release artifacts exist), edit the release, uncheck "Pre-release", and save.
- [ ] Wait 24 hours
- [ ] Release Captain: Tag RKE2 packaging "stable"
- [ ] Release Captain: Update stable release in channels.yaml
- [ ] Release Captain: Prepare PRs as needed to update [KDM](https://github.com/rancher/kontainer-driver-metadata/) in the appropriate dev branches to go from RC to non-RC release. Link this PR to rancher/rancher issue that is tracking the version bump (created in the "Prep work" phase)
- [ ] EM: Review and merge above PR. Update issue so that QA knows to test
- [ ] QA: Final validation of above PR and tracked through the linked ticket
- [ ] PJM: Close the milestone in GitHub.
`

type Author struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Date  string `json:"date"`
}

type Commit struct {
	Author    Author `json:"author"`
	Committer Author `json:"committer"`
	Message   string `json:"message"`
	URL       string `json:"url"`
}

type CommitResponse struct {
	SHA         string `json:"sha"`
	NodeID      string `json:"node_id"`
	Commit      Commit `json:"commit"`
	URL         string `json:"url"`
	HTMLURL     string `json:"html_url"`
	CommentsURL string `json:"comments_url"`
}

func CommitInfo(owner, repo, commitHash string, httpClient *http.Client) (*Commit, error) {
	var commitResponseMutex sync.Mutex

	apiUrl := fmt.Sprintf(ghAPIURL+"/repos/%s/%s/commits/%s", owner, repo, commitHash)

	response, err := httpClient.Get(apiUrl)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, errors.New("failed to fetch commit information. status code: " + strconv.Itoa(response.StatusCode))
	}

	var commitResponse CommitResponse

	commitResponseMutex.Lock()
	err = json.NewDecoder(response.Body).Decode(&commitResponse)
	commitResponseMutex.Unlock()
	if err != nil {
		return nil, err
	}

	return &commitResponse.Commit, nil
}

func ContentByFileNameAndCommit(owner, repo, commitHash, filePath string, httpClient *http.Client) ([]byte, error) {
	rawURL := fmt.Sprintf(ghContentURL+"/%s/%s/%s/%s", owner, repo, commitHash, filePath)

	response, err := http.Get(rawURL)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, errors.New("failed to fetch raw file. status code: " + strconv.Itoa(response.StatusCode))
	}
	data, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	return data, nil
}
