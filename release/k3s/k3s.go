package k3s

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/google/go-github/v39/github"
	ecmExec "github.com/rancher/ecm-distro-tools/exec"
	"github.com/rancher/ecm-distro-tools/release"
	"github.com/rancher/ecm-distro-tools/repository"
	"github.com/sirupsen/logrus"
	ssh2 "golang.org/x/crypto/ssh"
	"gopkg.in/yaml.v2"
)

const (
	k3sRepo            = "k3s"
	k8sUpstreamURL     = "https://github.com/kubernetes/kubernetes"
	rancherRemote      = "k3s-io"
	k8sRancherURL      = "git@github.com:k3s-io/kubernetes.git"
	k8sUserURL         = "git@github.com:user/kubernetes.git"
	k3sUpstreamRepoURL = "https://github.com/k3s-io/k3s"
	gitconfig          = `[safe]
directory = /home/go/src/kubernetes
[user]
email = %email%
name = %user%`
	dockerDevImage = `FROM %goimage%
RUN apk add --no-cache bash git make tar gzip curl git coreutils rsync alpine-sdk binutils-gold
ARG UID=1000
ARG GID=1000
RUN addgroup -S -g $GID ecmgroup && adduser -S -G ecmgroup -u $UID user
USER user`

	modifyScript = `#!/bin/bash
set -ex
OS=$(uname -s)
DRY_RUN={{ .DryRun }}
BRANCH_NAME={{ .NewK8SVersion }}-{{ .NewK3SSuffix }}
cd {{ .Workspace }}
# using ls | grep is not a good idea because it doesn't support non-alphanumeric filenames, but since we're only ever checking 'k3s' it isn't a problem https://www.shellcheck.net/wiki/SC2010
ls | grep -w k3s || git clone "git@github.com:{{ .Handler }}/k3s.git"
cd {{ .Workspace }}/k3s
git remote -v | grep -w upstream || git remote add upstream {{ .K3sUpstreamURL }}
git fetch upstream
git stash
git branch -D "${BRANCH_NAME}" &>/dev/null || true
git checkout -B "${BRANCH_NAME}" upstream/{{.ReleaseBranch}}
git clean -xfd

case ${OS} in
Darwin)
	sed -Ei '' "\|github.com/k3s-io/kubernetes| s|{{ replaceAll .OldK8SVersion "." "\\." }}-{{ .OldK3SSuffix }}|{{ replaceAll .NewK8SVersion "." "\\." }}-{{ .NewK3SSuffix }}|" go.mod
	sed -Ei '' "s/k8s.io\/kubernetes v\S+/k8s.io\/kubernetes {{ replaceAll .NewK8SVersion "." "\\." }}/" go.mod
	sed -Ei '' "s/{{ replaceAll .OldK8SClient "." "\\." }}/{{ replaceAll .NewK8SClient "." "\\." }}/g" go.mod # This should only change ~6 lines in go.mod
	sed -Ei '' "s/golang:.*-/golang:{{ .NewGoVersion }}-/g" Dockerfile.*
	sed -Ei '' "s/go-version:.*$/go-version:\ '{{ .NewGoVersion }}'/g" .github/workflows/integration.yaml .github/workflows/unitcoverage.yaml
	;;
Linux)
	sed -Ei "\|github.com/k3s-io/kubernetes| s|{{ replaceAll .OldK8SVersion "." "\\." }}-{{ .OldK3SSuffix }}|{{ replaceAll .NewK8SVersion "." "\\." }}-{{ .NewK3SSuffix }}|" go.mod
	sed -Ei "s/k8s.io\/kubernetes v\S+/k8s.io\/kubernetes {{ replaceAll .NewK8SVersion "." "\\." }}/" go.mod
	sed -Ei "s/{{ replaceAll .OldK8SClient "." "\\." }}/{{ replaceAll .NewK8SClient "." "\\." }}/g" go.mod # This should only change ~6 lines in go.mod
	sed -Ei "s/golang:.*-/golang:{{ .NewGoVersion }}-/g" Dockerfile.*
	sed -Ei "s/go-version:.*$/go-version:\ '{{ .NewGoVersion }}'/g" .github/workflows/integration.yaml .github/workflows/unitcoverage.yaml
	;;
*)
	>&2 echo "$(OS) not supported yet"
	exit 1
	;;
esac

go mod tidy

git add go.mod go.sum Dockerfile.* .github/workflows/integration.yaml .github/workflows/unitcoverage.yaml
	git commit --signoff -m "Update to {{ .NewK8SVersion }}"
if [ "${DRY_RUN}" = false ]; then
	git push --set-upstream origin "${BRANCH_NAME}" # run git remote -v for your origin
fi`
)

