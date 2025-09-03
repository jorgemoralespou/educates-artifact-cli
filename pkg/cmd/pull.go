package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"educates-artifact-cli/pkg/artifact"
	"educates-artifact-cli/pkg/artifact/educates"
	"educates-artifact-cli/pkg/artifact/imgpkg"
	"educates-artifact-cli/pkg/artifact/oci"
	"educates-artifact-cli/pkg/utils"
)

type PullCmdOpts struct {
	RepoRef      string
	PlatformStr  string
	OutputDir    string
	ArtifactType ArtifactType
}

// NewPullCmd creates the 'pull' command
func NewPullCmd() *cobra.Command {
	var opts PullCmdOpts
	opts.ArtifactType = DefaultArtifactType

	cmd := &cobra.Command{
		Use:   "pull <repository> -o <target_dir> [-p <platform>]",
		Short: "Pull and extract an OCI artifact folder",
		Example: `  # Pull the artifact matching the current system's architecture
  artifact-cli pull ghcr.io/my-user/my-app:1.0.1 -o ./restored-app

  # Pull a specific platform
  artifact-cli pull ghcr.io/my-user/my-app:1.0.1 -o ./restored-app -p linux/amd64`,
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			repoRef := args[0]

			// Use default platforms if no platform is specified
			if opts.PlatformStr == "" {
				opts.PlatformStr = utils.GetOSPlatformStr()
			}

			platforms := utils.SlicePlatforms(opts.PlatformStr)
			if err := utils.ValidatePlatforms(platforms); err != nil {
				return err
			}

			// Do some validation
			if opts.ArtifactType == ArtifactTypeImgpkg && len(platforms) != 0 {
				fmt.Println("when pushing an Imgpkg artifact, platforms will be ignored")
				platforms = nil
			}

			// Ensure the output directory exists
			if err := os.MkdirAll(opts.OutputDir, 0755); err != nil {
				return fmt.Errorf("failed to create output directory: %w", err)
			}

			var artifact artifact.Artifact

			switch opts.ArtifactType {
			case ArtifactTypeOci:
				artifact = oci.NewOciImageArtifact(repoRef, nil, opts.PlatformStr, opts.OutputDir)
			case ArtifactTypeImgpkg:
				artifact = imgpkg.NewImgpkgImageArtifact(repoRef, nil, opts.PlatformStr, opts.OutputDir)
			case ArtifactTypeEducates:
				artifact = educates.NewEducatesImageArtifact(repoRef, nil, opts.PlatformStr, opts.OutputDir)
			}
			return artifact.Pull()

			// return Run(repoRef, opts.PlatformStr, opts.OutputDir, opts.ArtifactType)
		},
	}

	cmd.Flags().StringVarP(&opts.OutputDir, "output", "o", "", "Path to the target directory for extraction (required)")
	cmd.Flags().StringVarP(&opts.PlatformStr, "platform", "p", "", "Target platform (e.g., 'linux/amd64'). If not specified, uses current system platform")
	cmd.Flags().Var(&opts.ArtifactType, "as", "Type of artifact to push (oci, imgpkg, educates). Defaults to oci")
	_ = cmd.MarkFlagRequired("output")

	return cmd
}

// func Run(repoRef string, platformStr string, outputDir string, artifactType ArtifactType) error {
// 	ctx := context.Background()

// 	// Create a registry client
// 	repo, err := remote.NewRepository(repoRef)
// 	if err != nil {
// 		return fmt.Errorf("failed to create repository client: %w", err)
// 	}
// 	repo.PlainHTTP = true

// 	// Create a memory store to hold the pulled content
// 	memStore := memory.New()

// 	// Define copy options to specify the target platform
// 	var targetPlatform *ocispec.Platform
// 	if platformStr != "" {

// 		err = utils.ParsePlatform(targetPlatform, platformStr)
// 		if err != nil {
// 			return fmt.Errorf("failed to parse platform: %w", err)
// 		}

// 		copyOpts := oras.DefaultCopyOptions
// 		copyOpts.WithTargetPlatform(targetPlatform)

// 		// Use oras.Copy to pull the artifact
// 		pulledDesc, err := oras.Copy(ctx, repo, repoRef, memStore, repoRef, copyOpts)
// 		if err != nil {
// 			return fmt.Errorf("failed to pull artifact: %w", err)
// 		}

// 		return processPulledArtifact(ctx, memStore, pulledDesc, outputDir)
// 	} else {
// 		// No platform specified - try fallback strategies
// 		return pullWithFallbackStrategies(ctx, repo, repoRef, memStore, outputDir)
// 	}

// }

// // pullWithFallbackStrategies tries different strategies to pull an artifact when no platform is specified
// func pullWithFallbackStrategies(ctx context.Context, repo *remote.Repository, repoRef string, memStore *memory.Store, outputDir string) error {
// 	// Strategy 1: Try to pull an image generated with artifact-cli push (no platform selector)
// 	fmt.Println("Strategy 1: Trying to pull artifact-cli generated image (no platform selector)...")
// 	pulledDesc, err := oras.Copy(ctx, repo, repoRef, memStore, repoRef, oras.DefaultCopyOptions)
// 	if err == nil {
// 		// Check if this is actually an artifact-cli artifact
// 		if isOciCliArtifact(ctx, memStore, pulledDesc) {
// 			fmt.Printf("Successfully pulled artifact-cli artifact: %s\n", pulledDesc.Digest)
// 			return processPulledArtifact(ctx, memStore, pulledDesc, outputDir)
// 		} else {
// 			fmt.Printf("Strategy 1: Found artifact but not artifact-cli generated, trying next strategy...\n")
// 		}
// 	} else {
// 		fmt.Printf("Strategy 1 failed: %v\n", err)
// 	}

