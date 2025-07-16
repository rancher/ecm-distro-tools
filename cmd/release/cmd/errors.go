package cmd

type ErrVersionNotFound struct {
	Version string
	Config  string
}

func (e *ErrVersionNotFound) Error() string {
	return "verify your config file: version " + e.Version + " not found for " + e.Config
}

func NewVersionNotFoundError(version, config string) error {
	return &ErrVersionNotFound{
		Version: version,
		Config:  config,
	}
}
