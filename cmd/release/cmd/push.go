package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/google/go-github/v39/github"
	"github.com/spf13/cobra"

	"github.com/rancher/ecm-distro-tools/release/charts"
	"github.com/rancher/ecm-distro-tools/release/k3s"
	"github.com/rancher/ecm-distro-tools/repository"
)

var pushCmd = &cobra.Command{
	Use:   "push",
	Short: "Push things from your local to a remote",
}

var pushK3sCmd = &cobra.Command{
	Use:   "k3s [version]",
	Short: "Push k3s artifacts",
}

var pushK3sTagsCmd = &cobra.Command{
	Use:   "tags [version]",
	Short: "Push k3s-io/kubernetes tags to remote",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.New("expected at least one argument: [version]")
		}
		version := args[0]
		k3sRelease, found := rootConfig.K3s.Versions[version]
		if !found {
			return errors.New("verify your config file, version not found: " + version)
		}
		ctx := context.Background()
		ghClient := repository.NewGithub(ctx, rootConfig.Auth.GithubToken)
		return k3s.PushTags(ghClient, &k3sRelease, rootConfig.User, rootConfig.Auth.SSHKeyPath)
	},
}

var pushChartsCmd = &cobra.Command{
	Use:     "charts [branch-line] [debug (optional)]",
	Short:   "Push charts updates to remote upstream charts repository",
	Example: "release push charts 2.9",
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if err := validateChartConfig(); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		if len(args) == 0 {
			return rootConfig.Charts.BranchLines, cobra.ShellCompDirectiveNoFileComp
		}

		return nil, cobra.ShellCompDirectiveNoFileComp
	},
	Args: func(cmd *cobra.Command, args []string) error {
		if err := validateChartConfig(); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		if len(args) < 1 {
			return errors.New("expected 1 argument: [branch-line]")
		}

		var (
			releaseBranch string
			found         bool
		)

		found = charts.IsBranchAvailable(args[0], rootConfig.Charts.BranchLines)
		if !found {
			return errors.New("release branch not available: " + releaseBranch)
		}

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		var (
			releaseBranch string // given release branch

			ctx context.Context // background context
			t   string          // token
			ghc *github.Client  // github client
		)

		// arguments
		releaseBranch = charts.MountReleaseBranch(args[0])
		debug, err := cmd.Flags().GetBool("debug")
		if err != nil {
			return err
		}

		ctx = context.Background()
		t = rootConfig.Auth.GithubToken
		ghc = repository.NewGithub(ctx, t)

		prURL, err := charts.Push(ctx, rootConfig.Charts, rootConfig.User, ghc, releaseBranch, t, debug)
		if err != nil {
			return err
		}

		fmt.Printf("Pull request created: %s\n", prURL)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(pushCmd)
	pushCmd.AddCommand(pushK3sCmd)
	pushCmd.AddCommand(pushChartsCmd)
	pushK3sCmd.AddCommand(pushK3sTagsCmd)
}
