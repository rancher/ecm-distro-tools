package k3s

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/google/go-github/v39/github"
	ecmConfig "github.com/rancher/ecm-distro-tools/cmd/release/config"
	ecmExec "github.com/rancher/ecm-distro-tools/exec"
	"github.com/rancher/ecm-distro-tools/release"
	"github.com/rancher/ecm-distro-tools/repository"
	ssh2 "golang.org/x/crypto/ssh"
	"golang.org/x/mod/semver"
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
	updateK3sScriptName       = "update_k3s_references.sh"
	updateK3sReferencesScript = `#!/bin/bash
set -ex
OS=$(uname -s)
DRY_RUN={{ .K3s.DryRun }}
BRANCH_NAME={{ .K3s.NewK8sVersion }}-{{ .K3s.NewSuffix }}
cd {{ .K3s.Workspace }}
# using ls | grep is not a good idea because it doesn't support non-alphanumeric filenames, but since we're only ever checking 'k3s' it isn't a problem https://www.shellcheck.net/wiki/SC2010
ls | grep -w k3s || git clone "git@github.com:{{ .User.GithubUsername }}/k3s.git"
cd {{ .K3s.Workspace }}/k3s
git remote -v | grep -w upstream || git remote add upstream {{ .K3s.K3sUpstreamURL }}
git fetch upstream
git stash
git branch -D "${BRANCH_NAME}" &>/dev/null || true
git checkout -B "${BRANCH_NAME}" upstream/{{.K3s.ReleaseBranch}}
git clean -xfd

case ${OS} in
Darwin)
	sed -Ei '' "\|github.com/k3s-io/kubernetes| s|{{ replaceAll .K3s.OldK8sVersion "." "\\." }}-{{ .K3s.OldSuffix }}|{{ replaceAll .K3s.NewK8sVersion "." "\\." }}-{{ .K3s.NewSuffix }}|" go.mod
	sed -Ei '' "s/k8s.io\/kubernetes v\S+/k8s.io\/kubernetes {{ replaceAll .K3s.NewK8sVersion "." "\\." }}/" go.mod
	sed -Ei '' "s/{{ replaceAll .K3s.OldK8sClient "." "\\." }}/{{ replaceAll .K3s.NewK8sClient "." "\\." }}/g" go.mod # This should only change ~6 lines in go.mod
	sed -Ei '' "s/golang:.*-/golang:{{ .K3s.NewGoVersion }}-/g" Dockerfile.*
	sed -Ei '' "s/go-version:.*$/go-version:\ '{{ .K3s.NewGoVersion }}'/g" .github/workflows/integration.yaml .github/workflows/unitcoverage.yaml
	;;
Linux)
	sed -Ei "\|github.com/k3s-io/kubernetes| s|{{ replaceAll .K3s.OldK8sVersion "." "\\." }}-{{ .K3s.OldSuffix }}|{{ replaceAll .K3s.NewK8sVersion "." "\\." }}-{{ .K3s.NewSuffix }}|" go.mod
	sed -Ei "s/k8s.io\/kubernetes v\S+/k8s.io\/kubernetes {{ replaceAll .K3s.NewK8sVersion "." "\\." }}/" go.mod
	sed -Ei "s/{{ replaceAll .K3s.OldK8sClient "." "\\." }}/{{ replaceAll .K3s.NewK8sClient "." "\\." }}/g" go.mod # This should only change ~6 lines in go.mod
	sed -Ei "s/golang:.*-/golang:{{ .K3s.NewGoVersion }}-/g" Dockerfile.*
	sed -Ei "s/go-version:.*$/go-version:\ '{{ .K3s.NewGoVersion }}'/g" .github/workflows/integration.yaml .github/workflows/unitcoverage.yaml
	;;
*)
	>&2 echo "$(OS) not supported yet"
	exit 1
	;;
esac

go mod tidy

git add go.mod go.sum Dockerfile.* .github/workflows/integration.yaml .github/workflows/unitcoverage.yaml
	git commit --signoff -m "Update to {{ .K3s.NewK8sVersion }}"
if [ "${DRY_RUN}" = false ]; then
	git push --set-upstream origin "${BRANCH_NAME}" # run git remote -v for your origin
fi`
)

type UpdateScriptVars struct {
	K3s  *ecmConfig.K3sRelease
	User *ecmConfig.User
}

