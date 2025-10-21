package oci

import (
	"bytes"
	"context"
	"educates-artifact-cli/pkg/artifact"
	"educates-artifact-cli/pkg/utils"
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"github.com/opencontainers/go-digest"
	"github.com/opencontainers/image-spec/specs-go"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content"
	"oras.land/oras-go/v2/content/memory"
	"oras.land/oras-go/v2/registry/remote"
)

type OciImageArtifact struct {
	repoRef       *artifact.RepositoryRef
	pushPlatforms []string
	pullPlatform  string
	path          string
}

func NewOciImageArtifact(repoRef *artifact.RepositoryRef, pushPlatforms []string, pullPlatform string, path string) *OciImageArtifact {
	return &OciImageArtifact{repoRef: repoRef, pushPlatforms: pushPlatforms, pullPlatform: pullPlatform, path: path}
}

func (a *OciImageArtifact) Push(ctx context.Context) error {
	fmt.Printf("OCI Artifact Push\n")
	fmt.Printf("Packaging folder '%s'...\n", a.path)
	// Create a tarball of the folder in memory
	tarballBytes, err := utils.CreateTarGz(a.path)
	if err != nil {
		return fmt.Errorf("failed to create tarball: %w", err)
	}

	// Create a new registry client with authentication
	// repo, err := artifact.CreateAuthenticatedRepository(ctx, a.repoRef)
	repo, err := a.repoRef.Authenticate(ctx)
	if err != nil {
		return fmt.Errorf("failed to create repository client: %w", err)
	}

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
	utils.VerbosePrintf("Pushed layer: %s\n", layerDesc.Digest)

	var rootDesc ocispec.Descriptor

	// When no platforms are provided, we use the default platforms (linux/amd64 and linux/arm64) as well
	// as the current platform
	if len(a.pushPlatforms) == 0 {
		a.pushPlatforms = utils.SlicePlatforms(utils.DefaultPlatforms)
		currentPlatform := utils.GetOSPlatformStr()
		// If current platform is not in the default platforms, add it
		if !slices.Contains(a.pushPlatforms, currentPlatform) {
			a.pushPlatforms = append(a.pushPlatforms, currentPlatform)
		}
	}

	// Create annotations
	annotations := map[string]string{
		"org.opencontainers.image.title":       "artifact-cli artifact",
		"org.opencontainers.image.description": "Created by artifact-cli",
	}
	// --- Multi-Platform (Index) Push ---
	utils.VerbosePrintf("Performing a multi-platform push for: %s\n", a.pushPlatforms)
	rootDesc, err = PushImageIndex(ctx, repo, layerDesc, a.pushPlatforms, annotations)
	if err != nil {
		return err
	}

	// Tag the root manifest/index with the provided tag
	tag := utils.GetTagFromRef(a.repoRef.String())
	if err := repo.Tag(ctx, rootDesc, tag); err != nil {
		return fmt.Errorf("failed to tag root descriptor: %w", err)
	}

	fmt.Printf("\nSuccessfully pushed and tagged artifact: %s\n", a.repoRef.String())
	utils.VerbosePrintf("Root digest: %s\n", rootDesc.Digest)

	return nil

}

func (a *OciImageArtifact) Pull(ctx context.Context) error {
	fmt.Printf("OCI Artifact Pull\n")

	// Create a new registry client with authentication
	// repo, err := artifact.CreateAuthenticatedRepository(ctx, a.repoRef)
	repo, err := a.repoRef.Authenticate(ctx)
	if err != nil {
		return err
	}

	imageMetadata := artifact.ImageMetadata{
		ImageRef: a.repoRef.String(),
	}
	err = artifact.GetImageMetadata(ctx, repo, &imageMetadata)
	if err != nil {
		return err
	}

	// Create a memory store to hold the pulled content
	memStore := memory.New()

	// Define copy options to specify the target platform
	// If image is multi-platform, we need to pull the image for the target platform
	// If image is single-platform, we need to pull the image for the current platform
	copyOpts := oras.DefaultCopyOptions
	if imageMetadata.MediaType == artifact.OCIMultiPlatform {
		var targetPlatform ocispec.Platform
		err = utils.ParsePlatform(&targetPlatform, a.pullPlatform)
		if err != nil {
			return fmt.Errorf("failed to parse platform: %w", err)
		}
		copyOpts.WithTargetPlatform(&targetPlatform)
		fmt.Printf("Pulling artifact for platform %s/%s...\n", targetPlatform.OS, targetPlatform.Architecture)
	} else {
		currentPlatform := utils.GetOSPlatformStr()
		fmt.Printf("Pulling artifact for current platform: %s\n", currentPlatform)
	}

	// Use oras.Copy to pull the artifact
	pulledDesc, err := oras.Copy(ctx, repo, a.repoRef.String(), memStore, a.repoRef.String(), copyOpts)
	if err != nil {
		// Check if the error is a CopyError and return details
		var copyErr *oras.CopyError
		if errors.As(err, &copyErr) {
			return copyErr.Err
		}

		return err
	}

	err = processPulledArtifact(ctx, memStore, pulledDesc, a.path)
	if err != nil {
		return err
	}

	fmt.Printf("\nSuccessfully pulled and extracted artifact to %s.\n", a.path)
	return nil
}

