package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/google/go-containerregistry/pkg/name"
	reg "github.com/rancher/ecm-distro-tools/registry"
	"github.com/rancher/ecm-distro-tools/release"
	"github.com/rancher/ecm-distro-tools/release/rke2"
	"github.com/rancher/ecm-distro-tools/repository"
	"github.com/spf13/cobra"
)

const (
	ossRegistry = "docker.io"
)

func displayResults(results []rke2.ImageStatus) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)
	defer w.Flush()

	fmt.Fprintln(w, "image\toss\tprime\tsig\tamd64\tarm64\twin")
	fmt.Fprintln(w, "-----\t---\t-----\t---\t-----\t-----\t-------")

	for _, result := range results {
		ossStatus := boolToMark(result.OSSImage.Exists)
		primeStatus := "✗"
		if result.PrimeImage.Exists {
			primeStatus = "✓"
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			formatImageRef(result.Reference),
			ossStatus,
			primeStatus,
			"?", // sigstore not implemented
			archStatus(result.ExpectsLinuxAmd64, result.OSSImage, result.PrimeImage, reg.Platform{OS: "linux", Architecture: "amd64"}),
			archStatus(result.ExpectsLinuxArm64, result.OSSImage, result.PrimeImage, reg.Platform{OS: "linux", Architecture: "arm64"}),
			windowsStatus(result.ExpectsWindows, result.OSSImage.Exists && result.PrimeImage.Exists),
		)
	}

	return nil
}

func archStatus(expected bool, ossInfo, primeInfo reg.Image, platform reg.Platform) string {
	if !expected {
		return "-"
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
	return fmt.Sprintf("%s:%s", ref.Context().RepositoryStr(), ref.Identifier())
}

func boolToMark(b bool) string {
	if b {
		return "✓"
	}
	return "✗"
}

var inspectCmd = &cobra.Command{
	Use:   "inspect [version]",
	Short: "Inspect release artifacts",
	Long: `Inspect release artifacts for a given version.
Currently supports inspecting the image list for published rke2 releases.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.New("expected at least one argument: [version]")
		}

		ctx := context.Background()
		gh := repository.NewGithub(ctx, rootConfig.Auth.GithubToken)
		filesystem, err := release.NewFS(ctx, gh, "rancher", "rke2", args[0])
		if err != nil {
			return err
		}

		ossClient := reg.NewClient(ossRegistry, debug)

		var primeClient *reg.Client
		if rootConfig.PrimeRegistry != "" {
			primeClient = reg.NewClient(rootConfig.PrimeRegistry, debug)
		}

		inspector := rke2.NewReleaseInspector(filesystem, ossClient, primeClient, debug)

		results, err := inspector.InspectRelease(ctx, args[0])
		if err != nil {
			return err
		}

		return displayResults(results)
	},
}

func init() {
	rootCmd.AddCommand(inspectCmd)
}
