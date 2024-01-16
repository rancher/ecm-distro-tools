package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/rancher/ecm-distro-tools/release/rke2"
	"github.com/rancher/ecm-distro-tools/repository"
	"github.com/spf13/cobra"
)

var (
	alpineVersion  *string
	releaseVersion *string
	rcVersion      *string
	rpmVersion     *int
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
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(*releaseVersion) != 2 {
			return errors.New("invalid release version")
		}

		ctx := context.Background()
		client := repository.NewGithub(ctx, rootConfig.Auth.GithubToken)

		switch args[0] {
		case "image-build-base":
			if err := rke2.ImageBuildBaseRelease(ctx, client, *alpineVersion, *dryRun); err != nil {
				return err
			}
		case "image-build-kubernetes":
			now := time.Now().UTC().Format("20060201")
			suffix := "+rke2" + *releaseVersion + "-build" + now

			if *dryRun {
				fmt.Println("dry-run:")
				for _, version := range rootConfig.RKE2.Versions {
					fmt.Println("\t" + version + suffix)
				}
			} else {
				for _, version := range rootConfig.RKE2.Versions {
					cro := repository.CreateReleaseOpts{
						Owner:      "rancher",
						Repo:       "image-build-kubernetes",
						Branch:     "master",
						Name:       version + suffix,
						Prerelease: false,
					}
					if _, err := repository.CreateRelease(ctx, client, &cro); err != nil {
						return err
					}

					fmt.Println("tag " + version + suffix + " created successfully")
				}
			}
		case "rke2":
			//
		case "rpm":
			if len(args) == 1 {
				return errors.New("invalid rpm tag. expected {testinglatest|stable}")
			}

			rpmTag := fmt.Sprintf("+rke2%s.%s.%d", *releaseVersion, args[1], *rpmVersion)
			if *rcVersion != "" {
				rpmTag = fmt.Sprintf("+rke2%s-rc%s.%s.%d", *releaseVersion, *rcVersion, args[1], *rpmVersion)
			}

			if *dryRun {
				fmt.Println("(dry-run)\n\nTagging github.com/rancher/rke2-packaging: \n")
				for _, version := range rootConfig.RKE2.Versions {
					fmt.Println("\t" + version + rpmTag)
				}
			} else {
				for _, version := range rootConfig.RKE2.Versions {
					cro := repository.CreateReleaseOpts{
						Owner:      "rancher",
						Repo:       "rke2-packaging",
						Branch:     "master",
						Name:       version + rpmTag,
						Prerelease: false,
					}
					if _, err := repository.CreateRelease(ctx, client, &cro); err != nil {
						return err
					}
				}
			}
		default:
			return errors.New("unrecognized resource")
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(tagCmd)

	tagCmd.AddCommand(k3sTagSubCmd)
	tagCmd.AddCommand(rke2TagSubCmd)

	dryRun = tagCmd.PersistentFlags().BoolP("dry-run", "d", false, "Dry run")

	alpineVersion = rke2TagSubCmd.Flags().StringP("alpine-version", "a", "", "Alpine version")
	releaseVersion = rke2TagSubCmd.Flags().StringP("release-version", "r", "r1", "Release version")
	rcVersion = rke2TagSubCmd.Flags().String("rc", "", "RC version")
	rpmVersion = rke2TagSubCmd.Flags().Int("rpm-version", 0, "RPM version")

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// tagCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:

}
