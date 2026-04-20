package opcbrowser

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/ua"
)

type TreeNode struct {
	Match    Match
	Children []TreeNode
}

type TreeResult struct {
	Endpoint       string
	StartNode      string
	NodesVisited   int
	BrowseDuration time.Duration
	Truncated      bool
	NamespaceArray []string
	Root           TreeNode
}

type FindResult struct {
	Endpoint       string
	StartNode      string
	Query          string
	NodesVisited   int
	BrowseDuration time.Duration
	Truncated      bool
	NamespaceArray []string
	Matches        []Match
}

type ListResult struct {
	Endpoint       string
	StartNode      string
	Path           string
	NodesVisited   int
	BrowseDuration time.Duration
	Truncated      bool
	NamespaceArray []string
	CurrentNodes   []Match
	Children       []Match
}

func Tree(ctx context.Context, req Request) (TreeResult, error) {
	client, err := connectClient(ctx, req)
	if err != nil {
		return TreeResult{}, err
	}
	defer closeClient(client)

	return treeWithClient(ctx, client, req)
}

func Find(ctx context.Context, req Request, query string) (FindResult, error) {
	trimmed := strings.TrimSpace(query)
	if trimmed == "" {
		return FindResult{}, fmt.Errorf("name is required")
	}

	client, err := connectClient(ctx, req)
	if err != nil {
		return FindResult{}, err
	}
	defer closeClient(client)

	return findWithClient(ctx, client, req, trimmed)
}

func List(ctx context.Context, req Request, path string) (ListResult, error) {
	client, err := connectClient(ctx, req)
	if err != nil {
		return ListResult{}, err
	}
	defer closeClient(client)

	return listWithClient(ctx, client, req, path)
}

func treeWithClient(ctx context.Context, client *opcua.Client, req Request) (TreeResult, error) {
	if req.MaxDepth <= 0 {
		req.MaxDepth = 8
	}
	if req.MaxNodes <= 0 {
		req.MaxNodes = 1000
	}
	if req.Timeout <= 0 {
		req.Timeout = 10 * time.Second
	}

	browseCtx, cancel := context.WithTimeout(ctx, req.Timeout)
	defer cancel()

	startNodeID, err := resolveStartNode(req.StartNode)
	if err != nil {
		return TreeResult{}, err
	}

	start := time.Now()
	namespaces, _ := client.NamespaceArray(browseCtx)
	rootNode := client.Node(startNodeID)
	browseName, displayName, nodeClass, err := describeCurrentNode(browseCtx, rootNode)
	if err != nil {
		browseName = startNodeID.String()
		displayName = startNodeID.String()
		nodeClass = ua.NodeClassUnspecified
	}

	result := TreeResult{
		Endpoint:       normalizeEndpoint(req.Endpoint),
		StartNode:      startNodeID.String(),
		NamespaceArray: namespaces,
		Root: TreeNode{
			Match: newMatch(browseCtx, rootNode, "/", browseName, displayName, nodeClass, 0, req.ReadValues),
		},
	}

	seen := map[string]struct{}{startNodeID.String(): {}}
	result.Root.Children, err = buildTreeChildren(browseCtx, client, browseItem{
		Node:        rootNode,
		Path:        "/",
		Depth:       0,
		BrowseName:  browseName,
		DisplayName: displayName,
		NodeClass:   nodeClass,
		HasMetadata: true,
	}, req, &result, seen)
	if err != nil {
		return TreeResult{}, err
	}

	result.BrowseDuration = time.Since(start)
	return result, nil
}

