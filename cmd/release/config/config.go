package config

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
)

// K3sRelease
type K3sRelease struct {
	OldK8sVersion                 string `mapstructure:"old_k8s_version" validate:"required"`
	NewK8sVersion                 string `mapstructure:"new_k8s_version" validate:"required"`
	OldK8sClient                  string `mapstructure:"old_k8s_client" validate:"required"`
	NewK8sClient                  string `mapstructure:"new_k8s_client" validate:"required"`
	OldSuffix                     string `mapstructure:"old_suffix" validate:"required,startswith=k3s"`
	NewSuffix                     string `mapstructure:"new_suffix" validate:"required,startswith=k3s"`
	ReleaseBranch                 string `mapstructure:"release_branch" validate:"required"`
	Workspace                     string `mapstructure:"workspace" validate:"required,dirpath"`
	NewGoVersion                  string `mapstructure:"-"`
	K3sRepoOwner                  string `mapstructure:"k3s_repo_owner" validate:"required"`
	SystemAgentInstallerRepoOwner string `mapstructure:"system_agent_installer_repo_owner" validate:"required"`
	K8sRancherURL                 string `mapstructure:"k8s_rancher_url" validate:"required"`
	K3sUpstreamURL                string `mapstructure:"k3s_upstream_url" validate:"required"`
	DryRun                        bool   `mapstructure:"dry_run"`
}

// RancherRelease
type RancherRelease struct {
	ReleaseBranch        string   `mapstructure:"release_branch" validate:"required"`
	RancherRepoOwner     string   `mapstructure:"rancher_repo_owner" validate:"required"`
	IssueNumber          string   `mapstructure:"issue_number" validate:"number"`
	CheckImages          []string `mapstructure:"check_images" validate:"required"`
	BaseRegistry         string   `mapstructure:"base_registry" validate:"required,hostname"`
	Registry             string   `mapstructure:"registry" validate:"required,hostname"`
	PrimeArtifactsBucket string   `mapstructure:"prime_artifacts_bucket" validate:"required"`
	DryRun               bool     `mapstructure:"dry_run"`
	SkipStatusCheck      bool     `mapstructure:"skip_status_check"`
}

// RKE2
type RKE2 struct {
	Versions []string `mapstructure:"versions"`
}

// ChartsRelease
type ChartsRelease struct {
	Workspace     string `mapstructure:"workspace" validate:"required,dirpath"`
	ChartsRepoURL string `mapstructure:"charts_repo_url" validate:"required"`
	ChartsForkURL string `mapstructure:"charts_fork_url" validate:"required"`
	DevBranch     string `mapstructure:"dev_branch" validate:"required"`
	ReleaseBranch string `mapstructure:"release_branch" validate:"required"`
}

// User
type User struct {
	Email          string `mapstructure:"email" validate:"required,email"`
	GithubUsername string `mapstructure:"github_username" validate:"required"`
}

// K3s
type K3s struct {
	Versions map[string]K3sRelease `mapstructure:"versions" validate:"dive"`
}

// Rancher
type Rancher struct {
	Versions map[string]RancherRelease `mapstructure:"versions" validate:"dive"`
}

// Auth
type Auth struct {
	GithubToken string `mapstructure:"github_token"`
	SSHKeyPath  string `mapstructure:"ssh_key_path" validate:"file"`
}

// Config
type Config struct {
	User    *User          `mapstructure:"user"`
	K3s     *K3s           `mapstructure:"k3s"`
	Rancher *Rancher       `mapstructure:"rancher"`
	RKE2    *RKE2          `mapstructure:"rke2"`
	Charts  *ChartsRelease `mapstructure:"charts"`
	Auth    *Auth          `mapstructure:"auth"`
}

// Load reads the given config file and returns a struct
// containing the necessary values to perform a release.
func OpenOnEditor(configPath string) error {
	cmd := exec.Command(textEditorName(), filepath.Join(os.ExpandEnv(configPath), "config.json"))
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout

	return cmd.Run()
}

