package main

import (
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const configFileName = ".config/.ecm-distro-tools.json"

func configCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage the k3s release configuration file",
	}
	cmd.PersistentFlags().StringP("path", "p", defaultConfigPath(), "ecm distro tools config file path")
	cmd.AddCommand(configInitcommand())
	return cmd
}

func defaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		logrus.Fatal("failed to get user home directory", err)
	}
	return filepath.Join(home, configFileName)
}
