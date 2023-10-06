package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"html/template"
	"os"
	"strings"

	"github.com/rancher/ecm-distro-tools/release/semver"
	"gopkg.in/yaml.v3"
)

func Format(v *semver.Version, format string) (string, error) {
	switch {
	case format == "":
		str := fmt.Sprintf(
			"Version: %s\nMajor: %d\nMinor: %d\nPatch: %d\nPrerelease: %s\nBuild: %s\n",
			v.Version,
			v.Major,
			v.Minor,
			v.Patch,
			v.Prerelease,
			v.Build,
		)
		return str, nil
	case format == "json":
		jsonData, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			return "", err
		}
		return string(jsonData), nil
	case format == "yaml":
		yml, err := yaml.Marshal(v)
		if err != nil {
			return "", err
		}
		return string(yml), nil
	case strings.HasPrefix(format, "go-template="):
		goTemplate := strings.TrimPrefix(format, "go-template=")
		tmpl, err := template.New("output").Parse(goTemplate)
		if err != nil {
			return "", err
		}
		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, v); err != nil {
			return "", err
		}
		return buf.String(), nil
	}
	return "", errors.New("invalid output format")
}

var (
	name    string
	version string
	gitSHA  string
)

const usage = `version: %s
Usage: %[2]s [-test] [-parse]
Options:
    -h            help
    -v            show version and exit
    -test         test a complete version against a semantic version pattern
    -parse        parse a semantic version

Examples: 
    # parse
    %[2]s -parse v1.2.3-rc1 -o go-template="{{.Major}}.{{.Minor}}.{{.Patch}}{{.Prerelease}}"
    #test
    %[2]s -pattern v1.x -test v1.2.3
`

func main() {
	flag.Usage = func() {
		w := os.Stderr
		for _, arg := range os.Args {
			if arg == "-h" {
				w = os.Stdout
				break
			}
		}
		fmt.Fprintf(w, usage, version, name)
	}

	var vers bool
	var parseArg, patternArg, testArg, formatArg string

	flag.BoolVar(&vers, "v", false, "")
	flag.StringVar(&parseArg, "parse", "", "Perform parse")
	flag.StringVar(&patternArg, "pattern", "", "Perform test")
	flag.StringVar(&testArg, "test", "", "Perform test")
	flag.StringVar(&formatArg, "o", "", "Output format")

	flag.Parse()

	if vers {
		fmt.Fprintf(os.Stdout, "version: %s - git sha: %s\n", version, gitSHA)
		return
	}

	if testArg != "" {
		if patternArg == "" || parseArg != "" {
			fmt.Println("Invalid arguments. Usage:" + name + " -pattern <pattern> -test <version>")
			os.Exit(1)
		}

		pattern, err := semver.ParsePattern(patternArg)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		version, err := semver.ParseVersion(testArg)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		pattern.Test(version)
	}

	if parseArg != "" {
		version, err := semver.ParseVersion(parseArg)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		result, err := Format(version, formatArg)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		fmt.Print(result)
	}
}
