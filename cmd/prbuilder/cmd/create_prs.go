package cmd

import (
	"context"
	"os"

	"github.com/rancher/ecm-distro-tools/cmd/prbuilder/config"
	"github.com/rancher/ecm-distro-tools/prbuilder"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type createPRsOpts struct {
	Tag        string
	ConfigFile string
	DryRun     bool
	TargetDir  string
	Remote     string
}

var createPRsCmdOpts createPRsOpts

var createPRsCmd = &cobra.Command{
	Use:   "create-prs",
	Short: "Create PRs in configured target repositories",
	Long: `Creates pull requests in downstream/consumer repositories when a new version is tagged.

The mode is determined by your config file:
  - Single-target mode: Use "target" (singular) in config, supports --target-dir
  - Multi-target mode: Use "targets" (plural) in config, processes all targets

Examples:
  # Single-target mode with local clone
  prbuilder create-prs --tag v10.3.2 --target-dir ~/repos/rancher

  # Multi-target mode (automation/CI)
  prbuilder create-prs --tag v10.3.2`,
	RunE: createPRs,
}

func init() {
	rootCmd.AddCommand(createPRsCmd)

	createPRsCmd.Flags().StringVarP(&createPRsCmdOpts.Tag, "tag", "t", os.Getenv("TAG"), "The tag that was released (e.g., v10.3.2)")
	createPRsCmd.Flags().StringVarP(&createPRsCmdOpts.ConfigFile, "config", "c", getEnvOrDefault("CONFIG_FILE", ".github/pr-consumer-config.yml"), "Path to config file")
	createPRsCmd.Flags().BoolVarP(&createPRsCmdOpts.DryRun, "dry-run", "n", getEnvBool("DRY_RUN"), "Dry run mode (show changes but don't create PRs)")
	createPRsCmd.Flags().StringVarP(&createPRsCmdOpts.TargetDir, "target-dir", "d", "", "Path to already-cloned target repository (only for single-target configs)")
	createPRsCmd.Flags().StringVarP(&createPRsCmdOpts.Remote, "remote", "r", "origin", "Git remote name to use for push")
}

func createPRs(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	if err := validateInputs(cmd); err != nil {
		return err
	}

	cfg := loadAndLogConfig()

	sourceRepoDir := getSourceRepoDir()

	pb := buildPRBuilder(cfg, sourceRepoDir)

	results, err := pb.ProcessTargets(ctx)
	if err != nil {
		logrus.Fatalf("Failed to process targets: %v", err)
	}

	return outputResults(results)
}

// validateInputs validates required inputs and environment variables
func validateInputs(cmd *cobra.Command) error {
	if createPRsCmdOpts.Tag == "" {
		logrus.Error("Tag is required (use --tag or set TAG environment variable)")
		return cmd.Usage()
	}

	githubToken := os.Getenv("GH_TOKEN")
	if githubToken == "" {
		githubToken = os.Getenv("GITHUB_TOKEN")
	}
	if githubToken == "" {
		logrus.Fatal("GH_TOKEN or GITHUB_TOKEN environment variable is required")
	}

	return nil
}

// loadAndLogConfig loads the config file and logs configuration details
func loadAndLogConfig() *config.Config {
	cfg, err := config.Load(createPRsCmdOpts.ConfigFile)
	if err != nil {
		logrus.Fatalf("Failed to load config file: %v", err)
	}

	mode := "multi-target"
	if cfg.IsSingleTarget() {
		mode = "single-target"
	}

	logrus.Infof("Tag: %s", createPRsCmdOpts.Tag)
	logrus.Infof("Config: %s", createPRsCmdOpts.ConfigFile)
	logrus.Infof("Mode: %s", mode)
	logrus.Infof("Version mapping type: %s", cfg.VersionMappingType)
	if createPRsCmdOpts.DryRun {
		logrus.Info("Dry run: true")
	}

	// Validate --target-dir is only used with single-target configs
	if createPRsCmdOpts.TargetDir != "" && !cfg.IsSingleTarget() {
		logrus.Fatal("--target-dir flag requires single-target config mode. Your config uses 'targets' (plural) which enables multi-target mode. Use 'target' (singular) in your config to enable single-target mode with --target-dir support")
	}

	if createPRsCmdOpts.TargetDir != "" {
		logrus.Infof("Using local repository at: %s", createPRsCmdOpts.TargetDir)
	}

	return cfg
}

// getSourceRepoDir returns the source repository directory
func getSourceRepoDir() string {
	sourceRepoDir := os.Getenv("GITHUB_WORKSPACE")
	if sourceRepoDir == "" {
		var err error
		sourceRepoDir, err = os.Getwd()
		if err != nil {
			logrus.Fatalf("Failed to get current directory: %v", err)
		}
	}
	return sourceRepoDir
}

// buildPRBuilder creates a new PRBuilder instance
func buildPRBuilder(cfg *config.Config, sourceRepoDir string) *prbuilder.PRBuilder {
	pb, err := prbuilder.NewPRBuilder(prbuilder.Options{
		Config:        cfg,
		Tag:           createPRsCmdOpts.Tag,
		SourceRepoDir: sourceRepoDir,
		DryRun:        createPRsCmdOpts.DryRun,
		TargetDir:     createPRsCmdOpts.TargetDir,
		Remote:        createPRsCmdOpts.Remote,
	})
	if err != nil {
		logrus.Fatalf("Failed to create PR builder: %v", err)
	}
	return pb
}

// outputResults processes and outputs the results of PR creation
func outputResults(results []prbuilder.PRResult) error {
	successCount := 0
	for _, result := range results {
		if result.Error == nil && result.PRURL != "" {
			successCount++
		}
	}

	if successCount > 0 {
		logrus.Info("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		logrus.Infof("Created %d PR(s):", successCount)
		for _, result := range results {
			if result.Error == nil && result.PRURL != "" {
				logrus.Infof("  - %s", result.PRURL)
			}
		}
	} else {
		logrus.Warn("No PRs were created")
	}

	if err := prbuilder.WriteGitHubOutput(results); err != nil {
		logrus.Warnf("Failed to write GitHub Actions output: %v", err)
	}

	// Exit with error if all targets failed
	if successCount == 0 && len(results) > 0 {
		hasErrors := false
		for _, result := range results {
			if result.Error != nil {
				hasErrors = true
				break
			}
		}
		if hasErrors {
			logrus.Fatal("All targets failed")
		}
	}

	return nil
}

// getEnvOrDefault returns environment variable value or default
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvBool returns true if environment variable is set to "true"
func getEnvBool(key string) bool {
	return os.Getenv(key) == "true"
}
