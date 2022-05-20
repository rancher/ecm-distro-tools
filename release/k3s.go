package release

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/google/go-github/v39/github"
	"github.com/sirupsen/logrus"
	ssh2 "golang.org/x/crypto/ssh"
	"gopkg.in/yaml.v2"
)

const (
	k8sUpstreamURL = "https://github.com/kubernetes/kubernetes"
	rancherRemote  = "k3s-io"
	k8sRancherURL  = "git@github.com:rancher/kubernetes.git"
	k8sUserURL     = "git@github.com:user/kubernetes.git"
	gitconfig      = `
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
)

type K8STagsOptions struct {
	OldK8SVersion  string
	NewK8SVersion  string
	OldK8SClient   string
	NewK8SClient   string
	OldK3SVersion  string
	NewK3SVersion  string
	ReleaseBranche string
}

// SetupK8sRemotes will clone the kubernetes upstream repo and proceed with setting up remotes
// for rancher and user's forks, then it will fetch branches and tags for all remotes
func SetupK8sRemotes(ctx context.Context, ghClient *github.Client, ghUser, workspace string) error {
	_, err := os.Stat(workspace)
	if err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(workspace, 0755); err != nil {
				return err
			}
		} else {
			return err
		}
	}
	// clone the repo
	repo, err := git.PlainClone(filepath.Join(workspace, "kubernetes"), false, &git.CloneOptions{
		URL:             k8sUpstreamURL,
		Progress:        os.Stdout,
		InsecureSkipTLS: true,
	})

	if err != nil {
		if err == git.ErrRepositoryAlreadyExists {
			logrus.Warnf("Repository already exists")
			repo, err = git.PlainOpen(filepath.Join(workspace, "kubernetes"))
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}
	_, err = repo.CreateRemote(&config.RemoteConfig{
		Name: rancherRemote,
		URLs: []string{k8sRancherURL},
	})
	if err != nil {
		if err != git.ErrRemoteExists {
			return err
		}
	}
	logrus.Infof("Remote %s created for url %s, fetching tags", rancherRemote, k8sRancherURL)
	gitAuth, err := getAuth("")
	if err != nil {
		return err
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
	userRemoteURL := strings.Replace(k8sUserURL, "user", ghUser, -1)
	_, err = repo.CreateRemote(&config.RemoteConfig{
		Name: ghUser,
		URLs: []string{userRemoteURL},
	})
	if err != nil {
		if err != git.ErrRemoteExists {
			return err
		}
	}
	logrus.Infof("Remote %s created for url %s, fetching tags", ghUser, userRemoteURL)
	if err := repo.Fetch(&git.FetchOptions{
		RemoteName: ghUser,
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

func RebaseAndTag(ctx context.Context, ghClient *github.Client, ghUser, ghEmail, workspace string, options K8STagsOptions) ([]string, error) {
	if err := gitRebaseOnto(filepath.Join(workspace, "kubernetes"), options); err != nil {
		return nil, err
	}
	goVersion, err := golangVersion(workspace)
	if err != nil {
		return nil, err
	}
	// setup gitconfig
	gitconfigFile := filepath.Join(workspace, ".gitconfig")
	gitconfigFileContent := strings.ReplaceAll(gitconfig, "%email%", ghEmail)
	gitconfigFileContent = strings.ReplaceAll(gitconfigFileContent, "%user%", ghUser)
	if err := os.WriteFile(gitconfigFile, []byte(gitconfigFileContent), 0644); err != nil {
		return nil, err
	}
	dir := filepath.Join(workspace, "kubernetes")
	goImageVersion := fmt.Sprintf("golang:%s-alpine3.15", goVersion)
	devDockerfile := strings.ReplaceAll(dockerDevImage, "%goimage%", goImageVersion)
	if err := os.WriteFile(filepath.Join(workspace, "dockerfile"), []byte(devDockerfile), 0644); err != nil {
		return nil, err
	}
	out, err := runCommand(workspace, "docker", "build", "-t", goImageVersion+"-dev", ".")
	if err != nil {
		return nil, err
	}
	goWrapper := fmt.Sprintf("run --rm -v %s:/home/go/src/kubernetes -v %s:/home/go/.gitconfig -e HOME=/home/go -w /home/go/src/kubernetes %s", filepath.Join(workspace, "kubernetes"), gitconfigFile, goImageVersion+"-dev")
	args := append(strings.Split(goWrapper, " "), "sh ./tag.sh "+options.NewK8SVersion+"-k3s1")
	logrus.Info(args)
	out, err = runCommand(dir, "docker", args...)
	if err != nil {
		return nil, err
	}
	logrus.Infof(out)
	return nil, nil
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

func gitRebaseOnto(dir string, options K8STagsOptions) error {
	commandArgs := strings.Split(fmt.Sprintf("rebase --onto %s %s %s-k3s1~1",
		options.NewK8SVersion,
		options.OldK8SVersion,
		options.OldK8SVersion), " ")
	cmd := exec.Command("git", commandArgs...)
	var outb, errb bytes.Buffer
	cmd.Stdout = &outb
	cmd.Stderr = &errb
	cmd.Dir = dir
	err := cmd.Run()
	if err != nil {
		return errors.New(err.Error() + ": " + errb.String())
	}
	logrus.Infof("%s", outb.String())
	return nil
}

func golangVersion(workspace string) (string, error) {
	var dep map[string]interface{}
	depFile := filepath.Join(workspace, "kubernetes", "build", "dependencies.yaml")
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
	return "", errors.New("can not find golang dependency")
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
