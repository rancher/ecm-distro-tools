package prbuilder

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// WriteGitHubOutput writes the PR results to the GitHub Actions output file.
// This is used in CI/CD environments to pass PR URLs to subsequent workflow steps.
//
// If GITHUB_OUTPUT environment variable is not set, this function does nothing.
// The output format is: prs=["url1","url2",...]
func WriteGitHubOutput(results []Result) error {
	outputFile := os.Getenv("GITHUB_OUTPUT")
	if outputFile == "" {
		return nil
	}

	prURLs := make([]string, 0)
	for _, result := range results {
		if result.Error == nil && result.PRURL != nil {
			prURLs = append(prURLs, *result.PRURL)
		}
	}

	b, err := json.Marshal(prURLs)
	if err != nil {
		return fmt.Errorf("failed to marshal PR URLs to JSON: %w", err)
	}

	cleanPath := filepath.Clean(outputFile)
	f, err := os.OpenFile(cleanPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("failed to open GITHUB_OUTPUT file: %w", err)
	}
	defer func() {
		if cerr := f.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("failed to close GITHUB_OUTPUT file: %w", cerr)
		}
	}()

	_, err = fmt.Fprintf(f, "prs=%s\n", string(b))
	if err != nil {
		return fmt.Errorf("failed to write to GITHUB_OUTPUT file: %w", err)
	}

	return err
}
