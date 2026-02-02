package cmd

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/rancher/ecm-distro-tools/release/verify"
	"github.com/rancher/ecm-distro-tools/repository"
	"github.com/spf13/cobra"
)

var verifyCmd = &cobra.Command{
	Use:   "verify",
	Short: "Variety of release railguards.",
}

var verifyGACmd = &cobra.Command{
	Use:   "ga [version]",
	Short: "Check if distro's GA release contains at least one previous RC release and for RPMs contains the required previous release (testing -> latest -> stable)",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.New("expected at least one argument: [version]")
		}

		version := args[0]

		// if the version is an RC/Alpha, we can just return nil
		if strings.Contains(version, "-rc") ||
			strings.Contains(version, "-alpha") {
			fmt.Printf("The release '%s' is not GA, skipping verification...", version)
			return nil
		}
		ctx := context.Background()
		ghClient := repository.NewGithub(ctx, rootConfig.Auth.GithubToken)

		return verify.GA(ctx, ghClient, version)
	},
}

func init() {
	rootCmd.AddCommand(verifyCmd)
	verifyCmd.AddCommand(verifyGACmd)
}
