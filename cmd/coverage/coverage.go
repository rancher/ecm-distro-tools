package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/urfave/cli"
	"gopkg.in/yaml.v2"
)

type VagrantCov struct {
	path            string
	serverArguments map[string]bool
	agentArguments  map[string]bool
}

var k3sRepoUrl = "https://github.com/k3s-io/k3s.git"

func downloadSource(program string) (*git.Repository, error) {
	programDir := filepath.Join(".", program)
	_, err := os.Stat(program)
	if err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(program, 0755); err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}
	// clone the repo
	repo, err := git.PlainClone(programDir, false, &git.CloneOptions{
		URL:             k3sRepoUrl,
		Progress:        os.Stdout,
		InsecureSkipTLS: true,
	})
	if err != nil {
		if err == git.ErrRepositoryAlreadyExists {
			return repo, nil
		}
		return nil, err
	}
	if err := repo.Fetch(&git.FetchOptions{
		RemoteName:      "origin",
		Progress:        os.Stdout,
		Tags:            git.AllTags,
		InsecureSkipTLS: true,
	}); err != nil {
		if err != git.NoErrAlreadyUpToDate {
			return nil, err
		}
	}
	return repo, nil
}

func checkoutCommit(repo *git.Repository, commit string) error {
	if commit != "" {
		wt, err := repo.Worktree()
		if err != nil {
			return err
		}
		commitRef := plumbing.NewHash(commit)
		return wt.Checkout(&git.CheckoutOptions{Hash: commitRef})
	}
	return nil
}

// discoverE2EFiles returns a list of all the e2e files in the program directory
func discoverE2EFiles(programName string) ([]string, error) {
	vagrantFiles := []string{}
	err := filepath.Walk(programName, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if strings.Contains(path, "tests") && strings.HasPrefix(info.Name(), "Vagrantfile") {
			vagrantFiles = append(vagrantFiles, path)
		}
		return nil
	})
	return vagrantFiles, err
}

func extractConfigYaml(e2eFile string) ([]VagrantCov, error) {
	vagrantCoverage := []VagrantCov{}

	b, err := ioutil.ReadFile(e2eFile)
	if err != nil {
		return vagrantCoverage, err
	}
	reType := regexp.MustCompile(`k3s.args =(?:\s[\"]|\s[\%][w,W][\[])(.*)`)
	reYaml := regexp.MustCompile(`(?m)YAML([\S\s]*?)YAML`)
	typeMatches := reType.FindAllStringSubmatch(string(b), -1)
	yamlMatches := reYaml.FindAllStringSubmatch(string(b), -1)
	for i, match := range typeMatches {
		test := VagrantCov{
			path:            e2eFile,
			serverArguments: make(map[string]bool),
			agentArguments:  make(map[string]bool),
		}

		tester := make(map[string]interface{})
		if len(yamlMatches) > i {
			if err := yaml.Unmarshal([]byte(yamlMatches[i][1]), &tester); err != nil {
				return vagrantCoverage, err
			}
		}
		m := strings.Trim(match[1], `"]`)
		nodeArgs := strings.Split(m, " ")

		if nodeArgs[0] == "server" {
			for _, arg := range nodeArgs[1:] {
				if arg != " " && arg != "" {
					test.serverArguments[arg] = true
				}
			}
			for k := range tester {
				test.serverArguments[k] = true
			}

		} else if nodeArgs[0] == "agent" {
			for _, arg := range nodeArgs[1:] {
				test.agentArguments[arg] = true
			}
			for k := range tester {
				test.agentArguments[k] = true
			}
		}
		vagrantCoverage = append(vagrantCoverage, test)
	}
	return vagrantCoverage, nil
}

func runCommand(dir string, env []string, cmd string, args ...string) (string, error) {
	command := exec.Command(cmd, args...)
	var outb, errb bytes.Buffer
	command.Stdout = &outb
	command.Stderr = &errb
	command.Dir = dir
	command.Env = env
	err := command.Run()
	if err != nil {
		return outb.String(), fmt.Errorf("%v: %w", errb.String(), err)
	}
	return outb.String(), nil
}

func buildBinary(programName string) error {
	binary := filepath.Join(programName, "dist", "artifacts", programName)
	if _, err := os.Stat(binary); err != nil {
		fmt.Println("Building binary")
		if os.IsNotExist(err) {
			out, err2 := runCommand(programName, []string{"SKIP_VALIDATE=true", "SKIP_AIRGAP=true"}, "pwd")
			if err2 != nil {
				fmt.Println(out, err2)
				return err2
			}
		} else {
			return err
		}
	}
	return nil
}

func extractHelp(programName, role string) (map[string]bool, error) {
	var re = regexp.MustCompile(`(?m)--(.*?) `)
	artifactFolder := filepath.Join(programName, "dist", "artifacts", programName)
	out, err := runCommand("", []string{}, artifactFolder, role, "--help")
	if err != nil {
		return nil, err
	}
	serverFlags := make(map[string]bool)
	matches := re.FindAllStringSubmatch(out, -1)
	for _, match := range matches {
		serverFlags[strings.TrimSpace(match[1])] = false
	}
	return serverFlags, nil
}

func coverage(c *cli.Context) error {

	// var re = regexp.MustCompile(`(?m)YAML([\S\s]*?)YAML`)
	programName := c.String("program")
	repo, err := downloadSource(programName)
	if err != nil {
		return err
	}
	if err := checkoutCommit(repo, c.String("commit")); err != nil {
		return err
	}

	e2eFiles, err := discoverE2EFiles(programName)
	if err != nil {
		return err
	}
	vagrantCoverage := []VagrantCov{}
	for _, e2eFile := range e2eFiles {
		vC, err := extractConfigYaml(e2eFile)
		if err != nil {
			return err
		}
		vagrantCoverage = append(vagrantCoverage, vC...)
	}
	fmt.Println(vagrantCoverage)
	if err := buildBinary(programName); err != nil {
		return err
	}
	serverFlagSet, err := extractHelp(programName, "server")
	if err != nil {
		return err
	}
	for _, v := range vagrantCoverage {
		for k := range v.serverArguments {
			serverFlagSet[k] = true
		}
	}
	totalTrue := 0
	for k := range serverFlagSet {
		if serverFlagSet[k] {
			totalTrue++
		}
	}
	percentageCover := float32(totalTrue) / float32(len(serverFlagSet)) * 100
	fmt.Printf("Covering %.2f%% of server flags\n", percentageCover)

	agentFlagSet, err := extractHelp(programName, "agent")
	if err != nil {
		return err
	}
	for _, v := range vagrantCoverage {
		for k := range v.serverArguments {
			agentFlagSet[k] = true
		}
	}
	totalTrue = 0
	for k := range agentFlagSet {
		if agentFlagSet[k] {
			totalTrue++
		}
	}
	percentageCover = float32(totalTrue) / float32(len(agentFlagSet)) * 100
	fmt.Printf("Covering %.2f%% of agent flags\n", percentageCover)
	return nil
}