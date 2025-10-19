/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"fmt"
	"os"
	"golang.org/x/mod/modfile"

	"github.com/spf13/cobra"
)

// Version of kobot (this get overridden on each build)		
var CliVersion = "v0.0.0"

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version of the kobot cli tool.",
	Long: `This will print a representation the version of Kobot.
The output will look something like this:`,
	Run: func(cmd *cobra.Command, args []string) {
		data, err := os.ReadFile("go.mod")
		if err != nil {
			fmt.Println("Error reading go.mod:", err)
			return
		}

		f, err := modfile.Parse("go.mod", data, nil)
		if err != nil {
			fmt.Println("Error parsing go.mod:", err)
			return
		}



		fmt.Printf("Kobot CLI version: %s\n", CliVersion)
		fmt.Printf("Go runtime version: %s\n", f.Go.Version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// versionCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// versionCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
