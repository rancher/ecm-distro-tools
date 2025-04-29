package cmd

type ErrVersionNotFound struct {
	Version string
}

func (e *ErrVersionNotFound) Error() string {
	return "verify your config file, version not found: " + e.Version
}

func NewVersionNotFoundError(version string) error {
	return &ErrVersionNotFound{
		Version: version,
	}
}
