# Reusing the OpenCHAMI Release Workflow

This document explains how to use the centralized GitHub Actions workflow from the OpenCHAMI cloud-init repository in your own projects.

## Overview

The OpenCHAMI cloud-init project uses a centralized, reusable GitHub Actions workflow that handles:
- Go application building with GoReleaser
- Multi-architecture builds (amd64, arm64) 
- Container image creation and publishing
- Release artifact generation
- Build attestation and security scanning

The workflow is centralized in the [OpenCHAMI/github-actions](https://github.com/OpenCHAMI/github-actions) repository and can be reused by any Go project.

## Quick Start

To use this workflow in your own repository, create a `.github/workflows/Release.yml` file:

```yaml
name: Release with goreleaser

on:
  workflow_dispatch:
  push:
    tags:
      - v*

permissions: write-all # Necessary for the generate-build-provenance action with containers

jobs:
  release:
    uses: OpenCHAMI/github-actions/workflows/go-build-release.yml@v2
    with:
      cgo-enabled: "1"
      pre-build-commands: |
        go install github.com/swaggo/swag/cmd/swag@latest
      attestation-binary-path: "dist/cloud-init*"
      registry-name: ghcr.io/openchami/cloud-init
```

> 💡 **Tip**: You can also copy the [example workflow file](example-release-workflow.yml) and customize it for your project.

## Configuration Options

The reusable workflow accepts several input parameters that you can customize:

### Required Inputs

- `registry-name`: The container registry where images will be pushed (e.g., `ghcr.io/yourusername/yourproject`)

### Optional Inputs

- `cgo-enabled`: Enable or disable CGO compilation (default: "0", set to "1" if needed)
- `pre-build-commands`: Commands to run before building (e.g., code generation, dependency installation)
- `attestation-binary-path`: Glob pattern for binaries to create attestations for
- `go-version`: Go version to use (defaults to latest stable)
- `goreleaser-version`: GoReleaser version to use (defaults to latest)

## Usage Examples

### Basic Go Application

For a simple Go application without special requirements:

```yaml
name: Release

on:
  push:
    tags:
      - v*

permissions: write-all

jobs:
  release:
    uses: OpenCHAMI/github-actions/workflows/go-build-release.yml@v2
    with:
      registry-name: ghcr.io/myorg/myapp
```

### Application with Code Generation

For applications that need code generation (like Swagger docs):

```yaml
name: Release

on:
  push:
    tags:
      - v*

permissions: write-all

jobs:
  release:
    uses: OpenCHAMI/github-actions/workflows/go-build-release.yml@v2
    with:
      cgo-enabled: "0"
      pre-build-commands: |
        go install github.com/swaggo/swag/cmd/swag@latest
        swag init -g cmd/myapp/main.go
      registry-name: ghcr.io/myorg/myapp
      attestation-binary-path: "dist/myapp*"
```

### Application with CGO Dependencies

For applications that require CGO (C bindings):

```yaml
name: Release

on:
  push:
    tags:
      - v*

permissions: write-all

jobs:
  release:
    uses: OpenCHAMI/github-actions/workflows/go-build-release.yml@v2
    with:
      cgo-enabled: "1"
      registry-name: ghcr.io/myorg/myapp
```

## Prerequisites

For the workflow to work properly in your repository, you need:

### 1. GoReleaser Configuration

Create a `.goreleaser.yaml` file in your repository root. You can use the [cloud-init example](./.goreleaser.yaml) as a template:

```yaml
version: 2.4
project_name: your-project-name

before:
  hooks:
    - go mod tidy
    # Add any additional pre-build commands here

builds:
  - id: your-app
    main: ./cmd/your-app
    binary: your-app
    ldflags:
      - "-X 'main.Version={{.Version}}' -X 'main.GitCommit={{.Commit}}'"
    goos:
      - linux
      - darwin
    goarch:
      - amd64
      - arm64
    env:
      - CGO_ENABLED=0  # or 1 if you need CGO

# Add docker, archives, and other configurations as needed
```

### 2. Repository Permissions

Ensure your repository has the necessary permissions:
- **Actions**: Enabled for running workflows
- **Packages**: Write permissions for container registry pushes
- **Contents**: Write permissions for creating releases

### 3. Container Registry Setup

If publishing to GitHub Container Registry (ghcr.io):
1. Enable the Package feature in your repository settings
2. Configure package visibility (public/private)
3. The workflow will automatically authenticate using the `GITHUB_TOKEN`

## Environment Variables

The workflow automatically sets these environment variables for GoReleaser:

- `GIT_STATE`: "clean" or "dirty" based on repository state
- `BUILD_HOST`: Hostname of the build machine
- `GO_VERSION`: Go version being used
- `BUILD_USER`: Username performing the build

You can reference these in your `.goreleaser.yaml` file:

```yaml
builds:
  - ldflags:
      - "-X 'main.GitState={{ .Env.GIT_STATE }}'"
      - "-X 'main.BuildHost={{ .Env.BUILD_HOST }}'"
      - "-X 'main.GoVersion={{ .Env.GO_VERSION }}'"
      - "-X 'main.BuildUser={{ .Env.BUILD_USER }}'"
```

## Triggering Releases

The workflow is typically triggered by:

1. **Tag pushes**: When you push a tag starting with 'v'
   ```bash
   git tag v1.0.0
   git push origin v1.0.0
   ```

2. **Manual dispatch**: Using the GitHub UI or CLI
   ```bash
   gh workflow run Release.yml
   ```

## Troubleshooting

### Common Issues

1. **GoReleaser version mismatch**: Ensure your local GoReleaser version matches the one used in CI
2. **CGO errors**: Set `cgo-enabled: "1"` if your application uses C dependencies
3. **Permission errors**: Verify repository has write permissions for packages and contents
4. **Registry authentication**: Ensure container registry name matches your organization/username

### Debugging

To debug locally:

1. Install GoReleaser locally
2. Set the required environment variables:
   ```bash
   export GIT_STATE=$(if git diff-index --quiet HEAD --; then echo 'clean'; else echo 'dirty'; fi)
   export BUILD_HOST=$(hostname)
   export GO_VERSION=$(go version | awk '{print $3}')
   export BUILD_USER=$(whoami)
   ```
3. Run in snapshot mode:
   ```bash
   goreleaser release --snapshot --clean
   ```

## Security Considerations

The workflow includes security features:
- **Build attestation**: Creates signed attestations for build artifacts
- **Container scanning**: Automatically scans container images for vulnerabilities
- **Secure defaults**: Uses minimal permissions and secure build practices

## More Information

- [OpenCHAMI GitHub Actions Repository](https://github.com/OpenCHAMI/github-actions)
- [GoReleaser Documentation](https://goreleaser.com/)
- [GitHub Actions Reusable Workflows](https://docs.github.com/en/actions/using-workflows/reusing-workflows)