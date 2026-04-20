package ui

import (
	"fmt"
	"strings"

	"github.com/KalebHawkins/opcua-query/internal/opcbrowser"
)

func RenderFindReport(request opcbrowser.Request, result opcbrowser.FindResult) string {
	s := newStyles()
	status := s.Success.Render("CONNECTED")
	if result.Truncated {
		status = s.Warning.Render("CONNECTED WITH LIMITS")
	}

	sections := []string{
		s.Title.Render("OPC UA Find"),
		"",
		s.Card.Render(strings.Join([]string{
			linePair(s, "Status", status),
			linePair(s, "Server", result.Endpoint),
			linePair(s, "Start node", result.StartNode),
			linePair(s, "Query", result.Query),
			linePair(s, "Max depth", fmt.Sprintf("%d", request.MaxDepth)),
			linePair(s, "Visited", fmt.Sprintf("%d nodes in %s", result.NodesVisited, result.BrowseDuration.Round(10_000_000))),
			linePair(s, "Matches", fmt.Sprintf("%d", len(result.Matches))),
		}, "\n")),
		s.Section.Render("Matches"),
	}

	if len(result.Matches) == 0 {
		sections = append(sections, s.Warning.Render("No nodes matched the supplied search query."))
	} else {
		sections = append(sections, renderMatchesTable(s, result.Matches))
	}

	if len(result.NamespaceArray) > 0 {
		sections = append(sections, s.Section.Render("Namespaces"))
		sections = append(sections, s.Muted.Render(strings.Join(result.NamespaceArray, "\n")))
	}

	return strings.Join(sections, "\n") + "\n"
}
