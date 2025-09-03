package utils

import (
	"fmt"
	"runtime"
	"strings"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

const DefaultPlatforms = "linux/amd64, linux/arm64"

const SupportedPlatforms = "linux/amd64, linux/arm64, darwin/amd64, darwin/arm64"

func ValidatePlatforms(platforms []string) error {
	for _, platform := range platforms {
		if !strings.Contains(SupportedPlatforms, platform) {
			return fmt.Errorf("unsupported platform: %s", platform)
		}
	}
	return nil
}

// SlicePlatforms converts a comma-separated string into a slice of OCI Platform structs.
func SlicePlatforms(platformStr string) []string {
	if platformStr == "" {
		return nil
	}
	parts := strings.Split(platformStr, ",")
	platforms := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			platforms = append(platforms, trimmed)
		}
	}
	return platforms
}

/**
 * Parse the provided platform and set the target platform
 *
 * @param targetPlatform - The target platform to set
 * @param platformStr - The platform string to parse
 * @return error - An error if the platform string is invalid
 */
func ParsePlatform(targetPlatform *ocispec.Platform, platformStr string) error {
	osArch := strings.Split(platformStr, "/")
	if len(osArch) != 2 {
		return fmt.Errorf("invalid platform format: %s (expected format: os/arch)", platformStr)
	}
	*targetPlatform = ocispec.Platform{
		OS:           osArch[0],
		Architecture: osArch[1],
	}
	return nil
}

// GetTagFromRef extracts the tag from a full repository reference string.
// Example: ghcr.io/user/repo:tag -> tag
func GetTagFromRef(ref string) string {
	if i := strings.LastIndex(ref, ":"); i != -1 {
		return ref[i+1:]
	}
	return "latest" // Default tag
}

func GetOSPlatformStr() string {
	return fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)
}
