package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/go-github/v78/github"
	"github.com/rancher/ecm-distro-tools/release"
	"github.com/rancher/ecm-distro-tools/release/k3s"
	"github.com/rancher/ecm-distro-tools/release/kdm"
	"github.com/rancher/ecm-distro-tools/release/metrics"
	"github.com/rancher/ecm-distro-tools/release/prime"
	"github.com/rancher/ecm-distro-tools/release/rancher"
	"github.com/rancher/ecm-distro-tools/repository"
	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"
)

const defaultConcurrencyLimit = 3

var (
	k3sPrevMilestone string
	k3sMilestone     string

	dashboardPrevMilestone string
	dashboardMilestone     string

	cliPrevMilestone string
	cliMilestone     string

	concurrencyLimit                      int
	imagesListURL                         string
	registry                              string
	ignoreImages                          []string
	checkImages                           []string
	registries                            []string
	username                              string
	password                              string
	rancherMissingImagesJSONOutput        bool
	rke2PrevMilestone                     string
	rke2Milestone                         string
	rancherArtifactsIndexWriteToPath      string
	rancherArtifactsDir                   string
	rancherArtifactsIndexIgnoreVersions   []string
	rancherImagesDigestsOutputFile        string
	rancherImagesDigestsRegistry          string
	rancherImagesDigestsImagesURL         string
	rancherSyncImages                     []string
	rancherSourceRegistry                 string
	rancherTargetRegistry                 string
	rancherSyncConfigOutputPath           string
	rancherMetricsRancherReleasesFilePath string
	rancherMetricsWorkflowsFilePath       string
	rancherMetricsPrimeReleasesFilePath   string
	releases                              []string
)

// generateCmd represents the generate command
var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Various utilities to generate release artifacts",
}

var k3sGenerateSubCmd = &cobra.Command{
	Use:   "k3s",
	Short: "Generate k3s related artifacts",
}

var k3sGenerateReleaseNotesSubCmd = &cobra.Command{
	Use:   "release-notes",
	Short: "Generate k3s release notes",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		client := repository.NewGithub(ctx, rootConfig.Auth.GithubToken)

		notes, err := release.GenReleaseNotes(ctx, "k3s-io", "k3s", k3sMilestone, k3sPrevMilestone, client)
		if err != nil {
			return err
		}

		fmt.Print(notes.String())

		return nil
	},
}

var k3sGenerateTagsSubCmd = &cobra.Command{
	Use:   "tags [version]",
	Short: "Generate k8s tags for a given version",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.New("expected at least one argument: [version]")
		}
		version := args[0]
		k3sRelease, found := rootConfig.K3s.Versions[version]
		if !found {
			return NewVersionNotFoundError(version, "k3s")
		}
		ctx := context.Background()
		ghClient := repository.NewGithub(ctx, rootConfig.Auth.GithubToken)
		return k3s.GenerateTags(ctx, ghClient, &k3sRelease, rootConfig.User, rootConfig.Auth.SSHKeyPath)
	},
}

var rke2GenerateSubCmd = &cobra.Command{
	Use:   "rke2",
	Short: "Generate rke2 related artifacts",
}

var rke2GenerateReleaseNotesSubCmd = &cobra.Command{
	Use:   "release-notes",
	Short: "Generate rke2 release notes",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		client := repository.NewGithub(ctx, rootConfig.Auth.GithubToken)

		notes, err := release.GenReleaseNotes(ctx, "rancher", "rke2", rke2Milestone, rke2PrevMilestone, client)
		if err != nil {
			return err
		}

		fmt.Print(notes.String())

		return nil
	},
}

var rancherGenerateSubCmd = &cobra.Command{
	Use:   "rancher",
	Short: "Generate rancher related artifacts",
}

