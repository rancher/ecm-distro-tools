package main

import (
	"github.com/rancher/ecm-distro-tools/repository"
	"github.com/spf13/cobra"
)

var configFilePath string

func configInitcommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Generate a k3s release configuration file",
		RunE:  generateConfig,
	}
	cmd.PersistentFlags().StringVarP(&configFilePath, "path", "p", defaultConfigPath(), "ecm distro tools config file path, default is $HOME/.config/ecm-distro-tools/config.json")
	return cmd
}

func generateConfig(cmd *cobra.Command, args []string) error {
	return repository.GenerateConfig(configFilePath)
}
