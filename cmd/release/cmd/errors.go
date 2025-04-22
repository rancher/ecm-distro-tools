package cmd

type VersionNotFoundError struct {
	Version string
}

func (e *VersionNotFoundError) Error() string {
	return "verify your config file, version not found: " + e.Version
}

func NewVersionNotFoundError(version string) error {
	return &VersionNotFoundError{
		Version: version,
	}
}
