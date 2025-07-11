package config

import (
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"
)

const (
	RancherGithubOrganization = "rancher"
	RancherRepositoryName     = "rancher"
	K3sGithubOrganization     = "k3s-io"
	K3sRepostitoryName        = "k3s"
)

const (
	// K3sSuffixBase represents the K3s specific version as part of the whole tag used for a release.
	// When using this constant, be sure to append the release number to the end; e.g. k3s1.
	K3sSuffixBase = "k3s"
)

const (
	ImageBuildBaseRepositoryURL    = "https://github.com/rancher/image-build-base"
	ImageBuildBaseRepositoryGitURI = "git@github.com:rancher/image-build-base.git"

	K3sKubernetesRepositoryURL = "https://github.com/k3s-io/kubernetes"
	K3sKubernetesGitURI        = "git@github.com:k3s-io/kubernetes.git"

	RancherRepositoryURL    = "https://github.com/rancher/rancher"
	RancherRepositoryGitURI = "git@github.com:rancher/rancher.git"
	RKE2RepositoryURL       = "https://github.com/rancher/rancher"
	RKE2RepositoryGitURI    = "git@github.com:rancher/rancher.git"
	K3sRepositoryURL        = "https://github.com/k3s-io/k3s"
	K3sRepositoryGitURI     = "git@github.com:k3s-io/k3s"

	RKE2SystemAgentInstallerRepositoryURL = "https://github.com/rancher/system-agent-installer-rke2"
	RKE2SystemAgentInstallerGitURI        = "git@github.com:rancher/system-agent-installer-rke2.git"
	K3sSystemAgentInstallerRepositoryURL  = "https://github.com/rancher/system-agent-installer-k3s"
	K3sSystemAgentInstallerGitURI         = "git@github.com:rancher/system-agent-installer-k3s.git"

	RKE2PackagingRepositoryURL = "https://github.com/rancher/rke2-packaging"
	RKE2PackagingGitURI        = "git@github.com:rancher/rke2-packaging.git"

	RKE2SELinuxRepositoryURL = "https://github.com/rancher/rke2-selinux"
	RKE2SELinuxGitURI        = "git@github.com:rancher/rke2-selinux.git"
	K3sSELinuxRepositoryURL  = "https://github.com/k3s-io/k3s-selinux"
	K3sSELinuxGitURI         = "git@github.com:k3s-io/k3s-selinux.git"

	RKE2UpgradeRepositoryURL = "https://github.com/rancher/rke2-upgrade"
	RKE2UpgradeGitURI        = "git@github.com:rancher/rke2-upgrade.git"
	K3sUpgradeRepositoryURL  = "https://github.com/k3s-io/k3s-upgrade"
	K3sUpgradeGitURI         = "git@github.com:k3s-io/k3s-upgrade.git"

	RancherChartsRepositoryURL    = "https://github.com/rancher/charts"
	RancherChartsRepositoryGitURI = "git@github.com/rancher/charts.git"
)

const (
	SuseStageRegistry    = "stgregistry.suse.com"
	PrimeArtifactsBucket = "prime-artifacts"
)

// K3sRelease
type K3sRelease struct {
	OldK8sVersion                 string `json:"old_k8s_version"`
	NewK8sVersion                 string `json:"new_k8s_version"`
	OldK8sClient                  string `json:"old_k8s_client"`
	NewK8sClient                  string `json:"new_k8s_client"`
	OldSuffix                     string `json:"old_suffix"`
	NewSuffix                     string `json:"new_suffix"`
	ReleaseBranch                 string `json:"release_branch"`
	Workspace                     string `json:"workspace"`
	NewGoVersion                  string `json:"-"`
	K3sRepoOwner                  string `json:"k3s_repo_owner"`
	SystemAgentInstallerRepoOwner string `json:"system_agent_installer_repo_owner"`
	K8sRancherURL                 string `json:"k8s_rancher_url"`
	K3sUpstreamURL                string `json:"k3s_upstream_url"`
	DryRun                        bool   `json:"dry_run"`
}

// RancherRelease
type RancherRelease struct {
	ReleaseBranch string `json:"release_branch"`
}

type UIRelease struct {
	UIRepoOwner   string `json:"ui_repo_owner"`
	UIRepoName    string `json:"ui_repo_name"`
	PreviousTag   string `json:"previous_tag"`
	ReleaseBranch string `json:"release_branch"`
	DryRun        bool   `json:"dry_run"`
}

type DashboardRelease struct {
	PreviousTag          string `json:"previous_tag"`
	ReleaseBranch        string `json:"release_branch"`
	UIReleaseBranch      string `json:"ui_release_branch"`
	UIPreviousTag        string `json:"ui_previous_tag"`
	Tag                  string
	RancherReleaseBranch string `json:"rancher_release_branch"`
	RancherUpstreamURL   string
	DryRun               bool `json:"dry_run"`
}

