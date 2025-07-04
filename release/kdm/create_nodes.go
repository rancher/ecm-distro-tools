package kdm

import (
	"strconv"
	"strings"
	"unicode"

	"gopkg.in/yaml.v3"
)

// createScalarNode creates a scalar YAML node (for keys or simple string values)
func createScalarNode(value string) *yaml.Node {
	return &yaml.Node{
		Kind:  yaml.ScalarNode,
		Tag:   "!!str", // Explicitly a string
		Value: value,
	}
}

// createSequenceNode creates a sequence node (array) from a slice of string values
func createSequenceNode(values []string) *yaml.Node {
	sequenceNode := &yaml.Node{
		Kind: yaml.SequenceNode,
		Tag:  "!!seq", // Explicitly a sequence
		// Style: 0, // Default block style. Use yaml.FlowStyle for [item1, item2]
	}

	// Populate the Content of the sequence node
	for _, valStr := range values {
		itemNode := createScalarNode(valStr) // Each item in the array is a scalar node
		sequenceNode.Content = append(sequenceNode.Content, itemNode)
	}
	return sequenceNode
}

// createArgsEntryNode creates a Mapping Node that follows the
// expected structure for serverArgs and agentArgs fields based on
// the provided Arg instance.
func createArgsEntryNode(arg Arg) *yaml.Node {
	content := []*yaml.Node{
		createScalarNode("default"), createScalarNode(arg.Default),
		createScalarNode("type"), createScalarNode(arg.Type),
	}

	if arg.Nullable {
		content = append(content,
			createScalarNode("nullable"), createScalarNode(strconv.FormatBool(arg.Nullable)),
		)
	}

	if len(arg.Options) > 0 {
		content = append(content,
			createScalarNode("options"), createSequenceNode(arg.Options),
		)
	}

	return &yaml.Node{
		Kind:    yaml.MappingNode,
		Tag:     "!!map",
		Content: content,
	}
}

// createChartEntryNode creates a mapping node for a chart entry {repo: ..., version: ...}
func createChartEntryNode(repo, version string) *yaml.Node {
	return &yaml.Node{
		Kind: yaml.MappingNode,
		Tag:  "!!map",
		Content: []*yaml.Node{
			createScalarNode("repo"), createScalarNode(repo),
			createScalarNode("version"), createScalarNode(version),
		},
	}
}

// createAliasNode creates a new alias node that points to an anchor.
func createAliasNode(anchorName string) *yaml.Node {
	return &yaml.Node{
		Kind:  yaml.AliasNode,
		Value: anchorName,
	}
}

// strictlyAlphanumeric sanitizes a string to be purely alphanumeric.
func strictlyAlphanumeric(input string) string {
	var sb strings.Builder
	for _, r := range input {
		if unicode.IsDigit(r) || unicode.IsLetter(r) {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

// anchorName sanitizes a string to a valid yaml anchor format
func anchorName(input string) string {
	input = strings.ReplaceAll(input, "+", "-")
	input = strings.ReplaceAll(input, ".", "-")

	return input
}
