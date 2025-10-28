/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"gitlab.com/kobot/kobot/pkg/common"
	"github.com/spf13/cobra"
	"gitlab.com/kobot/kobot/pkg/checks"
)

var (
	namespace       []string
	htmlOutput      bool
	helmRelease     bool
	fluxGracePeriod int
	podDeepCheck bool
)

var clusterCmd = &cobra.Command{
	Use:   "cluster",
	Short: "Check overall cluster health across all namespaces",
	Run: func(cmd *cobra.Command, args []string) {

		// if the user want to run helmrelease checks
		if helmRelease {
			dynamicClient := common.EnsureDynamicClusterConnection()
			if dynamicClient == nil {
				return
			}
			checks.RunHelmReleaseCheck(dynamicClient, namespace, htmlOutput, fluxGracePeriod)
			return
		}

		// if the user wants to run a deep pod health check
		if podDeepCheck {
			clientset := common.EnsureClusterConnection()
			if clientset == nil {
				return
			}
			checks.RunPodDeepCheck(clientset, namespace, htmlOutput)
			return
		}

		// default behavior of running a low level pod health check
		clientset := common.EnsureClusterConnection()
		if clientset == nil {
			return
		}

		checks.RunPodCheck(clientset, namespace, htmlOutput)
	},
}

func init() {
	checkCmd.AddCommand(clusterCmd)
	clusterCmd.Flags().StringSliceVarP(
		&namespace,
		"namespace",
		"n",
		[]string{},
		"Comma-separated list of namespaces to check (default: all)",
	)
	clusterCmd.Flags().BoolVar(&htmlOutput, "html", false, "Generate an HTML report (kobot-report.html)")
	clusterCmd.Flags().BoolVar(&helmRelease, "helmrelease-only", false, "Run only HelmRelease checks")
	clusterCmd.Flags().IntVar(&fluxGracePeriod, "flux-grace", 5, "Time (in seconds) to wait for Flux-managed resources to become Ready (default: 5s)")
	clusterCmd.Flags().BoolVar(&podDeepCheck, "deep", false, "Performs a deeper pod health analysis when running the check cluster command")
}
