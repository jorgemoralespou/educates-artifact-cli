package main

import (
	"github.com/spf13/cobra"

	"educates-artifact-cli/pkg/cmd"
	"educates-artifact-cli/pkg/utils"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "artifact-cli",
		Short: "A CLI tool to push, pull, and sync folders as OCI artifacts",
		Long:  `A command-line interface to package local folders, push them as OCI artifacts to a registry, pull them back down, and sync multiple artifacts based on configuration.`,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			verbose, _ := cmd.Flags().GetBool("verbose")
			utils.SetVerbose(verbose)
		},
	}

	// Add persistent flags
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose output")

	rootCmd.AddCommand(cmd.NewPushCmd())
	rootCmd.AddCommand(cmd.NewPullCmd())
	rootCmd.AddCommand(cmd.NewSyncCmd())
	rootCmd.AddCommand(cmd.NewManifestCmd())

	rootCmd.Execute()
}
