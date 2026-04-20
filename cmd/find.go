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
	findCmd := &cobra.Command{
		Use:   "find",
		Short: "Search OPC UA nodes by name, display name, path, or node id",
		Example: strings.TrimSpace(`opcua-query find --server localhost:4840 --name Counter
opcua-query find --server opc.tcp://ignition.local:62541 --name "Area 1" --max-depth 5
opcua-query find --server 10.0.0.25:53530 --name ns=2;s=Counter01 --read-values`),
		RunE: runFind,
	}

	findCmd.Flags().String("name", "", "name, path segment, display name, or node id fragment to search for")
	findCmd.Flags().String("start-node", "", "optional starting node id, such as ns=0;i=85")
	findCmd.Flags().Int("max-depth", 8, "maximum browse depth from the starting node")
	findCmd.Flags().Int("max-nodes", 1000, "maximum nodes to inspect before stopping")
	findCmd.Flags().Bool("read-values", false, "read live values for matched variable nodes")

	_ = viper.BindPFlag("find_name", findCmd.Flags().Lookup("name"))
	_ = viper.BindPFlag("find_start_node", findCmd.Flags().Lookup("start-node"))
	_ = viper.BindPFlag("find_max_depth", findCmd.Flags().Lookup("max-depth"))
	_ = viper.BindPFlag("find_max_nodes", findCmd.Flags().Lookup("max-nodes"))
	_ = viper.BindPFlag("find_read_values", findCmd.Flags().Lookup("read-values"))

	rootCmd.AddCommand(findCmd)
}

func runFind(cmd *cobra.Command, _ []string) error {
	server := strings.TrimSpace(viper.GetString("server"))
	if server == "" {
		return fmt.Errorf("server is required")
	}

	query := strings.TrimSpace(viper.GetString("find_name"))
	if query == "" {
		return fmt.Errorf("name is required")
	}

	request := opcbrowser.Request{
		Endpoint:   server,
		Username:   strings.TrimSpace(viper.GetString("username")),
		Password:   viper.GetString("password"),
		Timeout:    viper.GetDuration("timeout"),
		StartNode:  strings.TrimSpace(viper.GetString("find_start_node")),
		MaxDepth:   viper.GetInt("find_max_depth"),
		MaxNodes:   viper.GetInt("find_max_nodes"),
		ReadValues: viper.GetBool("find_read_values"),
	}

	result, err := ui.RunTask("Searching OPC UA nodes", func(ctx context.Context) (opcbrowser.FindResult, error) {
		return opcbrowser.Find(ctx, request, query)
	})
	if err != nil {
		return err
	}

	fmt.Print(ui.RenderFindReport(request, result))
	return nil
}
