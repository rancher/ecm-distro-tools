package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
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

		ghToken := os.Getenv("GITHUB_TOKEN")
		if ghToken == "" {
			return errors.New("GITHUB_TOKEN env is empty")
		}

		ghClient := repository.NewGithub(ctx, ghToken)

		return verify.GA(ctx, ghClient, owner, *repo, version)
	},
}

func init() {
	rootCmd.AddCommand(verifyCmd)
	verifyCmd.AddCommand(verifyGACmd)

	verifyGACmd.Flags().StringVar(repo, "repo", "", "Repository name")
	if err := verifyGACmd.MarkFlagRequired("repo"); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	verifyGACmd.Flags().StringVar(&owner, "owner", "", "Repository owner")
	if err := verifyGACmd.MarkFlagRequired("owner"); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}
