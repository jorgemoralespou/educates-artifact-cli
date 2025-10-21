package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"educates-artifact-cli/pkg/sync"
	"educates-artifact-cli/pkg/utils"
)

type SyncCmdOpts struct {
	ConfigFile string
	Timeout    string
}

// NewSyncCmd creates the 'sync' command
func NewSyncCmd() *cobra.Command {
	var opts SyncCmdOpts

	cmd := &cobra.Command{
		Use:   "sync -c <config.yaml>",
		Short: "Sync artifacts from OCI registries to local folders based on configuration",
		Long: `Sync artifacts from OCI registries to local folders based on a configuration YAML file.
The configuration file defines which artifacts to pull, where to extract them,
and which files to include or exclude.`,
		Example: `  # Sync artifacts using configuration file
  artifact-cli sync -c config.yaml

  # Verbose sync
  artifact-cli sync -c config.yaml -v

  # Example config.yaml structure:
  spec:
    dest: ./workshops
    artifacts:
      - image:
          url: ghcr.io/my-org/workshop-files:v1.0.0
        includePaths:
          - /workshop/**
          - /exercises/**
        excludePaths:
          - /README.md`,
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSync(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.ConfigFile, "config", "c", "", "Path to the configuration YAML file (required)")
	cmd.Flags().StringVarP(&opts.Timeout, "timeout", "t", "", "Timeout for the operation (e.g., '30s', '5m', '1h'). Defaults to 5m")
	_ = cmd.MarkFlagRequired("config")

	return cmd
}

func runSync(opts SyncCmdOpts) error {
	// Create cancellable context with signal handling
	ctx, cancel, err := utils.ContextWithSignalHandling(opts.Timeout)
	if err != nil {
		return fmt.Errorf("invalid timeout: %w", err)
	}
	defer cancel()

	var config sync.SyncConfig

	// Load and parse configuration file
	if err := sync.LoadSyncConfig(&config, opts.ConfigFile); err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Validate configuration
	if err := sync.ValidateSyncConfig(&config); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	err = sync.Sync(ctx, config)
	if err != nil {
		// Check if the error was due to user cancellation
		if utils.IsCancelledByUser(ctx) {
			// User cancelled the operation, don't return an error
			return nil
		}
		// Return the actual error for other cases
		return err
	}

	return nil
}
