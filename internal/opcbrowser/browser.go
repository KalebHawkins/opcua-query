package opcbrowser

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/id"
	"github.com/gopcua/opcua/ua"

	"github.com/KalebHawkins/opcua-query/internal/sitewise"
)

type Request struct {
	Endpoint   string
	Username   string
	Password   string
	Timeout    time.Duration
	Filter     string
	StartNode  string
	MaxDepth   int
	MaxNodes   int
	ReadValues bool
	Interval   time.Duration
}

type Match struct {
	Path        string
	NodeID      string
	BrowseName  string
	DisplayName string
	NodeClass   string
	Value       string
	Depth       int
	ReadError   string
}

type Result struct {
	Endpoint          string
	StartNode         string
	NodesVisited      int
	MatchedNodes      []Match
	Filter            string
	BrowseDuration    time.Duration
	Truncated         bool
	NamespaceArray    []string
	DiscoveredFilters []string
	ResolvedPrefix    string
	Strategy          string
}

func Browse(ctx context.Context, req Request) (Result, error) {
	endpoint := normalizeEndpoint(req.Endpoint)
	authType := ua.UserTokenTypeAnonymous
	if req.Username != "" {
		authType = ua.UserTokenTypeUserName
	}

	client, err := newClient(ctx, endpoint, req, authType)
	if err != nil {
		return Result{}, err
	}
	defer closeClient(client)

	return browseWithClient(ctx, client, req)
}

func browseWithClient(ctx context.Context, client *opcua.Client, req Request) (Result, error) {
	if req.MaxDepth <= 0 {
		req.MaxDepth = 8
	}
	if req.MaxNodes <= 0 {
		req.MaxNodes = 1000
	}
	if req.Timeout <= 0 {
		req.Timeout = 10 * time.Second
	}

	endpoint := normalizeEndpoint(req.Endpoint)

	browseCtx, cancel := context.WithTimeout(ctx, req.Timeout)
	defer cancel()

	startNodeID, err := resolveStartNode(req.StartNode)
	if err != nil {
		return Result{}, err
	}

	start := time.Now()
	namespaces, _ := client.NamespaceArray(browseCtx)
	result := Result{
		Endpoint:       endpoint,
		StartNode:      startNodeID.String(),
		Filter:         req.Filter,
		NamespaceArray: namespaces,
		ResolvedPrefix: "/",
		Strategy:       "recursive browse from start node",
	}

	root := client.Node(startNodeID)
	frontier := []browseItem{{Node: root, Path: "/", Depth: 0}}
	seen := map[string]struct{}{startNodeID.String(): {}}
	matched := map[string]struct{}{}
	discovered := map[string]struct{}{}

	literalPrefix, hasWildcard := splitLiteralPrefix(req.Filter)
	if len(literalPrefix) > 0 {
		result.Strategy = "targeted prefix browse"
		frontier, err = resolveLiteralPrefix(browseCtx, client, frontier, literalPrefix, req.MaxNodes, req.Timeout, &result, discovered)
		if err != nil {
			return Result{}, err
		}
		if len(frontier) == 0 {
			result.DiscoveredFilters = discoveredFilters(discovered)
			result.BrowseDuration = time.Since(start)
			return result, nil
		}
		result.ResolvedPrefix = frontier[0].Path
		for _, item := range frontier {
			seen[item.Node.ID.String()] = struct{}{}
		}
	}

	collectCurrentMatches(browseCtx, frontier, req, &result, matched)

	if !hasWildcard {
		sort.Slice(result.MatchedNodes, func(i, j int) bool {
			return result.MatchedNodes[i].Path < result.MatchedNodes[j].Path
		})
		result.DiscoveredFilters = discoveredFilters(discovered)
		result.BrowseDuration = time.Since(start)
		return result, nil
	}

	queue := append([]browseItem(nil), frontier...)

	for len(queue) > 0 {
		if result.NodesVisited >= req.MaxNodes {
			result.Truncated = true
			break
		}

		current := queue[0]
		queue = queue[1:]
		if current.Depth >= req.MaxDepth {
			continue
		}
		result.NodesVisited++

		children, err := browseChildren(browseCtx, client, current)
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				return Result{}, fmt.Errorf("browse timed out after %s", req.Timeout)
			}
			continue
		}

		for _, child := range children {
			discovered[child.Path] = struct{}{}

			if sitewise.Match(req.Filter, child.Path) {
				result.MatchedNodes = append(result.MatchedNodes, buildMatch(browseCtx, child.Node, child.Path, child.BrowseName, child.DisplayName, child.NodeClass, child.Depth, req.ReadValues, matched)...)
			}

			if current.Depth+1 >= req.MaxDepth {
				continue
			}

			childKey := child.Node.ID.String()
			if _, ok := seen[childKey]; ok {
				continue
			}
			seen[childKey] = struct{}{}
			queue = append(queue, child)
		}
	}

	sort.Slice(result.MatchedNodes, func(i, j int) bool {
		return result.MatchedNodes[i].Path < result.MatchedNodes[j].Path
	})
	result.DiscoveredFilters = discoveredFilters(discovered)
	result.BrowseDuration = time.Since(start)
	return result, nil
}

