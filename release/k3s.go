package release

import (
	"context"
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
)

const (
	k8sUpstreamURL = "https://github.com/kubernetes/kubernetes"
	rancherRemote  = "rancher"
	k8sRancherURL  = "git@github.com:rancher/kubernetes.git"
	k8sUserURL     = "git@github.com:user/kubernetes.git"
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

func RebaseAndTag(ctx context.Context, ghClient *github.Client, ghUser, workspace string, options K8STagsOptions) ([]string, error) {
	if err := gitRebaseOnto(filepath.Join(workspace, "kubernetes"), options); err != nil {
		return nil, err
	}
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
	commandArgs := fmt.Sprintf("git rebase --onto %s %s %s-k3s1~1",
		options.NewK8SVersion,
		options.OldK8SVersion,
		options.OldK8SVersion)
	cmd := exec.Command("git", commandArgs)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		logrus.Error(out)
		return err
	}
	logrus.Printf("%s", out)
	return nil
}
