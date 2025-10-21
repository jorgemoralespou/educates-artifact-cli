# Docker Usage Guide

This document explains how to use Docker with the artifact-cli project.

## Quick Start

### Build and Run

```bash
# Build the Docker image
docker build -t artifact-cli .

# Run the CLI
docker run --rm artifact-cli --help

# Run with volume mounts for file operations
docker run --rm -v $(pwd)/test:/workspace/test -v $(pwd)/output:/workspace/output artifact-cli push registry:5000/test:latest -f /workspace/test/my-app
```

### Using Docker Compose

```bash
# Start registry and CLI services
docker-compose up -d registry

# Run CLI commands
docker-compose run --rm artifact-cli push registry:5000/test:latest -f /workspace/test/my-app

# Development mode with live reload
docker-compose --profile dev up artifact-cli-dev
```

## Available Dockerfiles

### 1. `Dockerfile` (Production)
- **Purpose**: Minimal production image
- **Size**: ~15MB
- **Features**: 
  - Multi-stage build
  - Static binary
  - Non-root user
  - Health check
  - Security hardened

### 2. `Dockerfile.multiarch` (Multi-Platform)
- **Purpose**: Build for multiple architectures
- **Platforms**: linux/amd64, linux/arm64
- **Usage**: 
  ```bash
  docker buildx build --platform linux/amd64,linux/arm64 -t artifact-cli:latest .
  ```

### 3. `Dockerfile.dev` (Development)
- **Purpose**: Development with live reload
- **Features**:
  - Go development tools
  - Air for live reloading
  - Development dependencies

## Docker Compose Services

### Registry Service
- **Image**: `registry:2`
- **Port**: `5000:5000`
- **Purpose**: Local OCI registry for testing

### CLI Service
- **Profile**: `cli`
- **Purpose**: Run artifact-cli commands
- **Volumes**: Test data and output directories

### Development Service
- **Profile**: `dev`
- **Purpose**: Development with live reload
- **Features**: Source code mounted for live editing

## Examples

### Push Artifacts
```bash
# Using Docker directly
docker run --rm \
  -v $(pwd)/test:/workspace/test \
  -v $(pwd)/output:/workspace/output \
  artifact-cli push registry:5000/my-app:latest -f /workspace/test/my-app

# Using Docker Compose
docker-compose run --rm artifact-cli push registry:5000/my-app:latest -f /workspace/test/my-app
```

### Pull Artifacts
```bash
# Using Docker directly
docker run --rm \
  -v $(pwd)/output:/workspace/output \
  artifact-cli pull registry:5000/my-app:latest -o /workspace/output

# Using Docker Compose
docker-compose run --rm artifact-cli pull registry:5000/my-app:latest -o /workspace/output
```

### Sync Artifacts
```bash
# Using Docker Compose
docker-compose run --rm artifact-cli sync -c /workspace/sync-config.yaml
```

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `ARTIFACT_CLI_USERNAME` | Registry username | - |
| `ARTIFACT_CLI_PASSWORD` | Registry password | - |

## Volume Mounts

| Host Path | Container Path | Purpose |
|-----------|----------------|---------|
| `./test` | `/workspace/test` | Test data (read-only) |
| `./test-output` | `/workspace/output` | Output directory |
| `./sync-config.yaml` | `/workspace/sync-config.yaml` | Sync configuration |

## Security Features

- **Non-root user**: Runs as `artifact` user (UID 1001)
- **Minimal base image**: Alpine Linux
- **Static binary**: No external dependencies
- **Health check**: Built-in health monitoring
- **Read-only filesystem**: Where possible

## Troubleshooting

### Permission Issues
```bash
# Fix ownership of output directory
sudo chown -R 1001:1001 ./test-output
```

### Registry Connection Issues
```bash
# Check registry health
docker-compose ps registry

# View registry logs
docker-compose logs registry
```

### Build Issues
```bash
# Clean build
docker build --no-cache -t artifact-cli .

# Multi-platform build
docker buildx build --platform linux/amd64,linux/arm64 -t artifact-cli:latest .
```