type Release struct {
	OldK8SVersion  string `json:"old_k8s_version"`
	NewK8SVersion  string `json:"new_k8s_version"`
	OldK8SClient   string `json:"old_k8s_client"`
	NewK8SClient   string `json:"new_k8s_client"`
	OldK3SSuffix   string `json:"old_k3s_suffix"`
	NewK3SSuffix   string `json:"new_k3s_suffix"`
	NewGoVersion   string `json:"-"`
	ReleaseBranch  string `json:"release_branch"`
	Workspace      string `json:"workspace"`
	K3sRemote      string `json:"k3s_remote"`
	Handler        string `json:"handler"`
	Email          string `json:"email"`
	GithubToken    string `json:"-"`
	K8sRancherURL  string `json:"k8s_rancher_url"`
	K3sUpstreamURL string `json:"k3s_upstream_url"`
	SSHKeyPath     string `json:"ssh_key_path"`
	DryRun         bool   `json:"dry_run"`
}

func NewRelease(configPath string) (*Release, error) {
	var release Release

	if configPath == "" {
		return nil, errors.New("config file required")
	}

	b, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(b, &release); err != nil {
		return nil, err
	}

	if release.Workspace == "" {
		return nil, errors.New("workspace path required")
	}

	if !filepath.IsAbs(release.Workspace) {
		return nil, errors.New("workspace path must be an absolute path")
	}

	githubToken := os.Getenv("GITHUB_TOKEN")
	if githubToken == "" {
		return nil, errors.New("missing GITHUB_TOKEN env var")
	}
	release.GithubToken = githubToken

	if release.K3sRemote == "" {
		release.K3sRemote = rancherRemote
	}

	if release.K3sUpstreamURL == "" {
		release.K3sUpstreamURL = k3sUpstreamRepoURL
	}

	if release.K8sRancherURL == "" {
		release.K8sRancherURL = k8sRancherURL
	}

	return &release, nil
}

// SetupK8sRemotes will clone the kubernetes upstream repo and proceed with setting up remotes
// for rancher and user's forks, then it will fetch branches and tags for all remotes
func (r *Release) SetupK8sRemotes(_ context.Context, ghClient *github.Client) error {
	k8sDir := filepath.Join(r.Workspace, "kubernetes")

	if _, err := os.Stat(r.Workspace); err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(r.Workspace, 0755); err != nil {
				return err
			}
		} else {
			return err
		}
	}

	// clone the repo
	repo, err := git.PlainClone(k8sDir, false, &git.CloneOptions{
		URL:             k8sUpstreamURL,
		Progress:        os.Stdout,
		InsecureSkipTLS: true,
	})
	if err != nil {
		if err == git.ErrRepositoryAlreadyExists {
			repo, err = git.PlainOpen(k8sDir)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}

	gitAuth, err := getAuth(r.SSHKeyPath)
	if err != nil {
		return err
	}

	if err := repo.Fetch(&git.FetchOptions{
		RemoteName:      "origin",
		Progress:        os.Stdout,
		Tags:            git.AllTags,
		InsecureSkipTLS: true,
	}); err != nil {
		if err != git.NoErrAlreadyUpToDate {
			return err
		}
	}

	if _, err := repo.CreateRemote(&config.RemoteConfig{
		Name: r.K3sRemote,
		URLs: []string{r.K8sRancherURL},
	}); err != nil {
		if err != git.ErrRemoteExists {
			return err
		}
	}

	if err := repo.Fetch(&git.FetchOptions{
		RemoteName: r.K3sRemote,
		Progress:   os.Stdout,
		Tags:       git.AllTags,
		Auth:       gitAuth,
	}); err != nil {
		if err != git.NoErrAlreadyUpToDate {
			return err
		}
	}

	userRemoteURL := strings.Replace(k8sUserURL, "user", r.Handler, -1)
	if _, err := repo.CreateRemote(&config.RemoteConfig{
		Name: r.Handler,
		URLs: []string{userRemoteURL},
	}); err != nil {
		if err != git.ErrRemoteExists {
			return err
		}
	}
	if err := repo.Fetch(&git.FetchOptions{
		RemoteName: r.Handler,
		Progress:   os.Stdout,
		Tags:       git.AllTags,
		Auth:       gitAuth,
	}); err != nil {
		if err != git.NoErrAlreadyUpToDate {
			return err
		}
	}

	return nil
}

