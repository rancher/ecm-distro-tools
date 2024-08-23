package config

import (
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"

	"github.com/go-playground/validator/v10"
)

// K3sRelease
type K3sRelease struct {
	OldK8sVersion                 string `json:"old_k8s_version" validate:"required"`
	NewK8sVersion                 string `json:"new_k8s_version" validate:"required"`
	OldK8sClient                  string `json:"old_k8s_client" validate:"required"`
	NewK8sClient                  string `json:"new_k8s_client" validate:"required"`
	OldSuffix                     string `json:"old_suffix" validate:"required,startswith=k3s"`
	NewSuffix                     string `json:"new_suffix" validate:"required,startswith=k3s"`
	ReleaseBranch                 string `json:"release_branch" validate:"required"`
	Workspace                     string `json:"workspace" validate:"required,dirpath"`
	NewGoVersion                  string `json:"-"`
	K3sRepoOwner                  string `json:"k3s_repo_owner" validate:"required"`
	SystemAgentInstallerRepoOwner string `json:"system_agent_installer_repo_owner" validate:"required"`
	K8sRancherURL                 string `json:"k8s_rancher_url" validate:"required"`
	K3sUpstreamURL                string `json:"k3s_upstream_url" validate:"required"`
	DryRun                        bool   `json:"dry_run"`
}

// RancherRelease
type RancherRelease struct {
	ReleaseBranch        string `json:"release_branch" validate:"required"`
	RancherRepoOwner     string `json:"rancher_repo_owner" validate:"required"`
	IssueNumber          string `json:"issue_number" validate:"number"`
	BaseRegistry         string `json:"base_registry" validate:"required,hostname"`
	Registry             string `json:"registry" validate:"required,hostname"`
	PrimeArtifactsBucket string `json:"prime_artifacts_bucket" validate:"required"`
	DryRun               bool   `json:"dry_run"`
	SkipStatusCheck      bool   `json:"skip_status_check"`
}

// RKE2
type RKE2 struct {
	Versions []string `json:"versions"`
}

// ChartsRelease
type ChartsRelease struct {
	Workspace     string   `json:"workspace" validate:"required,dirpath"`
	ChartsRepoURL string   `json:"charts_repo_url" validate:"required"`
	ChartsForkURL string   `json:"charts_fork_url" validate:"required"`
	BranchLines   []string `json:"branch_lines" validate:"required"`
}

// User
type User struct {
	Email          string `json:"email" validate:"required,email"`
	GithubUsername string `json:"github_username" validate:"required"`
}

// K3s
type K3s struct {
	Versions map[string]K3sRelease `json:"versions" validate:"dive,omitempty"`
}

// Rancher
type Rancher struct {
	Versions map[string]RancherRelease `json:"versions" validate:"dive,omitempty"`
}

// Auth
type Auth struct {
	GithubToken        string `json:"github_token"`
	SSHKeyPath         string `json:"ssh_key_path" validate:"filepath"`
	AWSAccessKeyID     string `json:"aws_access_key_id"`
	AWSSecretAccessKey string `json:"aws_secret_access_key"`
	AWSSessionToken    string `json:"aws_session_token"`
	AWSDefaultRegion   string `json:"aws_default_region"`
}

// Config
type Config struct {
	User    *User          `json:"user"`
	K3s     *K3s           `json:"k3s" validate:"omitempty"`
	Rancher *Rancher       `json:"rancher" validate:"omitempty"`
	RKE2    *RKE2          `json:"rke2" validate:"omitempty"`
	Charts  *ChartsRelease `json:"charts" validate:"omitempty"`
	Auth    *Auth          `json:"auth"`
}

// OpenOnEditor opens the given config file on the user's default text editor.
func OpenOnEditor(configFile string) error {
	cmd := exec.Command(textEditorName(), configFile)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout

	return cmd.Run()
}

func textEditorName() string {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}

	return editor
}

// Load reads the given config file and returns a struct
// containing the necessary values to perform a release.
func Load(configFile string) (*Config, error) {
	f, err := os.Open(configFile)
	if err != nil {
		return nil, err
	}

	return Read(f)
}

// Read reads the given JSON file with the config and returns a struct
func Read(r io.Reader) (*Config, error) {
	var c Config
	if err := json.NewDecoder(r).Decode(&c); err != nil {
		return nil, err
	}

	return &c, nil
}

// ExampleConfig returns a valid JSON string with the config structure
func ExampleConfig() (string, error) {
	gopath := os.Getenv("GOPATH")

	conf := Config{
		User: &User{
			Email:          "your.name@suse.com",
			GithubUsername: "your-github-username",
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
					Workspace:                     filepath.Join(gopath, "src", "github.com", "k3s-io", "kubernetes", "v1.x.z") + "/",
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
					IssueNumber:          "1234",
					BaseRegistry:         "stgregistry.suse.com",
					Registry:             "registry.rancher.com",
					PrimeArtifactsBucket: "prime-artifacts",
				},
			},
		},
		Charts: &ChartsRelease{
			Workspace:     filepath.Join(gopath, "src", "github.com", "rancher", "charts") + "/",
			ChartsRepoURL: "https://github.com/rancher/charts",
			ChartsForkURL: "https://github.com/your-github-username/charts",
			BranchLines:   []string{"2.10", "2.9", "2.8"},
		},
		Auth: &Auth{
			GithubToken:        "YOUR_TOKEN",
			SSHKeyPath:         "path/to/your/ssh/key",
			AWSAccessKeyID:     "XXXXXXXXXXXXXXXXXXX",
			AWSSecretAccessKey: "xxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
			AWSSessionToken:    "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
			AWSDefaultRegion:   "us-east-1",
		},
	}
	b, err := json.MarshalIndent(conf, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// View prints a simplified view of the config to the standard output
func View(config *Config) error {
	tmp, err := template.New("ecm").Parse(configViewTemplate)
	if err != nil {
		return err
	}

	return tmp.Execute(os.Stdout, config)
}

func (c *Config) Validate() error {
	return validator.New(validator.WithRequiredStructEnabled()).Struct(c)
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
    BranchLines:     {{.Charts.BranchLines}}
`
