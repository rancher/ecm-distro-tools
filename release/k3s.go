package release

import (
	"bytes"
	"context"
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
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/google/go-github/v39/github"
	"github.com/sirupsen/logrus"
	ssh2 "golang.org/x/crypto/ssh"
	"gopkg.in/yaml.v2"
)

const (
	k8sUpstreamURL     = "https://github.com/kubernetes/kubernetes"
	rancherRemote      = "k3s-io"
	k8sRancherURL      = "git@github.com:k3s-io/kubernetes.git"
	k8sUserURL         = "git@github.com:user/kubernetes.git"
	k3sRepoURL         = "git@github.com:user/k3s.git"
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
)

type K8STagsOptions struct {
	OldK8SVersion string
	NewK8SVersion string
	OldK8SClient  string
	NewK8SClient  string
	OldK3SVersion string
	NewK3SVersion string
	ReleaseBranch string
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
	wrapperImageTag, err := buildGoWrapper(workspace)
	if err != nil {
		return nil, err
	}

	// setup gitconfig
	gitconfigFile, err := setupGitArtifacts(workspace, ghEmail, ghUser)
	if err != nil {
		return nil, err
	}

	out, err := runTagScript(workspace, gitconfigFile, wrapperImageTag, options.NewK8SVersion)
	if err != nil {
		if "" != err.Error() {
			return nil, err
		}
	}
	logrus.Infof("Running Tag Script: %s %v", out, err)
	tagFile := filepath.Join(workspace, "tags-"+options.NewK8SVersion)
	tags := tagPushLines(out, tagFile)
	if len(tags) != 28 {
		return nil, errors.New("failed to extract tag push lines")
	}
	err = os.WriteFile(tagFile, []byte(strings.Join(tags, "\n")), 0644)
	if err != nil {
		return nil, err
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

func buildGoWrapper(workspace string) (string, error) {
	goVersion, err := golangVersion(workspace)
	if err != nil {
		return "", err
	}
	goImageVersion := fmt.Sprintf("golang:%s-alpine3.15", goVersion)
	devDockerfile := strings.ReplaceAll(dockerDevImage, "%goimage%", goImageVersion)
	if err := os.WriteFile(filepath.Join(workspace, "dockerfile"), []byte(devDockerfile), 0644); err != nil {
		return "", err
	}
	wrapperImageTag := goImageVersion + "-dev"
	out, err := runCommand(workspace, "docker", "build", "-t", wrapperImageTag, ".")
	if err != nil {
		return "", err
	}
	logrus.Infof("Building Wrapper image: %s", out)
	return wrapperImageTag, nil
}

func setupGitArtifacts(workspace, email, handler string) (string, error) {
	gitconfigFile := filepath.Join(workspace, ".gitconfig")

	// setting up username and email for tagging purposes
	gitconfigFileContent := strings.ReplaceAll(gitconfig, "%email%", email)
	gitconfigFileContent = strings.ReplaceAll(gitconfigFileContent, "%user%", handler)

	if err := os.WriteFile(gitconfigFile, []byte(gitconfigFileContent), 0644); err != nil {
		return "", err
	}
	return gitconfigFile, nil
}

func runTagScript(workspace, gitConfigFile, wrapperImageTag, newK8SVersion string) (string, error) {
	uid := strconv.Itoa(os.Getuid())
	gid := strconv.Itoa(os.Getgid())
	gopath, err := runCommand(workspace, "go", "env", "GOPATH")
	if err != nil {
		return "", err
	}
	k8sDir := filepath.Join(workspace, "kubernetes")
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

	args := append(goWrapper, "./tag.sh", newK8SVersion+"-k3s1")
	return runCommand(k8sDir, "docker", args...)
}

func tagPushLines(out string, tagFile string) []string {
	tagCmds := []string{}
	_, err := os.Stat(tagFile)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil
		}
	} else {
		dat, err := os.ReadFile(tagFile)
		if err != nil {
			return nil
		}
		out = string(dat)
	}

	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "git push $REMOTE") {
			tagCmds = append(tagCmds, line)
		}
	}
	return tagCmds
}