var rancherGenerateArtifactsIndexSubCmd = &cobra.Command{
	Use:   "artifacts-index",
	Short: "Generate artifacts index page",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		cfg, err := config.LoadDefaultConfig(ctx,
			config.WithCredentialsProvider(aws.AnonymousCredentials{}),
			config.WithDefaultRegion("us-east-1"),
		)
		if err != nil {
			return err
		}

		var lister prime.ArtifactLister

		if rancherArtifactsDir != "" {
			lister = prime.NewArtifactDir(rancherArtifactsDir)
		} else {
			client := s3.NewFromConfig(cfg, func(o *s3.Options) {
				o.BaseEndpoint = aws.String("https://s3.us-east-1.amazonaws.com")
			})
			lister = prime.NewArtifactBucket(client)
		}

		// knownOmissions contains versions that should always be omitted for various reasons
		knownOmissions := []string{
			"v2.6.4", // test version of rancher
		}
		ignoreVersions := append(rancherArtifactsIndexIgnoreVersions, knownOmissions...)

		return prime.GenerateArtifactsIndex(ctx, rancherArtifactsIndexWriteToPath, ignoreVersions, lister)
	},
}

var rancherGenerateImagesLocationsSubCmd = &cobra.Command{
	Use:   "images-locations",
	Short: "Generate a json with images locations and if there any missing images",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(checkImages) == 0 {
			if imagesListURL == "" {
				return errors.New("either --images-list-url or --check-images must be provided")
			}

			rancherImages, err := rancher.ImagesFromArtifact(imagesListURL)
			if err != nil {
				return errors.New("failed to get rancher images: " + err.Error())
			}

			checkImages = append(checkImages, rancherImages...)
		}

		imagesLocations, err := rancher.ImagesLocations(username, password, concurrencyLimit, checkImages, ignoreImages, registry, registries)
		if err != nil {
			return err
		}
		b, err := json.MarshalIndent(imagesLocations, "", " ")
		if err != nil {
			return err
		}
		fmt.Println(string(b))
		return nil
	},
}

var rancherGenerateMissingImagesListSubCmd = &cobra.Command{
	Use:   "missing-images-list",
	Short: "Generate a missing images list",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(checkImages) == 0 {
			if imagesListURL == "" {
				return errors.New("either --images-list-url or --check-images must be provided")
			}

			rancherImages, err := rancher.ImagesFromArtifact(imagesListURL)
			if err != nil {
				return errors.New("failed to get rancher images: " + err.Error())
			}

			checkImages = append(checkImages, rancherImages...)
		}

		missingImages, err := rancher.MissingImagesFromRegistry(username, password, registry, concurrencyLimit, checkImages, ignoreImages)
		if err != nil {
			return err
		}
		if len(missingImages) != 0 && !rancherMissingImagesJSONOutput {
			return errors.New("found missing images: " + strings.Join(missingImages, ","))
		}
		if rancherMissingImagesJSONOutput {
			if len(missingImages) == 0 {
				missingImages = make([]string, 0)
			}
			b, err := json.MarshalIndent(missingImages, "", " ")
			if err != nil {
				return err
			}
			fmt.Println(string(b))
		}
		return nil
	},
}

var rancherGenerateDockerImagesDigestsSubCmd = &cobra.Command{
	Use:   "docker-images-digests",
	Short: "Generate a file with images digests from an images list",
	RunE: func(cmd *cobra.Command, args []string) error {
		return rancher.GenerateDockerImageDigests(rancherImagesDigestsOutputFile, rancherImagesDigestsImagesURL, rancherImagesDigestsRegistry, username, password, verbose)
	},
}

var rancherGenerateImagesSyncConfigSubCmd = &cobra.Command{
	Use:   "images-sync-config",
	Short: "Generate a regsync config file for images sync",
	RunE: func(cmd *cobra.Command, args []string) error {
		return rancher.GenerateImagesSyncConfig(rancherSyncImages, rancherSourceRegistry, rancherTargetRegistry, rancherSyncConfigOutputPath)
	},
}

