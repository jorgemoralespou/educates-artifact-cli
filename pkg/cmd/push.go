package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"educates-artifact-cli/pkg/artifact"
	"educates-artifact-cli/pkg/artifact/educates"
	"educates-artifact-cli/pkg/artifact/imgpkg"
	"educates-artifact-cli/pkg/artifact/oci"
	"educates-artifact-cli/pkg/utils"
)

type PushCmdOpts struct {
	RepoRef      string
	Platforms    string
	FolderPath   string
	ArtifactType ArtifactType
}

const DefaultArtifactType = ArtifactTypeOci

// NewPushCmd creates the 'push' command
func NewPushCmd() *cobra.Command {
	var opts PushCmdOpts
	opts.ArtifactType = DefaultArtifactType

	cmd := &cobra.Command{
		Use:   "push <repository> -f <folder> [-p <platforms>]",
		Short: "Package and push a folder to an OCI registry",
		Example: `  # Push a single artifact
  artifact-cli push ghcr.io/my-user/my-app:1.0.0 -f ./app-folder

  # Push a multi-platform artifact
  artifact-cli push ghcr.io/my-user/my-app:1.0.1 -f ./app-folder -p linux/amd64,linux/arm64

  # Push an artifact with a specific artifact type
  artifact-cli push ghcr.io/my-user/my-app:1.0.1 -f ./app-folder -a imgpkg`,

		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			repoRef := args[0]
			platforms := utils.SlicePlatforms(opts.Platforms)

			// Do some validation
			if opts.ArtifactType == ArtifactTypeImgpkg && len(platforms) != 0 {
				fmt.Println("when pushing an Imgpkg artifact, platforms will be ignored")
				platforms = nil
			}

			if err := utils.ValidatePlatforms(platforms); err != nil {
				return err
			}

			var artifact artifact.Artifact

			switch opts.ArtifactType {
			case ArtifactTypeOci:
				artifact = oci.NewOciImageArtifact(repoRef, platforms, "", opts.FolderPath)
			case ArtifactTypeImgpkg:
				artifact = imgpkg.NewImgpkgImageArtifact(repoRef, platforms, "", opts.FolderPath)
			case ArtifactTypeEducates:
				artifact = educates.NewEducatesImageArtifact(repoRef, platforms, "", opts.FolderPath)
			}
			return artifact.Push()
		},
	}

	cmd.Flags().StringVarP(&opts.FolderPath, "folder", "f", "", "Path to the folder to package and push (required)")
	cmd.Flags().StringVarP(&opts.Platforms, "platforms", "p", "", "A comma-separated list of platforms (e.g., 'linux/amd64,linux/arm64')")
	cmd.Flags().Var(&opts.ArtifactType, "as", "Type of artifact to push (oci, imgpkg, educates). Defaults to oci")
	_ = cmd.MarkFlagRequired("folder")

	return cmd
}
