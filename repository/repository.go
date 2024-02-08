package repository

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/google/go-github/v39/github"
	"github.com/rancher/ecm-distro-tools/exec"
	"github.com/rancher/ecm-distro-tools/types"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

const (
	releaseNoteSection = "```release-note"
	emptyReleaseNote   = "```release-note\r\n\r\n```"
	noneReleaseNote    = "```release-note\r\nNONE\r\n```"
	httpTimeout        = time.Second * 10
	ghContentURL       = "https://raw.githubusercontent.com"
)

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
	if token == "" {
		return github.NewClient(nil)
	}

	ts := TokenSource{
		AccessToken: token,
	}
	oauthClient := oauth2.NewClient(ctx, &ts)
	oauthClient.Timeout = httpTimeout

	return github.NewClient(oauthClient)
}

// CreateReleaseOpts
type CreateReleaseOpts struct {
	Owner        string `json:"owner"`
	Repo         string `json:"repo"`
	Name         string `json:"name"`
	Tag          string `json:"tag"`
	Prerelease   bool   `json:"pre_release"`
	Branch       string `json:"branch"`
	ReleaseNotes string `json:"release_notes"`
	Draft        bool   `json:"draft"`
}

// ListReleases
func ListReleases(ctx context.Context, client *github.Client, owner, repo string) ([]*github.RepositoryRelease, error) {
	releases, _, err := client.Repositories.ListReleases(ctx, owner, repo, &github.ListOptions{})
	if err != nil {
		return nil, err
	}

	return releases, nil
}

// ListTags
func ListTags(ctx context.Context, client *github.Client, owner, repo string) ([]*github.RepositoryTag, error) {
	tags, _, err := client.Repositories.ListTags(ctx, owner, repo, &github.ListOptions{})
	if err != nil {
		return nil, err
	}

	return tags, nil
}

func LatestTag(ctx context.Context, client *github.Client, owner, repo string) (*github.RepositoryTag, error) {
	tags, err := ListTags(ctx, client, owner, repo)
	if err != nil {
		return nil, err
	}

	if len(tags) == 0 {
		return nil, nil
	}

	return tags[0], nil
}

