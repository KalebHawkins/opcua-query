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
	lsCmd := &cobra.Command{
		Use:   "ls",
		Short: "List the immediate child nodes for a start node or browse path",
		Example: strings.TrimSpace(`opcua-query ls --server localhost:4840
opcua-query ls --server opc.tcp://ignition.local:62541 --path "/Plant/Area 1"
opcua-query ls --server 10.0.0.25:53530 --start-node "ns=0;i=85" --read-values`),
		RunE: runList,
	}

	lsCmd.Flags().String("path", "/", "exact browse path to resolve before listing child nodes")
	lsCmd.Flags().String("start-node", "", "optional starting node id, such as ns=0;i=85")
	lsCmd.Flags().Int("max-nodes", 1000, "maximum nodes to inspect while resolving the supplied path")
	lsCmd.Flags().Bool("read-values", false, "read live values for listed variable nodes")

	_ = viper.BindPFlag("ls_path", lsCmd.Flags().Lookup("path"))
	_ = viper.BindPFlag("ls_start_node", lsCmd.Flags().Lookup("start-node"))
	_ = viper.BindPFlag("ls_max_nodes", lsCmd.Flags().Lookup("max-nodes"))
	_ = viper.BindPFlag("ls_read_values", lsCmd.Flags().Lookup("read-values"))

	rootCmd.AddCommand(lsCmd)
}

func runList(cmd *cobra.Command, _ []string) error {
	server := strings.TrimSpace(viper.GetString("server"))
	if server == "" {
		return fmt.Errorf("server is required")
	}

	request := opcbrowser.Request{
		Endpoint:   server,
		Username:   strings.TrimSpace(viper.GetString("username")),
		Password:   viper.GetString("password"),
		Timeout:    viper.GetDuration("timeout"),
		StartNode:  strings.TrimSpace(viper.GetString("ls_start_node")),
		MaxNodes:   viper.GetInt("ls_max_nodes"),
		ReadValues: viper.GetBool("ls_read_values"),
	}

	path := strings.TrimSpace(viper.GetString("ls_path"))
	result, err := ui.RunTask("Listing OPC UA child nodes", func(ctx context.Context) (opcbrowser.ListResult, error) {
		return opcbrowser.List(ctx, request, path)
	})
	if err != nil {
		return err
	}

	fmt.Print(ui.RenderListReport(request, result))
	return nil
}
