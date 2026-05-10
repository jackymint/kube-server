package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/kube-server/kube-server/internal/cluster"
	"github.com/kube-server/kube-server/internal/config"
	"github.com/kube-server/kube-server/internal/metrics"
	"github.com/kube-server/kube-server/internal/tui"
	"github.com/kube-server/kube-server/internal/vm"
)

var cfg *config.Config

var rootCmd = &cobra.Command{
	Use:   "kube-server",
	Short: "Lightweight Kubernetes cluster manager for macOS",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		var err error
		cfg, err = config.Load()
		return err
	},
}

// cluster commands
var clusterCmd = &cobra.Command{Use: "cluster", Short: "Manage clusters"}

var clusterCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a new cluster",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c := cluster.New(args[0], cfg)
		fmt.Printf("Creating cluster %s...\n", args[0])
		return c.Create()
	},
}

var clusterStartCmd = &cobra.Command{
	Use:   "start [name]",
	Short: "Start a cluster",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c := cluster.New(args[0], cfg)
		fmt.Printf("Starting cluster %s...\n", args[0])
		return c.Start()
	},
}

var clusterStopCmd = &cobra.Command{
	Use:   "stop [name]",
	Short: "Stop a cluster",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c := cluster.New(args[0], cfg)
		fmt.Printf("Stopping cluster %s...\n", args[0])
		return c.Stop()
	},
}

var clusterListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all clusters",
	RunE: func(cmd *cobra.Command, args []string) error {
		clusters, err := cluster.List(cfg)
		if err != nil {
			return err
		}
		if len(clusters) == 0 {
			fmt.Println("No clusters found")
			return nil
		}
		for _, c := range clusters {
			fmt.Println(c)
		}
		return nil
	},
}

// node commands
var nodeCmd = &cobra.Command{Use: "node", Short: "Manage nodes"}

var nodeAddCmd = &cobra.Command{
	Use:   "add [cluster]",
	Short: "Add a worker node to cluster",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c := cluster.New(args[0], cfg)
		fmt.Printf("Adding node to cluster %s...\n", args[0])
		return c.AddNode()
	},
}

var nodeRemoveCmd = &cobra.Command{
	Use:   "remove [cluster] [node]",
	Short: "Remove a node from cluster",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		c := cluster.New(args[0], cfg)
		return c.RemoveNode(args[1])
	},
}

var nodeResizeCmd = &cobra.Command{
	Use:   "resize [cluster] [node]",
	Short: "Resize node disk",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		disk, _ := cmd.Flags().GetString("disk")
		v := vm.New(vm.VMConfig{Name: args[1], DataDir: cfg.DataDir, Disk: disk})
		return v.ResizeDisk(disk)
	},
}

// tui command
var tuiCmd = &cobra.Command{
	Use:   "tui [cluster]",
	Short: "Open interactive TUI",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		kubeconfig := fmt.Sprintf("%s/clusters/%s/kubeconfig", cfg.DataDir, args[0])
		collector, err := metrics.NewCollector(kubeconfig)
		if err != nil {
			return fmt.Errorf("connect to cluster: %w", err)
		}
		return tui.Run(collector, args[0])
	},
}

func Execute() {
	nodeResizeCmd.Flags().String("disk", "", "New disk size (e.g. 20GB)")
	nodeResizeCmd.MarkFlagRequired("disk")

	nodeCmd.AddCommand(nodeAddCmd, nodeRemoveCmd, nodeResizeCmd)
	clusterCmd.AddCommand(clusterCreateCmd, clusterStartCmd, clusterStopCmd, clusterListCmd)
	rootCmd.AddCommand(clusterCmd, nodeCmd, tuiCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
