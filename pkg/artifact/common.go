package artifact

import (
	"bytes"
	"context"
	"educates-artifact-cli/pkg/utils"
	"encoding/json"
	"fmt"

	"github.com/opencontainers/go-digest"
	"github.com/opencontainers/image-spec/specs-go"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2/registry/remote"
)

func PushSingleManifest(ctx context.Context, repo *remote.Repository, layerDesc ocispec.Descriptor, platform *ocispec.Platform, annotations map[string]string) (ocispec.Descriptor, error) {
	// Create and push a minimal config blob
	configBytes := []byte("{}")
	configDesc := ocispec.Descriptor{
		MediaType: OCIConfigMediaType,
		Digest:    digest.FromBytes(configBytes),
		Size:      int64(len(configBytes)),
	}
	if err := repo.Push(ctx, configDesc, bytes.NewReader(configBytes)); err != nil {
		return ocispec.Descriptor{}, fmt.Errorf("failed to push config blob: %w", err)
	}
	fmt.Printf("Pushed config: %s\n", configDesc.Digest)

	// Create the image manifest
	manifest := ocispec.Manifest{
		Versioned: specs.Versioned{
			SchemaVersion: 2,
		},
		Config:      configDesc,
		Layers:      []ocispec.Descriptor{layerDesc},
		MediaType:   OCIManifestMediaType,
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
		MediaType: OCIManifestMediaType,
		Digest:    digest.FromBytes(manifestBytes),
		Size:      int64(len(manifestBytes)),
		Platform:  platform,
	}

	if err := repo.Push(ctx, manifestDesc, bytes.NewReader(manifestBytes)); err != nil {
		return ocispec.Descriptor{}, fmt.Errorf("failed to push manifest: %w", err)
	}

	if platform != nil {
		fmt.Printf("Pushed manifest for platform %s/%s: %s\n", platform.OS, platform.Architecture, manifestDesc.Digest)
	} else {
		fmt.Printf("Pushed manifest without platform selector: %s\n", manifestDesc.Digest)
	}

	return manifestDesc, nil
}

func PushImageIndex(ctx context.Context, repo *remote.Repository, layerDesc ocispec.Descriptor, platforms []string, annotations map[string]string) (ocispec.Descriptor, error) {
	var manifestDescriptors []ocispec.Descriptor

	fmt.Printf("Pushing index...\n")

	for _, platformStr := range platforms {
		var platform ocispec.Platform
		err := utils.ParsePlatform(&platform, platformStr)
		if err != nil {
			return ocispec.Descriptor{}, fmt.Errorf("failed to parse platform: %w", err)
		}

		fmt.Printf("Processing platform %s/%s...\n", platform.OS, platform.Architecture)

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
		MediaType:   OCIIndexMediaType,
		Manifests:   manifestDescriptors,
		Annotations: annotations,
	}

	indexBytes, err := json.Marshal(index)
	if err != nil {
		return ocispec.Descriptor{}, fmt.Errorf("failed to marshal index: %w", err)
	}

	indexDesc := ocispec.Descriptor{
		MediaType: OCIIndexMediaType,
		Digest:    digest.FromBytes(indexBytes),
		Size:      int64(len(indexBytes)),
	}

	if err := repo.Push(ctx, indexDesc, bytes.NewReader(indexBytes)); err != nil {
		return ocispec.Descriptor{}, fmt.Errorf("failed to push index: %w", err)
	}
	fmt.Printf("Pushed index: %s\n", indexDesc.Digest)

	return indexDesc, nil
}
