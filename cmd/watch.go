package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/KalebHawkins/opcua-query/internal/opcbrowser"
	"github.com/KalebHawkins/opcua-query/internal/sitewise"
	"github.com/KalebHawkins/opcua-query/internal/ui"
)

func init() {
	watchCmd := &cobra.Command{
		Use:   "watch",
		Short: "Subscribe to matched OPC UA variable nodes and stream live values",
		Example: strings.TrimSpace(`opcua-query watch --server localhost:4840 --filter "/**/Counter*"
opcua-query watch --server opc.tcp://ignition.local:62541 --filter "/Plant/Area 1/Line 2/**" --interval 500ms
opcua-query watch --server 10.0.0.25:53530 --username operator --password secret --filter "/Line 2/Counter*"`),
		RunE: runWatch,
	}

	watchCmd.Flags().String("filter", "/**", "SiteWise OPC UA rootPath filter to watch")
	watchCmd.Flags().String("start-node", "", "optional starting node id, such as ns=0;i=85")
	watchCmd.Flags().Int("max-depth", 8, "maximum browse depth from the starting node")
	watchCmd.Flags().Int("max-nodes", 1000, "maximum nodes to inspect before stopping")
	watchCmd.Flags().Duration("interval", 1_000_000_000, "subscription publishing interval")

	_ = viper.BindPFlag("watch_filter", watchCmd.Flags().Lookup("filter"))
	_ = viper.BindPFlag("watch_start_node", watchCmd.Flags().Lookup("start-node"))
	_ = viper.BindPFlag("watch_max_depth", watchCmd.Flags().Lookup("max-depth"))
	_ = viper.BindPFlag("watch_max_nodes", watchCmd.Flags().Lookup("max-nodes"))
	_ = viper.BindPFlag("watch_interval", watchCmd.Flags().Lookup("interval"))

	rootCmd.AddCommand(watchCmd)
}

func runWatch(cmd *cobra.Command, _ []string) error {
	filter, err := sitewise.NormalizeRootPath(viper.GetString("watch_filter"))
	if err != nil {
		return err
	}

	server := strings.TrimSpace(viper.GetString("server"))
	if server == "" {
		return fmt.Errorf("server is required")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	request := opcbrowser.Request{
		Endpoint:   server,
		Username:   strings.TrimSpace(viper.GetString("username")),
		Password:   viper.GetString("password"),
		Timeout:    viper.GetDuration("timeout"),
		Filter:     filter,
		StartNode:  strings.TrimSpace(viper.GetString("watch_start_node")),
		MaxDepth:   viper.GetInt("watch_max_depth"),
		MaxNodes:   viper.GetInt("watch_max_nodes"),
		ReadValues: false,
		Interval:   viper.GetDuration("watch_interval"),
	}

	err = opcbrowser.Watch(ctx, request, opcbrowser.WatchCallbacks{
		Ready: func(session opcbrowser.WatchSession) {
			fmt.Print(ui.RenderWatchSession(session))
		},
		Event: func(event opcbrowser.WatchEvent) {
			fmt.Print(ui.RenderWatchEvent(event))
		},
	})
	if err != nil {
		return err
	}

	fmt.Print(ui.RenderWatchStopped())
	return nil
}