func Generate(configPath string) error {
	configExists := true

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
					OldK8sVersion:                 "v1.x.z",
					NewK8sVersion:                 "v1.x.y",
					OldK8sClient:                  "v0.x.z",
					NewK8sClient:                  "v0.x.y",
					OldSuffix:                     "k3s1",
					NewSuffix:                     "k3s1",
					ReleaseBranch:                 "release-1.x",
					DryRun:                        false,
					Workspace:                     filepath.Join(gopath, "src", "github.com", "k3s-io", "kubernetes", "v1.x.z"),
					SystemAgentInstallerRepoOwner: "rancher",
					K3sRepoOwner:                  "k3s-io",
					K8sRancherURL:                 "git@github.com:k3s-io/kubernetes.git",
					K3sUpstreamURL:                "git@github.com:k3s-io/k3s.git",
				},
			},
		},
		RKE2: &RKE2{
			Versions: []string{"v1.x.y"},
		},
		Rancher: &Rancher{
			Versions: map[string]RancherRelease{
				"v2.x.y": {
					ReleaseBranch:        "release/v2.x",
					DryRun:               false,
					SkipStatusCheck:      false,
					RancherRepoOwner:     "rancher",
					CheckImages:          []string{},
					BaseRegistry:         "stgregistry.suse.com",
					Registry:             "registry.rancher.com",
					PrimeArtifactsBucket: "prime-artifacts",
				},
			},
		},
		Charts: &ChartsRelease{
			Workspace:     filepath.Join(gopath, "src", "github.com", "rancher", "charts"),
			ChartsRepoURL: "https://github.com/rancher/charts",
			ChartsForkURL: "",
			DevBranch:     "dev-v2.9",
			ReleaseBranch: "release-v2.9",
		},
		Auth: &Auth{
			GithubToken: "YOUR_TOKEN",
			SSHKeyPath:  "path/to/your/ssh/key",
		},
	}
}

func View(config *Config) error {
	tmp, err := template.New("ecm").Parse(configViewTemplate)
	if err != nil {
		return err
	}

	return tmp.Execute(os.Stdout, config)
}

const configViewTemplate = `Release config

User
	Email:           {{ .User.Email }}
	Github Username: {{ .User.GithubUsername }}

K3s {{ range $k3sVersion, $k3sValue := .K3s.Versions }}
	{{ $k3sVersion }}:
		Old K8s Version:  {{ $k3sValue.OldK8sVersion}}
		New K8s Version:  {{ $k3sValue.NewK8sVersion}}
		Old K8s Client:   {{ $k3sValue.OldK8sClient}}
		New K8s Client:   {{ $k3sValue.NewK8sClient}}
		Old Suffix:       {{ $k3sValue.OldSuffix}}
		New Suffix:       {{ $k3sValue.NewSuffix}}
		Release Branch:   {{ $k3sValue.ReleaseBranch}}
		Dry Run:          {{ $k3sValue.DryRun}}
		K3s Repo Owner:   {{ $k3sValue.K3sRepoOwner}}
		K8s Rancher URL:  {{ $k3sValue.K8sRancherURL}}
		Workspace:        {{ $k3sValue.Workspace}}
		K3s Upstream URL: {{ $k3sValue.K3sUpstreamURL}}{{ end }}

Rancher {{ range $rancherVersion, $rancherValue := .Rancher.Versions }}
	{{ $rancherVersion }}:
		Release Branch:     {{ $rancherValue.ReleaseBranch }}
		Dry Run:            {{ $rancherValue.DryRun }}
		Skip Status Check:  {{ $rancherValue.SkipStatusCheck }}
		Issue Number:       {{ $rancherValue.IssueNumber }}
		Rancher Repo Owner: {{ $rancherValue.RancherRepoOwner }}{{ end }}

RKE2{{ range .RKE2.Versions }}
	{{ . }}{{ end}}

Charts
    Workspace:     {{.Charts.Workspace}}
    ChartsRepoURL: {{.Charts.ChartsRepoURL}}
    ChartsForkURL: {{.Charts.ChartsForkURL}}
    DevBranch:     {{.Charts.DevBranch}}
    ReleaseBranch: {{.Charts.ReleaseBranch}}
`