func PushTags(ctx context.Context, tagsCmds []string, ghClient *github.Client, workspace, email, handler, remote string) error {
	logrus.Infof("pushing tags to github")
	// here we can use go-git library or runCommand function
	// I am using go-git library to enhance code quality
	gitConfigFile, err := setupGitArtifacts(workspace, email, handler)
	if err != nil {
		return err
	}
	file, err := os.Open(gitConfigFile)
	if err != nil {
		return err
	}
	cfg, err := config.ReadConfig(file)
	if err != nil {
		return err
	}

	repo, err := git.PlainOpen(filepath.Join(workspace, "kubernetes"))
	if err != nil {
		return err
	}
	userRemote, err := repo.Remote(handler)
	if err != nil {
		return err
	}
	k3sRemote, err := repo.Remote("k3s-io")
	if err != nil {
		return err
	}
	cfg.Remotes[handler] = userRemote.Config()
	cfg.Remotes["k3s-io"] = k3sRemote.Config()

	if err := repo.SetConfig(cfg); err != nil {
		return err
	}
	gitAuth, err := getAuth("")
	if err != nil {
		return err
	}
	logrus.Info(tagsCmds)
	for _, tagCmd := range tagsCmds {
		tag := strings.Split(tagCmd, " ")[3]
		logrus.Infof(tag)
		if err := repo.Push(&git.PushOptions{
			RemoteName: remote,
			Auth:       gitAuth,
			Progress:   os.Stdout,
			RefSpecs: []config.RefSpec{
				config.RefSpec("+refs/tags/" + tag + ":refs/tags/" + tag),
			},
		}); err != nil {
			return err
		}
	}
	return nil
}

func ModifyAndPush(ctx context.Context, options K8STagsOptions, workspace, handle, email string) error {
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

	k3sFork := strings.ReplaceAll(k3sRepoURL, "user", handle)
	modifyScript = strings.Replace(modifyScript, "%Workspace%", workspace, -1)
	modifyScript = strings.Replace(modifyScript, "%K3SFork%", k3sFork, -1)
	modifyScript = strings.Replace(modifyScript, "%Handler%", handle, -1)

	f, err := os.Create(filepath.Join(workspace, "modify_script.sh"))
	if err != nil {
		return err
	}

	if err := os.Chmod(filepath.Join(workspace, "modify_script.sh"), 0755); err != nil {
		return err
	}

	tmpl, err := template.New("modify_script.sh").Parse(modifyScript)
	if err != nil {
		return err
	}

	if err := tmpl.Execute(f, options); err != nil {
		return err
	}

	out, err := runCommand(workspace, "bash", "./modify_script.sh")
	if err != nil {
		return err
	}
	logrus.Info(out)
	return nil
}

func CreatePRFromK3S(ctx context.Context, ghClient *github.Client, workspace, org string, options K8STagsOptions) error {
	repo := "k3s"
	pull := &github.NewPullRequest{
		Title:               github.String(fmt.Sprintf("Update to %s-%s", options.NewK8SVersion, options.NewK3SVersion)),
		Base:                github.String(options.ReleaseBranch),
		Head:                github.String(options.NewK8SVersion + "-" + options.NewK3SVersion),
		MaintainerCanModify: github.Bool(true),
	}

	_, resp, err := ghClient.PullRequests.Create(ctx, org, repo, pull)
	logrus.Info(*resp)
	return err
}

func UpdateAndPush(ctx context.Context, options K8STagsOptions, workspace, handle, email string) error {
	updateScript = strings.Replace(updateScript, "%Workspace%", workspace, -1)
	updateScript = strings.Replace(updateScript, "%Handler%", handle, -1)

	f, err := os.Create(filepath.Join(workspace, "update_script.sh"))
	if err != nil {
		return err
	}

	if err := os.Chmod(filepath.Join(workspace, "update_script.sh"), 0755); err != nil {
		return err
	}

	tmpl, err := template.New("update_script.sh").Parse(updateScript)
	if err != nil {
		return err
	}

	if err := tmpl.Execute(f, options); err != nil {
		return err
	}

	out, err := runCommand(workspace, "bash", "./update_script.sh")
	if err != nil {
		return err
	}
	logrus.Info(out)

	return nil
}
