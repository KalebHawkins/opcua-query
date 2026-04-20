package ui

import (
	"fmt"
	"strings"

	"github.com/KalebHawkins/opcua-query/internal/opcbrowser"
)

func RenderListReport(request opcbrowser.Request, result opcbrowser.ListResult) string {
	s := newStyles()
	status := s.Success.Render("CONNECTED")
	if result.Truncated {
		status = s.Warning.Render("CONNECTED WITH LIMITS")
	}

	sections := []string{
		s.Title.Render("OPC UA ls"),
		"",
		s.Card.Render(strings.Join([]string{
			linePair(s, "Status", status),
			linePair(s, "Server", result.Endpoint),
			linePair(s, "Start node", result.StartNode),
			linePair(s, "Path", result.Path),
			linePair(s, "Visited", fmt.Sprintf("%d nodes in %s", result.NodesVisited, result.BrowseDuration.Round(10_000_000))),
			linePair(s, "Resolved nodes", fmt.Sprintf("%d", len(result.CurrentNodes))),
			linePair(s, "Children", fmt.Sprintf("%d", len(result.Children))),
		}, "\n")),
	}

	sections = append(sections, s.Section.Render("Current Nodes"))
	if len(result.CurrentNodes) == 0 {
		sections = append(sections, s.Warning.Render("No nodes matched the supplied path."))
	} else {
		sections = append(sections, renderMatchesTable(s, result.CurrentNodes))
	}

	sections = append(sections, s.Section.Render("Children"))
	if len(result.Children) == 0 {
		sections = append(sections, s.Warning.Render("No child nodes found for the resolved location."))
	} else {
		sections = append(sections, renderMatchesTable(s, result.Children))
	}

	if len(result.NamespaceArray) > 0 {
		sections = append(sections, s.Section.Render("Namespaces"))
		sections = append(sections, s.Muted.Render(strings.Join(result.NamespaceArray, "\n")))
	}

	return strings.Join(sections, "\n") + "\n"
}