type browseItem struct {
	Node        *opcua.Node
	Path        string
	Depth       int
	BrowseName  string
	DisplayName string
	NodeClass   ua.NodeClass
	HasMetadata bool
}

func resolveLiteralPrefix(ctx context.Context, client *opcua.Client, frontier []browseItem, segments []string, maxNodes int, timeout time.Duration, result *Result, discovered map[string]struct{}) ([]browseItem, error) {
	for _, segment := range segments {
		next := make([]browseItem, 0)
		seen := make(map[string]struct{})

		for _, current := range frontier {
			if result.NodesVisited >= maxNodes {
				result.Truncated = true
				return uniqueBrowseItems(next), nil
			}

			result.NodesVisited++
			children, err := browseChildren(ctx, client, current)
			if err != nil {
				if errors.Is(err, context.DeadlineExceeded) {
					return nil, fmt.Errorf("browse timed out after %s", timeout)
				}
				continue
			}

			for _, child := range children {
				discovered[child.Path] = struct{}{}
				if child.BrowseName != segment {
					continue
				}

				childKey := child.Node.ID.String()
				if _, ok := seen[childKey]; ok {
					continue
				}
				seen[childKey] = struct{}{}
				next = append(next, child)
			}
		}

		frontier = uniqueBrowseItems(next)
		if len(frontier) == 0 {
			return nil, nil
		}
	}

	return frontier, nil
}

func collectCurrentMatches(ctx context.Context, frontier []browseItem, req Request, result *Result, matched map[string]struct{}) {
	for _, current := range frontier {
		if current.Path == "/" || !sitewise.Match(req.Filter, current.Path) {
			continue
		}

		browseName := current.BrowseName
		displayName := current.DisplayName
		nodeClass := current.NodeClass
		if !current.HasMetadata {
			var err error
			browseName, displayName, nodeClass, err = describeCurrentNode(ctx, current.Node)
			if err != nil {
				continue
			}
		}
		result.MatchedNodes = append(result.MatchedNodes, buildMatch(ctx, current.Node, current.Path, browseName, displayName, nodeClass, current.Depth, req.ReadValues, matched)...)
	}
}

func browseChildren(ctx context.Context, client *opcua.Client, current browseItem) ([]browseItem, error) {
	references, err := current.Node.References(ctx, id.HierarchicalReferences, ua.BrowseDirectionForward, ua.NodeClassAll, true)
	if err != nil {
		return nil, err
	}

	children := make([]browseItem, 0, len(references))
	for _, reference := range references {
		if reference == nil || reference.NodeID == nil || reference.BrowseName == nil {
			continue
		}

		browseName := strings.TrimSpace(reference.BrowseName.Name)
		if browseName == "" {
			continue
		}

		displayName := browseName
		if reference.DisplayName != nil && strings.TrimSpace(reference.DisplayName.Text) != "" {
			displayName = reference.DisplayName.Text
		}

		node := client.NodeFromExpandedNodeID(reference.NodeID)
		if node == nil {
			continue
		}

		children = append(children, browseItem{
			Node:        node,
			Path:        joinPath(current.Path, browseName),
			Depth:       current.Depth + 1,
			BrowseName:  browseName,
			DisplayName: displayName,
			NodeClass:   reference.NodeClass,
			HasMetadata: true,
		})
	}

	return uniqueBrowseItems(children), nil
}

func buildMatch(ctx context.Context, node *opcua.Node, nodePath, browseName, displayName string, nodeClass ua.NodeClass, depth int, readValues bool, matched map[string]struct{}) []Match {
	key := node.ID.String() + "|" + nodePath
	if _, ok := matched[key]; ok {
		return nil
	}
	matched[key] = struct{}{}

	match := Match{
		Path:        nodePath,
		NodeID:      node.ID.String(),
		BrowseName:  browseName,
		DisplayName: displayName,
		NodeClass:   nodeClass.String(),
		Depth:       depth,
	}
	if readValues && nodeClass == ua.NodeClassVariable {
		value, readErr := node.Value(ctx)
		if readErr != nil {
			match.ReadError = readErr.Error()
		} else if value != nil {
			match.Value = renderVariant(value)
		}
	}
	return []Match{match}
}

