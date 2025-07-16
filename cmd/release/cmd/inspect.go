package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/google/go-containerregistry/pkg/name"
	reg "github.com/rancher/ecm-distro-tools/registry"
	"github.com/rancher/ecm-distro-tools/release"
	"github.com/rancher/ecm-distro-tools/release/k3s"
	"github.com/rancher/ecm-distro-tools/release/rke2"
	"github.com/rancher/ecm-distro-tools/repository"
	"github.com/spf13/cobra"
)

const (
	ossRegistry = "docker.io"
)

type RKE2Contents struct {
	Images []rke2.Image
}

type K3SContents struct {
	Images []k3s.Image
}

func archStatus(expected bool, ossInfo, primeInfo reg.Image, platform reg.Platform) string {
	if !expected {
		return "-"
	}

	if primeInfo.Platforms == nil {
		if ossInfo.Platforms[platform] {
			return "✓"
		}
		return "✗"
	}
	hasArch := ossInfo.Platforms[platform] && primeInfo.Platforms[platform]
	if hasArch {
		return "✓"
	}
	return "✗"
}

func windowsStatus(expected, exists bool) string {
	if !expected {
		return "-"
	}
	if exists {
		return "✓"
	}
	return "✗"
}

func formatImageRef(ref name.Reference) string {
	return ref.Context().RepositoryStr() + ":" + ref.Identifier()
}

func (r RKE2Contents) Table(w io.Writer) {
	results := r.Images
	sort.Slice(results, func(i, j int) bool {
		return formatImageRef(results[i].Reference) < formatImageRef(results[j].Reference)
	})

	missingCount := 0
	for _, result := range results {
		if !result.OSSImage.Exists || !result.PrimeImage.Exists {
			missingCount++
		}
	}
	if missingCount > 0 {
		fmt.Fprintln(w, missingCount, "incomplete images")
	} else {
		fmt.Fprintln(w, "all images OK")
	}

	tw := tabwriter.NewWriter(w, 0, 8, 2, ' ', 0)
	defer tw.Flush()

	fmt.Fprintln(tw, "image\toss\tprime\tsig\tamd64\tarm64\twin")
	fmt.Fprintln(tw, "-----\t---\t-----\t---\t-----\t-----\t-------")

	for _, result := range results {
		ossStatus := "✗"
		if result.OSSImage.Exists {
			ossStatus = "✓"
		}
		primeStatus := "✗"
		if result.PrimeImage.Exists {
			primeStatus = "✓"
		}
		tw.Write([]byte(strings.Join([]string{
			formatImageRef(result.Reference),
			ossStatus,
			primeStatus,
			"?", // sigstore not implemented
			archStatus(result.ExpectsLinuxAmd64, result.OSSImage, result.PrimeImage, reg.Platform{OS: "linux", Architecture: "amd64"}),
			archStatus(result.ExpectsLinuxArm64, result.OSSImage, result.PrimeImage, reg.Platform{OS: "linux", Architecture: "arm64"}),
			windowsStatus(result.ExpectsWindows, result.OSSImage.Exists && result.PrimeImage.Exists),
			"",
		}, "\t") + "\n"))
	}
}

func (k K3SContents) Table(w io.Writer) {
	results := k.Images
	sort.Slice(results, func(i, j int) bool {
		return formatImageRef(results[i].Reference) < formatImageRef(results[j].Reference)
	})

	missingCount := 0
	for _, result := range results {
		if !result.OSSImage.Exists {
			missingCount++
		}
	}
	if missingCount > 0 {
		fmt.Fprintln(w, missingCount, "incomplete images")
	} else {
		fmt.Fprintln(w, "all images OK")
	}

	tw := tabwriter.NewWriter(w, 0, 8, 2, ' ', 0)
	defer tw.Flush()

	fmt.Fprintln(tw, "image\tamd64\tarm64")
	fmt.Fprintln(tw, "-----\t-----\t-----")

	for _, result := range results {
		tw.Write([]byte(strings.Join([]string{
			formatImageRef(result.Reference),
			archStatus(result.ExpectsLinuxAmd64, result.OSSImage, reg.Image{}, reg.Platform{OS: "linux", Architecture: "amd64"}),
			archStatus(result.ExpectsLinuxArm64, result.OSSImage, reg.Image{}, reg.Platform{OS: "linux", Architecture: "arm64"}),
			"",
		}, "\t") + "\n"))
	}
}

