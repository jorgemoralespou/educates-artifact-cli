package imgpkg

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content"
	"oras.land/oras-go/v2/content/memory"
	"oras.land/oras-go/v2/registry/remote"

	"educates-artifact-cli/pkg/artifact"
	"educates-artifact-cli/pkg/utils"
)

type ImgpkgImageArtifact struct {
	repoRef       string
	pushPlatforms []string
	pullPlatform  string
	path          string
}

func NewImgpkgImageArtifact(repoRef string, pushPlatforms []string, pullPlatform string, path string) *ImgpkgImageArtifact {
	return &ImgpkgImageArtifact{repoRef: repoRef, pushPlatforms: pushPlatforms, pullPlatform: pullPlatform, path: path}
}

func (a *ImgpkgImageArtifact) Push() error {
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

	// --- Single Artifact Push (no platform selector) ---
	fmt.Println("Performing a single artifact push without platform selector...")
	rootDesc, err = pushSingleManifest(ctx, repo, layerDesc, nil)
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

func (a *ImgpkgImageArtifact) Pull() error {
	ctx := context.Background()

	// Create a registry client
	repo, err := remote.NewRepository(a.repoRef)
	if err != nil {
		return fmt.Errorf("failed to create repository client: %w", err)
	}
	repo.PlainHTTP = true

	// Create a memory store to hold the pulled content
	memStore := memory.New()

	// Imgpkg does not support multiple platforms, so we don't pull the image for a specific platform
	copyOpts := oras.DefaultCopyOptions

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

	// Imgpkg does not support multiple platforms, so we don't pull the image if the manifest is an OCI Index
	if manifest.MediaType != artifact.DockerManifestMediaType {
		return fmt.Errorf("this is not an imgpkg artifact, please use the correct target to pull it")
	}

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

	// When this happens, maybe is because we are pulling an artifact generated with oci
	if folderLayerDesc == nil {
		return fmt.Errorf("could not find folder layer with any supported media type")
	}

	// Fetch the layer's content (the tarball)
	tarballBytes, err := content.FetchAll(ctx, memStore, *folderLayerDesc)
	if err != nil {
		return fmt.Errorf("failed to fetch layer content: %w", err)
	}

	// Extract the tarball to the output directory
	fmt.Printf("Extracting content to '%s'...\n", outputDir)
	if err := utils.ExtractTarGz(bytes.NewReader(tarballBytes), outputDir); err != nil {
		return fmt.Errorf("failed to extract tarball: %w", err)
	}

	fmt.Println("\nSuccessfully pulled and extracted artifact.")
	return nil
}

func pushSingleManifest(ctx context.Context, repo *remote.Repository, layerDesc ocispec.Descriptor, platform *ocispec.Platform) (ocispec.Descriptor, error) {
	annotations := map[string]string{
		"org.opencontainers.image.title":          "artifact-cli artifact",
		"org.opencontainers.image.description":    "Folder artifact created by artifact-cli",
		"dev.educates.artifact-cli.version":       artifact.ArtifactCliVersion,
		"dev.educates.artifact-cli.tool":          "artifact-cli",
		"dev.educates.artifact-cli.artifact-type": "imgpkg",
	}
	return artifact.PushSingleManifest(ctx, repo, layerDesc, platform, annotations)
}
