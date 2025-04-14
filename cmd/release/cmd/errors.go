package cmd

import "fmt"

type VersionNotFoundError struct {
	Version string
}

func (e *VersionNotFoundError) Error() string {
	return fmt.Sprintf("verify your config file, version not found: %s", e.Version)
}

func NewVersionNotFoundError(version string) error {
	return &VersionNotFoundError{
		Version: version,
	}
}
