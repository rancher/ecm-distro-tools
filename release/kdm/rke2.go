package kdm

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type (
	RKE2ChannelsUpdater struct {
		channels        RKE2Channels
		currentVersions []string
		rootNode        yaml.Node
		rootDoc         *yaml.Node
		releasesSeqNode *yaml.Node
		// replaceMap map is
		replaceMap map[string]string
	}

	RKE2Channels struct {
		Releases []Release `yaml:"releases"`
	}

	Release struct {
		Version                 string           `yaml:"version"`
		prevVersion             string           `yaml:"-"`
		MinChannelServerVersion string           `yaml:"minChannelServerVersion"`
		MaxChannelServerVersion string           `yaml:"maxChannelServerVersion"`
		ServerArgs              map[string]Arg   `yaml:"serverArgs"`
		serverArgsAnchor        string           `yaml:"-"`
		AgentArgs               map[string]Arg   `yaml:"agentArgs"`
		agentArgsAnchor         string           `yaml:"-"`
		Charts                  map[string]Chart `yaml:"charts"`
		chartsAnchor            string           `yaml:"-"`
	}

	Arg struct {
		Default  string   `yaml:"default"`
		Type     string   `yaml:"type"`
		Options  []string `yaml:"options"`
		Nullable bool     `yaml:"nullable"`
	}
)

const (
	rke2ChannelsFile = "channels-rke2.yaml"
)

func UpdateRKE2Channels(versions []string) error {
	u := &RKE2ChannelsUpdater{
		replaceMap:      make(map[string]string),
		currentVersions: make([]string, 0),
	}

	if err := u.parseYaml(rke2ChannelsFile); err != nil {
		return err
	}

	if err := u.setReleasesNode(); err != nil {
		return err
	}

	releases, err := u.getReleases(versions)
	if err != nil {
		return err
	}

	for _, release := range releases {
		if err := u.addRelease(release); err != nil {
			return err
		}
	}

	b, err := u.Bytes()
	if err != nil {
		return err
	}

	return os.WriteFile(rke2ChannelsFile, b, 0644)
}

func (u *RKE2ChannelsUpdater) parseYaml(filename string) error {
	yamlBytes, err := os.ReadFile(filename)
	if err != nil {
		return nil
	}

	var rke2channels RKE2Channels
	if err = yaml.Unmarshal(yamlBytes, &rke2channels); err != nil {
		return nil
	}

	u.channels = rke2channels

	var rootNode yaml.Node
	err = yaml.Unmarshal(yamlBytes, &rootNode)
	if err != nil {
		return nil
	}

	if rootNode.Kind != yaml.DocumentNode || len(rootNode.Content) == 0 {
		return fmt.Errorf("expected a YAML document node at the root")
	}

	u.rootNode = rootNode
	u.rootDoc = rootNode.Content[0]

	for _, v := range rke2channels.Releases {
		u.currentVersions = append(u.currentVersions, v.Version)
	}

	return nil
}

func (u *RKE2ChannelsUpdater) setReleasesNode() error {
	var releasesSeqNode *yaml.Node

	if u.rootDoc.Kind == yaml.MappingNode {
		// docContent.Content for a MappingNode is a flat list:
		// 	- [key1, value1, key2, value2, ...]
		// so here we need to iterate like i+=2
		for i := 0; i < len(u.rootDoc.Content); i += 2 {
			keyNode := u.rootDoc.Content[i]
			if keyNode.Kind == yaml.ScalarNode && keyNode.Value == "releases" {
				releasesSeqNode = u.rootDoc.Content[i+1]
				break
			}
		}
	}

	if releasesSeqNode == nil || releasesSeqNode.Kind != yaml.SequenceNode {
		return errors.New("could not find 'releases' sequence in YAML or it's not a sequence")
	}

	if len(releasesSeqNode.Content) == 0 {
		return errors.New("'releases' sequence is empty, cannot determine the last release")
	}

	u.releasesSeqNode = releasesSeqNode

	return nil
}

func (u *RKE2ChannelsUpdater) getReleases(versions []string) ([]Release, error) {
	releases := []Release{}
	for _, version := range versions {
		prevVersion, err := u.getPreviousVersion(version)
		if err != nil {
			return nil, err
		}

		chart, err := UpdatedCharts(version, prevVersion)
		if err != nil {
			return nil, err
		}

		releases = append(releases, Release{
			Version:     version,
			Charts:      chart,
			prevVersion: prevVersion,
		})

	}
	return releases, nil
}

const (
	rke2VersionTemplate = "v%d.%d.%d+rke2r%d"
)

