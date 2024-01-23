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

type TagRKE2CmdFlags struct {
	AlpineVersion  *string
	ReleaseVersion *string
	RCVersion      *string
	RPMVersion     *int
}

type TagRancherCmdFlags struct {
	Version *string
	Branch  *string
}

var (
	tagRKE2CmdFlags TagRKE2CmdFlags
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
		if len(*tagRKE2CmdFlags.AlpineVersion) != 2 {
			return errors.New("invalid release version")
		}

		ctx := context.Background()
		client := repository.NewGithub(ctx, rootConfig.Auth.GithubToken)

		switch args[0] {
		case "image-build-base":
			if err := rke2.ImageBuildBaseRelease(ctx, client, *tagRKE2CmdFlags.AlpineVersion, *dryRun); err != nil {
				return err
			}
		case "image-build-kubernetes":
			now := time.Now().UTC().Format("20060102")
			suffix := "-rke2" + *tagRKE2CmdFlags.ReleaseVersion + "-build" + now

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

			rpmTag := fmt.Sprintf("+rke2%s.%s.%d", *tagRKE2CmdFlags.ReleaseVersion, args[1], *tagRKE2CmdFlags.RPMVersion)
			if *tagRKE2CmdFlags.RCVersion != "" {
				rpmTag = fmt.Sprintf("+rke2%s-rc%s.%s.%d", *tagRKE2CmdFlags.ReleaseVersion, *tagRKE2CmdFlags.RCVersion, args[1], *tagRKE2CmdFlags.RPMVersion)
			}

			if *dryRun {
				fmt.Print("(dry-run)\n\nTagging github.com/rancher/rke2-packaging:\n\n")
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

var rancherTagSubCmd = &cobra.Command{
	Use:   "rancher",
	Short: "",
	Long:  ``,
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil
	},
}

func init() {
	rootCmd.AddCommand(tagCmd)

	tagCmd.AddCommand(k3sTagSubCmd)
	tagCmd.AddCommand(rke2TagSubCmd)
	tagCmd.AddCommand(rancherTagSubCmd)

	dryRun = tagCmd.PersistentFlags().BoolP("dry-run", "d", false, "Dry run")

	// rke2
	tagRKE2CmdFlags.AlpineVersion = rke2TagSubCmd.Flags().StringP("alpine-version", "a", "", "Alpine version")
	tagRKE2CmdFlags.ReleaseVersion = rke2TagSubCmd.Flags().StringP("release-version", "r", "r1", "Release version")
	tagRKE2CmdFlags.RCVersion = rke2TagSubCmd.Flags().String("rc", "", "RC version")
	tagRKE2CmdFlags.RPMVersion = rke2TagSubCmd.Flags().Int("rpm-version", 0, "RPM version")

	// rancher
}
