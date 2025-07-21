> [!IMPORTANT]
> **Migrate Images from *gcr.io* to *ghcr.io*.**
>
> To reduce costs, we've migrated all our new and old Tekton releases to the free tier on [ghcr.io/tektoncd](https://github.com/orgs/tektoncd/packages?repo_name=pipeline). <br />
> Read more [here](https://tekton.dev/blog/2025/04/03/migration-to-github-container-registry/).

---

# ![pipe](./pipe.png) Tekton Pipelines

[![pre-commit](https://img.shields.io/badge/pre--commit-enabled-brightgreen?logo=pre-commit)](https://github.com/pre-commit/pre-commit)
[![Go Report Card](https://goreportcard.com/badge/tektoncd/pipeline)](https://goreportcard.com/report/tektoncd/pipeline)
[![CII Best Practices](https://bestpractices.coreinfrastructure.org/projects/4020/badge)](https://bestpractices.coreinfrastructure.org/projects/4020)
[![Nightly Release](https://img.shields.io/badge/Nightly-Release-blue?logo=github)](docs/nightly-releases.md)

The Tekton Pipelines project provides k8s-style resources for declaring
CI/CD-style pipelines.

## 🚀 Quick Start

### For Users
- [Installing Tekton Pipelines](docs/install.md)
- [Getting Started Tutorial](https://tekton.dev/docs/getting-started/tasks/)
- [Browse Examples](examples/)

### For Contributors & Fork Maintainers
- [Contributing Guide](CONTRIBUTING.md)
- [Development Setup](DEVELOPMENT.md)
- **[🌙 Nightly Releases Setup](docs/nightly-releases.md)** - Complete guide for fork maintainers

---

## 🌙 Nightly Releases

This repository supports automated nightly releases that build and publish development images for testing and early feedback. The nightly release system is designed to work with any fork of the Tekton Pipeline repository.

### 🚀 Quick Start for Fork Maintainers

Get started with nightly releases:

1. **Set up GitHub secrets**:
   - `GHCR_TOKEN`: GitHub Personal Access Token with `packages:write` scope
   - `GCS_SERVICE_ACCOUNT_KEY`: Google Cloud Service Account key for bucket access

2. **Enable the workflow** in your fork's Actions tab

3. **Test your setup**:
   ```bash
   # Manual test run
   gh workflow run "Tekton Nightly Release"
   
   # Check the run status
   gh run watch
   ```

### ✨ Features

- 🔄 **Automated builds** every night at 03:00 UTC
- 🏗️ **Multi-platform images** (amd64, arm64, s390x, ppc64le)
- 🔍 **Basic validation** and health checks
- 📦 **GitHub Container Registry** publishing
- 🔧 **Fork-aware** with automatic configuration

### 📚 Documentation

| Document | Purpose |
|----------|---------|
| **[Setup Guide](docs/nightly-releases.md)** | Complete setup and configuration guide |
| **[CI/CD Integration](docs/cicd-integration.md)** | Simple integration patterns for your workflow |

### 📊 What Gets Published

Nightly builds publish these container images to `ghcr.io/{your-username}/pipeline/`:

- `cmd/controller:nightly-YYYYMMDD` - Tekton Pipeline Controller
- `cmd/webhook:nightly-YYYYMMDD` - Admission Webhook  
- `cmd/events:nightly-YYYYMMDD` - Event Handler
- `cmd/resolvers:nightly-YYYYMMDD` - Bundle and Git Resolvers

All images are:
- ✅ Multi-platform (linux/amd64, linux/arm64, linux/s390x, linux/ppc64le)
- ✅ Tagged with date and commit SHA for traceability

### 📚 Related Resources

- **[Tekton Pipeline Documentation](https://tekton.dev/docs/pipelines/)**
- **[GitHub Container Registry](https://docs.github.com/en/packages/working-with-a-github-packages-registry/working-with-the-container-registry)**

For detailed setup instructions and troubleshooting, see **[docs/nightly-releases.md](docs/nightly-releases.md)**.

---

The Tekton Pipelines project provides k8s-style resources for declaring
CI/CD-style pipelines.

Tekton Pipelines are **Cloud Native**:

- Run on Kubernetes
- Have Kubernetes clusters as a first class type
- Use containers as their building blocks

Tekton Pipelines are **Decoupled**:

- One Pipeline can be used to deploy to any k8s cluster
- The Tasks which make up a Pipeline can easily be run in isolation
- Resources such as git repos can easily be swapped between runs

Tekton Pipelines are **Typed**:

- The concept of typed resources means that for a resource such as an `Image`,
  implementations can easily be swapped out (e.g. building with
  [kaniko](https://github.com/GoogleContainerTools/kaniko) v.s.
  [buildkit](https://github.com/moby/buildkit))

## Want to start using Pipelines

- [Installing Tekton Pipelines](docs/install.md)
- Jump in with [the "Getting started" tutorial!](https://tekton.dev/docs/getting-started/tasks/)
- Take a look at our [roadmap](roadmap.md)
- Discover our [releases](releases.md)

### Required Kubernetes Version

- Starting from the v0.24.x release of Tekton: **Kubernetes version 1.18 or later**
- Starting from the v0.27.x release of Tekton: **Kubernetes version 1.19 or later**
- Starting from the v0.30.x release of Tekton: **Kubernetes version 1.20 or later**
- Starting from the v0.33.x release of Tekton: **Kubernetes version 1.21 or later**
- Starting from the v0.39.x release of Tekton: **Kubernetes version 1.22 or later**
- Starting from the v0.41.x release of Tekton: **Kubernetes version 1.23 or later**
- Starting from the v0.45.x release of Tekton: **Kubernetes version 1.24 or later**
- Starting from the v0.51.x release of Tekton: **Kubernetes version 1.25 or later**
- Starting from the v0.59.x release of Tekton: **Kubernetes version 1.27 or later**
- Starting from the v0.61.x release of Tekton: **Kubernetes version 1.28 or later**

### Read the docs

The latest version of our docs is available at:

- [Installation Guide @ HEAD](DEVELOPMENT.md#install-pipeline)
- [Docs @ HEAD](/docs/README.md)
- [Examples @ HEAD](/examples)

Version specific links are available in the [releases](releases.md) page and on the
[Tekton website](https://tekton.dev/docs).

_See [our API compatibility policy](api_compatibility_policy.md) for info on the
stability level of the API._

_See [our Deprecations table](docs/deprecations.md) for features that have been
deprecated and the earliest date they'll be removed._

## Migrating

### v1beta1 to v1

Several Tekton CRDs and API spec fields, including ClusterTask CRD and Pipeline
Resources fields, were updated or deprecated during the migration from `v1beta1`
to `v1`.

For users migrating their Tasks and Pipelines from `v1beta1` to `v1`, check
out [the v1beta1 to v1 migration guide](./docs/migrating-v1beta1-to-v1.md).

### v1alpha1 to v1beta1

In the move from v1alpha1 to v1beta1 several spec fields and Tekton
CRDs were updated or removed .

For users migrating their Tasks and Pipelines from v1alpha1 to v1beta1, check
out [the spec changes and migration paths](./docs/migrating-v1alpha1-to-v1beta1.md).

## Want to contribute

We are so excited to have you!

- See [CONTRIBUTING.md](CONTRIBUTING.md) for an overview of our processes
- See [DEVELOPMENT.md](DEVELOPMENT.md) for how to get started
- [Deep dive](./docs/developers/README.md) into demystifying the inner workings
  (advanced reading material)
- Look at our
  [good first issues](https://github.com/tektoncd/pipeline/issues?q=is%3Aissue+is%3Aopen+label%3A%22good+first+issue%22)
  and our
  [help wanted issues](https://github.com/tektoncd/pipeline/issues?q=is%3Aissue+is%3Aopen+label%3A%22help+wanted%22)