func (u *RKE2ChannelsUpdater) getPreviousVersion(version string) (string, error) {
	// TODO:
	// 1. Support (v1.33.0+rke2r1) -> (v1.32.9+rke2r1)
	// 2. Support (v1.33.0+rke2r2) -> (v1.33.0+rke2r1)

	major, minor, patch, release, err := parseRKE2Version(version)
	if err != nil {
		return "", err
	}
	if release > 1 {
		// for releases higher than 1, we can just return the previous one.
		return fmt.Sprintf(rke2VersionTemplate, major, minor, patch, release-1), nil
	}

	// when the patch number is 0, e.g "v1.33.0+rke2r1" we need
	// to get the latest previous minor.
	if patch == 0 {
		prevVersion, err := u.getRKE2LatestMinor(major, minor)
		if err != nil {
			return "", err
		}
		return prevVersion, nil
	}

	return fmt.Sprintf(rke2VersionTemplate, major, minor, patch-1, 1), nil
}

func (u *RKE2ChannelsUpdater) getRKE2LatestMinor(major, minor int) (string, error) {
	baseVersion := fmt.Sprintf("v%d.%d", major, minor)

	for i := len(u.currentVersions) - 1; i >= 0; i-- {
		if strings.Contains(u.currentVersions[i], baseVersion) {
			return u.currentVersions[i], nil
		}
	}

	return "", errors.New("not found latest patch for " + baseVersion)
}

// parseRKE2Version receives a version in this format: vX.Y.Z+rke2rN
// and returns the major, minor, patch, and release numbers as integers.
func parseRKE2Version(version string) (int, int, int, int, error) {
	formatErr := fmt.Errorf("version %q is not in the expected format vX.Y.Z+rke2rN", version)

	if !strings.HasPrefix(version, "v") {
		return 0, 0, 0, 0, formatErr
	}

	version = strings.TrimPrefix(version, "v")

	// "1.2.3+rke2r4" -> ["1.2.3", "4"]
	parts := strings.Split(version, "+rke2r")
	if len(parts) != 2 {
		return 0, 0, 0, 0, formatErr
	}
	versionStr := parts[0]
	releaseStr := parts[1]

	// "1.2.3" -> ["1", "2", "3"]
	versionParts := strings.Split(versionStr, ".")
	if len(versionParts) != 3 {
		return 0, 0, 0, 0, formatErr
	}

	majorStr := versionParts[0]
	major, err := strconv.Atoi(majorStr)
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("invalid major version part %q: %w", majorStr, err)
	}

	minorStr := versionParts[1]
	minor, err := strconv.Atoi(minorStr)
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("invalid minor version part %q: %w", minorStr, err)
	}

	patchStr := versionParts[2]
	patch, err := strconv.Atoi(patchStr)
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("invalid patch version part %q: %w", patchStr, err)
	}

	release, err := strconv.Atoi(releaseStr)
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("invalid release version part %q: %w", releaseStr, err)
	}

	return major, minor, patch, release, nil
}

func (u *RKE2ChannelsUpdater) addRelease(release Release) error {
	newReleaseContent := make([]*yaml.Node, 0)
	newReleaseContent = append(newReleaseContent, createScalarNode("version"), createScalarNode(release.Version))

	prevReleasePos, prevRelease, err := u.getPreviousRelease(release.Version)
	if err != nil {
		return err
	}

	newReleaseContent = append(newReleaseContent, createScalarNode("minChannelServerVersion"), createScalarNode(prevRelease.MinChannelServerVersion))
	newReleaseContent = append(newReleaseContent, createScalarNode("maxChannelServerVersion"), createScalarNode(prevRelease.MaxChannelServerVersion))

	sanitizedVersionForAnchor := strictlyAlphanumeric(release.Version) // e.g., "v1216rke2r1"
	versionForAnchor := getAnchorName(release.Version)

	// defining charts
	{
		newChartsAnchorName := "charts" + sanitizedVersionForAnchor
		u.replaceMap[newChartsAnchorName] = "charts" + versionForAnchor
		chartsContent := []*yaml.Node{
			{Kind: yaml.ScalarNode, Tag: "!!merge", Value: "<<"},
			{Kind: yaml.AliasNode, Value: prevRelease.chartsAnchor}, // Alias value is the name of the anchor
		}
		for chartName, chart := range release.Charts {
			chartsContent = append(chartsContent,
				createScalarNode(chartName),
				createChartEntryNode(chart.Repo, chart.Version),
			)
		}
		chartsValueMapNode := &yaml.Node{
			Kind:    yaml.MappingNode,
			Tag:     "!!map",
			Anchor:  newChartsAnchorName,
			Content: chartsContent,
		}
		newReleaseContent = append(newReleaseContent, createScalarNode("charts"), chartsValueMapNode)
	}

	// defining serverArgs
	{
		newServerArgsAnchorName := "serverArgs" + sanitizedVersionForAnchor // e.g., serverArgsv1216rke2r1
		u.replaceMap[newServerArgsAnchorName] = "serverArgs" + versionForAnchor
		serverArgsContent := []*yaml.Node{
			{Kind: yaml.ScalarNode, Tag: "!!merge", Value: "<<"},
			{Kind: yaml.AliasNode, Value: prevRelease.serverArgsAnchor}, // Alias value is the name of the anchor
		}
		serverArgsValueMapNode := &yaml.Node{
			Kind:    yaml.MappingNode,
			Tag:     "!!map",
			Anchor:  newServerArgsAnchorName,
			Content: serverArgsContent,
		}
		newReleaseContent = append(newReleaseContent, createScalarNode("serverArgs"), serverArgsValueMapNode)
	}

	// defining agentArgs
	{
		newAgentArgsAnchorName := "agentArgs" + sanitizedVersionForAnchor // e.g., agentArgsv1216rke2r1
		u.replaceMap[newAgentArgsAnchorName] = "agentArgs" + versionForAnchor
		agentArgsContent := []*yaml.Node{
			{Kind: yaml.ScalarNode, Tag: "!!merge", Value: "<<"},
			{Kind: yaml.AliasNode, Value: prevRelease.agentArgsAnchor}, // Alias value is the name of the anchor
		}
		agentArgsValueMapNode := &yaml.Node{
			Kind:    yaml.MappingNode,
			Tag:     "!!map",
			Anchor:  newAgentArgsAnchorName,
			Content: agentArgsContent,
		}
		newReleaseContent = append(newReleaseContent, createScalarNode("agentArgs"), agentArgsValueMapNode)
	}

	newReleaseNode := &yaml.Node{
		Kind:    yaml.MappingNode,
		Tag:     "!!map",
		Content: newReleaseContent,
	}

	u.releasesSeqNode.Content = slices.Insert(u.releasesSeqNode.Content, prevReleasePos+1, newReleaseNode)

	return nil
}

