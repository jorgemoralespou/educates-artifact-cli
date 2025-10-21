package artifact

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"

	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
)

// RepositoryRef represents a repository reference with optional authentication
type RepositoryRef struct {
	URL      string
	Username string
	Password string
	Insecure bool
}

// NewRepositoryRef creates a new RepositoryRef with optional authentication
func NewRepositoryRef(url, username, password string, insecure bool) *RepositoryRef {
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
		Insecure: insecure,
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

func (r *RepositoryRef) Authenticate(ctx context.Context) (*remote.Repository, error) {
	repo, err := remote.NewRepository(r.String())
	if err != nil {
		return nil, fmt.Errorf("failed to create new repository client: %v", err)
	}

	// Check if the repository is insecure
	if r.Insecure || strings.Contains(repo.Reference.Registry, "localhost") || strings.Contains(repo.Reference.Registry, "127.0.0.1") {
		repo.PlainHTTP = true
	}

	if r.HasAuth() {
		// Set up authentication if credentials are provided
		cred := auth.Credential{
			Username: r.Username,
			Password: r.Password,
		}

		// Extract registry host from the repository URL
		registryHost := extractRegistryHost(r.URL)

		authClient := &auth.Client{
			Client:     nil, // Use default client
			Cache:      auth.NewCache(),
			Credential: auth.StaticCredential(registryHost, cred),
		}
		repo.Client = authClient

		err = validateAuthentication(ctx, r, repo)
		if err != nil {
			return nil, err
		}
	}

	return repo, nil
}

// extractRegistryHost extracts the registry host from a repository URL
func extractRegistryHost(repoURL string) string {
	// Handle cases where the URL might not have a scheme
	if !strings.Contains(repoURL, "://") {
		repoURL = "https://" + repoURL
	}

	parsedURL, err := url.Parse(repoURL)
	if err != nil {
		// If parsing fails, try to extract host manually
		parts := strings.Split(repoURL, "/")
		if len(parts) > 0 {
			// Remove any scheme prefix
			host := parts[0]
			if strings.Contains(host, "://") {
				host = strings.Split(host, "://")[1]
			}
			return host
		}
		return repoURL
	}

	return parsedURL.Host
}

// ValidateAuthentication tests if the provided credentials work with the registry
// func ValidateAuthentication(ctx context.Context, repoRef *RepositoryRef) error {
// 	if !repoRef.HasAuth() {
// 		return nil // No credentials provided, skip validation
// 	}

// 	repo, err := CreateAuthenticatedRepository(ctx, repoRef)
// 	if err != nil {
// 		return fmt.Errorf("failed to create repository client: %w", err)
// 	}

// 	return validateAuthentication(ctx, repoRef, repo)
// }

func validateAuthentication(ctx context.Context, repoRef *RepositoryRef, repo *remote.Repository) error {
	// Try to access the repository to validate credentials
	// This is a lightweight operation that should fail if credentials are invalid
	_, err := repo.Resolve(ctx, repoRef.String())
	if err != nil {
		// Check if this is an authentication error
		if isAuthenticationError(err) {
			return fmt.Errorf("authentication failed: invalid credentials for repository %s", repoRef.URL)
		}
		// If it's not an authentication error, the credentials might be valid
		// but there could be other issues (network, repository not found, etc.)
		return fmt.Errorf("failed to access repository %s: %w", repoRef.URL, err)
	}

	return nil
}

// isAuthenticationError checks if the error is related to authentication
func isAuthenticationError(err error) bool {
	// Check for common authentication error patterns
	errStr := strings.ToLower(err.Error())
	authErrorPatterns := []string{
		"401",
		"unauthorized",
		"authentication failed",
		"invalid credentials",
		"access denied",
		"403",
		"forbidden",
		"denied",
	}

	for _, pattern := range authErrorPatterns {
		if strings.Contains(errStr, strings.ToLower(pattern)) {
			return true
		}
	}
	return false
}