// processPulledArtifact processes the pulled artifact and extracts it to the output directory
func processPulledArtifact(ctx context.Context, memStore *memory.Store, pulledDesc ocispec.Descriptor, outputDir string) error {
	utils.VerbosePrintf("Processing pulled artifact with digest: %s\n", pulledDesc.Digest)

	// Fetch the manifest to find our folder layer
	manifestBytes, err := content.FetchAll(ctx, memStore, pulledDesc)
	if err != nil {
		return fmt.Errorf("failed to fetch manifest from memory store: %w", err)
	}

	var manifest ocispec.Manifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		return fmt.Errorf("failed to unmarshal manifest: %w", err)
	}

	utils.VerbosePrintf("Found manifest with media type %s\n", manifest.MediaType)

	// Check if this is an artifact-cli generated artifact
	if manifest.Annotations != nil {
		if tool, exists := manifest.Annotations["dev.educates.artifact-cli.tool"]; exists && tool == "artifact-cli" {
			utils.VerbosePrintf("Detected artifact-cli generated artifact (version: %s)\n", manifest.Annotations["dev.educates.artifact-cli.version"])
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
				utils.VerbosePrintf("Found layer with media type %s: %s\n", mediaType, layer.Digest)
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

	utils.VerbosePrintln("Successfully pulled and extracted artifact.")
	return nil
}

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

// func pushImageIndex(ctx context.Context, repo *remote.Repository, layerDesc ocispec.Descriptor, platforms []string) (ocispec.Descriptor, error) {
// 	annotations := map[string]string{
// 		"org.opencontainers.image.title":          "artifact-cli artifact",
// 		"org.opencontainers.image.description":    "Folder artifact created by artifact-cli",
// 		"dev.educates.artifact-cli.version":       artifact.ArtifactCliVersion,
// 		"dev.educates.artifact-cli.tool":          "artifact-cli",
// 		"dev.educates.artifact-cli.artifact-type": "oci",
// 	}
// 	return artifact.PushImageIndex(ctx, repo, layerDesc, platforms, annotations)
// }

func PushImageIndex(ctx context.Context, repo *remote.Repository, layerDesc ocispec.Descriptor, platforms []string, annotations map[string]string) (ocispec.Descriptor, error) {
	var manifestDescriptors []ocispec.Descriptor

	utils.VerbosePrintf("Pushing index...\n")

	for _, platformStr := range platforms {
		var platform ocispec.Platform
		err := utils.ParsePlatform(&platform, platformStr)
		if err != nil {
			return ocispec.Descriptor{}, fmt.Errorf("failed to parse platform: %w", err)
		}

		utils.VerbosePrintf("Processing platform %s/%s...\n", platform.OS, platform.Architecture)

		manifestDesc, err := PushSingleManifest(ctx, repo, layerDesc, &platform, annotations)
		if err != nil {
			return ocispec.Descriptor{}, fmt.Errorf("failed to push manifest for platform %s/%s: %w", platform.OS, platform.Architecture, err)
		}
		manifestDescriptors = append(manifestDescriptors, manifestDesc)
	}

	// Create the image index
	index := ocispec.Index{
		Versioned: specs.Versioned{
			SchemaVersion: 2,
		},
		MediaType:   artifact.OCIIndexMediaType,
		Manifests:   manifestDescriptors,
		Annotations: annotations,
	}

	indexBytes, err := json.Marshal(index)
	if err != nil {
		return ocispec.Descriptor{}, fmt.Errorf("failed to marshal index: %w", err)
	}

	indexDesc := ocispec.Descriptor{
		MediaType: artifact.OCIIndexMediaType,
		Digest:    digest.FromBytes(indexBytes),
		Size:      int64(len(indexBytes)),
	}

	if err := repo.Push(ctx, indexDesc, bytes.NewReader(indexBytes)); err != nil {
		return ocispec.Descriptor{}, fmt.Errorf("failed to push index: %w", err)
	}
	utils.VerbosePrintf("Pushed index: %s\n", indexDesc.Digest)

	return indexDesc, nil
}

func PushSingleManifest(ctx context.Context, repo *remote.Repository, layerDesc ocispec.Descriptor, platform *ocispec.Platform, annotations map[string]string) (ocispec.Descriptor, error) {
	// Create and push a minimal config blob
	configBytes := []byte("{}")
	configDesc := ocispec.Descriptor{
		MediaType: artifact.OCIConfigMediaType,
		Digest:    digest.FromBytes(configBytes),
		Size:      int64(len(configBytes)),
	}
	if err := repo.Push(ctx, configDesc, bytes.NewReader(configBytes)); err != nil {
		return ocispec.Descriptor{}, fmt.Errorf("failed to push config blob: %w", err)
	}
	utils.VerbosePrintf("Pushed config: %s\n", configDesc.Digest)

	// Create the image manifest
	manifest := ocispec.Manifest{
		Versioned: specs.Versioned{
			SchemaVersion: 2,
		},
		Config:      configDesc,
		Layers:      []ocispec.Descriptor{layerDesc},
		MediaType:   artifact.OCIManifestMediaType,
		Annotations: annotations,
	}
	if platform != nil {
		manifest.Annotations["org.opencontainers.image.platform"] = fmt.Sprintf("%s/%s", platform.OS, platform.Architecture)
	}

	manifestBytes, err := json.Marshal(manifest)
	if err != nil {
		return ocispec.Descriptor{}, fmt.Errorf("failed to marshal manifest: %w", err)
	}

	manifestDesc := ocispec.Descriptor{
		MediaType: artifact.OCIManifestMediaType,
		Digest:    digest.FromBytes(manifestBytes),
		Size:      int64(len(manifestBytes)),
		Platform:  platform,
	}

	if err := repo.Push(ctx, manifestDesc, bytes.NewReader(manifestBytes)); err != nil {
		return ocispec.Descriptor{}, fmt.Errorf("failed to push manifest: %w", err)
	}

	if platform != nil {
		utils.VerbosePrintf("Pushed manifest for platform %s/%s: %s\n", platform.OS, platform.Architecture, manifestDesc.Digest)
	} else {
		utils.VerbosePrintf("Pushed manifest without platform selector: %s\n", manifestDesc.Digest)
	}

	return manifestDesc, nil
}
