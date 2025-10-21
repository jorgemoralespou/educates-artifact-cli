package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"educates-artifact-cli/pkg/artifact"
	"educates-artifact-cli/pkg/artifact/oci"
	"educates-artifact-cli/pkg/utils"
)

type PushCmdOpts struct {
	ImageRef   string
	Username   string
	Password   string
	Insecure   bool
	Platforms  string
	FolderPath string
	Timeout    string
	// ArtifactType ArtifactType
}

// const DefaultArtifactType = ArtifactTypeOci

// NewPushCmd creates the 'push' command
func NewPushCmd() *cobra.Command {
	var opts PushCmdOpts
	// opts.ArtifactType = DefaultArtifactType

	cmd := &cobra.Command{
		Use:   "push <repository> -f <folder> [-p <platforms>]",
		Short: "Package and push a folder to an OCI registry",
		Example: `  # Push a single artifact
  artifact-cli push ghcr.io/my-user/my-app:1.0.0 -f ./app-folder

  # Push a multi-platform artifact
  artifact-cli push ghcr.io/my-user/my-app:1.0.1 -f ./app-folder -p linux/amd64,linux/arm64

  # Push an artifact with a specific artifact type
  artifact-cli push ghcr.io/my-user/my-app:1.0.1 -f ./app-folder -a imgpkg

  # Verbose push
  artifact-cli push ghcr.io/my-user/my-app:1.0.0 -f ./app-folder -v`,

		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.ImageRef = args[0]
			return runPush(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.FolderPath, "folder", "f", "", "Path to the folder to package and push (required)")
	cmd.Flags().StringVarP(&opts.Platforms, "platforms", "p", "", "A comma-separated list of platforms (e.g., 'linux/amd64,linux/arm64')")
	cmd.Flags().StringVarP(&opts.Timeout, "timeout", "t", "", "Timeout for the operation (e.g., '30s', '5m', '1h'). Defaults to 5m")
	// cmd.Flags().Var(&opts.ArtifactType, "as", "Type of artifact to push (oci, imgpkg, educates). Defaults to oci")
	cmd.Flags().StringVarP(&opts.Username, "username", "u", "", "Username for registry authentication (can also use ARTIFACT_CLI_USERNAME env var)")
	cmd.Flags().StringVarP(&opts.Password, "password", "w", "", "Password or token for registry authentication (can also use ARTIFACT_CLI_PASSWORD env var)")
	cmd.Flags().BoolVarP(&opts.Insecure, "insecure", "", false, "Allow insecure registry communication")
	_ = cmd.MarkFlagRequired("folder")

	return cmd
}

func runPush(opts PushCmdOpts) error {
	// Create cancellable context with signal handling
	ctx, cancel, err := utils.ContextWithSignalHandling(opts.Timeout)
	if err != nil {
		return fmt.Errorf("invalid timeout: %w", err)
	}
	defer cancel()

	repoRef := artifact.NewRepositoryRef(opts.ImageRef, opts.Username, opts.Password, opts.Insecure)
	platforms := utils.SlicePlatforms(opts.Platforms)

	// Do some validation
	// if opts.ArtifactType == ArtifactTypeImgpkg && len(platforms) != 0 {
	// 	utils.VerbosePrintln("when pushing an Imgpkg artifact, platforms will be ignored")
	// 	platforms = nil
	// }

	if err := utils.ValidatePlatforms(platforms); err != nil {
		return err
	}

	var artifactInstance artifact.Artifact

	// switch opts.ArtifactType {
	// case ArtifactTypeOci:
	artifactInstance = oci.NewOciImageArtifact(repoRef, platforms, "", opts.FolderPath)
	// case ArtifactTypeImgpkg:
	// 	artifact = imgpkg.NewImgpkgImageArtifact(repoRef, platforms, "", opts.FolderPath)
	// case ArtifactTypeEducates:
	// 	artifact = educates.NewEducatesImageArtifact(repoRef, platforms, "", opts.FolderPath)
	// }

	err = artifactInstance.Push(ctx)
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