func (u *RKE2ChannelsUpdater) getPreviousRelease(version string) (int, Release, error) {
	prevVersion, err := u.getPreviousVersion(version)
	if err != nil {
		return 0, Release{}, err
	}

	prevReleasePos, err := u.getPreviousReleasePos(prevVersion)
	if err != nil {
		return 0, Release{}, err
	}
	release := Release{}
	node := u.releasesSeqNode.Content[prevReleasePos]

	if node.Kind != yaml.MappingNode {
		return 0, Release{}, fmt.Errorf("not a mapping node in '%s' release", prevVersion)
	}
	for i := 0; i < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		valueNode := node.Content[i+1]
		if keyNode.Kind == yaml.ScalarNode {
			switch keyNode.Value {
			case "version":
				release.Version = valueNode.Value
			case "minChannelServerVersion":
				release.MinChannelServerVersion = valueNode.Value
			case "maxChannelServerVersion":
				release.MaxChannelServerVersion = valueNode.Value
			case "agentArgs":
				if valueNode.Kind == yaml.MappingNode && valueNode.Anchor != "" {
					release.agentArgsAnchor = valueNode.Anchor // This anchor name is from the file, assume it's valid
					continue
				}
				if valueNode.Kind == yaml.AliasNode {
					release.agentArgsAnchor = valueNode.Value
					continue
				}
			case "serverArgs":
				if valueNode.Kind == yaml.MappingNode && valueNode.Anchor != "" {
					release.serverArgsAnchor = valueNode.Anchor // This anchor name is from the file, assume it's valid
					continue
				}
				if valueNode.Kind == yaml.AliasNode {
					release.serverArgsAnchor = valueNode.Value
					continue
				}
			case "charts":
				if valueNode.Kind == yaml.MappingNode && valueNode.Anchor != "" {
					release.chartsAnchor = valueNode.Anchor // This anchor name is from the file, assume it's valid
					continue
				}
			}
		}
	}
	return prevReleasePos, release, nil
}

func (u *RKE2ChannelsUpdater) getPreviousReleasePos(version string) (int, error) {
	for i := 0; i < len(u.releasesSeqNode.Content); i++ {
		node := u.releasesSeqNode.Content[i]
		if node.Kind == yaml.MappingNode {
			for j := 0; j < len(node.Content); j += 2 {
				keyNode := node.Content[j]
				valueNode := node.Content[j+1]
				if keyNode.Kind == yaml.ScalarNode {
					switch keyNode.Value {
					case "version":
						if valueNode.Value == version {
							return i, nil
						}
					}
				}
			}
		}
	}
	return -1, fmt.Errorf("unable to find release '%s'", version)
}

func (u *RKE2ChannelsUpdater) Bytes() ([]byte, error) {
	var buf bytes.Buffer

	encoder := yaml.NewEncoder(&buf)

	encoder.SetIndent(2)

	if err := encoder.Encode(&u.rootNode); err != nil {
		return nil, err
	}

	if err := encoder.Close(); err != nil {
		return nil, err
	}

	outputBytes := buf.Bytes()
	outputBytes = bytes.ReplaceAll(outputBytes, []byte("!!merge "), nil)
	outputBytes = bytes.ReplaceAll(outputBytes, []byte(" {}"), nil)
	for k, v := range u.replaceMap {
		outputBytes = bytes.ReplaceAll(outputBytes, []byte(k), []byte(v))
	}
	return outputBytes, nil
}