func findWithClient(ctx context.Context, client *opcua.Client, req Request, query string) (FindResult, error) {
	if req.MaxDepth <= 0 {
		req.MaxDepth = 8
	}
	if req.MaxNodes <= 0 {
		req.MaxNodes = 1000
	}
	if req.Timeout <= 0 {
		req.Timeout = 10 * time.Second
	}

	browseCtx, cancel := context.WithTimeout(ctx, req.Timeout)
	defer cancel()

	startNodeID, err := resolveStartNode(req.StartNode)
	if err != nil {
		return FindResult{}, err
	}

	start := time.Now()
	namespaces, _ := client.NamespaceArray(browseCtx)
	result := FindResult{
		Endpoint:       normalizeEndpoint(req.Endpoint),
		StartNode:      startNodeID.String(),
		Query:          query,
		NamespaceArray: namespaces,
	}

	root := client.Node(startNodeID)
	queue := []browseItem{{Node: root, Path: "/", Depth: 0}}
	seen := map[string]struct{}{startNodeID.String(): {}}
	matched := map[string]struct{}{}

	rootBrowseName, rootDisplayName, rootClass, err := describeCurrentNode(browseCtx, root)
	if err == nil {
		queue[0].BrowseName = rootBrowseName
		queue[0].DisplayName = rootDisplayName
		queue[0].NodeClass = rootClass
		queue[0].HasMetadata = true
		rootMatch := newMatch(browseCtx, root, "/", rootBrowseName, rootDisplayName, rootClass, 0, req.ReadValues)
		if matchesFindQuery(query, rootMatch) {
			result.Matches = append(result.Matches, buildMatch(browseCtx, root, "/", rootBrowseName, rootDisplayName, rootClass, 0, req.ReadValues, matched)...)
		}
	}

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
				return FindResult{}, fmt.Errorf("browse timed out after %s", req.Timeout)
			}
			continue
		}

		for _, child := range children {
			match := newMatch(browseCtx, child.Node, child.Path, child.BrowseName, child.DisplayName, child.NodeClass, child.Depth, req.ReadValues)
			if matchesFindQuery(query, match) {
				result.Matches = append(result.Matches, buildMatch(browseCtx, child.Node, child.Path, child.BrowseName, child.DisplayName, child.NodeClass, child.Depth, req.ReadValues, matched)...)
			}

			if child.Depth >= req.MaxDepth {
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

	sort.Slice(result.Matches, func(i, j int) bool {
		if result.Matches[i].Path == result.Matches[j].Path {
			return result.Matches[i].NodeID < result.Matches[j].NodeID
		}
		return result.Matches[i].Path < result.Matches[j].Path
	})
	result.BrowseDuration = time.Since(start)
	return result, nil
}

func listWithClient(ctx context.Context, client *opcua.Client, req Request, path string) (ListResult, error) {
	if req.MaxNodes <= 0 {
		req.MaxNodes = 1000
	}
	if req.Timeout <= 0 {
		req.Timeout = 10 * time.Second
	}

	browseCtx, cancel := context.WithTimeout(ctx, req.Timeout)
	defer cancel()

	startNodeID, err := resolveStartNode(req.StartNode)
	if err != nil {
		return ListResult{}, err
	}

	normalizedPath, err := normalizeBrowsePath(path)
	if err != nil {
		return ListResult{}, err
	}

	start := time.Now()
	namespaces, _ := client.NamespaceArray(browseCtx)
	result := ListResult{
		Endpoint:       normalizeEndpoint(req.Endpoint),
		StartNode:      startNodeID.String(),
		Path:           normalizedPath,
		NamespaceArray: namespaces,
	}

	rootNode := client.Node(startNodeID)
	rootItem, err := describeBrowseItem(browseCtx, rootNode, "/", 0)
	if err != nil {
		rootItem = browseItem{
			Node:        rootNode,
			Path:        "/",
			Depth:       0,
			BrowseName:  startNodeID.String(),
			DisplayName: startNodeID.String(),
			NodeClass:   ua.NodeClassUnspecified,
			HasMetadata: true,
		}
	}

	frontier := []browseItem{rootItem}
	if normalizedPath != "/" {
		segments, hasWildcard := splitLiteralPrefix(normalizedPath)
		if hasWildcard {
			return ListResult{}, fmt.Errorf("path must not include wildcards")
		}

		resolverStats := Result{}
		frontier, err = resolveLiteralPrefix(browseCtx, client, frontier, segments, req.MaxNodes, req.Timeout, &resolverStats, map[string]struct{}{})
		if err != nil {
			return ListResult{}, err
		}
		result.NodesVisited += resolverStats.NodesVisited
		result.Truncated = resolverStats.Truncated
	}

	if len(frontier) == 0 {
		result.BrowseDuration = time.Since(start)
		return result, nil
	}

	currentDedup := map[string]struct{}{}
	for _, item := range frontier {
		result.CurrentNodes = append(result.CurrentNodes, buildMatch(browseCtx, item.Node, item.Path, item.BrowseName, item.DisplayName, item.NodeClass, item.Depth, req.ReadValues, currentDedup)...)
	}

	childDedup := map[string]struct{}{}
	for _, current := range frontier {
		if result.NodesVisited >= req.MaxNodes {
			result.Truncated = true
			break
		}

		result.NodesVisited++
		children, err := browseChildren(browseCtx, client, current)
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				return ListResult{}, fmt.Errorf("browse timed out after %s", req.Timeout)
			}
			continue
		}

		for _, child := range children {
			result.Children = append(result.Children, buildMatch(browseCtx, child.Node, child.Path, child.BrowseName, child.DisplayName, child.NodeClass, child.Depth, req.ReadValues, childDedup)...)
		}
	}

	sort.Slice(result.CurrentNodes, func(i, j int) bool {
		if result.CurrentNodes[i].Path == result.CurrentNodes[j].Path {
			return result.CurrentNodes[i].NodeID < result.CurrentNodes[j].NodeID
		}
		return result.CurrentNodes[i].Path < result.CurrentNodes[j].Path
	})
	sort.Slice(result.Children, func(i, j int) bool {
		if result.Children[i].Path == result.Children[j].Path {
			return result.Children[i].NodeID < result.Children[j].NodeID
		}
		return result.Children[i].Path < result.Children[j].Path
	})

	result.BrowseDuration = time.Since(start)
	return result, nil
}