var rancherGenerateMetricsSubCmd = &cobra.Command{
	Use:   "metrics",
	Short: "Generate rancher release metrics",
	RunE: func(cmd *cobra.Command, args []string) error {
		var rancherReleases []github.RepositoryRelease
		var primeReleases []github.RepositoryRelease
		var workflows []github.WorkflowRun

		rancherReleasesFile, err := os.ReadFile(rancherMetricsRancherReleasesFilePath)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(rancherReleasesFile, &rancherReleases); err != nil {
			return err
		}

		primeReleasesFile, err := os.ReadFile(rancherMetricsPrimeReleasesFilePath)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(primeReleasesFile, &primeReleases); err != nil {
			return err
		}

		workflowsFile, err := os.ReadFile(rancherMetricsWorkflowsFilePath)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(workflowsFile, &workflows); err != nil {
			return err
		}

		metrics, err := metrics.ExtractMetrics(rancherReleases, primeReleases, workflows)
		if err != nil {
			return err
		}

		b, err := json.MarshalIndent(metrics, "", " ")
		if err != nil {
			return err
		}

		fmt.Println(string(b))

		return nil
	},
}

var uiGenerateSubCmd = &cobra.Command{
	Use:   "ui",
	Short: "Generate ui related artifacts",
}

var uiGenerateReleaseNotesSubCmd = &cobra.Command{
	Use:   "release-notes",
	Short: "Generate ui release notes",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		client := repository.NewGithub(ctx, rootConfig.Auth.GithubToken)

		notes, err := release.GenReleaseNotes(ctx, "rancher", "ui", dashboardMilestone, dashboardPrevMilestone, client)
		if err != nil {
			return err
		}

		fmt.Print(notes.String())

		return nil
	},
}

var dashboardGenerateSubCmd = &cobra.Command{
	Use:   "dashboard",
	Short: "Generate dashboard related artifacts",
}

var dashboardGenerateReleaseNotesSubCmd = &cobra.Command{
	Use:   "release-notes",
	Short: "Generate dashboard release notes",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		client := repository.NewGithub(ctx, rootConfig.Auth.GithubToken)

		notes, err := release.GenReleaseNotes(ctx, "rancher", "dashboard", dashboardMilestone, dashboardPrevMilestone, client)
		if err != nil {
			return err
		}

		fmt.Print(notes.String())

		return nil
	},
}

var cliGenerateSubCmd = &cobra.Command{
	Use:   "cli",
	Short: "Generate rancher/cli related artifacts",
}

var cliGenerateReleaseNotesSubCmd = &cobra.Command{
	Use:   "release-notes",
	Short: "Generate cli release notes",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		client := repository.NewGithub(ctx, rootConfig.Auth.GithubToken)

		notes, err := release.GenReleaseNotes(ctx, "rancher", "cli", cliMilestone, cliPrevMilestone, client)
		if err != nil {
			return err
		}

		fmt.Print(notes.String())

		return nil
	},
}

var kdmGenerateSubCmd = &cobra.Command{
	Use:   "kdm",
	Short: "Generate kdm related artifacts",
}

var kdmGenerateRKE2ChartsSubCmd = &cobra.Command{
	Use:   "rke2-charts",
	Short: "Generate rke2 charts updated charts in YAML",
	RunE: func(cmd *cobra.Command, args []string) error {
		charts, err := kdm.UpdatedCharts(rke2Milestone, rke2PrevMilestone)
		if err != nil {
			return err
		}

		b, err := yaml.Marshal(charts)
		if err != nil {
			return err
		}

		fmt.Print(string(b))

		return nil
	},
}

