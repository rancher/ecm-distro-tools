package config

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
)

const (
	ecmDistroDir   = ".ecm-distro-tools"
	configFileName = "config.json"
)

// K3sRelease
type K3sRelease struct {
	OldK8sVersion string `json:"old_k8s_version"`
	NewK8sVersion string `json:"new_k8s_version"`
	OldK8sClient  string `json:"old_k8s_client"`
	NewK8sClient  string `json:"new_k8s_client"`
	OldSuffix     string `json:"old_suffix"`
	NewSuffix     string `json:"new_suffix"`
	ReleaseBranch string `json:"release_branch"`
	DryRun        bool   `json:"dry_run"`
}

// RKE2
type RKE2 struct {
	Versions []string `json:"versions"`
}

// User
type User struct {
	Email string `json:"email"`
}

// K3s
type K3s struct {
	Version   map[string]K3sRelease `json:"version"`
	Workspace string                `json:"workspace"`
}

// Drone
type Drone struct {
	K3sPR          string `json:"k3s_pr"`
	K3sPublish     string `json:"k3s_publish"`
	RancherPR      string `json:"rancher_pr"`
	RancherPublish string `json:"rancher_publish"`
}

// Auth
type Auth struct {
	Drone       *Drone `json:"drone"`
	GithubToken string `json:"github_token"`
	SSHKeyPath  string `json:"ssh_key_path"`
}

// Config
type Config struct {
	User *User `json:"user"`
	K3s  *K3s  `json:"k3s"`
	RKE2 *RKE2 `json:"rke2"`
	Auth *Auth `json:"auth"`
}

func DefaultConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", nil
	}
	return filepath.Join(homeDir, ecmDistroDir, configFileName), nil
}

// Load reads the given config file and returns a struct
// containing the necessary values to perform a release.
func Load(configFile string) (*Config, error) {
	f, err := os.Open(configFile)
	if err != nil {
		return nil, err
	}
	return read(f)
}

func read(r io.Reader) (*Config, error) {
	var c Config
	if err := json.NewDecoder(r).Decode(&c); err != nil {
		return nil, err
	}
	return &c, nil
}

func (c *Config) String() (string, error) {
	b, err := json.MarshalIndent(c, "", " ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}
