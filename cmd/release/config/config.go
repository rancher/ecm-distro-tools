package config

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	ecmDistroDir   = ".ecm-distro-tools"
	configFileName = "config.json"
)

// K3sRelease
type K3sRelease struct {
	OldK8sVersion  string `json:"old_k8s_version"`
	NewK8sVersion  string `json:"new_k8s_version"`
	OldK8sClient   string `json:"old_k8s_client"`
	NewK8sClient   string `json:"new_k8s_client"`
	OldSuffix      string `json:"old_suffix"`
	NewSuffix      string `json:"new_suffix"`
	ReleaseBranch  string `json:"release_branch"`
	DryRun         bool   `json:"dry_run"`
	Workspace      string `json:"workspace"`
	NewGoVersion   string `json:"-"`
	K3sRepoOwner   string `json:"k3s_repo_owner"`
	K8sRancherURL  string `json:"k8s_rancher_url"`
	K3sUpstreamURL string `json:"k3s_upstream_url"`
}

// RKE2
type RKE2 struct {
	Versions []string `json:"versions"`
}

// User
type User struct {
	Email          string `json:"email"`
	GithubUsername string `json:"github_username"`
}

// K3s
type K3s struct {
	Versions map[string]K3sRelease `json:"versions"`
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

func OpenOnEditor() error {
	confPath, err := DefaultConfigPath()
	if err != nil {
		return err
	}
	cmd := exec.Command(textEditorName(), confPath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	return cmd.Run()
}

func Generate() error {
	configExists := true
	configPath, err := DefaultConfigPath()
	if err != nil {
		return err
	}
	if _, err := os.Stat(configPath); err != nil {
		if !strings.Contains(err.Error(), "no such file or directory") {
			return err
		}
		configExists = false
	}
	if configExists {
		return errors.New("config already exists at " + configPath)
	}
	confB, err := json.MarshalIndent(exampleConfig(), "", " ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath, confB, 0644)
}

func textEditorName() string {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}
	return editor
}

func exampleConfig() Config {
	gopath := os.Getenv("GOPATH")
	return Config{
		User: &User{
			Email: "your.name@suse.com",
		},
		K3s: &K3s{
			Versions: map[string]K3sRelease{
				"v1.x.y": {
					OldK8sVersion:  "v1.x.z",
					NewK8sVersion:  "v1.x.y",
					OldK8sClient:   "v0.x.z",
					NewK8sClient:   "v0.x.y",
					OldSuffix:      "k3s1",
					NewSuffix:      "k3s1",
					ReleaseBranch:  "release-1.x",
					DryRun:         false,
					Workspace:      filepath.Join(gopath, "src", "github.com", "k3s-io", "kubernetes"),
					K3sRepoOwner:   "k3s-io",
					K8sRancherURL:  "git@github.com:k3s-io/kubernetes.git",
					K3sUpstreamURL: "git@github.com:k3s-io/k3s.git",
				},
			},
		},
		RKE2: &RKE2{
			Versions: []string{"v1.x.y"},
		},
		Auth: &Auth{
			Drone: &Drone{
				K3sPR:          "YOUR_TOKEN",
				K3sPublish:     "YOUR_TOKEN",
				RancherPR:      "YOUR_TOKEN",
				RancherPublish: "YOUR_TOKEN",
			},
			GithubToken: "YOUR_TOKEN",
			SSHKeyPath:  "path/to/your/ssh/key",
		},
	}
}

const configViewTemplate = `RKE2 Version
------------
{{- range .RKE2.Versions }}
{{ . -}}+rke2r1
{{- end}}
`