func (r *Release) RebaseAndTag(_ context.Context, ghClient *github.Client) ([]string, string, error) {
	rebaseOut, err := r.gitRebaseOnto()
	if err != nil {
		return nil, "", err
	}
	wrapperImageTag, err := r.buildGoWrapper()
	if err != nil {
		return nil, "", err
	}

	// setup gitconfig
	gitconfigFile, err := r.setupGitArtifacts()
	if err != nil {
		return nil, "", err
	}
	// make sure that tag doesnt exist first
	tagExists, err := r.isTagExists()
	if err != nil {
		return nil, "", err
	}
	if tagExists {
		if err := r.removeExistingTags(); err != nil {
			return nil, "", err
		}
	}
	out, err := r.runTagScript(gitconfigFile, wrapperImageTag)
	if err != nil {
		return nil, "", err
	}

	tags := tagPushLines(out)
	if len(tags) == 0 {
		return nil, "", errors.New("failed to extract tag push lines")
	}

	return tags, rebaseOut, nil
}

// getAuth is a utility function which is used to get the ssh authentication method for connecting to an ssh server.
// the function takes a single parameter, privateKey, which is a string representing the path to a private key file.
// If the privateKey is an empty string, the function uses the default private key located at $HOME/.ssh/id_rsa.
// The function then creates a new ssh.AuthMethod using the ssh.NewPublicKeysFromFile function, passing in the "git" user, the privateKey path, and an empty password.
// If this returns an error, the function returns nil and the error.
// Finally, the function returns the publicKeys variable, which is now an ssh.AuthMethod, and a nil error.
func getAuth(privateKey string) (ssh.AuthMethod, error) {
	if privateKey == "" {
		privateKey = fmt.Sprintf("%s/.ssh/id_rsa", os.Getenv("HOME"))
	}

	publicKeys, err := ssh.NewPublicKeysFromFile("git", privateKey, "")
	if err != nil {
		return nil, err
	}
	publicKeys.HostKeyCallback = ssh2.InsecureIgnoreHostKey()

	return publicKeys, nil
}

func (r *Release) gitRebaseOnto() (string, error) {
	dir := filepath.Join(r.Workspace, "kubernetes")

	// clean kubernetes directory before rebase
	if err := cleanGitRepo(dir); err != nil {
		return "", err
	}
	if _, err := ecmExec.RunCommand(dir, "rm", "-rf", "_output"); err != nil {
		return "", err
	}

	commandArgs := strings.Split(fmt.Sprintf("rebase --onto %s %s %s-k3s1~1",
		r.NewK8SVersion,
		r.OldK8SVersion,
		r.OldK8SVersion), " ")
	cmd := exec.Command("git", commandArgs...)
	var outb bytes.Buffer
	cmd.Stdout = &outb
	cmd.Stderr = &outb
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		return "", errors.New(err.Error() + ": " + outb.String())
	}

	return outb.String(), nil
}

