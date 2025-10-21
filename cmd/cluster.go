/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"log/slog"
	"github.com/spf13/cobra"
	"gitlab.com/kobot/kobot/pkg/cluster"
	"gitlab.com/kobot/kobot/pkg/logging"
)

// clusterCmd represents the cluster command
var clusterCmd = &cobra.Command{
	Use:   "cluster",
	Short: "Check overall cluster health across all namespaces",
	RunE: func(cmd *cobra.Command, args []string) error {
		
		// empty string defaults to json now
		logging.Init("", slog.LevelInfo)

		// connect to the cluster
		clientset, err := cluster.GetClientset()
		if err != nil {
			slog.Error("failed to connect to cluster", "error", err)
		}
		
		// get version of the kubernetes client (e.g v1.34)
		version, err := clientset.Discovery().ServerVersion()
		if err != nil {
			slog.Warn("connected to cluster but failed to obtain the version", "error", err)
		}

		slog.Info("successfully connected to cluster", "version", version.String())
		return nil
	},
}

func init() {
	checkCmd.AddCommand(clusterCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// clusterCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// clusterCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