// GenerateTags will clone the kubernetes repository, rebase it with the k3s-io fork and
// generate tags to be pushed
func GenerateTags(ctx context.Context, ghClient *github.Client, r *ecmConfig.K3sRelease, u *ecmConfig.User, sshKeyPath string) error {
	fmt.Println("setting up k8s remotes")
	if err := setupK8sRemotes(ghClient, r, u, sshKeyPath); err != nil {
		return errors.New("failed to clone and setup remotes for k8s repos: " + err.Error())
	}
	tagsExists, err := tagsFileExists(r)
	if err != nil {
		return errors.New("failed to verify if tags file already exists: " + err.Error())
	}
	if tagsExists {
		return errors.New("tag file already exists, skipping rebase and tag")
	}
	fmt.Println("rebasing and tagging")
	tags, err := rebaseAndTag(ghClient, r, u)
	if err != nil {
		return errors.New("failed to rebase and tag: " + err.Error())
	}
	fmt.Println("successfully rebased and tagged")
	return writeTagsFile(r, tags)
}

func writeTagsFile(r *ecmConfig.K3sRelease, tags []string) error {
	tagFile := filepath.Join(r.Workspace, "tags-"+r.NewK8sVersion)
	return os.WriteFile(tagFile, []byte(strings.Join(tags, "\n")), 0644)
}