func (r *Release) goVersion() (string, error) {
	var dep map[string]interface{}

	depFile := filepath.Join(r.Workspace, "kubernetes", "build", "dependencies.yaml")
	dat, err := os.ReadFile(depFile)
	if err != nil {
		return "", err
	}

	if err := yaml.Unmarshal(dat, &dep); err != nil {
		return "", err
	}

	depList := dep["dependencies"].([]interface{})

	for _, v := range depList {
		item := v.(map[interface{}]interface{})
		itemName := item["name"]
		if itemName == "golang: upstream version" {
			version := item["version"]
			return version.(string), nil
		}
	}

	return "", errors.New("can not find Go dependency")
}

func (r *Release) buildGoWrapper() (string, error) {
	goVersion, err := r.goVersion()
	if err != nil {
		return "", err
	}

	goImageVersion := fmt.Sprintf("golang:%s-alpine", goVersion)

	devDockerfile := strings.ReplaceAll(dockerDevImage, "%goimage%", goImageVersion)

	if err := os.WriteFile(filepath.Join(r.Workspace, "dockerfile"), []byte(devDockerfile), 0644); err != nil {
		return "", err
	}

	wrapperImageTag := goImageVersion + "-dev"
	if _, err := ecmExec.RunCommand(r.Workspace, "docker", "build", "-t", wrapperImageTag, "."); err != nil {
		return "", err
	}

	return wrapperImageTag, nil
}

func (r *Release) setupGitArtifacts() (string, error) {
	gitconfigFile := filepath.Join(r.Workspace, ".gitconfig")

	// setting up username and email for tagging purposes
	gitconfigFileContent := strings.ReplaceAll(gitconfig, "%email%", r.Email)
	gitconfigFileContent = strings.ReplaceAll(gitconfigFileContent, "%user%", r.Handler)

	// disable gpg signing direct in .gitconfig
	if strings.Contains(gitconfigFileContent, "[commit]") {
		gitconfigFileContent = strings.Replace(gitconfigFileContent, "gpgsign = true", "gpgsign = false", 1)
	} else {
		gitconfigFileContent += "[commit]\n\tgpgsign = false\n"
	}

	if err := os.WriteFile(gitconfigFile, []byte(gitconfigFileContent), 0644); err != nil {
		return "", err
	}

	return gitconfigFile, nil
}

func (r *Release) runTagScript(gitConfigFile, wrapperImageTag string) (string, error) {
	const containerK8sPath = "/home/go/src/kubernetes"
	const containerGoCachePath = "/home/go/.cache"
	uid := strconv.Itoa(os.Getuid())
	gid := strconv.Itoa(os.Getgid())

	gopath, err := ecmExec.RunCommand(r.Workspace, "go", "env", "GOPATH")
	if err != nil {
		return "", err
	}
	gopath = strings.Trim(gopath, "\n")

	k8sDir := filepath.Join(r.Workspace, "kubernetes")

	// prep the docker run command
	args := []string{
		"run",
		"-u", uid + ":" + gid,
		"-v", gopath + ":/home/go:rw",
		"-v", gitConfigFile + ":/home/go/.gitconfig:rw",
		"-v", k8sDir + ":" + containerK8sPath + ":rw",
		"-v", gopath + "/.cache:" + containerGoCachePath + ":rw",
		"-e", "HOME=/home/go",
		"-e", "GOCACHE=" + containerGoCachePath,
		"-w", containerK8sPath,
		wrapperImageTag,
		"./tag.sh", r.NewK8SVersion + "-k3s1",
	}

	return ecmExec.RunCommand(k8sDir, "docker", args...)
}

func tagPushLines(out string) []string {
	var tagCmds []string

	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "git push $REMOTE") {
			tagCmds = append(tagCmds, line)
		}
	}

	return tagCmds
}

