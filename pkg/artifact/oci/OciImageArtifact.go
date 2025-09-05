package oci

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"runtime"

	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content"
	"oras.land/oras-go/v2/content/memory"
	"oras.land/oras-go/v2/registry/remote"

	"educates-artifact-cli/pkg/artifact"
	"educates-artifact-cli/pkg/utils"
)

type OciImageArtifact struct {
	repoRef       string
	pushPlatforms []string
	pullPlatform  string
	path          string
}

func NewOciImageArtifact(repoRef string, pushPlatforms []string, pullPlatform string, path string) *OciImageArtifact {
	return &OciImageArtifact{repoRef: repoRef, pushPlatforms: pushPlatforms, pullPlatform: pullPlatform, path: path}
}

func (a *OciImageArtifact) Push() error {
	fmt.Printf("OCI Image Artifact Push\n")
	fmt.Printf("Packaging folder '%s'...\n", a.path)
	// Create a tarball of the folder in memory
	tarballBytes, err := utils.CreateTarGz(a.path)
	if err != nil {
		return fmt.Errorf("failed to create tarball: %w", err)
	}

	ctx := context.Background()

	// Create a new registry client
	repo, err := remote.NewRepository(a.repoRef)
	if err != nil {
		return fmt.Errorf("failed to create repository client: %w", err)
	}
	// Use plain HTTP if the registry is insecure
	repo.PlainHTTP = true

	// Push the folder layer (blob) to the registry. This is shared across all platforms.
	// Use OCI layer media type for compatibility
	layerDesc := ocispec.Descriptor{
		MediaType: artifact.OCILayerMediaType,
		Digest:    digest.FromBytes(tarballBytes),
		Size:      int64(len(tarballBytes)),
	}
	if err := repo.Push(ctx, layerDesc, bytes.NewReader(tarballBytes)); err != nil {
		return fmt.Errorf("failed to push layer blob: %w", err)
	}
	fmt.Printf("Pushed layer: %s\n", layerDesc.Digest)

	var rootDesc ocispec.Descriptor

	// When no platforms are provided, we use the default platforms (linux/amd64 and linux/arm64)
	if len(a.pushPlatforms) == 0 {
		a.pushPlatforms = utils.SlicePlatforms(utils.DefaultPlatforms)
	}
	// --- Multi-Platform (Index) Push ---
	fmt.Printf("Performing a multi-platform push for: %s\n", a.pushPlatforms)
	rootDesc, err = pushImageIndex(ctx, repo, layerDesc, a.pushPlatforms)
	if err != nil {
		return err
	}

	// Tag the root manifest/index with the provided tag
	tag := utils.GetTagFromRef(a.repoRef)
	if err := repo.Tag(ctx, rootDesc, tag); err != nil {
		return fmt.Errorf("failed to tag root descriptor: %w", err)
	}

	fmt.Printf("\nSuccessfully pushed and tagged artifact: %s\n", a.repoRef)
	fmt.Printf("Root digest: %s\n", rootDesc.Digest)

	return nil

}

func (a *OciImageArtifact) Pull() error {
	ctx := context.Background()

	fmt.Printf("OCI Image Artifact Pull\n")

	// Create a registry client
	repo, err := remote.NewRepository(a.repoRef)
	if err != nil {
		return fmt.Errorf("failed to create repository client: %w", err)
	}
	repo.PlainHTTP = true

	// Create a memory store to hold the pulled content
	memStore := memory.New()

	// Define copy options to specify the target platform
	var targetPlatform ocispec.Platform
	err = utils.ParsePlatform(&targetPlatform, a.pullPlatform)
	if err != nil {
		return fmt.Errorf("failed to parse platform: %w", err)
	}
	copyOpts := oras.DefaultCopyOptions
	copyOpts.WithTargetPlatform(&targetPlatform)

	// Use oras.Copy to pull the artifact
	pulledDesc, err := oras.Copy(ctx, repo, a.repoRef, memStore, a.repoRef, copyOpts)
	if err != nil {
		// Check if the error is a CopyError and return details
		var copyErr *oras.CopyError
		if errors.As(err, &copyErr) {
			return copyErr.Err
		}

		return err
	}

	return processPulledArtifact(ctx, memStore, pulledDesc, a.path)
}

// pullWithFallbackStrategies tries different strategies to pull an artifact when no platform is specified
func pullWithFallbackStrategies(ctx context.Context, repo *remote.Repository, repoRef string, memStore *memory.Store, outputDir string) error {
	// Strategy 1: Try to pull an image generated with artifact-cli push (no platform selector)
	fmt.Println("Strategy 1: Trying to pull artifact-cli generated image (no platform selector)...")
	pulledDesc, err := oras.Copy(ctx, repo, repoRef, memStore, repoRef, oras.DefaultCopyOptions)
	if err == nil {
		// Check if this is actually an artifact-cli artifact
		if isOciCliArtifact(ctx, memStore, pulledDesc) {
			fmt.Printf("Successfully pulled artifact-cli artifact: %s\n", pulledDesc.Digest)
			return processPulledArtifact(ctx, memStore, pulledDesc, outputDir)
		} else {
			fmt.Printf("Strategy 1: Found artifact but not artifact-cli generated, trying next strategy...\n")
		}
	} else {
		fmt.Printf("Strategy 1 failed: %v\n", err)
	}

	// Strategy 2: Try to pull an image generated via imgpkg (Docker manifest format)
	fmt.Println("Strategy 2: Trying to pull imgpkg generated image...")
	// For imgpkg, we need to handle Docker manifest format
	// This is more complex and would require custom handling of Docker manifests
	// For now, we'll skip this and go to strategy 3

	// Strategy 3: Try to pull an image generated with docker buildx using current architecture
	fmt.Println("Strategy 3: Trying to pull docker buildx image with current platform...")
	currentPlatform := &ocispec.Platform{
		OS:           runtime.GOOS,
		Architecture: runtime.GOARCH,
	}
	copyOpts := oras.DefaultCopyOptions
	copyOpts.WithTargetPlatform(currentPlatform)

	pulledDesc, err = oras.Copy(ctx, repo, repoRef, memStore, repoRef, copyOpts)
	if err == nil {
		fmt.Printf("Successfully pulled docker buildx artifact for platform %s/%s: %s\n",
			currentPlatform.OS, currentPlatform.Architecture, pulledDesc.Digest)
		return processPulledArtifact(ctx, memStore, pulledDesc, outputDir)
	}
	fmt.Printf("Strategy 3 failed: %v\n", err)

	return fmt.Errorf("all pull strategies failed. Last error: %w", err)
}

