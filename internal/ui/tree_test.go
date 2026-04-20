package ui

import (
	"testing"

	"github.com/KalebHawkins/opcua-query/internal/opcbrowser"
)

func TestRenderTree(t *testing.T) {
	t.Parallel()

	root := opcbrowser.TreeNode{
		Match: opcbrowser.Match{DisplayName: "Objects", NodeClass: "Object"},
		Children: []opcbrowser.TreeNode{
			{
				Match: opcbrowser.Match{DisplayName: "Plant", NodeClass: "Object"},
				Children: []opcbrowser.TreeNode{
					{Match: opcbrowser.Match{DisplayName: "Counter", NodeClass: "Variable", Value: "42"}},
				},
			},
		},
	}

	rendered := renderTree(root)
	want := "Objects [Object]\n`- Plant [Object]\n   `- Counter [Variable] = 42"
	if rendered != want {
		t.Fatalf("renderTree() = %q, want %q", rendered, want)
	}
}