func (r *Release) TagsFromFile(_ context.Context) ([]string, error) {
	var tagCmds []string

	tagFile := filepath.Join(r.Workspace, "tags-"+r.NewK8SVersion)
	if _, err := os.Stat(tagFile); err != nil {
		return nil, err
	}

	dat, err := os.ReadFile(tagFile)
	if err != nil {
		return nil, err
	}

	for _, line := range strings.Split(string(dat), "\n") {
		if strings.Contains(line, "git push $REMOTE") {
			tagCmds = append(tagCmds, line)
		}
	}

	return tagCmds, nil

}

func (r *Release) PushTags(_ context.Context, tagsCmds []string, ghClient *github.Client) error {
	gitConfigFile, err := r.setupGitArtifacts()
	if err != nil {
		return err
	}

	file, err := os.Open(gitConfigFile)
	if err != nil {
		return err
	}
	defer file.Close()

	cfg, err := config.ReadConfig(file)
	if err != nil {
		return err
	}

	repo, err := git.PlainOpen(filepath.Join(r.Workspace, "kubernetes"))
	if err != nil {
		return err
	}

	userRemote, err := repo.Remote(r.Handler)
	if err != nil {
		return err
	}

	originRemote, err := repo.Remote("origin")
	if err != nil {
		return err
	}

	k3sRemote, err := repo.Remote(r.K3sRemote)
	if err != nil {
		return fmt.Errorf("failed to find remote %s: %s", r.K3sRemote, err.Error())
	}

	cfg.Remotes["origin"] = originRemote.Config()
	cfg.Remotes[r.Handler] = userRemote.Config()
	cfg.Remotes[r.K3sRemote] = k3sRemote.Config()

	if err := repo.SetConfig(cfg); err != nil {
		return err
	}

	gitAuth, err := getAuth(r.SSHKeyPath)
	if err != nil {
		return err
	}

	for i, tagCmd := range tagsCmds {
		tagCmdStr := tagCmd
		tag := strings.Split(tagCmdStr, " ")[3]
		logrus.Infof("pushing tag %d/%d: %s", i+1, len(tagsCmds), tag)
		if r.DryRun {
			logrus.Info("Dry run, skipping tag creation")
			continue
		}
		if err := repo.Push(&git.PushOptions{
			RemoteName: r.K3sRemote,
			Auth:       gitAuth,
			Progress:   os.Stdout,
			RefSpecs: []config.RefSpec{
				config.RefSpec("+refs/tags/" + tag + ":refs/tags/" + tag),
			},
		}); err != nil {
			if err != git.NoErrAlreadyUpToDate {
				return errors.New("failed to push tag: " + err.Error())
			}
		}
	}

	return nil
}

func (r *Release) ModifyAndPush(_ context.Context) error {
	if _, err := os.Stat(r.Workspace); err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(r.Workspace, 0755); err != nil {
				return err
			}
		} else {
			return err
		}
	}

	goVersion, err := r.goVersion()
	if err != nil {
		return err
	}
	r.NewGoVersion = goVersion

	logrus.Info("creating modify script")
	modifyScriptPath := filepath.Join(r.Workspace, "modify_script.sh")
	f, err := os.Create(modifyScriptPath)
	if err != nil {
		return err
	}

	if err := os.Chmod(modifyScriptPath, 0755); err != nil {
		return err
	}

	funcMap := template.FuncMap{
		"replaceAll": strings.ReplaceAll,
	}
	tmpl, err := template.New("modify_script.sh").Funcs(funcMap).Parse(modifyScript)
	if err != nil {
		return err
	}

	if err := tmpl.Execute(f, r); err != nil {
		return err
	}

	logrus.Info("running modify script")
	out, err := ecmExec.RunCommand(r.Workspace, "bash", "./modify_script.sh")
	if err != nil {
		return err
	}
	logrus.Info(out)

	return nil
}

