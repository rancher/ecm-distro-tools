package main

import (
	"github.com/charmbracelet/lipgloss"
)

type kubernetesUpstreamRelease struct {
	version   string
	published bool
}

// var kubernetesReleaseRunningStyle = lipgloss.NewStyle().Background(lipgloss.Color("220")).Foreground(lipgloss.AdaptiveColor{Light: "236", Dark: "248"})
// var kubernetesReleaseFailingStyle = lipgloss.NewStyle().Background(lipgloss.Color("160")).Foreground(lipgloss.AdaptiveColor{Light: "236", Dark: "248"})
// var kubernetesReleasePassingStyle = lipgloss.NewStyle().Background(lipgloss.Color("40")).Foreground(lipgloss.AdaptiveColor{Light: "236", Dark: "248"})
var kubernetesReleaseNotStartedStyle = lipgloss.NewStyle().Background(lipgloss.Color("255")).Foreground(lipgloss.AdaptiveColor{Light: "236", Dark: "248"})

func (i kubernetesUpstreamRelease) Title() string {
	return "[k8s] " + i.version
}
func (i kubernetesUpstreamRelease) Description() string {

	return kubernetesReleaseNotStartedStyle.Render("not started")
}
func (i kubernetesUpstreamRelease) FilterValue() string {
	return i.version
}
func (i kubernetesUpstreamRelease) ID() string {
	return i.version
}
func (i kubernetesUpstreamRelease) Org() string {
	return "kubernetes"
}
func (i kubernetesUpstreamRelease) Repo() string {
	return "kubernetes"
}
func (i kubernetesUpstreamRelease) Type() string {
	return "kubernetes_upstream_release"
}
