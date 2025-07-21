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
	"github.com/rancher/ecm-distro-tools/cmd/release/config"
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
	if len(r.Images) == 0 {
		fmt.Fprintln(w, "no images found")
		return
	}

	results := r.Images
	sort.Slice(results, func(i, j int) bool {
		return formatImageRef(results[i].Reference) < formatImageRef(results[j].Reference)
	})

	registryNames := make(map[string]bool)
	for _, result := range results {
		for regName := range result.RegistryResults {
			registryNames[regName] = true
		}
	}

	var registryList []string
	for regName := range registryNames {
		registryList = append(registryList, regName)
	}
	sort.Strings(registryList)

	missingCount := 0
	for _, result := range results {
		for _, regName := range registryList {
			if img, ok := result.RegistryResults[regName]; !ok || !img.Exists {
				missingCount++
			}
		}
	}
	if missingCount > 0 {
		fmt.Fprintln(w, missingCount, "incomplete images")
	} else {
		fmt.Fprintln(w, "all images OK")
	}

	tw := tabwriter.NewWriter(w, 0, 8, 2, ' ', 0)
	defer tw.Flush()

	header := []string{"image"}
	header = append(header, registryList...)
	header = append(header, "sig", "amd64", "arm64", "win")
	fmt.Fprintln(tw, strings.Join(header, "\t"))

	separator := []string{"-----"}
	for range registryList {
		separator = append(separator, "---")
	}
	separator = append(separator, "---", "-----", "-----", "-------")
	fmt.Fprintln(tw, strings.Join(separator, "\t"))

	for _, result := range results {
		row := []string{formatImageRef(result.Reference)}

		for _, regName := range registryList {
			status := "✗"
			if img, ok := result.RegistryResults[regName]; ok && img.Exists {
				status = "✓"
			}
			row = append(row, status)
		}

		ossImg := result.OSSImage()
		primeImg := result.PrimeImage()
		row = append(row,
			"?", // sigstore not implemented
			archStatus(result.ExpectsLinuxAmd64, ossImg, primeImg, reg.Platform{OS: "linux", Architecture: "amd64"}),
			archStatus(result.ExpectsLinuxArm64, ossImg, primeImg, reg.Platform{OS: "linux", Architecture: "arm64"}),
			windowsStatus(result.ExpectsWindows, ossImg.Exists && primeImg.Exists),
		)

		fmt.Fprintln(tw, strings.Join(row, "\t"))
	}
}

func (k K3SContents) Table(w io.Writer) {
	if len(k.Images) == 0 {
		fmt.Fprintln(w, "no images found")
		return
	}

	results := k.Images
	sort.Slice(results, func(i, j int) bool {
		return formatImageRef(results[i].Reference) < formatImageRef(results[j].Reference)
	})

	registryNames := make(map[string]bool)
	for _, result := range results {
		for regName := range result.RegistryResults {
			registryNames[regName] = true
		}
	}

	var registryList []string
	for regName := range registryNames {
		registryList = append(registryList, regName)
	}
	sort.Strings(registryList)

	missingCount := 0
	for _, result := range results {
		for _, regName := range registryList {
			if img, ok := result.RegistryResults[regName]; !ok || !img.Exists {
				missingCount++
			}
		}
	}
	if missingCount > 0 {
		fmt.Fprintln(w, missingCount, "incomplete images")
	} else {
		fmt.Fprintln(w, "all images OK")
	}

	tw := tabwriter.NewWriter(w, 0, 8, 2, ' ', 0)
	defer tw.Flush()

	header := []string{"image"}
	header = append(header, registryList...)
	header = append(header, "amd64", "arm64")
	fmt.Fprintln(tw, strings.Join(header, "\t"))

	separator := []string{"-----"}
	for range registryList {
		separator = append(separator, "---")
	}
	separator = append(separator, "-----", "-----")
	fmt.Fprintln(tw, strings.Join(separator, "\t"))

	for _, result := range results {
		row := []string{formatImageRef(result.Reference)}

		for _, regName := range registryList {
			status := "✗"
			if img, ok := result.RegistryResults[regName]; ok && img.Exists {
				status = "✓"
			}
			row = append(row, status)
		}

		ossImg := result.OSSImage()
		row = append(row,
			archStatus(result.ExpectsLinuxAmd64, ossImg, reg.Image{}, reg.Platform{OS: "linux", Architecture: "amd64"}),
			archStatus(result.ExpectsLinuxArm64, ossImg, reg.Image{}, reg.Platform{OS: "linux", Architecture: "arm64"}),
		)

		fmt.Fprintln(tw, strings.Join(row, "\t"))
	}
}

