package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"

	"github.com/spdx/tools-golang/builder"
	"github.com/spdx/tools-golang/tvsaver"
)

const (
	ORGANIZATION_SUSE = "SUSE"
	DEFAULT_FORMAT    = "spdx"
)

const usage = `version: %s
Usage: %[2]s [-v] [-p] [-m upstream version]
Options:
	-h                   help
	-v version		 	 show version and exit
	-r                   Repository name (rke2, k3s) needed for the output file
	-m upstream version  upstream version used to identify output file name
	-p packageRootDir    package root dir or root path to generate SBOMs from
Examples: 
# generate SBOM SPDX file for the given root file
%[2]s -r rke2 -v v1.21.5 
`

var (
	vers           bool
	repo           string
	upVersion      string
	packageRootDir string
	namespaces     map[string]string
)

func init() {
	namespaces = make(map[string]string)
	namespaces["rke2"] = "https://rke2.io/"
	namespaces["k3s"] = "https://k3s.io/"
}

func main() {
	flag.Usage = func() {
		w := os.Stderr
		for _, arg := range os.Args {
			if arg == "-h" {
				w = os.Stdout
				break
			}
		}
		fmt.Fprintf(w, usage, upVersion, "sbom-spdx")
	}

	flag.BoolVar(&vers, "v", false, "")
	flag.StringVar(&repo, "r", "", "")
	flag.StringVar(&packageRootDir, "p", "", "")
	flag.StringVar(&upVersion, "m", "", "")
	flag.Parse()

	if vers {
		fmt.Fprintf(os.Stdout, "version: %s\n", upVersion)
		return
	}

	if repo == "" {
		fmt.Println("error: please provide a valid repo")
		os.Exit(1)
	}

	if upVersion == "" {
		fmt.Println("error: please provide the upstream version")
		os.Exit(1)
	}

	if packageRootDir == "" {
		fmt.Println("error: please provide the packageRootDir")
		os.Exit(1)
	}

	fileOut := generateFileName()
	format := DEFAULT_FORMAT
	config := getBuildImagesConfig()

	docBuildImages, err := builder.Build2_2("buildimages", packageRootDir, config)
	if err != nil {
		fmt.Printf("Error while building document: %v\n", err)
		return
	}

	docBuildImages.CreationInfo.DocumentName = fmt.Sprintf("SBOM_SPDX_%s-%s", repo, upVersion)

	fmt.Printf("Successfully created document for package %s\n", "images")
	goFileOut := fmt.Sprintf("%s_%s.%s", fileOut, "buildimages", format)
	w, err := os.Create(goFileOut)
	if err != nil {
		fmt.Printf("Error while opening %v for writing: %v\n", goFileOut, err)
		return
	}
	defer w.Close()

	err = tvsaver.Save2_2(docBuildImages, w)
	if err != nil {
		fmt.Printf("Error while saving %v: %v", goFileOut, err)
		return
	}

	fmt.Printf("Successfully buildimages module saved %v\n", fileOut)

	err = callSbomSpdxTool(packageRootDir, "")
	if err != nil {
		fmt.Printf("Error callSbomSpdxTool execution: %+v", err)
		os.Exit(1)
	}

	os.Exit(0)
}

func generateFileName() string {
	return fmt.Sprintf("sbom-spdf-%s-%s", repo, upVersion)
}

func getBuildImagesConfig() *builder.Config2_2 {
	return &builder.Config2_2{

		NamespacePrefix: namespaces[repo],

		// "Person", "Organization" or "Tool" for the SPDX specs
		CreatorType: "Organization",

		Creator: ORGANIZATION_SUSE,

		PathsIgnored: []string{

			// ignore all files in the given directory at the package root
			"/.git/",
			"/.github/",
			"/.vagrant/",
			"/bin/",
			"/bundle/",
			"/charts/",
			"/contrib/",
			"/developer-docs/",
			"/dist/",
			"/docs/",
			"/pkg/",
			"/scripts/",
			"/tests/",
			"/windows/",

			"/.codespellignore",
			"/.dapper",
			"/.dockerignore",
			"/.drone.yml",
			"/.gitignore",
			"/.golangci.json",
			"/.vagrant",
			"/BUILDING.md",
			"/channels.yaml",
			"/CODEOWNERS",
			"/Dockerfile",
			"/Dockerfile.docs",
			"/Dockerfile.windows",
			"/LICENSE",
			"/MAINTAINERS",
			"/Makefile",
			"/README.md",
			"/ROADMAP.md",
			"/SECURITY.md",
			"/mkdocs.yml",
			"/go-deps-list.json",
			"/go.sum",
			"/go.mod",
			"/install.ps1",
			"/install.sh",
			"/main.go",
			"/manifest-runtime.tmpl",
			"/Vagrantfile",

			// ignore folder anywhere within the directory tree
			"**/.DS_Store",
			"**/images/",
		},
	}
}

func callSbomSpdxTool(path, outPath string) error {

	var args = []string{
		"-p",
		path,
	}
	if outPath != "" {
		args = append(args, []string{"-o", outPath}...)
	}

	cmd := exec.Command("./bin/spdx-sbom-generator", args...)
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		return err
	}
	return nil
}