type CLIRelease struct {
	PreviousTag          string `json:"previous_tag"`
	ReleaseBranch        string `json:"release_branch"`
	Tag                  string `json:"-"`
	CLIUpstreamURL       string `json:"-"`
	RancherReleaseBranch string `json:"rancher_release_branch"`
	RancherUpstreamURL   string `json:"rancher_upstream_url"`
	RancherCommitSHA     string `json:"-"`
	RancherTag           string `json:"-"`
	DryRun               bool   `json:"dry_run"`
}

// RKE2
type RKE2 struct {
	Versions []string `json:"versions"`
}

// ChartsRelease
type ChartsRelease struct {
	Workspace     string   `json:"workspace"`
	ChartsRepoURL string   `json:"charts_repo_url"`
	ChartsForkURL string   `json:"charts_fork_url"`
	BranchLines   []string `json:"branch_lines"`
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

// Rancher
type Rancher struct {
	Versions map[string]RancherRelease `json:"versions"`
}

// Dashboard
type Dashboard struct {
	Versions           map[string]DashboardRelease `json:"versions"`
	RepoOwner          string                      `json:"repo_owner"`
	RepoName           string                      `json:"repo_name"`
	UIRepoOwner        string                      `json:"ui_repo_owner"`
	UIRepoName         string                      `json:"ui_repo_name"`
	RancherRepoOwner   string                      `json:"rancher_repo_owner"`
	RancherRepoName    string                      `json:"rancher_repo_name"`
	RancherUpstreamURL string                      `json:"rancher_upstream_url"`
}

type CLI struct {
	Versions           map[string]CLIRelease `json:"versions"`
	RepoOwner          string                `json:"repo_owner"`
	RepoName           string                `json:"repo_name"`
	RancherRepoOwner   string                `json:"rancher_repo_owner"`
	RancherRepoName    string                `json:"rancher_repo_name"`
	RancherUpstreamURL string                `json:"rancher_upstream_url"`
}

// Auth
type Auth struct {
	GithubToken        string `json:"github_token"`
	SSHKeyPath         string `json:"ssh_key_path"`
	AWSAccessKeyID     string `json:"aws_access_key_id"`
	AWSSecretAccessKey string `json:"aws_secret_access_key"`
	AWSSessionToken    string `json:"aws_session_token"`
	AWSDefaultRegion   string `json:"aws_default_region"`
}

// Config
type Config struct {
	User                      *User          `json:"user"`
	K3s                       *K3s           `json:"k3s"`
	Rancher                   *Rancher       `json:"rancher"`
	RKE2                      *RKE2          `json:"rke2"`
	Charts                    *ChartsRelease `json:"charts"`
	Auth                      *Auth          `json:"auth"`
	Dashboard                 *Dashboard     `json:"dashboard"`
	CLI                       *CLI           `json:"cli"`
	PrimeRegistry             string         `json:"prime_registry"`
	RancherGithubOrganization string         `json:"rancher_github_organization"`
	RancherRepositoryName     string         `json:"rancher_repository_name"`
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
					ReleaseBranch: "release/v2.x",
				},
			},
		},
		Dashboard: &Dashboard{
			RepoName:           "dashboard",
			RepoOwner:          "rancher",
			UIRepoName:         "ui",
			UIRepoOwner:        "rancher",
			RancherRepoName:    "rancher",
			RancherRepoOwner:   "rancher",
			RancherUpstreamURL: "git@github.com:rancher/rancher.git",
			Versions: map[string]DashboardRelease{
				"v2.x.y": {
					PreviousTag:          "v2.x.y",
					UIPreviousTag:        "v2.x.y",
					ReleaseBranch:        "release-v2.x",
					UIReleaseBranch:      "release-v2.x",
					RancherReleaseBranch: "release/v2.x",
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
		PrimeRegistry:             "example.com",
		RancherGithubOrganization: RancherGithubOrganization,
		RancherRepositoryName:     RancherRepositoryName,
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

func ValueOrDefault(v string, d string) string {
	if v == "" {
		return d
	}
	return v
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
		Rancher Repo Owner: {{ $rancherValue.RancherRepoOwner }}{{ end }}

RKE2{{ range .RKE2.Versions }}
	{{ . }}{{ end}}

Charts
    Workspace:     {{.Charts.Workspace}}
    ChartsRepoURL: {{.Charts.ChartsRepoURL}}
    ChartsForkURL: {{.Charts.ChartsForkURL}}
    BranchLines:     {{.Charts.BranchLines}}
`