// 	// Strategy 2: Try to pull an image generated via imgpkg (Docker manifest format)
// 	fmt.Println("Strategy 2: Trying to pull imgpkg generated image...")
// 	// For imgpkg, we need to handle Docker manifest format
// 	// This is more complex and would require custom handling of Docker manifests
// 	// For now, we'll skip this and go to strategy 3

// 	// Strategy 3: Try to pull an image generated with docker buildx using current architecture
// 	fmt.Println("Strategy 3: Trying to pull docker buildx image with current platform...")
// 	currentPlatform := &ocispec.Platform{
// 		OS:           runtime.GOOS,
// 		Architecture: runtime.GOARCH,
// 	}
// 	copyOpts := oras.DefaultCopyOptions
// 	copyOpts.WithTargetPlatform(currentPlatform)

// 	pulledDesc, err = oras.Copy(ctx, repo, repoRef, memStore, repoRef, copyOpts)
// 	if err == nil {
// 		fmt.Printf("Successfully pulled docker buildx artifact for platform %s/%s: %s\n",
// 			currentPlatform.OS, currentPlatform.Architecture, pulledDesc.Digest)
// 		return processPulledArtifact(ctx, memStore, pulledDesc, outputDir)
// 	}
// 	fmt.Printf("Strategy 3 failed: %v\n", err)

// 	return fmt.Errorf("all pull strategies failed. Last error: %w", err)
// }

// // processPulledArtifact processes the pulled artifact and extracts it to the output directory
// func processPulledArtifact(ctx context.Context, memStore *memory.Store, pulledDesc ocispec.Descriptor, outputDir string) error {
// 	fmt.Printf("Processing pulled artifact with digest: %s\n", pulledDesc.Digest)

// 	// Fetch the manifest to find our folder layer
// 	manifestBytes, err := content.FetchAll(ctx, memStore, pulledDesc)
// 	if err != nil {
// 		return fmt.Errorf("failed to fetch manifest from memory store: %w", err)
// 	}

// 	var manifest ocispec.Manifest
// 	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
// 		return fmt.Errorf("failed to unmarshal manifest: %w", err)
// 	}

// 	fmt.Printf("Found manifest with media type %s\n", manifest.MediaType)

// 	// Check if this is an artifact-cli generated artifact
// 	if manifest.Annotations != nil {
// 		if tool, exists := manifest.Annotations["dev.educates.artifact-cli.tool"]; exists && tool == "artifact-cli" {
// 			fmt.Printf("Detected artifact-cli generated artifact (version: %s)\n", manifest.Annotations["dev.educates.artifact-cli.version"])
// 		}
// 	}

// 	// Find the specific layer containing our folder tarball
// 	// Try different media types for compatibility
// 	var folderLayerDesc *ocispec.Descriptor
// 	layerMediaTypes := []string{
// 		artifact.OCILayerMediaType,    // Our OCI layer type
// 		artifact.DockerLayerMediaType, // Docker layer type (imgpkg/docker buildx)
// 		artifact.FolderLayerMediaType, // Legacy folder layer type
// 	}

// 	for _, mediaType := range layerMediaTypes {
// 		for _, layer := range manifest.Layers {
// 			if layer.MediaType == mediaType {
// 				folderLayerDesc = &layer
// 				fmt.Printf("Found layer with media type %s: %s\n", mediaType, layer.Digest)
// 				break
// 			}
// 		}
// 		if folderLayerDesc != nil {
// 			break
// 		}
// 	}

// 	if folderLayerDesc == nil {
// 		return fmt.Errorf("could not find folder layer with any supported media type")
// 	}

// 	// Fetch the layer's content (the tarball)
// 	tarballBytes, err := content.FetchAll(ctx, memStore, *folderLayerDesc)
// 	if err != nil {
// 		return fmt.Errorf("failed to fetch layer content: %w", err)
// 	}

// 	// Extract the tarball to the output directory
// 	fmt.Printf("Extracting content to '%s'...\n", outputDir)
// 	if err := utils.ExtractTarGz(bytes.NewReader(tarballBytes), outputDir); err != nil {
// 		return fmt.Errorf("failed to extract tarball: %w", err)
// 	}

// 	fmt.Println("\nSuccessfully pulled and extracted artifact.")
// 	return nil
// }

// // isOciCliArtifact checks if the pulled descriptor is an artifact-cli generated artifact
// func isOciCliArtifact(ctx context.Context, memStore *memory.Store, desc ocispec.Descriptor) bool {
// 	// Fetch the manifest to check annotations
// 	manifestBytes, err := content.FetchAll(ctx, memStore, desc)
// 	if err != nil {
// 		return false
// 	}

// 	var manifest ocispec.Manifest
// 	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
// 		return false
// 	}

// 	// Check for artifact-cli specific annotations
// 	if manifest.Annotations != nil {
// 		if tool, exists := manifest.Annotations["dev.educates.artifact-cli.tool"]; exists && tool == "artifact-cli" {
// 			return true
// 		}
// 	}

// 	return false
// }