var kdmGenerateRKE2SubCmd = &cobra.Command{
	Use:   "rke2",
	Short: "Generate rke2 KDM artifacts updating channels-rke2.yaml file",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if len(releases) == 0 {
			return errors.New("'releases' flag is empty")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := kdm.UpdateRKE2Channels(releases); err != nil {
			return err
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(generateCmd)

	k3sGenerateSubCmd.AddCommand(k3sGenerateReleaseNotesSubCmd)
	k3sGenerateSubCmd.AddCommand(k3sGenerateTagsSubCmd)

	rke2GenerateSubCmd.AddCommand(rke2GenerateReleaseNotesSubCmd)

	rancherGenerateSubCmd.AddCommand(rancherGenerateArtifactsIndexSubCmd)
	rancherGenerateSubCmd.AddCommand(rancherGenerateMissingImagesListSubCmd)
	rancherGenerateSubCmd.AddCommand(rancherGenerateImagesLocationsSubCmd)
	rancherGenerateSubCmd.AddCommand(rancherGenerateDockerImagesDigestsSubCmd)
	rancherGenerateSubCmd.AddCommand(rancherGenerateImagesSyncConfigSubCmd)
	rancherGenerateSubCmd.AddCommand(rancherGenerateMetricsSubCmd)

	uiGenerateSubCmd.AddCommand(uiGenerateReleaseNotesSubCmd)
	dashboardGenerateSubCmd.AddCommand(dashboardGenerateReleaseNotesSubCmd)
	cliGenerateSubCmd.AddCommand(cliGenerateReleaseNotesSubCmd)

	kdmGenerateSubCmd.AddCommand(kdmGenerateRKE2ChartsSubCmd)
	kdmGenerateSubCmd.AddCommand(kdmGenerateRKE2SubCmd)

	generateCmd.AddCommand(k3sGenerateSubCmd)
	generateCmd.AddCommand(rke2GenerateSubCmd)
	generateCmd.AddCommand(rancherGenerateSubCmd)
	generateCmd.AddCommand(uiGenerateSubCmd)
	generateCmd.AddCommand(dashboardGenerateSubCmd)
	generateCmd.AddCommand(cliGenerateSubCmd)
	generateCmd.AddCommand(kdmGenerateSubCmd)

	// k3s release notes
	k3sGenerateReleaseNotesSubCmd.Flags().StringVarP(&k3sPrevMilestone, "prev-milestone", "p", "", "Previous Milestone")
	k3sGenerateReleaseNotesSubCmd.Flags().StringVarP(&k3sMilestone, "milestone", "m", "", "Milestone")
	if err := k3sGenerateReleaseNotesSubCmd.MarkFlagRequired("prev-milestone"); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	if err := k3sGenerateReleaseNotesSubCmd.MarkFlagRequired("milestone"); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	// rke2 release notes
	rke2GenerateReleaseNotesSubCmd.Flags().StringVarP(&rke2PrevMilestone, "prev-milestone", "p", "", "Previous Milestone")
	rke2GenerateReleaseNotesSubCmd.Flags().StringVarP(&rke2Milestone, "milestone", "m", "", "Milestone")
	if err := rke2GenerateReleaseNotesSubCmd.MarkFlagRequired("prev-milestone"); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	if err := rke2GenerateReleaseNotesSubCmd.MarkFlagRequired("milestone"); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	// ui release notes
	uiGenerateReleaseNotesSubCmd.Flags().StringVarP(&dashboardPrevMilestone, "prev-milestone", "p", "", "Previous Milestone")
	uiGenerateReleaseNotesSubCmd.Flags().StringVarP(&dashboardMilestone, "milestone", "m", "", "Milestone")
	if err := uiGenerateReleaseNotesSubCmd.MarkFlagRequired("prev-milestone"); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	if err := uiGenerateReleaseNotesSubCmd.MarkFlagRequired("milestone"); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	// dashboard release notes
	dashboardGenerateReleaseNotesSubCmd.Flags().StringVarP(&dashboardPrevMilestone, "prev-milestone", "p", "", "Previous Milestone")
	dashboardGenerateReleaseNotesSubCmd.Flags().StringVarP(&dashboardMilestone, "milestone", "m", "", "Milestone")
	if err := dashboardGenerateReleaseNotesSubCmd.MarkFlagRequired("prev-milestone"); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	if err := dashboardGenerateReleaseNotesSubCmd.MarkFlagRequired("milestone"); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	// cli release notes
	cliGenerateReleaseNotesSubCmd.Flags().StringVarP(&cliPrevMilestone, "prev-milestone", "p", "", "Previous Milestone")
	cliGenerateReleaseNotesSubCmd.Flags().StringVarP(&cliMilestone, "milestone", "m", "", "Milestone")
	if err := cliGenerateReleaseNotesSubCmd.MarkFlagRequired("prev-milestone"); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	if err := cliGenerateReleaseNotesSubCmd.MarkFlagRequired("milestone"); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	// rancher artifacts-index
	rancherGenerateArtifactsIndexSubCmd.Flags().StringSliceVarP(&rancherArtifactsIndexIgnoreVersions, "ignore-versions", "i", []string{}, "Versions to ignore on the index")
	rancherGenerateArtifactsIndexSubCmd.Flags().StringVarP(&rancherArtifactsIndexWriteToPath, "write-path", "w", ".", "Output directory, defaults to current working directory")
	rancherGenerateArtifactsIndexSubCmd.Flags().StringVarP(&rancherArtifactsDir, "dir", "d", "", "Local artifacts directory, for testing purposes")

	// rancher generate images-locations
	rancherGenerateImagesLocationsSubCmd.Flags().IntVarP(&concurrencyLimit, "concurrency-limit", "l", defaultConcurrencyLimit, "Concurrency Limit")
	rancherGenerateImagesLocationsSubCmd.Flags().StringVarP(&imagesListURL, "images-list-url", "i", "", "URL of the artifact containing all images for a given version 'rancher-images.txt' (required)")
	rancherGenerateImagesLocationsSubCmd.Flags().StringSliceVarP(&ignoreImages, "ignore-images", "g", make([]string, 0), "Images to ignore when checking for missing images without the version. e.g: rancher/rancher")
	rancherGenerateImagesLocationsSubCmd.Flags().StringSliceVarP(&checkImages, "check-images", "k", make([]string, 0), "Images to check for when checking for missing images with the version. e.g: rancher/rancher-agent:v2.9.0")
	rancherGenerateImagesLocationsSubCmd.Flags().StringSliceVarP(&registries, "registries", "r", make([]string, 0), "Registries where the images should be located")
	rancherGenerateImagesLocationsSubCmd.Flags().StringVarP(&registry, "target-registry", "t", "registry.rancher.com", "Registry where the images should be located")
	rancherGenerateImagesLocationsSubCmd.Flags().StringVarP(&username, "username", "u", "", "Docker registry username")
	rancherGenerateImagesLocationsSubCmd.Flags().StringVarP(&password, "password", "p", "", "Docker registry password")

	// rancher generate missing-images-list
	rancherGenerateMissingImagesListSubCmd.Flags().IntVarP(&concurrencyLimit, "concurrency-limit", "l", defaultConcurrencyLimit, "Concurrency Limit")
	rancherGenerateMissingImagesListSubCmd.Flags().BoolVarP(&rancherMissingImagesJSONOutput, "json", "j", false, "JSON Output")
	rancherGenerateMissingImagesListSubCmd.Flags().StringVarP(&imagesListURL, "images-list-url", "i", "", "URL of the artifact containing all images for a given version 'rancher-images.txt' (required)")
	rancherGenerateMissingImagesListSubCmd.Flags().StringSliceVarP(&ignoreImages, "ignore-images", "g", make([]string, 0), "Images to ignore when checking for missing images without the version. e.g: rancher/rancher")
	rancherGenerateMissingImagesListSubCmd.Flags().StringSliceVarP(&checkImages, "check-images", "k", make([]string, 0), "Images to check for when checking for missing images with the version. e.g: rancher/rancher-agent:v2.9.0")
	rancherGenerateMissingImagesListSubCmd.Flags().StringVarP(&registry, "registry", "r", "registry.rancher.com", "Registry where the images should be located")
	rancherGenerateMissingImagesListSubCmd.Flags().StringVarP(&username, "username", "u", "", "Docker registry username")
	rancherGenerateMissingImagesListSubCmd.Flags().StringVarP(&password, "password", "p", "", "Docker registry password")

	// rancher generate docker-images-digests
	rancherGenerateDockerImagesDigestsSubCmd.Flags().StringVarP(&rancherImagesDigestsOutputFile, "output-file", "o", "", "Output file with images digests")
	rancherGenerateDockerImagesDigestsSubCmd.Flags().StringVarP(&username, "username", "u", "", "Docker registry username")
	rancherGenerateDockerImagesDigestsSubCmd.Flags().StringVarP(&password, "password", "p", "", "Docker registry password")
	if err := rancherGenerateDockerImagesDigestsSubCmd.MarkFlagRequired("output-file"); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	rancherGenerateDockerImagesDigestsSubCmd.Flags().StringVarP(&rancherImagesDigestsImagesURL, "images-url", "i", "", "Images list artifact URL")
	if err := rancherGenerateDockerImagesDigestsSubCmd.MarkFlagRequired("images-url"); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	rancherGenerateDockerImagesDigestsSubCmd.Flags().StringVarP(&rancherImagesDigestsRegistry, "registry", "r", "", "Docker Registry e.g: docker.io")
	if err := rancherGenerateDockerImagesDigestsSubCmd.MarkFlagRequired("registry"); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	// rancher generate images-sync-config
	rancherGenerateImagesSyncConfigSubCmd.Flags().StringSliceVarP(&rancherSyncImages, "images", "k", make([]string, 0), "List of images to sync to a registry")
	rancherGenerateImagesSyncConfigSubCmd.Flags().StringVarP(&rancherSourceRegistry, "source-registry", "s", "", "Source registry, where the images are located")
	rancherGenerateImagesSyncConfigSubCmd.Flags().StringVarP(&rancherTargetRegistry, "target-registry", "t", "", "Target registry, where the images should be synced to")
	rancherGenerateImagesSyncConfigSubCmd.Flags().StringVarP(&rancherSyncConfigOutputPath, "output", "o", "./config.yaml", "Output path of the generated config file")
	if err := rancherGenerateImagesSyncConfigSubCmd.MarkFlagRequired("images"); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	if err := rancherGenerateImagesSyncConfigSubCmd.MarkFlagRequired("source-registry"); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	if err := rancherGenerateImagesSyncConfigSubCmd.MarkFlagRequired("target-registry"); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	// rancher generate metrics
	rancherGenerateMetricsSubCmd.Flags().StringVarP(&rancherMetricsRancherReleasesFilePath, "rancher-releases-file", "r", "", "Path to the releases file")
	rancherGenerateMetricsSubCmd.Flags().StringVarP(&rancherMetricsWorkflowsFilePath, "workflows-file", "w", "", "Path to the workflows file")
	rancherGenerateMetricsSubCmd.Flags().StringVarP(&rancherMetricsPrimeReleasesFilePath, "prime-releases-file", "p", "", "Path to the prime releases file")
	if err := rancherGenerateMetricsSubCmd.MarkFlagRequired("rancher-releases-file"); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	if err := rancherGenerateMetricsSubCmd.MarkFlagRequired("workflows-file"); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	if err := rancherGenerateMetricsSubCmd.MarkFlagRequired("prime-releases-file"); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	// kdm charts
	kdmGenerateRKE2ChartsSubCmd.Flags().StringVarP(&rke2PrevMilestone, "prev-milestone", "p", "", "Previous Milestone")
	kdmGenerateRKE2ChartsSubCmd.Flags().StringVarP(&rke2Milestone, "milestone", "m", "", "Milestone")
	if err := kdmGenerateRKE2ChartsSubCmd.MarkFlagRequired("prev-milestone"); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	if err := kdmGenerateRKE2ChartsSubCmd.MarkFlagRequired("milestone"); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	kdmGenerateRKE2SubCmd.Flags().StringSliceVarP(&releases, "releases", "r", make([]string, 0), "List of releases")
	if err := kdmGenerateRKE2SubCmd.MarkFlagRequired("releases"); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}