func (r *Release) CreatePRFromK3S(ctx context.Context, ghClient *github.Client) error {
	const repo = "k3s"

	pull := &github.NewPullRequest{
		Title:               github.String(fmt.Sprintf("Update to %s-%s", r.NewK8SVersion, r.NewK3SSuffix)),
		Base:                github.String(r.ReleaseBranch),
		Head:                github.String(r.Handler + ":" + r.NewK8SVersion + "-" + r.NewK3SSuffix),
		MaintainerCanModify: github.Bool(true),
	}

	// creating a pr from your fork branch
	_, _, err := ghClient.PullRequests.Create(ctx, r.K3sRemote, repo, pull)

	return err
}

func NewGithubClient(ctx context.Context, token string) (*github.Client, error) {
	if token == "" {
		return nil, errors.New("error: github token required")
	}

	return repository.NewGithub(ctx, token), nil
}

func (r *Release) TagsCreated(_ context.Context) (bool, error) {
	tagFile := filepath.Join(r.Workspace, "tags-"+r.NewK8SVersion)

	if _, err := os.Stat(tagFile); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func (r *Release) isTagExists() (bool, error) {
	dir := filepath.Join(r.Workspace, "kubernetes")

	repo, err := git.PlainOpen(dir)
	if err != nil {
		return false, err
	}

	tag := r.NewK8SVersion + "-" + r.NewK3SSuffix

	if _, err := repo.Tag(tag); err != nil {
		if err == git.ErrTagNotFound {
			return false, nil
		}
		return false, errors.New("invalid tag " + tag + " object: " + err.Error())
	}

	return true, nil
}

func (r *Release) removeExistingTags() error {
	dir := filepath.Join(r.Workspace, "kubernetes")

	repo, err := git.PlainOpen(dir)
	if err != nil {
		return err
	}

	tagsIter, err := repo.Tags()
	if err != nil {
		return err
	}

	if err := tagsIter.ForEach(func(ref *plumbing.Reference) error {
		if strings.Contains(ref.Name().String(), r.NewK8SVersion+"-"+r.NewK3SSuffix) {
			if err := repo.DeleteTag(ref.Name().Short()); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return err
	}

	return nil
}

func cleanGitRepo(dir string) error {
	if _, err := ecmExec.RunCommand(dir, "rm", "-rf", "_output"); err != nil {
		return err
	}

	if _, err := ecmExec.RunCommand(dir, "git", "clean", "-xfd"); err != nil {
		return err
	}

	if _, err := ecmExec.RunCommand(dir, "git", "checkout", "."); err != nil {
		return err
	}

	return nil
}

func (r *Release) CreateRelease(ctx context.Context, client *github.Client, rc bool) error {
	rcNum := 1
	name := r.NewK8SClient + "+" + r.NewK3SSuffix
	oldName := r.OldK8SVersion + "+" + r.OldK8SVersion

	for {
		if rc {
			name = r.NewK8SVersion + "-rc" + strconv.Itoa(rcNum) + "+" + r.NewK3SSuffix
		}

		opts := &repository.CreateReleaseOpts{
			Repo:         k3sRepo,
			Name:         name,
			Owner:        r.K3sRemote,
			Prerelease:   rc,
			Branch:       r.ReleaseBranch,
			Draft:        !rc,
			ReleaseNotes: "",
		}

		if !rc {
			latestRc, err := release.LatestRC(ctx, r.K3sRemote, k3sRepo, r.NewK8SVersion, client)
			if err != nil {
				return err
			}

			buff, err := release.GenReleaseNotes(ctx, r.K3sRemote, k3sRepo, latestRc, oldName, client)
			if err != nil {
				return err
			}
			opts.ReleaseNotes = buff.String()
		}

		if _, err := repository.CreateRelease(ctx, client, opts); err != nil {
			githubErr := err.(*github.ErrorResponse)
			logrus.Printf("error: %+v", githubErr)
			if strings.Contains(githubErr.Errors[0].Code, "already_exists") {
				if !rc {
					return err
				}

				logrus.Printf("RC %d already exists, trying to create next", rcNum)
				rcNum += 1
				continue
			}

			return err
		}

		break
	}

	return nil
}
