package main

import (
	"github.com/spf13/cobra"

	"educates-artifact-cli/pkg/cmd"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "artifact-cli",
		Short: "A CLI tool to push and pull folders as OCI artifacts",
		Long:  `A command-line interface to package local folders, push them as OCI artifacts to a registry, and pull them back down.`,
	}

	rootCmd.AddCommand(cmd.NewPushCmd())
	rootCmd.AddCommand(cmd.NewPullCmd())

	rootCmd.Execute()
}
