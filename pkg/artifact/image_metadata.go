package artifact

import (
	"context"
	"encoding/json"
	"fmt"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"gopkg.in/yaml.v3"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/registry/remote"
)

type MediaType int

const (
	// Custom media type for our folder layer
	DockerIndexMediaType    = "application/vnd.docker.distribution.manifest.list.v2+json" // Multi-platform
	DockerManifestMediaType = "application/vnd.docker.distribution.manifest.v2+json"      // Single-platform
	OCIIndexMediaType       = "application/vnd.oci.image.index.v1+json"                   // Multi-platform
	OCIManifestMediaType    = "application/vnd.oci.image.manifest.v1+json"                // Single-platform

	OCIConfigMediaType = "application/vnd.oci.image.config.v1+json"
	OCILayerMediaType  = "application/vnd.oci.image.layer.v1.tar+gzip"
)

const (
	DockerLayerMediaType = "application/vnd.docker.image.rootfs.diff.tar.gzip"
	FolderLayerMediaType = "application/vnd.oci.image.layer.v1.tar+gzip"
)

const (
	Undefined            MediaType = 0
	DockerMultiPlatform  MediaType = 1
	OCIMultiPlatform     MediaType = 2
	DockerSinglePlatform MediaType = 3
	OCISinglePlatform    MediaType = 4
)

func (m MediaType) String() string {
	return [...]string{
		Undefined:            "Undefined",
		DockerMultiPlatform:  "Docker-MultiPlatform",
		OCIMultiPlatform:     "OCI-MultiPlatform",
		DockerSinglePlatform: "Docker-SinglePlatform",
		OCISinglePlatform:    "OCI-SinglePlatform",
	}[m]
}

func (m MediaType) IsMultiPlatform() bool {
	return m == DockerMultiPlatform || m == OCIMultiPlatform
}

func (m MediaType) IsSinglePlatform() bool {
	return m == DockerSinglePlatform || m == OCISinglePlatform
}

func (m MediaType) IsOci() bool {
	return m == OCIMultiPlatform || m == OCISinglePlatform
}

func (m MediaType) IsDocker() bool {
	return m == DockerMultiPlatform || m == DockerSinglePlatform
}

// MarshalJSON implements json.Marshaler interface
func (m MediaType) MarshalJSON() ([]byte, error) {
	return json.Marshal(m.String())
}

// UnmarshalJSON implements json.Unmarshaler interface
func (m *MediaType) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	// Convert string back to MediaType
	switch s {
	case "Undefined":
		*m = Undefined
	case "Docker-MultiPlatform":
		*m = DockerMultiPlatform
	case "OCI-MultiPlatform":
		*m = OCIMultiPlatform
	case "Docker-SinglePlatform":
		*m = DockerSinglePlatform
	case "OCI-SinglePlatform":
		*m = OCISinglePlatform
	default:
		*m = Undefined
	}
	return nil
}

// MarshalYAML implements yaml.Marshaler interface
func (m MediaType) MarshalYAML() (interface{}, error) {
	return m.String(), nil
}

// UnmarshalYAML implements yaml.Unmarshaler interface
func (m *MediaType) UnmarshalYAML(value *yaml.Node) error {
	var s string
	if err := value.Decode(&s); err != nil {
		return err
	}

	// Convert string back to MediaType
	switch s {
	case "Undefined":
		*m = Undefined
	case "Docker-MultiPlatform":
		*m = DockerMultiPlatform
	case "OCI-MultiPlatform":
		*m = OCIMultiPlatform
	case "Docker-SinglePlatform":
		*m = DockerSinglePlatform
	case "OCI-SinglePlatform":
		*m = OCISinglePlatform
	default:
		*m = Undefined
	}
	return nil
}

func DetectArtifactType(mediaTypeString string) MediaType {
	switch mediaTypeString {
	case OCIIndexMediaType:
		return OCIMultiPlatform
	case OCIManifestMediaType:
		return OCISinglePlatform
	case DockerManifestMediaType:
		return OCISinglePlatform
	case DockerIndexMediaType:
		return DockerMultiPlatform
	default:
		return Undefined
	}
}

// ManifestWrapper holds either a Manifest or Index
type ManifestWrapper struct {
	Manifest *ocispec.Manifest `json:"manifest,omitempty"`
	Index    *ocispec.Index    `json:"index,omitempty"`
}

// IsManifest returns true if this wrapper contains a Manifest
func (mw *ManifestWrapper) IsManifest() bool {
	return mw.Manifest != nil
}

// IsIndex returns true if this wrapper contains an Index
func (mw *ManifestWrapper) IsIndex() bool {
	return mw.Index != nil
}

// GetMediaType returns the media type of the contained manifest/index
func (mw *ManifestWrapper) GetMediaType() string {
	if mw.Manifest != nil {
		return mw.Manifest.MediaType
	}
	if mw.Index != nil {
		return mw.Index.MediaType
	}
	return ""
}

type ImageMetadata struct {
	ImageRef      string           `json:"image_ref"`
	MediaType     MediaType        `json:"media_type"`
	OciCompliant  bool             `json:"oci_compliant"`
	MultiPlatform bool             `json:"multi_platform"`
	Platforms     []PlatformInfo   `json:"platforms,omitempty"`
	Manifest      *ManifestWrapper `json:"raw_manifest,omitempty"`
}

type PlatformInfo struct {
	OS           string `json:"os"`
	Architecture string `json:"architecture"`
}

func GetImageMetadata(ctx context.Context, repo *remote.Repository, imageMetadata *ImageMetadata) error {
	// Resolve the tag to a descriptor.
	// This gets the metadata of the manifest/index without downloading the content.
	descriptor, err := repo.Resolve(ctx, repo.Reference.String())
	if err != nil {
		return fmt.Errorf("failed to fetch descriptor: %v", err)
	}

	mediaType := DetectArtifactType(descriptor.MediaType)

	imageMetadata.MediaType = mediaType
	imageMetadata.OciCompliant = mediaType.IsOci()
	imageMetadata.MultiPlatform = mediaType.IsMultiPlatform()
	imageMetadata.Manifest = &ManifestWrapper{
		Manifest: nil,
		Index:    nil,
	}

	// Fetch the manifest from the repository by tag
	_, fetchedManifestContent, err := oras.FetchBytes(ctx, repo, imageMetadata.ImageRef, oras.DefaultFetchBytesOptions)
	if err != nil {
		return fmt.Errorf("failed to fetch manifest: %w", err)
	}

	if imageMetadata.MediaType == OCIMultiPlatform || imageMetadata.MediaType == OCISinglePlatform {
		// Parse the fetched manifest content and get the layers
		if err := json.Unmarshal(fetchedManifestContent, &imageMetadata.Manifest.Index); err != nil {
			return fmt.Errorf("failed to unmarshal manifest: %w", err)
		}
		// Copy the platforms from the manifests in the index to the image metadata
		for _, manifest := range imageMetadata.Manifest.Index.Manifests {
			imageMetadata.Platforms = append(imageMetadata.Platforms, PlatformInfo{
				OS:           manifest.Platform.OS,
				Architecture: manifest.Platform.Architecture,
			})
		}
	} else {
		if err := json.Unmarshal(fetchedManifestContent, &imageMetadata.Manifest.Manifest); err != nil {
			return fmt.Errorf("failed to unmarshal manifest: %w", err)
		}
	}

	return nil
}