// processPulledArtifact processes the pulled artifact and extracts it to the output directory
func processPulledArtifact(ctx context.Context, memStore *memory.Store, pulledDesc ocispec.Descriptor, outputDir string) error {
	fmt.Printf("Processing pulled artifact with digest: %s\n", pulledDesc.Digest)

	// Fetch the manifest to find our folder layer
	manifestBytes, err := content.FetchAll(ctx, memStore, pulledDesc)
	if err != nil {
		return fmt.Errorf("failed to fetch manifest from memory store: %w", err)
	}

	var manifest ocispec.Manifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		return fmt.Errorf("failed to unmarshal manifest: %w", err)
	}

	fmt.Printf("Found manifest with media type %s\n", manifest.MediaType)

	// Check if this is an artifact-cli generated artifact
	if manifest.Annotations != nil {
		if tool, exists := manifest.Annotations["dev.educates.artifact-cli.tool"]; exists && tool == "artifact-cli" {
			fmt.Printf("Detected artifact-cli generated artifact (version: %s)\n", manifest.Annotations["dev.educates.artifact-cli.version"])
		}
	}

	// Find the specific layer containing our folder tarball
	// Try different media types for compatibility
	var folderLayerDesc *ocispec.Descriptor
	layerMediaTypes := []string{
		artifact.OCILayerMediaType,    // Our OCI layer type
		artifact.DockerLayerMediaType, // Docker layer type (imgpkg/docker buildx)
		artifact.FolderLayerMediaType, // Legacy folder layer type
	}

	for _, mediaType := range layerMediaTypes {
		for _, layer := range manifest.Layers {
			if layer.MediaType == mediaType {
				folderLayerDesc = &layer
				fmt.Printf("Found layer with media type %s: %s\n", mediaType, layer.Digest)
				break
			}
		}
		if folderLayerDesc != nil {
			break
		}
	}

	if folderLayerDesc == nil {
		return fmt.Errorf("could not find folder layer with any supported media type")
	}

	// Fetch the layer's content (the tarball)
	tarballBytes, err := content.FetchAll(ctx, memStore, *folderLayerDesc)
	if err != nil {
		return fmt.Errorf("failed to fetch layer content: %w", err)
	}

	// Extract the tarball to the output directory
	if err := utils.ExtractTarGz(bytes.NewReader(tarballBytes), outputDir); err != nil {
		return fmt.Errorf("failed to extract tarball: %w", err)
	}

	fmt.Println("\nSuccessfully pulled and extracted artifact.")
	return nil
}

// isOciCliArtifact checks if the pulled descriptor is an artifact-cli generated artifact
func isOciCliArtifact(ctx context.Context, memStore *memory.Store, desc ocispec.Descriptor) bool {
	// Fetch the manifest to check annotations
	manifestBytes, err := content.FetchAll(ctx, memStore, desc)
	if err != nil {
		return false
	}

	var manifest ocispec.Manifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		return false
	}

	// Check for artifact-cli specific annotations
	if manifest.Annotations != nil {
		if tool, exists := manifest.Annotations["dev.educates.artifact-cli.tool"]; exists && tool == "artifact-cli" {
			return true
		}
	}

	return false
}

func pushImageIndex(ctx context.Context, repo *remote.Repository, layerDesc ocispec.Descriptor, platforms []string) (ocispec.Descriptor, error) {
	annotations := map[string]string{
		"org.opencontainers.image.title":          "artifact-cli artifact",
		"org.opencontainers.image.description":    "Folder artifact created by artifact-cli",
		"dev.educates.artifact-cli.version":       artifact.ArtifactCliVersion,
		"dev.educates.artifact-cli.tool":          "artifact-cli",
		"dev.educates.artifact-cli.artifact-type": "oci",
	}
	return artifact.PushImageIndex(ctx, repo, layerDesc, platforms, annotations)
}

func pushSingleManifest(ctx context.Context, repo *remote.Repository, layerDesc ocispec.Descriptor, platform *ocispec.Platform) (ocispec.Descriptor, error) {
	annotations := map[string]string{
		"org.opencontainers.image.title":          "artifact-cli artifact",
		"org.opencontainers.image.description":    "Folder artifact created by artifact-cli",
		"dev.educates.artifact-cli.version":       artifact.ArtifactCliVersion,
		"dev.educates.artifact-cli.tool":          "artifact-cli",
		"dev.educates.artifact-cli.artifact-type": "oci",
	}
	return artifact.PushSingleManifest(ctx, repo, layerDesc, platform, annotations)
}
