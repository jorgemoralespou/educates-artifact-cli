package artifact

const (
	// Custom media type for our folder layer
	FolderLayerMediaType  = "application/vnd.oci.image.layer.v1.tar+gzip"
	FolderConfigMediaType = "application/vnd.oci.image.config.v1+json"

	// Docker media types (used by imgpkg and docker buildx)
	DockerManifestMediaType = "application/vnd.docker.distribution.manifest.v2+json"
	DockerConfigMediaType   = "application/vnd.docker.container.image.v1+json"
	DockerLayerMediaType    = "application/vnd.docker.image.rootfs.diff.tar.gzip"

	// OCI media types (used by docker buildx and our CLI)
	OCIManifestMediaType = "application/vnd.oci.image.manifest.v1+json"
	OCIConfigMediaType   = "application/vnd.oci.image.config.v1+json"
	OCILayerMediaType    = "application/vnd.oci.image.layer.v1.tar+gzip"
	OCIIndexMediaType    = "application/vnd.oci.image.index.v1+json"

	// Artifact CLI version for annotations
	ArtifactCliVersion = "1.0.0"
)