// setupK8sRemotes will clone the kubernetes upstream repo and proceed with setting up remotes
// for rancher and user's forks, then it will fetch branches and tags for all remotes
func setupK8sRemotes(ghClient *github.Client, r *ecmConfig.K3sRelease, u *ecmConfig.User, sshKeyPath string) error {
	k8sDir := filepath.Join(r.Workspace, "kubernetes")

	fmt.Println("verifying if the k8s dir already exists: " + k8sDir)
	if _, err := os.Stat(r.Workspace); err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		fmt.Println("dir doesn't exists, creating")
		if err := os.MkdirAll(r.Workspace, 0755); err != nil {
			return err
		}
	}

	// clone the repo
	fmt.Println("cloning the repo")
	repo, err := git.PlainClone(k8sDir, false, &git.CloneOptions{
		URL:             k8sUpstreamURL,
		Progress:        os.Stdout,
		InsecureSkipTLS: true,
	})
	if err != nil {
		if err != git.ErrRepositoryAlreadyExists {
			return err
		}
		fmt.Println("repo already exists, opening it")
		repo, err = git.PlainOpen(k8sDir)
		if err != nil {
			return err
		}
	}

	fmt.Println("getting ssh auth")
	gitAuth, err := getAuth(sshKeyPath)
	if err != nil {
		return err
	}

	fmt.Println("fetching remote: origin")
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

	fmt.Println("creating remote: '" + r.K3sRepoOwner + " " + r.K8sRancherURL + "'")
	if _, err := repo.CreateRemote(&config.RemoteConfig{
		Name: r.K3sRepoOwner,
		URLs: []string{r.K8sRancherURL},
	}); err != nil {
		if err != git.ErrRemoteExists {
			return err
		}
	}

	fmt.Println("fetching remote: " + r.K3sRepoOwner)
	if err := repo.Fetch(&git.FetchOptions{
		RemoteName: r.K3sRepoOwner,
		Progress:   os.Stdout,
		Tags:       git.AllTags,
		Auth:       gitAuth,
	}); err != nil {
		if err != git.NoErrAlreadyUpToDate {
			return err
		}
	}

	userRemoteURL := strings.Replace(k8sUserURL, "user", u.GithubUsername, -1)
	fmt.Println("creating remote: '" + u.GithubUsername + " " + userRemoteURL + "'")
	if _, err := repo.CreateRemote(&config.RemoteConfig{
		Name: u.GithubUsername,
		URLs: []string{userRemoteURL},
	}); err != nil {
		if err != git.ErrRemoteExists {
			return err
		}
	}
	fmt.Println("fetching remote: " + u.GithubUsername)
	if err := repo.Fetch(&git.FetchOptions{
		RemoteName: u.GithubUsername,
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

func rebaseAndTag(ghClient *github.Client, r *ecmConfig.K3sRelease, u *ecmConfig.User) ([]string, error) {
	rebaseOut, err := gitRebaseOnto(r)
	if err != nil {
		return nil, err
	}
	fmt.Println(rebaseOut)
	wrapperImageTag, err := buildGoWrapper(r)
	if err != nil {
		return nil, err
	}

	// setup gitconfig
	gitconfigFile, err := setupGitArtifacts(r, u)
	if err != nil {
		return nil, err
	}
	// make sure that tag doesnt exist first
	tagExists, err := isTagExists(r)
	if err != nil {
		return nil, err
	}
	if tagExists {
		fmt.Println("tag exists, removing it")
		if err := removeExistingTags(r); err != nil {
			return nil, err
		}
	}
	out, err := runTagScript(r, gitconfigFile, wrapperImageTag)
	if err != nil {
		return nil, err
	}

	tags := tagPushLines(out)
	if len(tags) == 0 {
		return nil, errors.New("failed to extract tag push lines")
	}

	return tags, nil
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

func gitRebaseOnto(r *ecmConfig.K3sRelease) (string, error) {
	dir := filepath.Join(r.Workspace, "kubernetes")

	// clean kubernetes directory before rebase
	fmt.Println("cleaning git repo: " + dir)
	if err := cleanGitRepo(dir); err != nil {
		return "", err
	}

	commandArgs := strings.Split(fmt.Sprintf("rebase --onto %s %s %s-k3s1~1",
		r.NewK8sVersion,
		r.OldK8sVersion,
		r.OldK8sVersion), " ")
	fmt.Println("git ", commandArgs)
	return ecmExec.RunCommand(dir, "git", commandArgs...)
}

func goVersion(r *ecmConfig.K3sRelease) (string, error) {
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

func buildGoWrapper(r *ecmConfig.K3sRelease) (string, error) {
	fmt.Println("getting go version for k8s")
	goVersion, err := goVersion(r)
	if err != nil {
		return "", err
	}

	goImageVersion := fmt.Sprintf("golang:%s-alpine", goVersion)
	fmt.Println("go image version: " + goImageVersion)

	devDockerfile := strings.ReplaceAll(dockerDevImage, "%goimage%", goImageVersion)

	fmt.Println("writing dockerfile")
	if err := os.WriteFile(filepath.Join(r.Workspace, "dockerfile"), []byte(devDockerfile), 0644); err != nil {
		return "", err
	}

	wrapperImageTag := goImageVersion + "-dev"
	fmt.Println("building docker image")
	if _, err := ecmExec.RunCommand(r.Workspace, "docker", "build", "-t", wrapperImageTag, "."); err != nil {
		return "", err
	}

	return wrapperImageTag, nil
}

func setupGitArtifacts(r *ecmConfig.K3sRelease, u *ecmConfig.User) (string, error) {
	gitconfigFile := filepath.Join(r.Workspace, ".gitconfig")

	// setting up username and email for tagging purposes
	fmt.Println("updating git config contents")
	gitconfigFileContent := strings.ReplaceAll(gitconfig, "%email%", u.Email)
	gitconfigFileContent = strings.ReplaceAll(gitconfigFileContent, "%user%", u.GithubUsername)

	// disable gpg signing direct in .gitconfig
	fmt.Println("disabling gpg signing")
	if strings.Contains(gitconfigFileContent, "[commit]") {
		gitconfigFileContent = strings.Replace(gitconfigFileContent, "gpgsign = true", "gpgsign = false", 1)
	} else {
		gitconfigFileContent += "[commit]\n\tgpgsign = false\n"
	}

	fmt.Println("writing .gitconfig at: " + gitconfigFile)
	if err := os.WriteFile(gitconfigFile, []byte(gitconfigFileContent), 0644); err != nil {
		return "", err
	}

	return gitconfigFile, nil
}

func runTagScript(r *ecmConfig.K3sRelease, gitConfigFile, wrapperImageTag string) (string, error) {
	const containerK8sPath = "/home/go/src/kubernetes"
	const containerGoCachePath = "/home/go/.cache"
	uid := strconv.Itoa(os.Getuid())
	gid := strconv.Itoa(os.Getgid())

	gopath, err := ecmExec.RunCommand(r.Workspace, "go", "env", "GOPATH")
	if err != nil {
		return "", err
	}
	gopath = strings.Trim(gopath, "\n")
	fmt.Println("gopath: " + gopath)

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
		"./tag.sh", r.NewK8sVersion + "-k3s1",
	}

	fmt.Println("running tag script")
	return ecmExec.RunCommand(k8sDir, "docker", args...)
}

func tagPushLines(out string) []string {
	var tagCmds []string

	fmt.Println("getting git push lines")
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "git push $REMOTE") {
			tagCmds = append(tagCmds, line)
		}
	}

	return tagCmds
}

func tagsCmdsFromFile(r *ecmConfig.K3sRelease) ([]string, error) {
	var tagCmds []string

	tagFile := filepath.Join(r.Workspace, "tags-"+r.NewK8sVersion)
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

func PushTags(ghClient *github.Client, r *ecmConfig.K3sRelease, u *ecmConfig.User, sshKeyPath string) error {
	tagsCmds, err := tagsCmdsFromFile(r)
	if err != nil {
		return errors.New("failed to extract tags from file: " + err.Error())
	}
	fmt.Println("setting up git artifacts")
	gitConfigFile, err := setupGitArtifacts(r, u)
	if err != nil {
		return err
	}

	fmt.Println("opening git config file")
	file, err := os.Open(gitConfigFile)
	if err != nil {
		return err
	}
	defer file.Close()

	fmt.Println("reading git config")
	cfg, err := config.ReadConfig(file)
	if err != nil {
		return err
	}

	fmt.Println("opening kubernetes repo")
	repo, err := git.PlainOpen(filepath.Join(r.Workspace, "kubernetes"))
	if err != nil {
		return err
	}

	fmt.Println("getting remote: " + u.GithubUsername)
	userRemote, err := repo.Remote(u.GithubUsername)
	if err != nil {
		return err
	}

	fmt.Println("getting remote: origin")
	originRemote, err := repo.Remote("origin")
	if err != nil {
		return err
	}

	fmt.Println("getting remote: " + r.K3sRepoOwner)
	k3sRemote, err := repo.Remote(r.K3sRepoOwner)
	if err != nil {
		return errors.New("failed to find remote: '" + r.K3sRepoOwner + "' " + err.Error())
	}

	cfg.Remotes["origin"] = originRemote.Config()
	cfg.Remotes[u.GithubUsername] = userRemote.Config()
	cfg.Remotes[r.K3sRepoOwner] = k3sRemote.Config()

	fmt.Println("setting remotes in the config")
	if err := repo.SetConfig(cfg); err != nil {
		return err
	}

	fmt.Println("getting ssh key auth")
	gitAuth, err := getAuth(sshKeyPath)
	if err != nil {
		return err
	}

	fmt.Println("pushing tags")
	for i, tagCmd := range tagsCmds {
		tagCmdStr := tagCmd
		tag := strings.Split(tagCmdStr, " ")[3]
		fmt.Printf("pushing tag %d/%d: %s", i+1, len(tagsCmds), tag)
		if r.DryRun {
			fmt.Println("Dry run, skipping tag creation")
			continue
		}
		if err := repo.Push(&git.PushOptions{
			RemoteName: r.K3sRepoOwner,
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

func UpdateK3sReferences(ctx context.Context, ghClient *github.Client, r *ecmConfig.K3sRelease, u *ecmConfig.User) error {
	if err := updateK3sReferencesAndPush(r, u); err != nil {
		return err
	}
	if r.DryRun {
		fmt.Println("dry run, skipping creating PR")
		return nil
	}
	return createK3sReferencesPR(ctx, ghClient, r, u)
}

func updateK3sReferencesAndPush(r *ecmConfig.K3sRelease, u *ecmConfig.User) error {
	fmt.Println("verifying if workspace dir exists")
	if _, err := os.Stat(r.Workspace); err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		fmt.Println("workspace dir doesn't exists, creating it")
		if err := os.MkdirAll(r.Workspace, 0755); err != nil {
			return err
		}
	}

	fmt.Println("getting k8s go version")
	goVersion, err := goVersion(r)
	if err != nil {
		return err
	}
	r.NewGoVersion = goVersion

	funcMap := template.FuncMap{"replaceAll": strings.ReplaceAll}
	fmt.Println("creating update k3s references script template")
	scriptVars := UpdateScriptVars{K3s: r, User: u}
	updateScriptOut, err := ecmExec.RunTemplatedScript(r.Workspace, updateK3sScriptName, updateK3sReferencesScript, funcMap, scriptVars)
	if err != nil {
		return err
	}
	fmt.Println(updateScriptOut)
	return nil
}

func createK3sReferencesPR(ctx context.Context, ghClient *github.Client, r *ecmConfig.K3sRelease, u *ecmConfig.User) error {
	const repo = "k3s"

	pull := &github.NewPullRequest{
		Title:               github.String(fmt.Sprintf("Update to %s-%s and Go %s", r.NewK8sVersion, r.NewSuffix, r.NewGoVersion)),
		Base:                github.String(r.ReleaseBranch),
		Head:                github.String(u.GithubUsername + ":" + r.NewK8sVersion + "-" + r.NewSuffix),
		MaintainerCanModify: github.Bool(true),
	}

	// creating a pr from your fork branch
	_, _, err := ghClient.PullRequests.Create(ctx, r.K3sRepoOwner, repo, pull)

	return err
}

func NewGithubClient(ctx context.Context, token string) (*github.Client, error) {
	if token == "" {
		return nil, errors.New("error: github token required")
	}

	return repository.NewGithub(ctx, token), nil
}

// tagsFileExists verify if there is a tags file at the release workspace
func tagsFileExists(r *ecmConfig.K3sRelease) (bool, error) {
	tagFile := filepath.Join(r.Workspace, "tags-"+r.NewK8sVersion)
	fmt.Println("verifying if tags file exists at: " + tagFile)
	if _, err := os.Stat(tagFile); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func isTagExists(r *ecmConfig.K3sRelease) (bool, error) {
	dir := filepath.Join(r.Workspace, "kubernetes")

	fmt.Println("opening k8s repo: " + dir)
	repo, err := git.PlainOpen(dir)
	if err != nil {
		return false, err
	}

	tag := r.NewK8sVersion + "-" + r.NewSuffix

	fmt.Println("verifying if tag exists: " + tag)
	if _, err := repo.Tag(tag); err != nil {
		if err == git.ErrTagNotFound {
			return false, nil
		}
		return false, errors.New("invalid tag " + tag + " object: " + err.Error())
	}

	return true, nil
}

func removeExistingTags(r *ecmConfig.K3sRelease) error {
	dir := filepath.Join(r.Workspace, "kubernetes")

	fmt.Println("opening k8s repo: " + dir)
	repo, err := git.PlainOpen(dir)
	if err != nil {
		return err
	}

	fmt.Println("getting repo tags")
	tagsIter, err := repo.Tags()
	if err != nil {
		return err
	}

	if err := tagsIter.ForEach(func(ref *plumbing.Reference) error {
		if strings.Contains(ref.Name().String(), r.NewK8sVersion+"-"+r.NewSuffix) {
			tagRefName := ref.Name().Short()
			fmt.Println("tag ref found, deleting it: " + tagRefName)
			if err := repo.DeleteTag(tagRefName); err != nil {
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
	fmt.Println("cleaning _output")
	if _, err := ecmExec.RunCommand(dir, "rm", "-rf", "_output"); err != nil {
		return err
	}

	fmt.Println("removing unwanted files")
	if _, err := ecmExec.RunCommand(dir, "git", "clean", "-xfd"); err != nil {
		return err
	}

	fmt.Println("git checkout .")
	if _, err := ecmExec.RunCommand(dir, "git", "checkout", "."); err != nil {
		return err
	}

	return nil
}

func CreateRelease(ctx context.Context, client *github.Client, r *ecmConfig.K3sRelease, tag string, rc bool) error {
	fmt.Println("validating tag")
	if !semver.IsValid(tag) {
		return errors.New("tag isn't a valid semver: " + tag)
	}
	var createdReleaseURL string
	rcNum := 1
	name := r.NewK8sVersion + "+" + r.NewSuffix
	oldName := r.OldK8sVersion + "+" + r.OldSuffix

	for {
		if rc {
			name = r.NewK8sVersion + "-rc" + strconv.Itoa(rcNum) + "+" + r.NewSuffix
		}

		opts := &repository.CreateReleaseOpts{
			Repo:         k3sRepo,
			Name:         name,
			Owner:        r.K3sRepoOwner,
			Prerelease:   true,
			Branch:       r.ReleaseBranch,
			Draft:        !rc,
			ReleaseNotes: "",
		}

		fmt.Printf("create release options: %+v\n", *opts)

		if r.DryRun {
			fmt.Println("dry run, skipping creating release and verifying if rcs already were created")
			break
		}

		if !rc {
			latestRc, err := release.LatestRC(ctx, r.K3sRepoOwner, k3sRepo, r.NewK8sVersion, client)
			if err != nil {
				return err
			}

			buff, err := release.GenReleaseNotes(ctx, r.K3sRepoOwner, k3sRepo, latestRc, oldName, client)
			if err != nil {
				return err
			}
			opts.ReleaseNotes = buff.String()
		}

		createdRelease, err := repository.CreateRelease(ctx, client, opts)
		if err != nil {
			githubErr := err.(*github.ErrorResponse)
			if len(githubErr.Errors) > 0 {
				if strings.Contains(githubErr.Errors[0].Code, "already_exists") {
					if !rc {
						return err
					}

					fmt.Println("RC " + strconv.Itoa(rcNum) + " already exists, trying to create next")
					rcNum += 1
					continue
				}
			}

			return err
		}

		createdReleaseURL = *createdRelease.HTMLURL

		break
	}
	fmt.Println("release created: " + createdReleaseURL)
	return nil
}