func (r RKE2Contents) CSV(w io.Writer) {
	results := r.Images
	sort.Slice(results, func(i, j int) bool {
		return formatImageRef(results[i].Reference) < formatImageRef(results[j].Reference)
	})

	fmt.Fprintln(w, "image,oss,prime,sig,amd64,arm64,win")

	for _, result := range results {
		ossStatus := "N"
		if result.OSSImage.Exists {
			ossStatus = "Y"
		}
		primeStatus := "N"
		if result.PrimeImage.Exists {
			primeStatus = "Y"
		}

		amd64Status := ""
		if result.ExpectsLinuxAmd64 {
			if result.OSSImage.Platforms[reg.Platform{OS: "linux", Architecture: "amd64"}] &&
				result.PrimeImage.Platforms[reg.Platform{OS: "linux", Architecture: "amd64"}] {
				amd64Status = "Y"
			} else {
				amd64Status = "N"
			}
		}

		arm64Status := ""
		if result.ExpectsLinuxArm64 {
			if result.OSSImage.Platforms[reg.Platform{OS: "linux", Architecture: "arm64"}] &&
				result.PrimeImage.Platforms[reg.Platform{OS: "linux", Architecture: "arm64"}] {
				arm64Status = "Y"
			} else {
				arm64Status = "N"
			}
		}

		winStatus := ""
		if result.ExpectsWindows {
			if result.OSSImage.Exists && result.PrimeImage.Exists {
				winStatus = "Y"
			} else {
				winStatus = "N"
			}
		}

		values := []string{
			formatImageRef(result.Reference),
			ossStatus,
			primeStatus,
			"?", // sigstore not implemented
			amd64Status,
			arm64Status,
			winStatus,
		}
		fmt.Fprintln(w, strings.Join(values, ","))
	}
}

func (k K3SContents) CSV(w io.Writer) {
	results := k.Images
	sort.Slice(results, func(i, j int) bool {
		return formatImageRef(results[i].Reference) < formatImageRef(results[j].Reference)
	})

	fmt.Fprintln(w, "image,amd64,arm64")

	for _, result := range results {
		amd64Status := ""
		if result.ExpectsLinuxAmd64 {
			if result.OSSImage.Platforms[reg.Platform{OS: "linux", Architecture: "amd64"}] {
				amd64Status = "Y"
			} else {
				amd64Status = "N"
			}
		}

		arm64Status := ""
		if result.ExpectsLinuxArm64 {
			if result.OSSImage.Platforms[reg.Platform{OS: "linux", Architecture: "arm64"}] {
				arm64Status = "Y"
			} else {
				arm64Status = "N"
			}
		}

		values := []string{
			formatImageRef(result.Reference),
			amd64Status,
			arm64Status,
		}
		fmt.Fprintln(w, strings.Join(values, ","))
	}
}

var inspectCmd = &cobra.Command{
	Use:   "inspect",
	Short: "Inspect release artifacts",
	Long: `Inspect release artifacts for a given version.
Supports inspecting the image list for published k3s and rke2 releases.`,
}

var inspectK3SCmd = &cobra.Command{
	Use:   "k3s [version]",
	Short: "Inspect K3s release artifacts",
	Long:  `Inspect K3s release artifacts for a given version.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.New("expected at least one argument: [version]")
		}

		ctx := context.Background()
		version := args[0]
		return inspectK3SRelease(ctx, version, cmd)
	},
}

var inspectRKE2Cmd = &cobra.Command{
	Use:   "rke2 [version]",
	Short: "Inspect RKE2 release artifacts",
	Long:  `Inspect RKE2 release artifacts for a given version.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.New("expected at least one argument: [version]")
		}

		ctx := context.Background()
		version := args[0]
		return inspectRKE2Release(ctx, version, cmd)
	},
}

func inspectRKE2Release(ctx context.Context, version string, cmd *cobra.Command) error {
	gh := repository.NewGithub(ctx, rootConfig.Auth.GithubToken)
	releaseFs, err := release.NewFS(ctx, gh, "rancher", "rke2", version)
	if err != nil {
		return err
	}

	ossClient := reg.NewClient(ossRegistry, debug)

	var primeClient *reg.Client
	if rootConfig.PrimeRegistry != "" {
		primeClient = reg.NewClient(rootConfig.PrimeRegistry, debug)
	}

	inspector := rke2.NewReleaseInspector(releaseFs, ossClient, primeClient, debug)

	images, err := inspector.InspectRelease(ctx, version)
	if err != nil {
		return err
	}

	contents := RKE2Contents{Images: images}
	outputFormat, _ := cmd.Flags().GetString("output")
	switch outputFormat {
	case "csv":
		contents.CSV(os.Stdout)
	default:
		contents.Table(os.Stdout)
	}

	return nil
}

func inspectK3SRelease(ctx context.Context, version string, cmd *cobra.Command) error {
	gh := repository.NewGithub(ctx, rootConfig.Auth.GithubToken)
	releaseFs, err := release.NewFS(ctx, gh, "k3s-io", "k3s", version)
	if err != nil {
		return err
	}

	ossClient := reg.NewClient(ossRegistry, debug)

	inspector := k3s.NewReleaseInspector(releaseFs, ossClient, debug)

	images, err := inspector.InspectRelease(ctx, version)
	if err != nil {
		return err
	}

	contents := K3SContents{Images: images}
	outputFormat, _ := cmd.Flags().GetString("output")
	switch outputFormat {
	case "csv":
		contents.CSV(os.Stdout)
	default:
		contents.Table(os.Stdout)
	}

	return nil
}

func init() {
	rootCmd.AddCommand(inspectCmd)
	inspectCmd.AddCommand(inspectK3SCmd)
	inspectCmd.AddCommand(inspectRKE2Cmd)
	inspectK3SCmd.Flags().StringP("output", "o", "table", "Output format (table|csv)")
	inspectRKE2Cmd.Flags().StringP("output", "o", "table", "Output format (table|csv)")
}
