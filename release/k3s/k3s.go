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
	"github.com/rancher/ecm-distro-tools/repository"
	ssh2 "golang.org/x/crypto/ssh"
	"gopkg.in/yaml.v2"
)

const (
	k8sUpstreamURL     = "https://github.com/kubernetes/kubernetes"
	rancherRemote      = "k3s-io"
	k8sRancherURL      = "git@github.com:k3s-io/kubernetes.git"
	k8sUserURL         = "git@github.com:user/kubernetes.git"
	k3sUpstreamRepoURL = "https://github.com/k3s-io/k3s"
	gitconfig          = `
	[safe]
	directory = /home/go/src/kubernetes
	[user]
	email = %email%
	name = %user%
	`
	dockerDevImage = `
	FROM %goimage%
	RUN apk add --no-cache bash git make tar gzip curl git coreutils rsync alpine-sdk
	`

	modifyScript = `
		#!/bin/bash
		cd {{ .Workspace }}
		git clone "git@github.com:{{ .Handler }}/k3s.git"
		cd {{ .Workspace }}/k3s
		git remote add upstream https://github.com/k3s-io/k3s.git
		git fetch upstream
		git branch delete {{ .NewK8SVersion }}-{{ .NewK3SVersion }}
		git checkout -B {{ .NewK8SVersion }}-{{ .NewK3SVersion }} upstream/{{.ReleaseBranch}}
		git clean -xfd
		
		sed -Ei "\|github.com/k3s-io/kubernetes| s|{{ .OldK8SVersion }}-{{ .OldK3SVersion }}|{{ .NewK8SVersion }}-{{ .NewK3SVersion }}|" go.mod
		sed -Ei "s/k8s.io\/kubernetes v\S+/k8s.io\/kubernetes {{ .NewK8SVersion }}/" go.mod
		sed -Ei "s/{{ .OldK8SClient }}/{{ .NewK8SClient }}/g" go.mod # This should only change ~6 lines in go.mod
		
		go mod tidy
		# There is no need for running make since the changes will be only for go.mod
		# mkdir -p build/data && DRONE_TAG={{ .NewK8SVersion }}-{{ .NewK3SVersion }} make download && make generate
	
		git add go.mod go.sum
		git commit --all --signoff -m "Update to {{ .NewK8SVersion }}"
		git push --set-upstream origin {{ .NewK8SVersion }}-{{ .NewK3SVersion }} # run git remote -v for your origin
		`
)

type Release struct {
	OldK8SVersion string `json:"old_k8s_version"`
	NewK8SVersion string `json:"new_k8s_version"`
	OldK8SClient  string `json:"old_k8s_client"`
	NewK8SClient  string `json:"new_k8s_client"`
	OldK3SVersion string `json:"old_k3s_version"`
	NewK3SVersion string `json:"new_k3s_version"`
	ReleaseBranch string `json:"release_branch"`
	Workspace     string `json:"workspace"`
	Handler       string `json:"handler"`
	Email         string `json:"email"`
	Token         string `json:"token"`
}

func NewReleaseFromConfig(configPath string) (*Release, error) {

	var release *Release
	if configPath == "" {
		return nil, errors.New("error: config file required")
	}

	configData, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(configData, &release); err != nil {
		return nil, err
	}
	return release, nil
}

