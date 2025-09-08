package sync

import (
	"educates-artifact-cli/pkg/artifact"
	"educates-artifact-cli/pkg/artifact/educates"
	"educates-artifact-cli/pkg/artifact/imgpkg"
	"educates-artifact-cli/pkg/artifact/oci"
	"educates-artifact-cli/pkg/utils"
	"fmt"
	"os"
	"path/filepath"
)

func Sync(config SyncConfig) error {
	// Create destination directory
	// if folder is not absolute, make it absolute from the current working directory
	destDir := config.Spec.Dest
	if !filepath.IsAbs(destDir) {
		wd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current working directory: %w", err)
		}
		destDir = filepath.Join(wd, config.Spec.Dest)
	}
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Process each artifact
	for i, artifactConfig := range config.Spec.Artifacts {
		utils.VerbosePrintf("Processing artifact %d/%d: %s\n", i+1, len(config.Spec.Artifacts), artifactConfig.Image.URL)

		if err := processArtifact(artifactConfig, config.Spec.Dest); err != nil {
			return fmt.Errorf("failed to process artifact %s: %w", artifactConfig.Image.URL, err)
		}
	}

	fmt.Printf("Successfully synced %d artifacts to %s\n", len(config.Spec.Artifacts), config.Spec.Dest)
	return nil
}

// processArtifact processes a single artifact configuration
func processArtifact(artifactConfig SyncArtifact, destDir string) error {
	// Create temporary directory for extraction
	tempDir, err := os.MkdirTemp("", "artifact-cli-sync-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	// fmt.Printf("Created temp directory: %s\n", tempDir)
	// defer os.RemoveAll(tempDir)

	// Determine artifact type and create appropriate artifact handler
	var artifactHandler artifact.Artifact

	// Try to determine artifact type by attempting different pull strategies
	// This follows the same fallback logic as the pull command
	platformStr := utils.GetOSPlatformStr()

	// Create repository reference with credentials
	repoRef := artifact.NewRepositoryRef(artifactConfig.Image.URL, artifactConfig.Image.Username, artifactConfig.Image.Password)

	// Try OCI format first
	artifactHandler = oci.NewOciImageArtifact(repoRef, nil, platformStr, tempDir)
	if err := artifactHandler.Pull(); err != nil {
		// Try imgpkg format
		artifactHandler = imgpkg.NewImgpkgImageArtifact(repoRef, nil, platformStr, tempDir)
		if err := artifactHandler.Pull(); err != nil {
			// Try educates format
			artifactHandler = educates.NewEducatesImageArtifact(repoRef, nil, platformStr, tempDir)
			if err := artifactHandler.Pull(); err != nil {
				return fmt.Errorf("failed to pull artifact with any supported format: %w", err)
			}
		}
	}

	// Apply include/exclude patterns and copy files to destination
	fileFilter := FileFilter{
		IncludePaths: artifactConfig.IncludePaths,
		ExcludePaths: artifactConfig.ExcludePaths,
	}
	if err := fileFilter.Apply(tempDir); err != nil {
		return fmt.Errorf("failed to apply file filter: %w", err)
	}

	// We need to add the path to the destDir
	destDir = filepath.Join(destDir, artifactConfig.Path)
	return copyFiles(tempDir, destDir)
}
