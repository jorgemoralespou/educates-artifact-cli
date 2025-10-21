package sync

// SyncConfig represents the configuration structure for sync command
type SyncConfig struct {
	Spec SyncSpec `yaml:"spec" json:"spec"`
}

type SyncSpec struct {
	Dest      string         `yaml:"dest" json:"dest"`
	Artifacts []SyncArtifact `yaml:"artifacts" json:"artifacts"`
}

type SyncArtifact struct {
	Image        ArtifactImage `yaml:"image" json:"image"`
	Path         string        `yaml:"path" json:"path"`
	IncludePaths []string      `json:"includePaths,omitempty" yaml:"includePaths,omitempty"`
	ExcludePaths []string      `json:"excludePaths,omitempty" yaml:"excludePaths,omitempty"`
}

type ArtifactImage struct {
	URL      string `yaml:"url" json:"url"`
	Username string `yaml:"username,omitempty" json:"username,omitempty"`
	Password string `yaml:"password,omitempty" json:"password,omitempty"`
	Insecure bool   `yaml:"insecure,omitempty" json:"insecure,omitempty"`
}
