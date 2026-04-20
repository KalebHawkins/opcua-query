package ui

import (
	"fmt"
	"strings"

	"github.com/KalebHawkins/opcua-query/internal/opcbrowser"
	"github.com/KalebHawkins/opcua-query/internal/sitewise"
)

func RenderBrowseReport(request opcbrowser.Request, result opcbrowser.Result, payload sitewise.Payload, copied bool, copyFormat string) string {
	s := newStyles()
	var sections []string

	status := s.Success.Render("CONNECTED")
	if result.Truncated {
		status = s.Warning.Render("CONNECTED WITH LIMITS")
	}

	summary := []string{
		s.Title.Render("Node Filter Preview"),
		"",
		s.Card.Render(strings.Join([]string{
			linePair(s, "Status", status),
			linePair(s, "Server", result.Endpoint),
			linePair(s, "Start node", result.StartNode),
			linePair(s, "Filter", payload.RootPath),
			linePair(s, "Strategy", result.Strategy),
			linePair(s, "Resolved prefix", result.ResolvedPrefix),
			linePair(s, "Visited", fmt.Sprintf("%d nodes in %s", result.NodesVisited, result.BrowseDuration.Round(10_000_000))),
			linePair(s, "Matches", fmt.Sprintf("%d", len(result.MatchedNodes))),
		}, "\n")),
	}
	sections = append(sections, strings.Join(summary, "\n"))

	if copied {
		sections = append(sections, s.Success.Render(fmt.Sprintf("Copied %s to clipboard.", copyFormat)))
	}

	sections = append(sections, s.Section.Render("Matched Nodes"))
	if len(result.MatchedNodes) == 0 {
		sections = append(sections, s.Warning.Render("No nodes matched the supplied SiteWise filter."))
	} else {
		sections = append(sections, renderMatchesTable(s, result.MatchedNodes))
	}

	if len(result.DiscoveredFilters) > 0 {
		sections = append(sections, s.Section.Render("Discovered Paths"))
		var paths []string
		for _, candidate := range result.DiscoveredFilters {
			paths = append(paths, "  "+s.Path.Render(candidate))
		}
		sections = append(sections, strings.Join(paths, "\n"))
	}

	sections = append(sections, s.Section.Render("SiteWise Payload"))
	sections = append(sections, s.Card.Render(payload.JSON))

	if len(result.NamespaceArray) > 0 {
		sections = append(sections, s.Section.Render("Namespaces"))
		sections = append(sections, s.Muted.Render(strings.Join(result.NamespaceArray, "\n")))
	}

	return strings.Join(sections, "\n") + "\n"
}

func linePair(s styles, label, value string) string {
	return s.Label.Render(label+":") + " " + s.Value.Render(value)
}

func renderMatchesTable(s styles, matches []opcbrowser.Match) string {
	headers := []string{"PATH", "CLASS", "VALUE", "NODE ID"}
	widths := []int{44, 12, 24, 20}

	var lines []string
	lines = append(lines, formatRow(s, widths, headers, true))
	lines = append(lines, s.Border.Render(strings.Repeat("-", 108)))

	for _, match := range matches {
		value := match.Value
		if value == "" {
			value = "-"
		}
		if match.ReadError != "" {
			value = "read error: " + match.ReadError
		}

		row := []string{
			match.Path,
			match.NodeClass,
			value,
			match.NodeID,
		}
		lines = append(lines, formatRow(s, widths, row, false))
	}

	return strings.Join(lines, "\n")
}

func formatRow(s styles, widths []int, values []string, header bool) string {
	rendered := make([]string, 0, len(values))
	for i, value := range values {
		cell := truncate(value, widths[i])
		style := s.Value
		if header {
			style = s.HeaderCell
		}
		if !header {
			switch i {
			case 0:
				style = s.Path
			case 2:
				style = s.ValueCell
			case 3:
				style = s.NodeID
			}
		}
		padding := widths[i] - lipWidth(cell)
		if padding < 0 {
			padding = 0
		}
		rendered = append(rendered, style.Render(cell+strings.Repeat(" ", padding)))
	}
	return strings.Join(rendered, "  ")
}

func truncate(value string, limit int) string {
	clean := strings.ReplaceAll(value, "\n", " ")
	if lipWidth(clean) <= limit {
		return clean
	}
	if limit <= 1 {
		return clean[:limit]
	}
	runes := []rune(clean)
	if len(runes) <= limit {
		return clean
	}
	return string(runes[:limit-1]) + "…"
}

func lipWidth(value string) int {
	return len([]rune(value))
}
