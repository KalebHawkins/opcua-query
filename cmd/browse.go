package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/KalebHawkins/opcua-query/internal/opcbrowser"
	"github.com/KalebHawkins/opcua-query/internal/sitewise"
	"github.com/KalebHawkins/opcua-query/internal/ui"
)

func init() {
	browseCmd := &cobra.Command{
		Use:   "browse",
		Short: "Browse an OPC UA server and preview SiteWise-compatible filters",
		Example: strings.TrimSpace(`opcua-query browse --server localhost:4840 --filter "/**/PLC*"
opcua-query browse --server opc.tcp://ignition.local:62541 --filter "/Plant/Area 1/**" --copy
opcua-query browse --server 10.0.0.25:53530 --username operator --password secret --filter "/Line 2/Counter*"`),
		RunE: runBrowse,
	}

	browseCmd.Flags().String("filter", "/**", "SiteWise OPC UA rootPath filter to test")
	browseCmd.Flags().String("start-node", "", "optional starting node id, such as ns=0;i=85")
	browseCmd.Flags().Int("max-depth", 8, "maximum browse depth from the starting node")
	browseCmd.Flags().Int("max-nodes", 1000, "maximum nodes to inspect before stopping")
	browseCmd.Flags().Bool("read-values", true, "read live values for matched variable nodes")
	browseCmd.Flags().Bool("copy", false, "copy the SiteWise filter payload to the clipboard")
	browseCmd.Flags().String("copy-format", "json", "clipboard format: json or path")

	_ = viper.BindPFlag("filter", browseCmd.Flags().Lookup("filter"))
	_ = viper.BindPFlag("start_node", browseCmd.Flags().Lookup("start-node"))
	_ = viper.BindPFlag("max_depth", browseCmd.Flags().Lookup("max-depth"))
	_ = viper.BindPFlag("max_nodes", browseCmd.Flags().Lookup("max-nodes"))
	_ = viper.BindPFlag("read_values", browseCmd.Flags().Lookup("read-values"))
	_ = viper.BindPFlag("copy", browseCmd.Flags().Lookup("copy"))
	_ = viper.BindPFlag("copy_format", browseCmd.Flags().Lookup("copy-format"))

	rootCmd.AddCommand(browseCmd)
}

func runBrowse(cmd *cobra.Command, _ []string) error {
	filter, err := sitewise.NormalizeRootPath(viper.GetString("filter"))
	if err != nil {
		return err
	}

	copyFormat := strings.ToLower(strings.TrimSpace(viper.GetString("copy_format")))
	if copyFormat != "json" && copyFormat != "path" {
		return fmt.Errorf("copy-format must be json or path")
	}

	server := strings.TrimSpace(viper.GetString("server"))
	if server == "" {
		return fmt.Errorf("server is required")
	}

	request := opcbrowser.Request{
		Endpoint:   server,
		Username:   strings.TrimSpace(viper.GetString("username")),
		Password:   viper.GetString("password"),
		Timeout:    viper.GetDuration("timeout"),
		Filter:     filter,
		StartNode:  strings.TrimSpace(viper.GetString("start_node")),
		MaxDepth:   viper.GetInt("max_depth"),
		MaxNodes:   viper.GetInt("max_nodes"),
		ReadValues: viper.GetBool("read_values"),
	}

	result, err := ui.RunTask("Browsing OPC UA nodes", func(ctx context.Context) (opcbrowser.Result, error) {
		return opcbrowser.Browse(ctx, request)
	})
	if err != nil {
		return err
	}

	payload, err := sitewise.BuildPayload(filter)
	if err != nil {
		return err
	}

	if viper.GetBool("copy") {
		copyValue := payload.JSON
		if copyFormat == "path" {
			copyValue = payload.RootPath
		}
		if err := clipboard.WriteAll(copyValue); err != nil {
			return fmt.Errorf("copy filter to clipboard: %w", err)
		}
	}

	fmt.Print(ui.RenderBrowseReport(request, result, payload, viper.GetBool("copy"), copyFormat))
	return nil
}
