package artifact

import (
	"os"
	"strings"
)

// RepositoryRef represents a repository reference with optional authentication
type RepositoryRef struct {
	URL      string
	Username string
	Password string
}

// NewRepositoryRef creates a new RepositoryRef with optional authentication
func NewRepositoryRef(url, username, password string) *RepositoryRef {
	// Check for environment variables as fallback
	if username == "" {
		username = os.Getenv("ARTIFACT_CLI_USERNAME")
	}
	if password == "" {
		password = os.Getenv("ARTIFACT_CLI_PASSWORD")
	}

	return &RepositoryRef{
		URL:      url,
		Username: username,
		Password: password,
	}
}

// String returns the URL string representation
func (r *RepositoryRef) String() string {
	return r.URL
}

// HasAuth returns true if both username and password are provided
func (r *RepositoryRef) HasAuth() bool {
	return r.Username != "" && r.Password != ""
}

// GetAuthString returns the authentication string for registry operations
func (r *RepositoryRef) GetAuthString() string {
	if !r.HasAuth() {
		return ""
	}
	return r.Username + ":" + r.Password
}

// ParseRepositoryRef parses a repository reference string and returns a RepositoryRef
func ParseRepositoryRef(repoStr string) *RepositoryRef {
	// Handle cases where the repoStr might contain credentials
	// Format: username:password@registry.com/repo:tag
	if strings.Contains(repoStr, "@") {
		parts := strings.Split(repoStr, "@")
		if len(parts) == 2 {
			authPart := parts[0]
			urlPart := parts[1]

			if strings.Contains(authPart, ":") {
				authParts := strings.Split(authPart, ":")
				if len(authParts) == 2 {
					return &RepositoryRef{
						URL:      urlPart,
						Username: authParts[0],
						Password: authParts[1],
					}
				}
			}
		}
	}

	// No credentials in the string, use as-is
	return &RepositoryRef{
		URL: repoStr,
	}
}