func describeCurrentNode(ctx context.Context, node *opcua.Node) (string, string, ua.NodeClass, error) {
	_, browseName, displayName, nodeClass, err := describeNode(ctx, node)
	if err != nil {
		return "", "", ua.NodeClassUnspecified, err
	}
	return browseName, displayName, nodeClass, nil
}

func splitLiteralPrefix(filter string) ([]string, bool) {
	trimmed := strings.TrimPrefix(filter, "/")
	if trimmed == "" {
		return nil, false
	}

	segments := strings.Split(trimmed, "/")
	prefix := make([]string, 0, len(segments))
	for _, segment := range segments {
		if segment == "" {
			continue
		}
		if strings.Contains(segment, "*") {
			return prefix, true
		}
		prefix = append(prefix, segment)
	}
	return prefix, false
}

func uniqueBrowseItems(items []browseItem) []browseItem {
	unique := make([]browseItem, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		key := item.Node.ID.String() + "|" + item.Path
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		unique = append(unique, item)
	}
	return unique
}

func newClient(ctx context.Context, endpoint string, req Request, authType ua.UserTokenType) (*opcua.Client, error) {
	endpoints, err := opcua.GetEndpoints(ctx, endpoint)
	if err != nil {
		return nil, fmt.Errorf("discover endpoints for %s: %w", endpoint, err)
	}

	selected, err := opcua.SelectEndpoint(endpoints, ua.SecurityPolicyURINone, ua.MessageSecurityModeNone)
	if err != nil {
		return nil, fmt.Errorf("select insecure endpoint: %w", err)
	}

	options := []opcua.Option{
		opcua.SecurityFromEndpoint(selected, authType),
		opcua.RequestTimeout(req.Timeout),
		opcua.SessionTimeout(req.Timeout),
		opcua.AutoReconnect(false),
	}
	if authType == ua.UserTokenTypeUserName {
		options = append(options, opcua.AuthUsername(req.Username, req.Password))
	} else {
		options = append(options, opcua.AuthAnonymous())
	}

	client, err := opcua.NewClient(endpoint, options...)
	if err != nil {
		return nil, fmt.Errorf("create client: %w", err)
	}
	if err := client.Connect(ctx); err != nil {
		return nil, fmt.Errorf("connect to %s: %w", endpoint, err)
	}
	return client, nil
}

func resolveStartNode(value string) (*ua.NodeID, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ua.NewNumericNodeID(0, id.ObjectsFolder), nil
	}

	nodeID, err := ua.ParseNodeID(trimmed)
	if err != nil {
		return nil, fmt.Errorf("parse start-node %q: %w", trimmed, err)
	}
	return nodeID, nil
}

func describeNode(ctx context.Context, node *opcua.Node) (string, string, string, ua.NodeClass, error) {
	browseName, err := node.BrowseName(ctx)
	if err != nil {
		return "", "", "", ua.NodeClassUnspecified, err
	}
	if browseName == nil || strings.TrimSpace(browseName.Name) == "" {
		return "", "", "", ua.NodeClassUnspecified, fmt.Errorf("empty browse name")
	}

	displayNameText := browseName.Name
	displayName, err := node.DisplayName(ctx)
	if err == nil && displayName != nil && strings.TrimSpace(displayName.Text) != "" {
		displayNameText = displayName.Text
	}

	nodeClass, err := node.NodeClass(ctx)
	if err != nil {
		return "", "", "", ua.NodeClassUnspecified, err
	}

	return browseName.Name, browseName.Name, displayNameText, nodeClass, nil
}

func renderVariant(value *ua.Variant) string {
	if value == nil {
		return ""
	}

	rendered := strings.TrimSpace(fmt.Sprint(value.Value()))
	if rendered == "<nil>" {
		return value.String()
	}
	return rendered
}

func normalizeEndpoint(value string) string {
	trimmed := strings.TrimSpace(value)
	if strings.HasPrefix(trimmed, "opc.tcp://") {
		return trimmed
	}
	return "opc.tcp://" + trimmed
}

func joinPath(parent, child string) string {
	child = strings.TrimSpace(child)
	if parent == "/" {
		return "/" + child
	}
	return parent + "/" + child
}

func discoveredFilters(discovered map[string]struct{}) []string {
	filters := make([]string, 0, len(discovered))
	for candidate := range discovered {
		filters = append(filters, candidate)
	}
	sort.Strings(filters)
	if len(filters) > 12 {
		filters = filters[:12]
	}
	return filters
}

func closeClient(client *opcua.Client) {
	cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client.Close(cleanupCtx)
}
