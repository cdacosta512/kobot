/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"gitlab.com/kobot/kobot/pkg/common"
	"github.com/spf13/cobra"
	"gitlab.com/kobot/kobot/pkg/checks"
)

// clusterCmd represents the cluster command
var clusterCmd = &cobra.Command{
	Use:   "cluster",
	Short: "Check overall cluster health across all namespaces",
	Run: func(cmd *cobra.Command, args []string) {
		clientset := common.EnsureClusterConnection()
		if clientset == nil {
			return
		}

		checks.RunPodCheck(clientset)

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
