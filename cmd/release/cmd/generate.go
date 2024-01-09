package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/rancher/ecm-distro-tools/release"
	"github.com/rancher/ecm-distro-tools/repository"
	"github.com/spf13/cobra"
)

var (
	prevMilestone *string
	milestone     *string
)

// generateCmd represents the generate command
var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 || len(args) > 2 {
			rootCmd.Help()
			os.Exit(0)
		}

		ctx := context.Background()
		client := repository.NewGithub(ctx, rootConfig.Auth.GithubToken)

		var (
			owner string
			repo  string
		)

		switch args[0] {
		case "k3s":
			if args[1] == "release-notes" {
				owner = "k3s-io"
				repo = "k3s"
			}
		case "rke2":
			if args[1] == "release-notes" {
				owner = "rancher"
				repo = "rke2"
			}
		default:
			rootCmd.Help()
			os.Exit(0)
		}

		notes, err := release.GenReleaseNotes(ctx, owner, repo, *milestone, *prevMilestone, client)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		fmt.Print(notes.String())
	},
}

func init() {
	rootCmd.AddCommand(generateCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// generateCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	prevMilestone = generateCmd.Flags().StringP("prev-milestone", "p", "", "Previous Milestone")
	milestone = generateCmd.Flags().StringP("milestone", "m", "", "Milestone")
}
