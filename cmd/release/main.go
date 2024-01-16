package main

import "github.com/rancher/ecm-distro-tools/cmd/release/cmd"

var version = "development"

func main() {
	cmd.SetVersion(version)
	cmd.Execute()
}
