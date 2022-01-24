package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"k8s.io/helm/pkg/chartutil"
)

var (
	name    string
	version string
	gitSHA  string
)

const (
	envCharts   = "CHARTS"
	filePackage = "package.yaml"
	fileChart   = "Chart.yaml"
	fileValues  = "values.yaml"
)

const usage = `Version: %s
%[2]s - rke2-charts version references update

Usage: %[2]s [options] <chart name> [<version field>=<version value>]
Options:
  -h		show this help message
  -c dir	rke2-charts directory (defaults: $CHARTS value or "rke2-charts")
  -i		write changes into their respective files
  -p		print resulting yaml files on STDOUT
  -v		print version
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

	var (
		vers, inPlace, printOut bool
		chartsDir               string
	)

	chartDefault := "rke2-charts"
	if c, ok := os.LookupEnv(envCharts); ok {
		chartDefault = c
	}

	flag.BoolVar(&vers, "v", false, "print version")
	flag.BoolVar(&inPlace, "i", false, "write changes into their respective files")
	flag.BoolVar(&printOut, "p", false, "print resulting yaml files on STDOUT")
	flag.StringVar(&chartsDir, "c", chartDefault, "rke2-charts directory")
	flag.Parse()

	if vers {
		fmt.Fprintf(os.Stdout, "version: %s - git sha: %s\n", version, gitSHA)
		return
	}

	args := flag.Args()

	if len(args) == 0 {
		exitErr("no chart provided")
	}

	chartName := flag.Arg(0)
	chartPath := filepath.Join(chartsDir, "packages", chartName)

	overrides := make(map[string]string, len(args))
	for _, v := range args[1:] {
		a := strings.Split(v, "=")

		if len(a) < 2 {
			continue
		}

		overrides[a[0]] = a[1]
	}

	err := filepath.Walk(chartPath, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			if strings.Contains(path, "generated") {
				return filepath.SkipDir
			}
			return nil
		}

		var u yamlUpdater
		switch filepath.Base(path) {
		case fileChart:
			u = newUpdater([]string{"appVersion", "version"})
		case filePackage:
			u = newUpdater([]string{"packageVersion"})
		case fileValues:
			u = newImageUpdater("tag")
		default:
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		var b bytes.Buffer
		if _, err := io.Copy(&b, f); err != nil {
			return err
		}

		if err := u.Load(b.Bytes()); err != nil {
			return err
		}
		u.Update(overrides)

		if !(printOut || inPlace) {
			fmt.Print(u)
		}

		if printOut {
			fmt.Print(chartutil.ToYaml(u))
		}

		if inPlace && u.HasChanged() {
			if err := os.WriteFile(path, []byte(chartutil.ToYaml(u)), info.Mode()); err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		exitErr(err)
	}
}

func exitErr(msg interface{}) {
	fmt.Fprintln(os.Stderr, msg)
	os.Exit(1)
}

type yamlUpdater interface {
	Load([]byte) error
	Update(map[string]string)
	HasChanged() bool
}

func newUpdater(targets []string) yamlUpdater {
	m := make(map[string]interface{})

	return &versionTree{
		targets:  targets,
		yaml:     nil,
		Versions: m,
	}
}

func newImageUpdater(target string) yamlUpdater {
	v := make(map[string]interface{})
	r := make(map[string][]map[string]interface{})

	return &versionTreeR{
		target:            target,
		yaml:              nil,
		Versions:          v,
		versionReferences: r,
	}
}

type versionTree struct {
	yaml     chartutil.Values
	targets  []string
	modified bool
	Versions map[string]interface{}
}

func (v *versionTree) Load(b []byte) error {
	values, err := chartutil.ReadValues(b)
	if err != nil {
		return err
	}

	v.yaml = values
	for _, t := range v.targets {
		if val, ok := values[t]; ok {
			v.Versions[t] = val
		}
	}

	return nil
}

func (v *versionTree) Update(overrides map[string]string) {
	m := v.yaml.AsMap()
	for _, t := range v.targets {
		if val, ok := overrides[t]; ok {
			v.modified = true
			m[t] = val
			v.Versions[t] = val
		}
	}
}

func (v *versionTree) HasChanged() bool {
	return v.modified
}

func (v *versionTree) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.yaml)
}

func (v *versionTree) String() string {
	return chartutil.ToYaml(v.Versions)
}

type versionTreeR struct {
	yaml              chartutil.Values
	target            string
	Versions          map[string]interface{}
	versionReferences map[string][]map[string]interface{}
	modified          bool
}

func (v *versionTreeR) Load(b []byte) error {
	values, err := chartutil.ReadValues(b)
	if err != nil {
		return err
	}

	v.yaml = values
	targetLookup(v.target, values.AsMap(), v.versionReferences)

	for k, val := range v.versionReferences {
		v.Versions[k] = val[0][v.target]
	}

	return nil
}

func (v *versionTreeR) Update(overrides map[string]string) {
	for k := range v.versionReferences {
		if val, ok := overrides[k]; ok {
			v.modified = true
			v.Versions[k] = val
			for _, ref := range v.versionReferences[k] {
				ref[v.target] = val
			}
		}
	}
}

func (v *versionTreeR) HasChanged() bool {
	return v.modified
}

func (v *versionTreeR) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.yaml)
}

func (v *versionTreeR) String() string {
	return chartutil.ToYaml(v.Versions)
}

func targetLookup(target string, tree map[string]interface{}, dictionary map[string][]map[string]interface{}) {
	if _, found := tree[target]; !found {
		for _, v := range tree {
			if vv, ok := v.(map[string]interface{}); ok {
				targetLookup(target, vv, dictionary)
			}
		}
		return
	}

	var relative string
	for k, v := range tree {
		if k == target {
			continue
		}
		relative = v.(string)
	}

	l := dictionary[relative]
	dictionary[relative] = append(l, tree)
}