func buildTreeChildren(ctx context.Context, client *opcua.Client, current browseItem, req Request, result *TreeResult, seen map[string]struct{}) ([]TreeNode, error) {
	if current.Depth >= req.MaxDepth {
		return nil, nil
	}
	if result.NodesVisited >= req.MaxNodes {
		result.Truncated = true
		return nil, nil
	}

	result.NodesVisited++
	children, err := browseChildren(ctx, client, current)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, fmt.Errorf("browse timed out after %s", req.Timeout)
		}
		return nil, nil
	}

	nodes := make([]TreeNode, 0, len(children))
	for _, child := range children {
		node := TreeNode{
			Match: newMatch(ctx, child.Node, child.Path, child.BrowseName, child.DisplayName, child.NodeClass, child.Depth, req.ReadValues),
		}

		childKey := child.Node.ID.String()
		if _, ok := seen[childKey]; !ok && child.Depth < req.MaxDepth {
			seen[childKey] = struct{}{}
			node.Children, err = buildTreeChildren(ctx, client, child, req, result, seen)
			if err != nil {
				return nil, err
			}
		}

		nodes = append(nodes, node)
		if result.Truncated {
			break
		}
	}

	return nodes, nil
}

func matchesFindQuery(query string, match Match) bool {
	needle := strings.ToLower(strings.TrimSpace(query))
	if needle == "" {
		return false
	}

	fields := []string{match.Path, match.BrowseName, match.DisplayName, match.NodeID}
	for _, field := range fields {
		if strings.Contains(strings.ToLower(field), needle) {
			return true
		}
	}

	return false
}

func normalizeBrowsePath(path string) (string, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "/", nil
	}
	if strings.Contains(trimmed, "*") {
		return "", fmt.Errorf("path must not include wildcards")
	}
	if !strings.HasPrefix(trimmed, "/") {
		trimmed = "/" + trimmed
	}
	if trimmed != "/" {
		trimmed = strings.TrimRight(trimmed, "/")
	}
	return trimmed, nil
}

func describeBrowseItem(ctx context.Context, node *opcua.Node, path string, depth int) (browseItem, error) {
	browseName, displayName, nodeClass, err := describeCurrentNode(ctx, node)
	if err != nil {
		return browseItem{}, err
	}

	return browseItem{
		Node:        node,
		Path:        path,
		Depth:       depth,
		BrowseName:  browseName,
		DisplayName: displayName,
		NodeClass:   nodeClass,
		HasMetadata: true,
	}, nil
}
