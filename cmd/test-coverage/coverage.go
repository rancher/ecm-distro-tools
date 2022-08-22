package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	grob "github.com/MetalBlueberry/go-plotly/graph_objects"
	"github.com/MetalBlueberry/go-plotly/offline"
	"github.com/urfave/cli"
	"gopkg.in/yaml.v2"
)

type TestCov struct {
	path            string
	serverArguments map[string]bool
	agentArguments  map[string]bool
}

// discoverTestFiles returns a list of all the e2e files in the program directory
func discoverTestFiles(programName string) ([]string, []string, error) {
	vagrantFiles := []string{}
	integrationFiles := []string{}
	testRoot := filepath.Join(programName, "tests")
	err := filepath.Walk(testRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if strings.HasPrefix(info.Name(), "Vagrantfile") {
			vagrantFiles = append(vagrantFiles, path)
		}
		if strings.HasSuffix(info.Name(), "int_test.go") {
			integrationFiles = append(integrationFiles, path)
		}
		return nil
	})
	return vagrantFiles, integrationFiles, err
}

func extractConfigYaml(e2eFile string) (TestCov, error) {

	b, err := ioutil.ReadFile(e2eFile)
	if err != nil {
		return TestCov{}, err
	}
	reType := regexp.MustCompile(`k3s.args =(?:\s[\"]|\s[\%][w,W][\[])(.*)`)
	reYaml := regexp.MustCompile(`(?m)YAML([\S\s]*?)YAML`)
	typeMatches := reType.FindAllStringSubmatch(string(b), -1)
	yamlMatches := reYaml.FindAllStringSubmatch(string(b), -1)
	vagrantCoverage := TestCov{
		path:            e2eFile,
		serverArguments: make(map[string]bool),
		agentArguments:  make(map[string]bool),
	}
	for i, match := range typeMatches {
		yamlConfig := make(map[string]interface{})
		if len(yamlMatches) > i {
			if err := yaml.Unmarshal([]byte(yamlMatches[i][1]), &yamlConfig); err != nil {
				return vagrantCoverage, err
			}
		}
		m := strings.Trim(match[1], `"]`)
		nodeArgs := strings.Split(m, " ")

		if nodeArgs[0] == "server" {
			for _, arg := range nodeArgs[1:] {
				if arg != " " && arg != "" {
					vagrantCoverage.serverArguments[arg] = true
				}
			}
			for k := range yamlConfig {
				vagrantCoverage.serverArguments[k] = true
			}

		} else if nodeArgs[0] == "agent" {
			for _, arg := range nodeArgs[1:] {
				vagrantCoverage.agentArguments[arg] = true
			}
			for k := range yamlConfig {
				vagrantCoverage.agentArguments[k] = true
			}
		}
	}
	return vagrantCoverage, nil
}

func extractTestArgs(testFile string) (TestCov, error) {
	reArgs := regexp.MustCompile(`(?m)(?i)serverargs =.*(?s){(.*?)}`)
	b, err := ioutil.ReadFile(testFile)
	if err != nil {
		return TestCov{}, err
	}
	intCoverage := TestCov{
		path:            testFile,
		serverArguments: make(map[string]bool),
	}
	matches := reArgs.FindAllStringSubmatch(string(b), -1)
	for _, match := range matches {
		reQuotes := regexp.MustCompile(`(?m)"[^"]*"`)
		args := reQuotes.FindAllString(match[1], -1)
		// Double for loop to deal with nested arguments
		for _, arg := range args {
			arg = strings.Trim(arg, `"`)
			arg = strings.TrimPrefix(arg, "--")
			for _, a := range strings.Split(arg, " ") {
				if a != "" {
					intCoverage.serverArguments[a] = true
				}
			}
		}
	}

	return intCoverage, nil
}

func totalUsed(m map[string]int) int {
	count := 0
	for _, v := range m {
		if v > 0 {
			count++
		}
	}
	return count
}

func extractHelp(program, role string) (map[string]int, error) {
	var re = regexp.MustCompile(`(?m)--(.*?) `)
	command := exec.Command(program, role, "--help")
	out, err := command.CombinedOutput()

	if err != nil {
		return nil, fmt.Errorf("exec output: %s: %v", out, err)
	}
	roleFlags := make(map[string]int)
	matches := re.FindAllStringSubmatch(string(out), -1)
	for _, match := range matches {
		roleFlags[strings.TrimSpace(match[1])] = 0
	}
	return roleFlags, nil
}

