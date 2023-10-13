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
		Usage: "parse a semantic version",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "output",
				Aliases:  []string{"o"},
				Usage:    "Output format (table|json|yaml|name|go-template)",
				Required: false,
			},
		},
		Action: parse,
	}
}

func test(c *cli.Context) error {
	if c.Args().Get(0) == "" {
		return errors.New("constraint and version are required")
	}
	if c.Args().Get(1) == "" {
		return errors.New("version is required")
	}
	if c.Args().Get(2) != "" {
		return errors.New("too many arguments")
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
	if c.Args().Get(0) == "" {
		return errors.New("version is required")
	}
	v, err := semver.NewVersion(c.Args().Get(0))
	if err != nil {
		return err
	}
	result, err := format(v, c.String("format"))
	if err != nil {
		return err
	}
	fmt.Print(result)
	return nil
}

func format(v *semver.Version, f string) (string, error) {
	switch {
	case f == "" || f == "table":
		var buffer bytes.Buffer
		w := tabwriter.NewWriter(&buffer, 0, 0, 2, ' ', tabwriter.TabIndent)
		fmt.Fprintln(w, "Major\tMinor\tPatch\tPrerelease\tMetadata")
		fmt.Fprintf(w, "%d\t%d\t%d\t%s\t%s\t\n",
			v.Major(),
			v.Minor(),
			v.Patch(),
			v.Prerelease(),
			v.Metadata())
		w.Flush()
		return buffer.String(), nil
	case f == "json":
		jsonData, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			return "", err
		}
		return string(jsonData), nil
	case f == "yaml":
		yml, err := yaml.Marshal(v)
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
		if err := tmpl.Execute(&buf, v); err != nil {
			return "", err
		}
		return buf.String(), nil
	}
	return "", errors.New("invalid output format")
}
