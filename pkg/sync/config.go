package sync

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// loadSyncConfig loads and parses the sync configuration from YAML file
func LoadSyncConfig(config *SyncConfig, configPath string) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse YAML: %w", err)
	}

	return nil
}

// validateSyncConfig validates the sync configuration
func ValidateSyncConfig(config *SyncConfig) error {
	if config.Spec.Dest == "" {
		return fmt.Errorf("destination path is required")
	}

	if len(config.Spec.Artifacts) == 0 {
		return fmt.Errorf("at least one artifact must be specified")
	}

	for i, artifact := range config.Spec.Artifacts {
		if artifact.Image.URL == "" {
			return fmt.Errorf("artifact %d: image URL is required", i+1)
		}
	}

	return nil
}