// CreateRelease
func CreateRelease(ctx context.Context, client *github.Client, cro *CreateReleaseOpts) (*github.RepositoryRelease, error) {
	if cro == nil {
		return nil, errors.New("CreateReleaseOpts cannot be nil")
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

	release, _, err := client.Repositories.CreateRelease(ctx, cro.Owner, cro.Repo, &rr)
	if err != nil {
		return nil, err
	}

	return release, nil
}

// CreateReleaseIssueOpts
type CreateReleaseIssueOpts struct {
	Owner   string
	Repo    string
	Release string
	Captain string
}

// CreateReleaseIssue
func CreateReleaseIssue(ctx context.Context, client *github.Client, cri *CreateReleaseIssueOpts) (*github.Issue, error) {
	body := fmt.Sprintf(cutRKE2ReleaseIssue, cri.Release, cri.Release)
	ir := github.IssueRequest{
		Title:    types.StringPtr("Cut " + cri.Release),
		Body:     types.StringPtr(body),
		Assignee: types.StringPtr(cri.Captain),
		State:    types.StringPtr("open"),
	}

	issue, _, err := client.Issues.Create(ctx, cri.Owner, cri.Repo, &ir)
	if err != nil {
		return nil, err
	}

	return issue, nil
}

// RetrieveOriginalIssue
func RetrieveOriginalIssue(ctx context.Context, client *github.Client, owner, repo string, issueID uint) (*github.Issue, error) {
	issue, _, err := client.Issues.Get(ctx, owner, repo, int(issueID))
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
func CreateBackportIssues(ctx context.Context, client *github.Client, origIssue *github.Issue, owner, repo, branch, user string, i *Issue) (*github.Issue, error) {
	caser := cases.Title(language.English)
	title := fmt.Sprintf(i.Title, caser.String(branch), origIssue.GetTitle())
	body := fmt.Sprintf(i.Body, origIssue.GetTitle(), *origIssue.Number)

	var assignee *string
	if user != "" {
		assignee = types.StringPtr(user)
	} else if origIssue.GetAssignee() != nil {
		assignee = origIssue.GetAssignee().Login
	} else {
		assignee = types.StringPtr("")
	}
	issue, _, err := client.Issues.Create(ctx, owner, repo, &github.IssueRequest{
		Title:    github.String(title),
		Body:     github.String(body),
		Labels:   &[]string{"kind/backport"},
		Assignee: assignee,
	})
	if err != nil {
		return nil, err
	}

	return issue, nil
}

// PerformBackportOpts
type PerformBackportOpts struct {
	Owner           string   `json:"owner"`
	Repo            string   `json:"repo"`
	Commits         []string `json:"commits"`
	IssueID         uint     `json:"issue_id"`
	Branches        []string `json:"branches"`
	User            string   `json:"user"`
	DryRun          bool     `json:"dry_run"`
	SkipCreateIssue bool     `json:"skip_create_issue"`
}

// PerformBackport creates backport issues, performs a cherry-pick of the
// given commit if it exists.
func PerformBackport(ctx context.Context, client *github.Client, pbo *PerformBackportOpts) ([]*github.Issue, error) {
	var issues []*github.Issue
	var cwd string
	var r *git.Repository
	var w *git.Worktree
	var cherryPick bool

	cherryPick = false
	if len(pbo.Commits) != 0 {
		cherryPick = true
	}

	origIssue, err := RetrieveOriginalIssue(ctx, client, pbo.Owner, pbo.Repo, pbo.IssueID)
	if err != nil {
		return nil, err
	}
	issue := Issue{
		Title: "[%s] - %s",
		Body:  "Backport fix for %s\n\n* #%d",
	}
	if cherryPick {
		// we're assuming this code is called from the repository itself
		logrus.Info("getting working directory")
		cwd, err = os.Getwd()
		if err != nil {
			return nil, err
		}
		logrus.Info("working directory: " + cwd)

		logrus.Info("opening git repository at working directory")
		r, err = git.PlainOpen(cwd)
		if err != nil {
			return nil, errors.New("not in a git repository, make sure you are executing this inside the " + pbo.Repo + " repo: " + err.Error())
		}

		logrus.Info("getting repository worktree")
		w, err = r.Worktree()
		if err != nil {
			return nil, err
		}
		upstreamRemoteURL := "https://github.com/" + pbo.Owner + "/" + pbo.Repo + ".git"
		fmt.Println("creating remote: 'upstream " + upstreamRemoteURL + "'")
		if _, err := r.CreateRemote(&config.RemoteConfig{
			Name: "upstream",
			URLs: []string{upstreamRemoteURL},
		}); err != nil {
			if err != git.ErrRemoteExists {
				return nil, err
			}
		}
		fmt.Println("fetching remote: upstream")
		if err := r.Fetch(&git.FetchOptions{
			RemoteName: "upstream",
			Progress:   os.Stdout,
			Tags:       git.AllTags,
		}); err != nil {
			if err != git.NoErrAlreadyUpToDate {
				return nil, err
			}
		}
	}

	for _, branch := range pbo.Branches {
		if cherryPick {
			coo := git.CheckoutOptions{Branch: plumbing.ReferenceName("refs/remotes/upstream/" + branch)}
			logrus.Info("checking out on reference refs/remotes/upstream/" + branch)
			logrus.Infof("checkout options: %+v", coo)
			if err := w.Checkout(&coo); err != nil {
				return nil, errors.New("failed checkout: " + err.Error())
			}

			newBranchName := fmt.Sprintf("issue-%d_%s", pbo.IssueID, branch)
			logrus.Info("new branch name: " + newBranchName)

			logrus.Info("getting head reference")
			headRef, err := r.Head()
			if err != nil {
				return nil, err
			}

			logrus.Info("getting new hash reference")
			ref := plumbing.NewHashReference(plumbing.NewBranchReferenceName(newBranchName), headRef.Hash())
			logrus.Info("setting new hash reference")
			if err := r.Storer.SetReference(ref); err != nil {
				return nil, err
			}

			coo = git.CheckoutOptions{
				Branch: plumbing.ReferenceName("refs/heads/" + newBranchName),
			}
			logrus.Info("checkout out on reference refs/heads/" + newBranchName)
			if err := w.Checkout(&coo); err != nil {
				return nil, errors.New("failed checkout: " + err.Error())
			}

			for _, commit := range pbo.Commits {
				logrus.Info("cherry picking commit: " + commit)
				cherryPickOut, err := exec.RunCommand(cwd, "git", "cherry-pick", commit)
				if err != nil {
					return nil, err
				}
				logrus.Info(cherryPickOut)

				if pbo.DryRun {
					logrus.Info("dry run, skipping push to origin for branch " + newBranchName)
					continue
				}
				logrus.Info("pushing " + newBranchName + " to origin")
				pushOut, err := exec.RunCommand(cwd, "git", "push", "origin", newBranchName)
				if err != nil {
					return nil, err
				}
				logrus.Info(pushOut)
			}
		}

		logrus.Info("creating issue | owner: " + pbo.Owner + " | Repo: " + pbo.Repo + " | Branch: " + branch)
		if pbo.DryRun || pbo.SkipCreateIssue {
			logrus.Info("skipping issue creation")
			continue
		}
		newIssue, err := CreateBackportIssues(ctx, client, origIssue, pbo.Owner, pbo.Repo, branch, pbo.User, &issue)
		if err != nil {
			return nil, err
		}
		issues = append(issues, newIssue)
	}

	return issues, nil
}

// RetrieveChangeLogContents gets the relevant changes
// for the given release, formats, and returns them.
func RetrieveChangeLogContents(ctx context.Context, client *github.Client, owner, repo, prevMilestone, milestone string) ([]ChangeLog, error) {
	comp, _, err := client.Repositories.CompareCommits(ctx, owner, repo, prevMilestone, milestone, &github.ListOptions{})
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

		prs, _, err := client.PullRequests.ListPullRequestsWithCommit(ctx, owner, repo, sha, &github.PullRequestListOptions{})
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
