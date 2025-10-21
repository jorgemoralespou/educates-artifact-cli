package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"educates-artifact-cli/pkg/artifact"
	"educates-artifact-cli/pkg/artifact/oci"
	"educates-artifact-cli/pkg/utils"
)

type PullCmdOpts struct {
	RepoRef     string
	Username    string
	Password    string
	Insecure    bool
	PlatformStr string
	OutputDir   string
	Timeout     string
	// ArtifactType ArtifactType
}

// NewPullCmd creates the 'pull' command
func NewPullCmd() *cobra.Command {
	var opts PullCmdOpts
	// opts.ArtifactType = DefaultArtifactType

	cmd := &cobra.Command{
		Use:   "pull <repository> -o <target_dir> [-p <platform>]",
		Short: "Pull and extract an OCI artifact folder",
		Example: `  # Pull the artifact matching the current system's architecture
  artifact-cli pull ghcr.io/my-user/my-app:1.0.1 -o ./restored-app

  # Pull a specific platform
  artifact-cli pull ghcr.io/my-user/my-app:1.0.1 -o ./restored-app -p linux/amd64

  # Verbose pull
  artifact-cli pull ghcr.io/my-user/my-app:1.0.1 -o ./restored-app -v`,
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.RepoRef = args[0]
			return runPull(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.OutputDir, "output", "o", "", "Path to the target directory for extraction (required)")
	cmd.Flags().StringVarP(&opts.PlatformStr, "platform", "p", "", "Target platform (e.g., 'linux/amd64'). If not specified, uses current system platform")
	cmd.Flags().StringVarP(&opts.Timeout, "timeout", "t", "", "Timeout for the operation (e.g., '30s', '5m', '1h'). Defaults to 5m")
	// cmd.Flags().Var(&opts.ArtifactType, "as", "Type of artifact to push (oci, imgpkg, educates). Defaults to oci")
	cmd.Flags().StringVarP(&opts.Username, "username", "u", "", "Username for registry authentication (can also use ARTIFACT_CLI_USERNAME env var)")
	cmd.Flags().StringVarP(&opts.Password, "password", "w", "", "Password or token for registry authentication (can also use ARTIFACT_CLI_PASSWORD env var)")
	cmd.Flags().BoolVarP(&opts.Insecure, "insecure", "", false, "Allow insecure registry communication")
	_ = cmd.MarkFlagRequired("output")

	return cmd
}

func runPull(opts PullCmdOpts) error {
	// Create cancellable context with signal handling
	ctx, cancel, err := utils.ContextWithSignalHandling(opts.Timeout)
	if err != nil {
		return fmt.Errorf("invalid timeout: %w", err)
	}
	defer cancel()

	repoRef := artifact.NewRepositoryRef(opts.RepoRef, opts.Username, opts.Password, opts.Insecure)

	// Use default platforms if no platform is specified
	if opts.PlatformStr == "" {
		opts.PlatformStr = utils.GetOSPlatformStr()
	}

	platforms := utils.SlicePlatforms(opts.PlatformStr)
	if err := utils.ValidatePlatforms(platforms); err != nil {
		return err
	}

	// Do some validation
	if len(platforms) > 1 {
		return fmt.Errorf("when pulling an OCI artifact, can only pull for one platform")
	}

	// Ensure the output directory exists
	if err := os.MkdirAll(opts.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	var artifactInstance artifact.Artifact

	// switch opts.ArtifactType {
	// case ArtifactTypeOci:
	artifactInstance = oci.NewOciImageArtifact(repoRef, nil, opts.PlatformStr, opts.OutputDir)
	// case ArtifactTypeImgpkg:
	// 	artifact = imgpkg.NewImgpkgImageArtifact(repoRef, nil, opts.PlatformStr, opts.OutputDir)
	// case ArtifactTypeEducates:
	// 	artifact = educates.NewEducatesImageArtifact(repoRef, nil, opts.PlatformStr, opts.OutputDir)
	// }

	err = artifactInstance.Pull(ctx)
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

	// return Run(repoRef, opts.PlatformStr, opts.OutputDir, opts.ArtifactType)
}
