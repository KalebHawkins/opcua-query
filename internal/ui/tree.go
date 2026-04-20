package ui

import (
	"fmt"
	"strings"

	"github.com/KalebHawkins/opcua-query/internal/opcbrowser"
)

func RenderTreeReport(request opcbrowser.Request, result opcbrowser.TreeResult) string {
	s := newStyles()
	status := s.Success.Render("CONNECTED")
	if result.Truncated {
		status = s.Warning.Render("CONNECTED WITH LIMITS")
	}

	sections := []string{
		s.Title.Render("OPC UA Tree"),
		"",
		s.Card.Render(strings.Join([]string{
			linePair(s, "Status", status),
			linePair(s, "Server", result.Endpoint),
			linePair(s, "Start node", result.StartNode),
			linePair(s, "Max depth", fmt.Sprintf("%d", request.MaxDepth)),
			linePair(s, "Visited", fmt.Sprintf("%d nodes in %s", result.NodesVisited, result.BrowseDuration.Round(10_000_000))),
		}, "\n")),
		s.Section.Render("Tree"),
		renderTree(result.Root),
	}

	if len(result.NamespaceArray) > 0 {
		sections = append(sections, s.Section.Render("Namespaces"))
		sections = append(sections, s.Muted.Render(strings.Join(result.NamespaceArray, "\n")))
	}

	return strings.Join(sections, "\n") + "\n"
}

func renderTree(root opcbrowser.TreeNode) string {
	lines := []string{formatTreeLabel(root.Match)}
	for index, child := range root.Children {
		last := index == len(root.Children)-1
		lines = append(lines, renderTreeNode(child, "", last)...)
	}
	return strings.Join(lines, "\n")
}

func renderTreeNode(node opcbrowser.TreeNode, prefix string, last bool) []string {
	connector := "|- "
	nextPrefix := prefix + "|  "
	if last {
		connector = "`- "
		nextPrefix = prefix + "   "
	}

	lines := []string{prefix + connector + formatTreeLabel(node.Match)}
	for index, child := range node.Children {
		lines = append(lines, renderTreeNode(child, nextPrefix, index == len(node.Children)-1)...)
	}
	return lines
}

func formatTreeLabel(match opcbrowser.Match) string {
	label := match.DisplayName
	if strings.TrimSpace(label) == "" {
		label = match.BrowseName
	}
	if strings.TrimSpace(label) == "" {
		label = match.Path
	}

	parts := []string{label}
	if match.DisplayName != "" && match.BrowseName != "" && match.DisplayName != match.BrowseName {
		parts = append(parts, fmt.Sprintf("(%s)", match.BrowseName))
	}
	if match.NodeClass != "" {
		parts = append(parts, fmt.Sprintf("[%s]", match.NodeClass))
	}
	if match.Value != "" {
		parts = append(parts, fmt.Sprintf("= %s", match.Value))
	}
	if match.ReadError != "" {
		parts = append(parts, fmt.Sprintf("[read error: %s]", match.ReadError))
	}
	return strings.Join(parts, " ")
}
