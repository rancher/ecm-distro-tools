package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/rancher/ecm-distro-tools/release/rke2"
	"github.com/rancher/ecm-distro-tools/repository"
	"github.com/spf13/cobra"
)

var (
	alpineVersion *string
)

// tagCmd represents the tag command.
var tagCmd = &cobra.Command{
	Use:   "tag",
	Short: "",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			rootCmd.Help()
			os.Exit(0)
		}
	},
}

var k3sTagSubCmd = &cobra.Command{
	Use:   "k3s",
	Short: "",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Here we are!")
	},
}

var rke2TagSubCmd = &cobra.Command{
	Use:   "rke2",
	Short: "",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		client := repository.NewGithub(ctx, rootConfig.Auth.GithubToken)

		switch args[0] {
		case "image-build-kubernetes":
			if err := rke2.ImageBuildBaseRelease(ctx, client, *alpineVersion, false); err != nil {
				fmt.Println(err)
				os.Exit(1)
			}

			fmt.Println("Successfully tagged")
		}
	},
}

func init() {
	rootCmd.AddCommand(tagCmd)

	tagCmd.AddCommand(k3sTagSubCmd)
	tagCmd.AddCommand(rke2TagSubCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// tagCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	tagCmd.Flags().StringP("alpine-version", "a", "", "Alpine version")
}
