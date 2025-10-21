package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"educates-artifact-cli/pkg/artifact"
	"educates-artifact-cli/pkg/utils"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type ManifestCmdOpts struct {
	ImageRef     string
	Username     string
	Password     string
	Insecure     bool
	Timeout      string
	OutputFormat string
	// ArtifactType ArtifactType
}

const (
	OutputFormatTable = "table"
	OutputFormatJSON  = "json"
	OutputFormatYAML  = "yaml"
)

// NewManifestCmd creates the 'manifest' command
func NewManifestCmd() *cobra.Command {
	var opts ManifestCmdOpts
	// opts.ArtifactType = DefaultArtifactType
	opts.OutputFormat = OutputFormatTable

	cmd := &cobra.Command{
		Use:   "describe <repository>",
		Short: "Manifest an OCI artifact manifest",
		Long:  `Manifest provides human-readable information about an OCI artifact, including platform support, artifact type, and layer information.`,
		Example: `  # Manifest an artifact (auto-detect type)
  artifact-cli describe ghcr.io/my-user/my-app:1.0.0

  # Manifest with specific artifact type
  artifact-cli describe ghcr.io/my-user/my-app:1.0.0 --as oci

  # Output in JSON format
  artifact-cli describe ghcr.io/my-user/my-app:1.0.0 --output json

  # Output in YAML format
  artifact-cli describe ghcr.io/my-user/my-app:1.0.0 --output yaml

  # With authentication
  artifact-cli describe ghcr.io/my-user/my-app:1.0.0 --username myuser --password mypass`,

		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.ImageRef = args[0]
			return runManifest(opts)
		},
	}

	// cmd.Flags().Var(&opts.ArtifactType, "as", "Type of artifact to describe (oci, imgpkg, educates). Auto-detected if not specified")
	cmd.Flags().StringVarP(&opts.OutputFormat, "output", "o", OutputFormatTable, "Output format (table, json, yaml)")
	cmd.Flags().StringVarP(&opts.Timeout, "timeout", "t", "", "Timeout for the operation (e.g., '30s', '5m', '1h'). Defaults to 5m")
	cmd.Flags().StringVarP(&opts.Username, "username", "u", "", "Username for registry authentication (can also use ARTIFACT_CLI_USERNAME env var)")
	cmd.Flags().StringVarP(&opts.Password, "password", "w", "", "Password or token for registry authentication (can also use ARTIFACT_CLI_PASSWORD env var)")
	cmd.Flags().BoolVarP(&opts.Insecure, "insecure", "i", false, "Allow insecure registry communication")

	return cmd
}

func runManifest(opts ManifestCmdOpts) error {
	// Create cancellable context with signal handling
	ctx, cancel, err := utils.ContextWithSignalHandling(opts.Timeout)
	if err != nil {
		return fmt.Errorf("invalid timeout: %w", err)
	}
	defer cancel()

	repoRef := artifact.NewRepositoryRef(opts.ImageRef, opts.Username, opts.Password, opts.Insecure)

	repo, err := repoRef.Authenticate(ctx)
	if err != nil {
		return fmt.Errorf("failed to authenticate repository: %w", err)
	}

	imageMetadata := artifact.ImageMetadata{
		ImageRef: opts.ImageRef,
	}
	err = artifact.GetImageMetadata(ctx, repo, &imageMetadata)
	if err != nil {
		return fmt.Errorf("failed to pull image index: %w", err)
	}

	// Output the information in the requested format
	switch opts.OutputFormat {
	case OutputFormatJSON:
		return outputJSON(imageMetadata)
	case OutputFormatYAML:
		return outputYAML(imageMetadata)
	case OutputFormatTable:
		return outputTable(imageMetadata)
	default:
		return outputTable(imageMetadata)
	}
}

func outputJSON(v any) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(v)
}

func outputYAML(v any) error {
	data, err := yaml.Marshal(v)
	if err != nil {
		return fmt.Errorf("failed to marshal to YAML: %w", err)
	}
	fmt.Print(string(data))
	return nil
}

func outputTable(imageMetadata artifact.ImageMetadata) error {
	fmt.Printf("Image Ref: %s\n", imageMetadata.ImageRef)
	fmt.Printf("Media Type: %s\n", imageMetadata.MediaType)
	fmt.Printf("OCI Compliant: %t\n", imageMetadata.OciCompliant)
	fmt.Printf("Multi-Platform: %t\n", imageMetadata.MultiPlatform)

	if len(imageMetadata.Platforms) > 0 {
		fmt.Printf("Platforms:\n")
		for _, platform := range imageMetadata.Platforms {
			fmt.Printf("  - %s (%s)\n", platform.Architecture, platform.OS)
		}
	}

	// // Show relevant annotations
	// if len(manifestInfo.Annotations) > 0 {
	// 	fmt.Printf("Annotations:\n")
	// 	for key, value := range manifestInfo.Annotations {
	// 		if strings.HasPrefix(key, "dev.educates.artifact-cli.") ||
	// 			strings.HasPrefix(key, "org.opencontainers.image.") {
	// 			fmt.Printf("  %s: %s\n", key, value)
	// 		}
	// 	}
	// }

	return nil
}
