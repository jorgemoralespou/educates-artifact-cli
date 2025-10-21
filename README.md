# Artifact CLI

A command-line interface to package local folders, push them as OCI artifacts to a registry, pull them back down, and sync multiple artifacts based on configuration with support for multiple artifact formats.

## Features

- **Multi-Format Support**: Push and pull artifacts in different formats (OCI, imgpkg, educates)
- **Cross-Platform**: Build and push multi-platform artifacts
- **Compatibility**: Works with existing OCI registries and tools
- **Flexible Pulling**: Smart fallback strategies for pulling artifacts from different sources
- **Artifact Identification**: Automatically identifies and handles different artifact types
- **Batch Synchronization**: Sync multiple artifacts with file filtering using configuration files

## Installation

### Prerequisites

- Go 1.25.0 or later
- Task (for building) - [Install Task](https://taskfile.dev/installation/)

### Build from Source

```bash
git clone <repository-url>
cd educates-artifact-cli
task build
```

The binary will be created in `./bin/artifact-cli`.

## Usage

### Push Command

Package and push a folder to an OCI registry:

```bash
# Push a single artifact (no platform selector)
artifact-cli push ghcr.io/my-user/my-app:1.0.0 -f ./app-folder

# Push a multi-platform artifact
artifact-cli push ghcr.io/my-user/my-app:1.0.1 -f ./app-folder -p linux/amd64,linux/arm64

# Push with specific artifact type
artifact-cli push ghcr.io/my-user/my-app:1.0.0 -f ./app-folder -a imgpkg
```

#### Push Options

- `-f, --folder`: Path to the folder to package and push (required)
- `-p, --platforms`: Comma-separated list of platforms (e.g., 'linux/amd64,linux/arm64')
- `-a, --as`: Type of artifact to push (oci, imgpkg, educates). Defaults to oci

### Pull Command

Pull and extract an OCI artifact folder:

```bash
# Pull using fallback strategies (recommended)
artifact-cli pull ghcr.io/my-user/my-app:1.0.1 -o ./restored-app

# Pull a specific platform
artifact-cli pull ghcr.io/my-user/my-app:1.0.1 -o ./restored-app -p linux/amd64

# Pull with specific artifact type
artifact-cli pull ghcr.io/my-user/my-app:1.0.1 -o ./restored-app -a imgpkg
```

#### Pull Options

- `-o, --output`: Path to the target directory for extraction (required)
- `-p, --platform`: Target platform (e.g., 'linux/amd64'). If not specified, uses fallback strategies
- `-a, --as`: Type of artifact to pull (oci, imgpkg, educates). Defaults to oci

### Sync Command

Sync multiple artifacts from OCI registries to local folders based on a configuration file:

```bash
# Sync artifacts using configuration file
artifact-cli sync -c config.yaml
```

#### Sync Configuration

Create a `config.yaml` file to define which artifacts to sync:

```yaml
spec:
  # Target destination for pulled down artifacts
  dest: ./workshops
  
  # List of artifacts to pull
  artifacts:
    - image:
        # OCI repository where the image is located
        url: ghcr.io/my-org/workshop-files:v1.0.0
      # List of files (and file patterns) within the OCI artifact to extract (include)
      includePaths:
        - /workshop/**
        - /exercises/**
        - /resources/**
      # List of files (and file patterns) within the OCI artifact to not extract (exclude)
      excludePaths:
        - /README.md
        - /docs/**
        - /tests/**
    
    - image:
        url: ghcr.io/my-org/advanced-workshop:v2.1.0
      includePaths:
        - /content/**
        - /labs/**
      excludePaths:
        - /temp/**
        - /cache/**
```

#### Sync Options

- `-c, --config`: Path to the configuration YAML file (required)

#### Sync Features

- **Multiple Artifacts**: Sync multiple artifacts in a single command
- **File Filtering**: Use include/exclude patterns to control which files are extracted
- **Pattern Matching**: Support for glob patterns (`**` for recursive matching)
- **Fallback Strategies**: Automatically tries different artifact formats (OCI, imgpkg, educates)
- **Progress Tracking**: Shows progress for each artifact being processed

## Verbosity Control

The CLI supports a simple verbosity system to control output:

### Verbose Flag

Use the `-v` or `--verbose` flag to enable detailed output:

```bash
# Quiet mode (default) - only shows errors and final results
artifact-cli push ghcr.io/my-user/my-app:1.0.0 -f ./app-folder

# Verbose mode - shows progress and detailed information
artifact-cli push ghcr.io/my-user/my-app:1.0.0 -f ./app-folder -v

# Verbose sync
artifact-cli sync -c config.yaml -v

# Verbose pull
artifact-cli pull ghcr.io/my-user/my-app:1.0.1 -o ./restored-app -v
```

### Output Behavior

- **Default (no `-v`)**: Only error messages and final results are shown
- **Verbose (`-v`)**: Progress messages, validation warnings, and detailed status updates are displayed
- **Error messages**: Always shown regardless of verbosity level
- **Final results**: Always shown (e.g., "Successfully synced X artifacts")

## Artifact Types

### OCI Format (Default)

- **Manifest**: `application/vnd.oci.image.manifest.v1+json`
- **Config**: `application/vnd.oci.image.config.v1+json`
- **Layers**: `application/vnd.oci.image.layer.v1.tar+gzip`
- **Index**: `application/vnd.oci.image.index.v1+json` (for multi-platform)

### Imgpkg Format

- **Manifest**: `application/vnd.docker.distribution.manifest.v2+json`
- **Config**: `application/vnd.docker.container.image.v1+json`
- **Layers**: `application/vnd.docker.image.rootfs.diff.tar.gzip`
- **Note**: Imgpkg artifacts don't support multi-platform (platforms are ignored)

### Educates Format

- **Custom format**: Uses educates-specific media types and structure
- **Purpose**: Optimized for educates learning environments

## Pull Fallback Strategies

When no platform is specified, the pull command uses the following strategies:

1. **Strategy 1**: Try to pull an image generated with `artifact-cli push` (no platform selector)
2. **Strategy 2**: Try to pull an image generated via `imgpkg` (Docker manifest format)
3. **Strategy 3**: Try to pull an image generated with `docker buildx` using current architecture

## Artifact Identification

artifact-cli generated artifacts include specific annotations for identification:

```json
{
  "org.opencontainers.image.title": "artifact-cli artifact",
  "org.opencontainers.image.description": "Folder artifact created by artifact-cli",
  "dev.educates.artifact-cli.version": "1.0.0",
  "dev.educates.artifact-cli.tool": "artifact-cli"
}
```

## Development

### Project Structure

```
.
├── cmd/                    # Main application entry point
│   └── root.go
├── pkg/
│   ├── cmd/               # Command implementations
│   │   ├── push.go
│   │   └── pull.go
│   ├── artifact/          # Artifact type implementations
│   │   ├── oci/
│   │   ├── imgpkg/
│   │   └── educates/
│   └── utils/             # Utility functions
├── specs/                 # Project specifications
└── Taskfile.yml          # Build automation
```

### Available Tasks

```bash
# Build the binary
task build

# Run in development mode
task dev

# Cross-compile for all platforms
task build-all

# Run tests
task test

# Run linter
task lint

# Run all checks
task check

# Clean build artifacts
task clean

# Show all available tasks
task --list
```

### Example Workflows

```bash
# Push examples
task example-push                    # Push without platform selector
task example-push-multi-platform    # Push multi-platform artifact

# Pull examples
task example-pull                   # Pull with fallback strategies
task example-pull-platform          # Pull specific platform
```

## Compatibility

### Supported Tools

- **Carvel imgpkg**: Full compatibility with imgpkg generated artifacts
- **Docker buildx**: Compatible with docker buildx multi-platform images
- **Standard OCI**: Works with any OCI-compliant registry

### Registry Support

- GitHub Container Registry (ghcr.io)
- Docker Hub
- Amazon ECR
- Google Container Registry
- Any OCI-compliant registry

## Examples

### Basic Workflow (Defaults to OCI)

```bash
# 1. Package and push a folder
artifact-cli push ghcr.io/my-user/my-app:1.0.0 -f ./my-app

# 2. Pull it back
artifact-cli pull ghcr.io/my-user/my-app:1.0.0 -o ./restored-app
```

### Multi-Platform Workflow (Defaults to OCI)

```bash
# 1. Push for multiple platforms
artifact-cli push ghcr.io/my-user/my-app:1.0.0 -f ./my-app -p linux/amd64,linux/arm64,darwin/amd64

# 2. Pull for specific platform
artifact-cli pull ghcr.io/my-user/my-app:1.0.0 -o ./restored-app -p linux/amd64
```

### Imgpkg Compatibility

```bash
# Push in imgpkg format
artifact-cli push ghcr.io/my-user/my-app:1.0.0 -f ./my-app -a imgpkg

# Pull imgpkg artifact
artifact-cli pull ghcr.io/my-user/my-app:1.0.0 -o ./restored-app -a imgpkg
```

## Installation

### Using the Install Script (Recommended)

```bash
curl -fsSL https://raw.githubusercontent.com/jorgemoralespou/educates-artifact-cli/main/scripts/install.sh | bash
```

### Manual Installation

1. Download the latest release from the [releases page](https://github.com/jorgemoralespou/educates-artifact-cli/releases)
2. Extract the archive for your platform
3. Move the binary to your PATH

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run tests: `task test`
5. Run linter: `task lint`
6. Submit a pull request

## Release Process

The project uses GoReleaser for automated releases. To create a new release:

1. Update the version in the code
2. Update CHANGELOG.md
3. Create a git tag: `git tag v1.0.0`
4. Push the tag: `git push origin v1.0.0`
5. GitHub Actions will automatically build and publish the release

For manual releases, use the release script:
```bash
task release
```

To test the release process locally:
```bash
task release-dry-run
```

## License

[Add your license information here]

## Support

For issues and questions:
- Create an issue in the repository
- Check the [specifications](specs/specs.md) for detailed requirements

## TODO

- [*] Support secure/authenticated repositories
- [ ] Support .ignorefile for push, so that some files are not added to the OCI image