func (r RKE2Contents) CSV(w io.Writer) {
	if len(r.Images) == 0 {
		fmt.Fprintln(w, "no images found")
		return
	}

	results := r.Images
	sort.Slice(results, func(i, j int) bool {
		return formatImageRef(results[i].Reference) < formatImageRef(results[j].Reference)
	})

	registryNames := make(map[string]bool)
	for _, result := range results {
		for regName := range result.RegistryResults {
			registryNames[regName] = true
		}
	}

	var registryList []string
	for regName := range registryNames {
		registryList = append(registryList, regName)
	}
	sort.Strings(registryList)

	header := []string{"image"}
	header = append(header, registryList...)
	header = append(header, "sig", "amd64", "arm64", "win")
	fmt.Fprintln(w, strings.Join(header, ","))

	for _, result := range results {
		values := []string{formatImageRef(result.Reference)}

		for _, regName := range registryList {
			status := "N"
			if img, ok := result.RegistryResults[regName]; ok && img.Exists {
				status = "Y"
			}
			values = append(values, status)
		}

		ossImg := result.OSSImage()
		primeImg := result.PrimeImage()

		amd64Status := ""
		if result.ExpectsLinuxAmd64 {
			if ossImg.Platforms[reg.Platform{OS: "linux", Architecture: "amd64"}] &&
				primeImg.Platforms[reg.Platform{OS: "linux", Architecture: "amd64"}] {
				amd64Status = "Y"
			} else {
				amd64Status = "N"
			}
		}

		arm64Status := ""
		if result.ExpectsLinuxArm64 {
			if ossImg.Platforms[reg.Platform{OS: "linux", Architecture: "arm64"}] &&
				primeImg.Platforms[reg.Platform{OS: "linux", Architecture: "arm64"}] {
				arm64Status = "Y"
			} else {
				arm64Status = "N"
			}
		}

		winStatus := ""
		if result.ExpectsWindows {
			if ossImg.Exists && primeImg.Exists {
				winStatus = "Y"
			} else {
				winStatus = "N"
			}
		}

		values = append(values,
			"?", // sigstore not implemented
			amd64Status,
			arm64Status,
			winStatus,
		)

		fmt.Fprintln(w, strings.Join(values, ","))
	}
}

func (k K3SContents) CSV(w io.Writer) {
	if len(k.Images) == 0 {
		fmt.Fprintln(w, "no images found")
		return
	}

	results := k.Images
	sort.Slice(results, func(i, j int) bool {
		return formatImageRef(results[i].Reference) < formatImageRef(results[j].Reference)
	})

	registryNames := make(map[string]bool)
	for _, result := range results {
		for regName := range result.RegistryResults {
			registryNames[regName] = true
		}
	}

	var registryList []string
	for regName := range registryNames {
		registryList = append(registryList, regName)
	}
	sort.Strings(registryList)

	header := []string{"image"}
	header = append(header, registryList...)
	header = append(header, "amd64", "arm64")
	fmt.Fprintln(w, strings.Join(header, ","))

	for _, result := range results {
		values := []string{formatImageRef(result.Reference)}

		for _, regName := range registryList {
			status := "N"
			if img, ok := result.RegistryResults[regName]; ok && img.Exists {
				status = "Y"
			}
			values = append(values, status)
		}

		ossImg := result.OSSImage()

		amd64Status := ""
		if result.ExpectsLinuxAmd64 {
			if ossImg.Platforms[reg.Platform{OS: "linux", Architecture: "amd64"}] {
				amd64Status = "Y"
			} else {
				amd64Status = "N"
			}
		}

		arm64Status := ""
		if result.ExpectsLinuxArm64 {
			if ossImg.Platforms[reg.Platform{OS: "linux", Architecture: "arm64"}] {
				arm64Status = "Y"
			} else {
				arm64Status = "N"
			}
		}

		values = append(values, amd64Status, arm64Status)

		fmt.Fprintln(w, strings.Join(values, ","))
	}
}

func setupRegistries(registryConfigs map[string]config.RegistryConfig, includePrime bool) map[string]reg.Inspector {
	registries := map[string]reg.Inspector{
		"oss": reg.NewClient(ossRegistry, debug),
	}

	for name, regConfig := range registryConfigs {
		registries[name] = reg.NewInspectorFromConfig(&regConfig, debug)
	}

	if includePrime && rootConfig.PrimeRegistry != "" {
		if _, exists := registries["prime"]; !exists {
			registries["prime"] = reg.NewClient(rootConfig.PrimeRegistry, debug)
		}
	}

	return registries
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

	var registryConfigs map[string]config.RegistryConfig
	if rootConfig.RKE2 != nil {
		registryConfigs = rootConfig.RKE2.Registries
	}
	registries := setupRegistries(registryConfigs, true)

	inspector := rke2.NewReleaseInspector(releaseFs, registries, debug)

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

	var registryConfigs map[string]config.RegistryConfig
	if rootConfig.K3s != nil {
		registryConfigs = rootConfig.K3s.Registries
	}
	registries := setupRegistries(registryConfigs, false)

	inspector := k3s.NewReleaseInspector(releaseFs, registries, debug)

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
