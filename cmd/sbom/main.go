package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spdx/tools-golang/builder"
	"github.com/spdx/tools-golang/tvsaver"
)

const (
	ORGANIZATION_SUSE        = "SUSE"
	DEFAULT_FORMAT           = "spdx"
	SPDX_CREATOR_TYPE_ORG    = "Organization"
	SPDX_CREATOR_TYPE_PERSON = "Person"
	SPDX_CREATOR_TYPE_TOOL   = "Tool"
	REGEX_LICENSE_IDENTIFIER = `SPDX-License-Identifier`
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
	vers             bool
	repo             string
	upVersion        string
	packageRootDir   string
	namespaces       map[string]string
	cliDir           string
	licensesListSPDX []license
	sbomFileTool     string
)

func init() {
	namespaces = make(map[string]string)
	namespaces["rke2"] = "https://rke2.io/"
	namespaces["k3s"] = "https://k3s.io/"
	wd, err := os.Getwd()
	if err != nil {
		log.Println(err)
	}
	cliDir = wd
	fileBytes, err := os.ReadFile(filepath.Join(cliDir, "assets", "spdx-license-list.json"))
	if err != nil {
		fmt.Println("error: ", err.Error())
		os.Exit(1)
	}

	json.Unmarshal(fileBytes, &licensesListSPDX)
	sbomFileTool = "bom-go-mod.spdx"
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

	// retrieve official library name and version for the Creator: Tool part in SPDX
	docBuildImages.CreationInfo.DocumentName = fmt.Sprintf("SBOM_SPDX_%s-%s", repo, upVersion)
	moduleVers, err := getModuleVersion("github.com/spdx/tools-golang")
	if err != nil {
		fmt.Printf("error: unable to find module version. %s\n", err.Error())
		os.Exit(1)
	} else {
		for i, t := range docBuildImages.CreationInfo.CreatorTools {
			if t == "github.com/spdx/tools-golang/builder" {
				docBuildImages.CreationInfo.CreatorTools[i] = t + "-" + moduleVers
				break
			}
		}
		// docBuildImages.CreationInfo.CreatorTools = append(docBuildImages.CreationInfo.CreatorTools, "spdx/tools-golang-"+moduleVers)
	}

	fmt.Printf("Successfully created document for package %s\n", "images")

	goFileOut := fmt.Sprintf("%s_%s.%s", fileOut, "buildimages", format)
	w, err := os.Create(goFileOut)
	if err != nil {
		fmt.Printf("Error while opening %v for writing: %v\n", goFileOut, err)
		os.Exit(1)
	}
	defer w.Close()

	err = tvsaver.Save2_2(docBuildImages, w)
	if err != nil {
		fmt.Printf("Error while saving %v: %v", goFileOut, err)
		os.Exit(1)
	}

	fmt.Printf("Successfully buildimages module saved %v\n", fileOut)

	err = callSbomSpdxTool(packageRootDir, "")
	if err != nil {
		fmt.Printf("Error callSbomSpdxTool execution: %+v", err)
		os.Exit(1)
	}

	// replace root package license
	rootSpdxID, err := findSpdxIDInFile(packageRootDir)
	if err != nil {
		fmt.Println("error finding spdx ID in root Pkg License file: ", err.Error())
		os.Exit(1)
	}
	replaceRootPackageLicenses(packageRootDir, rootSpdxID, "")

	err = findLicensesInPackages(packageRootDir)
	if err != nil {
		fmt.Printf("Error finding LicensesInPackages: %+v", err)
		os.Exit(1)
	}

	renameSpdxFile(sbomFileTool, fmt.Sprintf("%s_%s.%s", fileOut, "gomod", format))
	os.Exit(0)
}

func renameSpdxFile(current, final string) {

	args := []string{current, final}
	cmd := exec.Command("mv", args...)
	_, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println("error final spdx file: ", err.Error())
		os.Exit(1)
	}
	fmt.Println("process completed\nOutput file: ", final)
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

// returns the version for the given go module
func getModuleVersion(module string) (string, error) {
	args := []string{"list", "-u", "-m", "-versions", module}
	cmd := exec.Command("go", args...)
	out, _ := cmd.CombinedOutput()
	outStr := string(out)
	verArr := strings.Split(outStr, " ")
	outVer := verArr[len(verArr)-1]

	if !strings.ContainsAny(outVer, "v") {
		return "", fmt.Errorf("no versions found for the module: %s", module)
	}

	return outVer, nil
}

type Module struct {
	Path      string
	Version   string
	Indirect  bool
	Dir       string
	GoVersion string
}
type goPkg struct {
	Dir        string
	ImportPath string
	Name       string
	Doc        string
	Root       string
	Module     Module
	SpdxID     string
}

type license struct {
	Reference       string `json:"reference"`
	IsDeprecated    bool   `json:"isDeprecatedLicenseId"`
	DetailsURL      string `json:"detailsUrl"`
	ReferenceNumber int    `json:"referenceNumber"`
	Name            string `json:"name"`
	LicenseID       string `json:"licenseId"`
	IsOSIapproved   string `json:"isOsiApproved"`
}

func returnToCliDir() {
	os.Chdir(cliDir)
}

