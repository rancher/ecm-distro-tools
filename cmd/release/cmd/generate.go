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
	},
}

var k3sGenerateSubCmd = &cobra.Command{
	Use:   "k3s",
	Short: "",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		client := repository.NewGithub(ctx, rootConfig.Auth.GithubToken)

		notes, err := release.GenReleaseNotes(ctx, "k3s-io", "k3s", *milestone, *prevMilestone, client)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		fmt.Print(notes.String())
	},
}

var rke2GenerateSubCmd = &cobra.Command{
	Use:   "rke2",
	Short: "",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		client := repository.NewGithub(ctx, rootConfig.Auth.GithubToken)

		notes, err := release.GenReleaseNotes(ctx, "rancher", "rke2", *milestone, *prevMilestone, client)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		fmt.Print(notes.String())
	},
}

func init() {
	rootCmd.AddCommand(generateCmd)

	generateCmd.AddCommand(k3sGenerateSubCmd)
	generateCmd.AddCommand(rke2GenerateSubCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// generateCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	prevMilestone = generateCmd.Flags().StringP("prev-milestone", "p", "", "Previous Milestone")
	milestone = generateCmd.Flags().StringP("milestone", "m", "", "Milestone")
}