func coverage(c *cli.Context) error {

	programPath := strings.ToLower(c.String("path"))
	programPath = filepath.Clean(programPath)
	fmt.Println(programPath)
	var program string
	if strings.Contains(programPath, "k3s") {
		program = filepath.Join(programPath, "bin", "k3s")
	} else if strings.Contains(programPath, "rke2") {
		program = filepath.Join(programPath, "bin", "rke2")
	}

	if _, err := os.Stat(program); err != nil {
		return fmt.Errorf("unable to find binary at %s", program)
	}
	e2eFiles, intTestFiles, err := discoverTestFiles(programPath)
	if err != nil {
		return err
	}
	vagrantCoverage := []TestCov{}
	for _, e2eFile := range e2eFiles {
		vC, err := extractConfigYaml(e2eFile)
		if err != nil {
			return err
		}
		vagrantCoverage = append(vagrantCoverage, vC)
	}

	intCoverage := []TestCov{}
	for _, integrationFile := range intTestFiles {
		iC, err := extractTestArgs(integrationFile)
		if err != nil {
			return err
		}
		intCoverage = append(intCoverage, iC)
	}

	serverFlagSet, err := extractHelp(program, "server")
	if err != nil {
		return err
	}

	// Record covered flags and filter out invalid entries
	for flag := range serverFlagSet {
		for _, vC := range vagrantCoverage {
			if vC.serverArguments[flag] {
				serverFlagSet[flag] += 1
			}
		}
		for _, intC := range intCoverage {
			if intC.serverArguments[flag] {
				serverFlagSet[flag] += 1
			}
		}
	}

	totalUsedFlags := totalUsed(serverFlagSet)
	percentageCover := float32(totalUsedFlags) / float32(len(serverFlagSet)) * 100
	fmt.Printf("Covering %d out of %d (%.2f%%) of server flags\n", totalUsedFlags, len(serverFlagSet), percentageCover)

	// integration tests don't have agent flags
	agentFlagSet, err := extractHelp(program, "agent")
	if err != nil {
		return err
	}
	for _, v := range vagrantCoverage {
		for k := range v.agentArguments {
			if _, ok := agentFlagSet[k]; ok {
				agentFlagSet[k] += 1
			}
		}
	}
	totalUsedFlags = totalUsed(agentFlagSet)
	percentageCover = float32(totalUsedFlags) / float32(len(agentFlagSet)) * 100
	fmt.Printf("Covering %d out of %d (%.2f%%) of agent flags\n", totalUsedFlags, len(agentFlagSet), percentageCover)

	if c.Bool("graph") {

		data := grob.Traces{}
		xFlagNames := []string{}

		for k := range serverFlagSet {
			xFlagNames = append(xFlagNames, k)
		}
		for _, test := range append(vagrantCoverage, intCoverage...) {
			condensedName := strings.TrimPrefix(test.path, programPath+"/tests/")
			var testGroup string
			if group := "integration"; strings.Contains(condensedName, group) {
				testGroup = group
			} else if group := "install"; strings.Contains(condensedName, group) {
				testGroup = group
			} else if group := "e2e"; strings.Contains(condensedName, group) {
				testGroup = group
			}

			flagHits := make([]int, len(xFlagNames))
			for i, flag := range xFlagNames {
				if test.serverArguments[flag] {
					flagHits[i] = 1
				}
			}
			data = append(data, &grob.Bar{
				Name:        condensedName,
				X:           xFlagNames,
				Y:           flagHits,
				Type:        grob.TraceTypeBar,
				Legendgroup: testGroup,
			})
		}

		fig := &grob.Fig{
			Data: data,
			Layout: &grob.Layout{
				Title: &grob.LayoutTitle{
					Text: "Server Argument Coverage",
				},
				Xaxis: &grob.LayoutXaxis{
					Tickangle: 60,
				},
				Yaxis: &grob.LayoutYaxis{
					Title: &grob.LayoutYaxisTitle{
						Text: "# of Tests Using Flag",
					},
				},
				Barmode: "stack",
			},
		}
		offline.ToHtml(fig, "graph.html")
		cd, _ := os.Getwd()
		fmt.Printf("Graph written to: file://%s\n", filepath.Join(cd, "graph.html"))

	}

	return nil
}