func getGoPackages(dir string) ([]*goPkg, error) {
	defer returnToCliDir()

	os.Chdir(dir)

	args := []string{"list", "-deps", "-json", "./..."}
	cmd := exec.Command("go", args...)
	out, err := cmd.Output()
	if err != nil {
		fmt.Printf("error getting go deps: %v \n", err)
		return nil, err
	}

	var modules []*goPkg

	dec := json.NewDecoder(bytes.NewReader(out))
	for {
		var m goPkg
		if err := dec.Decode(&m); err != nil {
			if err == io.EOF {
				break
			}
			log.Fatalf("reading go list output: %v", err)
		}
		modules = append(modules, &m)
	}

	return modules, nil
}

func replaceRootPackageLicenses(filePath string, licConcluded, licDeclared string) {
	noAssertions := map[string]string{
		// Governing license for the package
		"PackageLicenseConcluded: NOASSERTION": "PackageLicenseConcluded: " + licConcluded,

		// TBD: Licenses used in files
		// "PackageLicenseDeclared: NOASSERTION": licDeclared,

		// TBD
		// "PackageLicenseComments: NOASSERTION"
	}

	for old, new := range noAssertions {
		args := []string{fmt.Sprintf("0,/%s/s//%s/", old, new), filepath.Join(filePath, sbomFileTool)}
		_, err := exec.Command("sed", args...).Output()
		if err != nil {
			fmt.Println("error replacing root license file: ", err.Error())
			continue
		}
	}
}

func findLicensesInPackages(dir string) error {
	/*
		1. Read packages from go.mod
		2. Find Package in go $GOPATH/pkg/mod
		3. Read Pckage LICENSE in $GOPATH/pkg/mod
		4. If only one dir then enter the child dir and obtain the LICENSE if pissible
			4.1. retrieve SPDX-License-Identifier or SPDX license
		5. else find version in dir
	*/
	pkgs, err := getGoPackages(dir)
	if err != nil {
		return err
	}

	// lookup in pkgs files
	// Note: validate this process as it will need a whole search
	// for the PackageLicenseConcluded: NOASSERTION part for every package in the spdx file
	for _, pkg := range pkgs {
		if pkg.Module.Dir == "" {
			continue
		}
		spdxID, err := findSpdxIDInFile(pkg.Module.Dir)
		if err != nil {
			fmt.Println("Error: ", err.Error())
			continue
		}
		pkg.SpdxID = spdxID
		// TODO: Validate if Pkg Spdx IDs are required
		// TODO: if true then replace PackageLicenseConcluded: NOASSERTION in generated sbom file
	}
	return nil
}

func findSpdxIDInFile(fileAbsPath string) (string, error) {

	// get the SPDX ID if there exists the identifier(SPDX-License-Identifier) in the License file
	licenseFile := filepath.Join(fileAbsPath, "LICENSE")
	args := []string{fmt.Sprintf("/%s/", REGEX_LICENSE_IDENTIFIER), licenseFile}

	cmd := exec.Command("awk", args...)
	resBytes, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println("error awk: ", err.Error())
		return "", err
	}

	spdxIdFound := string(resBytes)

	// if no SPDX ientifier found then search for the ID in the Licenses File
	if spdxIdFound == "" {
		return findSpdxIDFromLicenseFile(licenseFile)
	}

	// valid SPDX identifiers
	// // SPDX-License-Identifier: MIT
	// /* SPDX-License-Identifier: MIT OR Apache-2.0 */
	// # SPDX-License-Identifer: GPL-2.0-or-later
	if bytes.Contains(resBytes, []byte("/*")) {
		resBytes = bytes.Replace(resBytes, []byte("/*"), []byte(""), 1)
		resBytes = bytes.Replace(resBytes, []byte("*/"), []byte(""), 1)
	}

	spdxID := bytes.Split(resBytes, []byte(":"))[1]
	spdxID = bytes.TrimSpace(spdxID)

	return string(spdxID), nil
}

// Read the given LICENSE file to retreieve the SPDX ID that matches
// returns empty string if error or no match
func findSpdxIDFromLicenseFile(licenseFile string) (string, error) {
	args := []string{"-4", licenseFile}
	resBytes, err := exec.Command("head", args...).Output()
	if err != nil {
		fmt.Println("error head license file: ", err.Error())
		return "", err
	}
	if len(resBytes) == 0 {
		return "", fmt.Errorf("error: no LICENSE text found in file")

	}

	outSplit := bytes.Split(resBytes, []byte("\n"))
	for i, s := range outSplit {
		outSplit[i] = bytes.TrimSpace(s)
	}

	resBytes = bytes.Join(outSplit, []byte(" "))
	gotSpdxID := searchSpdxIDLicensesList(resBytes)
	if gotSpdxID == "" {
		return "", fmt.Errorf("error: could not find SPDX ID")
	}

	return gotSpdxID, nil
}

// searchSpdxIDLicensesList loops over the SPDX licenses list trying to find the ID from the given copyright text
func searchSpdxIDLicensesList(textBytes []byte) string {

	// Done this way as Apache Copyright text does not contains License name
	if bytes.Contains(textBytes, []byte("1995-1999 The Apache Group")) {
		return "Apache-1.0"
	} else if bytes.Contains(textBytes, []byte("Apache License Version 2.0, January 2004")) {
		return "Apache-2.0"
	}

	for _, li := range licensesListSPDX {
		if bytes.Contains([]byte(li.Name), textBytes) {
			return li.LicenseID
		}
	}
	return ""
}
