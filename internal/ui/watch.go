package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/KalebHawkins/opcua-query/internal/opcbrowser"
)

func RenderWatchSession(session opcbrowser.WatchSession) string {
	s := newStyles()
	sections := []string{
		s.Title.Render("OPC UA Watch"),
		"",
		s.Card.Render(strings.Join([]string{
			linePair(s, "Status", s.Success.Render("WATCHING")),
			linePair(s, "Server", session.Result.Endpoint),
			linePair(s, "Start node", session.Result.StartNode),
			linePair(s, "Filter", session.Result.Filter),
			linePair(s, "Strategy", session.Result.Strategy),
			linePair(s, "Resolved prefix", session.Result.ResolvedPrefix),
			linePair(s, "Subscription", fmt.Sprintf("%d", session.SubscriptionID)),
			linePair(s, "Interval", session.RevisedInterval.String()),
			linePair(s, "Watched nodes", fmt.Sprintf("%d", len(session.Targets))),
			linePair(s, "Stop", "Press Ctrl+C to cancel and close the OPC UA session"),
		}, "\n")),
	}

	sections = append(sections, s.Section.Render("Watched Nodes"))
	sections = append(sections, renderMatchesTable(s, session.Targets))

	if len(session.MonitorFailures) > 0 {
		warnings := make([]string, 0, len(session.MonitorFailures)+1)
		warnings = append(warnings, s.Section.Render("Subscription Warnings"))
		for _, failure := range session.MonitorFailures {
			warnings = append(warnings, s.Warning.Render(failure))
		}
		sections = append(sections, warnings...)
	}

	sections = append(sections, s.Section.Render("Live Updates"))
	return strings.Join(sections, "\n") + "\n"
}

func RenderWatchEvent(event opcbrowser.WatchEvent) string {
	s := newStyles()
	timestamp := event.SourceTimestamp
	if timestamp.IsZero() {
		timestamp = event.ServerTimestamp
	}

	parts := []string{
		s.Muted.Render(formatWatchTimestamp(timestamp)),
		s.Path.Render(event.Path),
		s.ValueCell.Render(renderWatchValue(event.Value)),
	}
	if event.Status != "" {
		parts = append(parts, s.Warning.Render("["+event.Status+"]"))
	}

	return strings.Join(parts, "  ") + "\n"
}

func RenderWatchStopped() string {
	s := newStyles()
	return s.Success.Render("Watch stopped. Subscription canceled and connection closed.") + "\n"
}

func formatWatchTimestamp(timestamp time.Time) string {
	if timestamp.IsZero() {
		return "-"
	}
	return timestamp.Format(time.RFC3339)
}

func renderWatchValue(value string) string {
	if strings.TrimSpace(value) == "" {
		return "<nil>"
	}
	return value
}
