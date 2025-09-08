package artifact

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
)

// CreateAuthenticatedRepository creates a registry client with optional authentication
func CreateAuthenticatedRepository(ctx context.Context, repoRef *RepositoryRef) (*remote.Repository, error) {
	repo, err := remote.NewRepository(repoRef.URL)
	if err != nil {
		return nil, err
	}

	// Set up authentication if credentials are provided
	if repoRef.HasAuth() {
		cred := auth.Credential{
			Username: repoRef.Username,
			Password: repoRef.Password,
		}

		// Extract registry host from the repository URL
		registryHost := extractRegistryHost(repoRef.URL)

		authClient := &auth.Client{
			Client:     nil, // Use default client
			Cache:      auth.NewCache(),
			Credential: auth.StaticCredential(registryHost, cred),
		}
		repo.Client = authClient
	}

	// Use plain HTTP if the registry is insecure (for development/testing)
	repo.PlainHTTP = true

	err = validateAuthentication(ctx, repoRef, repo)
	if err != nil {
		return nil, err
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
func ValidateAuthentication(ctx context.Context, repoRef *RepositoryRef) error {
	if !repoRef.HasAuth() {
		return nil // No credentials provided, skip validation
	}

	repo, err := CreateAuthenticatedRepository(ctx, repoRef)
	if err != nil {
		return fmt.Errorf("failed to create repository client: %w", err)
	}

	return validateAuthentication(ctx, repoRef, repo)
}

func validateAuthentication(ctx context.Context, repoRef *RepositoryRef, repo *remote.Repository) error {
	if !repoRef.HasAuth() {
		return nil // No credentials provided, skip validation
	}

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
