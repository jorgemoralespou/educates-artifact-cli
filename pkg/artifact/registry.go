package artifact

import (
	"context"
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
