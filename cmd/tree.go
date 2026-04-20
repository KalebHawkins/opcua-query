package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/KalebHawkins/opcua-query/internal/opcbrowser"
	"github.com/KalebHawkins/opcua-query/internal/ui"
)

func init() {
	treeCmd := &cobra.Command{
		Use:   "tree",
		Short: "Browse OPC UA nodes as a tree when you do not know the path yet",
		Example: strings.TrimSpace(`opcua-query tree --server localhost:4840
opcua-query tree --server opc.tcp://ignition.local:62541 --max-depth 3
opcua-query tree --server 10.0.0.25:53530 --start-node "ns=0;i=85" --read-values`),
		RunE: runTree,
	}

	treeCmd.Flags().String("start-node", "", "optional starting node id, such as ns=0;i=85")
	treeCmd.Flags().Int("max-depth", 4, "maximum browse depth from the starting node")
	treeCmd.Flags().Int("max-nodes", 1000, "maximum nodes to inspect before stopping")
	treeCmd.Flags().Bool("read-values", false, "read live values for variable nodes included in the tree")

	_ = viper.BindPFlag("tree_start_node", treeCmd.Flags().Lookup("start-node"))
	_ = viper.BindPFlag("tree_max_depth", treeCmd.Flags().Lookup("max-depth"))
	_ = viper.BindPFlag("tree_max_nodes", treeCmd.Flags().Lookup("max-nodes"))
	_ = viper.BindPFlag("tree_read_values", treeCmd.Flags().Lookup("read-values"))

	rootCmd.AddCommand(treeCmd)
}

func runTree(cmd *cobra.Command, _ []string) error {
	server := strings.TrimSpace(viper.GetString("server"))
	if server == "" {
		return fmt.Errorf("server is required")
	}

	request := opcbrowser.Request{
		Endpoint:   server,
		Username:   strings.TrimSpace(viper.GetString("username")),
		Password:   viper.GetString("password"),
		Timeout:    viper.GetDuration("timeout"),
		StartNode:  strings.TrimSpace(viper.GetString("tree_start_node")),
		MaxDepth:   viper.GetInt("tree_max_depth"),
		MaxNodes:   viper.GetInt("tree_max_nodes"),
		ReadValues: viper.GetBool("tree_read_values"),
	}

	result, err := ui.RunTask("Browsing OPC UA tree", func(ctx context.Context) (opcbrowser.TreeResult, error) {
		return opcbrowser.Tree(ctx, request)
	})
	if err != nil {
		return err
	}

	fmt.Print(ui.RenderTreeReport(request, result))
	return nil
}
