package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/Masterminds/semver/v3"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v3"
)

var version string

func main() {
	app := &cli.App{
		Name:                   "semv",
		UseShortOptionHandling: true,
		Version:                version,
		Commands: []*cli.Command{
			parseCommand(),
			testCommand(),
		},
	}

	if err := app.Run(os.Args); err != nil {
		logrus.Fatal(err)
	}
}

func testCommand() *cli.Command {
	return &cli.Command{
		Name:   "test",
		Usage:  "test [constraint] [version]",
		Action: test,
	}
}

func parseCommand() *cli.Command {
	return &cli.Command{
		Name:  "parse",
		Usage: "parse [version]",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "output",
				Aliases:     []string{"o"},
				Usage:       "Output `format` (table|json|yaml|name|go-template)",
				DefaultText: "table",
				Required:    false,
			},
		},
		Action: parse,
	}
}

func test(c *cli.Context) error {
	if c.Args().Len() != 2 {
		return errors.New("invalid number of arguments")
	}
	constraint, err := semver.NewConstraint(c.Args().Get(0))
	if err != nil {
		return err
	}
	version, err := semver.NewVersion(c.Args().Get(1))
	if err != nil {
		return err
	}
	if constraint.Check(version) {
		os.Exit(0)
	} else {
		os.Exit(1)
	}
	return nil
}

func parse(c *cli.Context) error {
	if c.Args().Len() != 1 {
		return errors.New("invalid number of arguments")
	}
	v, err := semver.NewVersion(c.Args().Get(0))
	if err != nil {
		return err
	}
	result, err := format(v, c.String("output"))
	if err != nil {
		return err
	}
	fmt.Print(result)
	return nil
}

func format(v *semver.Version, f string) (string, error) {
	data := struct {
		Major      uint64 `json:"major" yaml:"major"`
		Minor      uint64 `json:"minor" yaml:"minor"`
		Patch      uint64 `json:"patch" yaml:"patch"`
		Prerelease string `json:"prerelease" yaml:"prerelease"`
		Metadata   string `json:"metadata" yaml:"metadata"`
	}{
		Major:      v.Major(),
		Minor:      v.Minor(),
		Patch:      v.Patch(),
		Prerelease: v.Prerelease(),
		Metadata:   v.Metadata(),
	}
	switch {
	case f == "" || f == "table":
		var buffer bytes.Buffer
		w := tabwriter.NewWriter(&buffer, 0, 0, 2, ' ', tabwriter.TabIndent)
		fmt.Fprintln(w, "Major\tMinor\tPatch\tPrerelease\tMetadata")
		fmt.Fprintf(w, "%d\t%d\t%d\t%s\t%s\t\n",
			data.Major,
			data.Minor,
			data.Patch,
			data.Prerelease,
			data.Metadata)
		w.Flush()
		return buffer.String(), nil
	case f == "json":
		jsonData, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			return "", err
		}
		return string(jsonData), nil
	case f == "yaml":
		yml, err := yaml.Marshal(data)
		if err != nil {
			return "", err
		}
		return string(yml), nil
	case strings.HasPrefix(f, "go-template="):
		goTemplate := strings.TrimPrefix(f, "go-template=")
		tmpl, err := template.New("output").Parse(goTemplate)
		if err != nil {
			return "", err
		}
		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, data); err != nil {
			return "", err
		}
		return buf.String(), nil
	}
	return "", errors.New("invalid output format")
}
