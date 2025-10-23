package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

// CliVersion is injected at build time using -ldflags
var CliVersion = "v0.0.0"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version of the kobot CLI tool.",
	Long: `Displays the version of Kobot and the Go runtime used
to build the binary. When built via 'go build -ldflags', the version
is injected automatically.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Kobot CLI version: %s\n", CliVersion)
		fmt.Printf("Go runtime version: %s\n", runtime.Version())
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