// SetupK8sRemotes will clone the kubernetes upstream repo and proceed with setting up remotes
// for rancher and user's forks, then it will fetch branches and tags for all remotes
func (r *Release) SetupK8sRemotes(ctx context.Context, ghClient *github.Client) error {
	k3sDir := filepath.Join(r.Workspace, "kubernetes")
	_, err := os.Stat(r.Workspace)
	if err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(r.Workspace, 0755); err != nil {
				return err
			}
		} else {
			return err
		}
	}
	// clone the repo
	repo, err := git.PlainClone(k3sDir, false, &git.CloneOptions{
		URL:             k8sUpstreamURL,
		Progress:        os.Stdout,
		InsecureSkipTLS: true,
	})
	if err != nil {
		if err == git.ErrRepositoryAlreadyExists {
			repo, err = git.PlainOpen(k3sDir)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}
	gitAuth, err := getAuth("")
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
		Name: rancherRemote,
		URLs: []string{k8sRancherURL},
	}); err != nil {
		if err != git.ErrRemoteExists {
			return err
		}
	}
	if err := repo.Fetch(&git.FetchOptions{
		RemoteName: rancherRemote,
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

func (r *Release) RebaseAndTag(ctx context.Context, ghClient *github.Client) ([]string, error) {
	if err := r.gitRebaseOnto(); err != nil {
		return nil, err
	}
	wrapperImageTag, err := r.buildGoWrapper()
	if err != nil {
		return nil, err
	}

	// setup gitconfig
	gitconfigFile, err := r.setupGitArtifacts()
	if err != nil {
		return nil, err
	}
	// make sure that tag doesnt exist first
	tagExists, err := r.isTagExists()
	if err != nil {
		return nil, err
	}
	if tagExists {
		if err := r.removeExistingTags(); err != nil {
			return nil, err
		}
	}
	out, err := r.runTagScript(gitconfigFile, wrapperImageTag)
	if err != nil {
		return nil, err
	}

	tags := tagPushLines(out)
	if len(tags) == 0 {
		return nil, errors.New("failed to extract tag push lines")
	}
	return tags, nil
}

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

func (r *Release) gitRebaseOnto() error {
	dir := filepath.Join(r.Workspace, "kubernetes")

	// clean kubernetes directory before rebase
	if err := cleanGitRepo(dir); err != nil {
		return err
	}
	if _, err := runCommand(dir, "rm", "-rf", "_output"); err != nil {
		return err
	}

	commandArgs := strings.Split(fmt.Sprintf("rebase --onto %s %s %s-k3s1~1",
		r.NewK8SVersion,
		r.OldK8SVersion,
		r.OldK8SVersion), " ")
	cmd := exec.Command("git", commandArgs...)
	var outb, errb bytes.Buffer
	cmd.Stdout = &outb
	cmd.Stderr = &errb
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		return errors.New(err.Error() + ": " + errb.String())
	}

	return nil
}

func (r *Release) goVersion() (string, error) {
	var dep map[string]interface{}
	depFile := filepath.Join(r.Workspace, "kubernetes", "build", "dependencies.yaml")
	dat, err := os.ReadFile(depFile)
	if err != nil {
		return "", err
	}

	err = yaml.Unmarshal(dat, &dep)
	if err != nil {
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
	return "", errors.New("can not find go dependency")
}

func runCommand(dir, cmd string, args ...string) (string, error) {
	command := exec.Command(cmd, args...)
	var outb, errb bytes.Buffer
	command.Stdout = &outb
	command.Stderr = &errb
	command.Dir = dir
	err := command.Run()
	if err != nil {
		return "", errors.New(errb.String())
	}
	return outb.String(), nil
}

func (r *Release) buildGoWrapper() (string, error) {
	goVersion, err := r.goVersion()
	if err != nil {
		return "", err
	}
	goImageVersion := fmt.Sprintf("golang:%s-alpine3.15", goVersion)
	devDockerfile := strings.ReplaceAll(dockerDevImage, "%goimage%", goImageVersion)
	if err := os.WriteFile(filepath.Join(r.Workspace, "dockerfile"), []byte(devDockerfile), 0644); err != nil {
		return "", err
	}
	wrapperImageTag := goImageVersion + "-dev"
	if _, err := runCommand(r.Workspace, "docker", "build", "-t", wrapperImageTag, "."); err != nil {
		return "", err
	}

	return wrapperImageTag, nil
}

func (r *Release) setupGitArtifacts() (string, error) {
	gitconfigFile := filepath.Join(r.Workspace, ".gitconfig")

	// setting up username and email for tagging purposes
	gitconfigFileContent := strings.ReplaceAll(gitconfig, "%email%", r.Email)
	gitconfigFileContent = strings.ReplaceAll(gitconfigFileContent, "%user%", r.Handler)

	if err := os.WriteFile(gitconfigFile, []byte(gitconfigFileContent), 0644); err != nil {
		return "", err
	}
	return gitconfigFile, nil
}

func (r *Release) runTagScript(gitConfigFile, wrapperImageTag string) (string, error) {
	uid := strconv.Itoa(os.Getuid())
	gid := strconv.Itoa(os.Getgid())
	gopath, err := runCommand(r.Workspace, "go", "env", "GOPATH")
	if err != nil {
		return "", err
	}
	k8sDir := filepath.Join(r.Workspace, "kubernetes")
	// prep the docker run command
	goWrapper := []string{
		"run",
		"-u",
		uid + ":" + gid,
		"-v",
		gopath + ":/home/go",
		"-v",
		gitConfigFile + ":/home/go/.gitconfig",
		"-v",
		k8sDir + ":/home/go/src/kubernetes",
		"-v",
		gopath + "/.cache:/home/go/.cache",
		"-e",
		"HOME=/home/go",
		"-e",
		"GOCACHE=/home/go/src/kubernetes/.cache",
		"-w",
		"/home/go/src/kubernetes",
		wrapperImageTag,
	}

	args := append(goWrapper, "./tag.sh", r.NewK8SVersion+"-k3s1")
	return runCommand(k8sDir, "docker", args...)
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

func (r *Release) TagsFromFile(ctx context.Context) ([]string, error) {
	tagCmds := []string{}
	tagFile := filepath.Join(r.Workspace, "tags-"+r.NewK8SVersion)
	_, err := os.Stat(tagFile)
	if err != nil {
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

func (r *Release) PushTags(ctx context.Context, tagsCmds []string, ghClient *github.Client, remote string) error {
	// here we can use go-git library or runCommand function
	// I am using go-git library to enhance code quality
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
	k3sRemote, err := repo.Remote("k3s-io")
	if err != nil {
		return err
	}
	cfg.Remotes[r.Handler] = userRemote.Config()
	cfg.Remotes["k3s-io"] = k3sRemote.Config()

	if err := repo.SetConfig(cfg); err != nil {
		return err
	}
	gitAuth, err := getAuth("")
	if err != nil {
		return err
	}
	for _, tagCmd := range tagsCmds {
		tag := strings.Split(tagCmd, " ")[3]

		if err := repo.Push(&git.PushOptions{
			RemoteName: remote,
			Auth:       gitAuth,
			Progress:   os.Stdout,
			RefSpecs: []config.RefSpec{
				config.RefSpec("+refs/tags/" + tag + ":refs/tags/" + tag),
			},
		}); err != nil {
			if err == git.NoErrAlreadyUpToDate {
				continue
			}
			return err
		}
	}

	return nil
}

func (r *Release) ModifyAndPush(ctx context.Context) error {
	_, err := os.Stat(r.Workspace)
	if err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(r.Workspace, 0755); err != nil {
				return err
			}
		} else {
			return err
		}
	}

	modifyScriptPath := filepath.Join(r.Workspace, "modify_script.sh")
	f, err := os.Create(modifyScriptPath)
	if err != nil {
		return err
	}

	if err := os.Chmod(modifyScriptPath, 0755); err != nil {
		return err
	}

	tmpl, err := template.New("modify_script.sh").Parse(modifyScript)
	if err != nil {
		return err
	}

	if err := tmpl.Execute(f, r); err != nil {
		return err
	}

	if _, err := runCommand(r.Workspace, "bash", "./modify_script.sh"); err != nil {
		return err
	}
	return nil
}

func (r *Release) CreatePRFromK3S(ctx context.Context, ghClient *github.Client) error {
	repo := "k3s"
	pull := &github.NewPullRequest{
		Title:               github.String(fmt.Sprintf("Update to %s-%s", r.NewK8SVersion, r.NewK3SVersion)),
		Base:                github.String(r.ReleaseBranch),
		Head:                github.String(r.Handler + ":" + r.NewK8SVersion + "-" + r.NewK3SVersion),
		MaintainerCanModify: github.Bool(true),
	}

	// creating a pr from your fork branch
	_, _, err := ghClient.PullRequests.Create(ctx, "k3s-io", repo, pull)
	return err
}

func NewGithubClient(ctx context.Context, token string) (*github.Client, error) {
	if token == "" {
		return nil, errors.New("error: github token required")
	}
	client := repository.NewGithub(ctx, token)
	return client, nil
}

func (r *Release) TagsCreated(ctx context.Context) (bool, error) {
	tagFile := filepath.Join(r.Workspace, "tags-"+r.NewK8SVersion)
	_, err := os.Stat(tagFile)
	if err != nil {
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
	tag := r.NewK8SVersion + "-" + r.NewK3SVersion
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
		if strings.Contains(ref.Name().String(), r.NewK8SVersion+"-"+r.NewK3SVersion) {
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
	if _, err := runCommand(dir, "rm", "-rf", "_output"); err != nil {
		return err
	}
	if _, err := runCommand(dir, "git", "clean", "-xfd"); err != nil {
		return err
	}
	if _, err := runCommand(dir, "git", "checkout", "."); err != nil {
		return err
	}
	return nil
}
