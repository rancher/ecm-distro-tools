package main

import "github.com/rancher/ecm-distro-tools/cmd/prbuilder/cmd"

var version = "development"

func main() {
	cmd.SetVersion(version)
	cmd.Execute()
}
