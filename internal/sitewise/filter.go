package sitewise

import (
	"encoding/json"
	"fmt"
	"path"
	"strings"
)

type Payload struct {
	RootPath string
	JSON     string
}

type nodeFilterRule struct {
	Action     string                   `json:"action"`
	Definition nodeFilterRuleDefinition `json:"definition"`
}

type nodeFilterRuleDefinition struct {
	Type     string `json:"type"`
	RootPath string `json:"rootPath"`
}

func NormalizeRootPath(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", fmt.Errorf("filter cannot be empty")
	}

	if !strings.HasPrefix(trimmed, "/") {
		trimmed = "/" + trimmed
	}

	trimmed = strings.ReplaceAll(trimmed, "\\", "/")
	trimmed = collapseSlashes(trimmed)
	if trimmed != "/" && strings.HasSuffix(trimmed, "/") {
		trimmed = strings.TrimSuffix(trimmed, "/")
	}

	return trimmed, nil
}

func BuildPayload(rootPath string) (Payload, error) {
	normalized, err := NormalizeRootPath(rootPath)
	if err != nil {
		return Payload{}, err
	}

	rules := []nodeFilterRule{{
		Action: "INCLUDE",
		Definition: nodeFilterRuleDefinition{
			Type:     "OpcUaRootPath",
			RootPath: normalized,
		},
	}}

	encoded, err := json.MarshalIndent(rules, "", "  ")
	if err != nil {
		return Payload{}, fmt.Errorf("marshal sitewise filter payload: %w", err)
	}

	return Payload{
		RootPath: normalized,
		JSON:     string(encoded),
	}, nil
}

func Match(rootPath, candidate string) bool {
	normalizedRule, err := NormalizeRootPath(rootPath)
	if err != nil {
		return false
	}

	normalizedCandidate, err := NormalizeRootPath(candidate)
	if err != nil {
		return false
	}

	if normalizedRule == "/" || normalizedRule == "/**" {
		return true
	}

	return matchSegments(splitSegments(normalizedRule), splitSegments(normalizedCandidate))
}

func matchSegments(rule []string, candidate []string) bool {
	if len(rule) == 0 {
		return len(candidate) == 0
	}

	if rule[0] == "**" {
		if matchSegments(rule[1:], candidate) {
			return true
		}
		for i := 0; i < len(candidate); i++ {
			if matchSegments(rule[1:], candidate[i+1:]) {
				return true
			}
		}
		return false
	}

	if len(candidate) == 0 {
		return false
	}

	matched, err := path.Match(rule[0], candidate[0])
	if err != nil || !matched {
		return false
	}

	return matchSegments(rule[1:], candidate[1:])
}

func splitSegments(value string) []string {
	if value == "/" {
		return nil
	}

	parts := strings.Split(strings.TrimPrefix(value, "/"), "/")
	segments := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == "" {
			continue
		}
		segments = append(segments, part)
	}
	return segments
}

func collapseSlashes(value string) string {
	for strings.Contains(value, "//") {
		value = strings.ReplaceAll(value, "//", "/")
	}
	return value
}
