# CI/CD Integration Guide for Tekton Nightly Releases

This guide covers integrating nightly releases into your existing CI/CD pipeline and development workflow.

## Table of Contents

- [Overview](#overview)
- [Integration Patterns](#integration-patterns)
- [Development Workflow](#development-workflow)
- [Quality Gates](#quality-gates)
- [Monitoring Integration](#monitoring-integration)
- [Security Integration](#security-integration)
- [Advanced Scenarios](#advanced-scenarios)

## Overview

The Tekton nightly release system is designed to integrate seamlessly with your existing CI/CD pipeline while providing:

- **Independent Operation**: Runs on its own schedule without interfering with regular CI/CD
- **Quality Assurance**: Built-in testing and validation before publishing
- **Observability**: Comprehensive monitoring and alerting capabilities
- **Security**: Secure container publishing and vulnerability scanning

## Integration Patterns

### Pattern 1: Parallel Pipeline

The nightly release runs independently alongside your main CI/CD pipeline:

```
Main Branch Development:
    ├── Feature PR → Tests → Merge
    ├── Release PR → Tests → Tag → Publish Release
    └── [Independent] Nightly Build → Test → Publish Nightly

Nightly Pipeline:
    ├── Scheduled Trigger (03:00 UTC)
    ├── Pre-flight Checks
    ├── Build Multi-platform Images
    ├── Security Scanning
    ├── Integration Testing
    └── Publish to GHCR
```

### Pattern 2: Integrated Quality Gates

Use nightly builds as additional quality gates:

```yaml
# Example: Use nightly images in downstream testing
name: Integration Tests
on:
  schedule:
    - cron: '0 6 * * *'  # Run after nightly build

jobs:
  test-with-nightly:
    runs-on: ubuntu-latest
    steps:
      - name: Test with Nightly Images
        run: |
          # Use latest nightly images for testing
          docker run ghcr.io/${{ github.repository_owner }}/pipeline/cmd/controller:nightly-$(date +%Y%m%d)
```

### Pattern 3: Environment Promotion

Use nightly releases for environment promotion:

```yaml
# Example: Auto-deploy nightly to staging
name: Deploy Nightly to Staging
on:
  workflow_run:
    workflows: ["Tekton Nightly Release"]
    types: [completed]

jobs:
  deploy-staging:
    if: ${{ github.event.workflow_run.conclusion == 'success' }}
    runs-on: ubuntu-latest
    steps:
      - name: Deploy to Staging
        run: |
          # Deploy nightly images to staging environment
          kubectl set image deployment/tekton-pipelines-controller \
            tekton-pipelines-controller=ghcr.io/${{ github.repository_owner }}/pipeline/cmd/controller:nightly-$(date +%Y%m%d)
```

## Development Workflow

### 1. Feature Development

When developing new features, consider nightly builds:

```bash
# Check if your feature breaks nightly builds
git checkout feature-branch
./scripts/quick-setup.sh --validate

# Monitor nightly builds after merging
git checkout main
git merge feature-branch
# Wait for next nightly build to verify
```

### 2. Release Preparation

Use nightly builds to validate release candidates:

```bash
# Test release candidate against nightly
export NIGHTLY_TAG="nightly-$(date +%Y%m%d)"
export RC_TAG="v0.50.0-rc.1"

# Compare performance and functionality
./scripts/test-nightly-release.sh --compare-tags "$NIGHTLY_TAG,$RC_TAG"
```

### 3. Hotfix Process

Nightly builds can help validate hotfixes:

```yaml
# .github/workflows/hotfix-validation.yaml
name: Hotfix Validation
on:
  push:
    branches: [hotfix/*]

jobs:
  validate-hotfix:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Trigger Test Build
        run: |
          # Trigger a test build similar to nightly
          gh workflow run nightly-release.yaml \
            --ref ${{ github.ref }} \
            -f dry_run=true
```

## Quality Gates

### 1. Pre-release Validation

Integrate nightly build status into your release process:

```yaml
# Release workflow with nightly validation
name: Release
on:
  push:
    tags: ['v*']

jobs:
  check-nightly-health:
    runs-on: ubuntu-latest
    steps:
      - name: Check Recent Nightly Status
        run: |
          # Verify recent nightly builds are successful
          RECENT_RUNS=$(gh run list \
            --workflow="nightly-release.yaml" \
            --limit=3 \
            --json=status,conclusion)
          
          FAILED_RUNS=$(echo "$RECENT_RUNS" | jq '[.[] | select(.conclusion != "success")] | length')
          
          if [ "$FAILED_RUNS" -gt 1 ]; then
            echo "❌ Multiple recent nightly builds failed"
            echo "Please investigate before releasing"
            exit 1
          fi

  release:
    needs: check-nightly-health
    # ... rest of release job
```

### 2. Automated Testing Integration

Use nightly images in your test suites:

```yaml
# test/e2e-nightly.yaml
name: E2E Tests with Nightly
on:
  schedule:
    - cron: '0 8 * * *'  # Run after nightly build

jobs:
  e2e-nightly:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        k8s-version: [1.26, 1.27, 1.28, 1.29]
    steps:
      - uses: actions/checkout@v4
      
      - name: Setup Test Cluster
        run: |
          kind create cluster --image kindest/node:v${{ matrix.k8s-version }}.0
      
      - name: Install Nightly Tekton
        run: |
          # Install using nightly images
          DATE=$(date +%Y%m%d)
          kubectl apply -f https://raw.githubusercontent.com/${{ github.repository }}/nightly-pipeline-gha/releases/nightly-$DATE/release.yaml
      
      - name: Run E2E Tests
        run: |
          go test ./test/e2e/... -tags=e2e
```

### 3. Performance Benchmarking

Track performance trends using nightly builds:

```yaml
# .github/workflows/nightly-benchmarks.yaml
name: Nightly Performance Benchmarks
on:
  workflow_run:
    workflows: ["Tekton Nightly Release"]
    types: [completed]

jobs:
  benchmark:
    if: ${{ github.event.workflow_run.conclusion == 'success' }}
    runs-on: ubuntu-latest
    steps:
      - name: Run Benchmarks
        run: |
          # Run performance benchmarks with nightly images
          DATE=$(date +%Y%m%d)
          docker run --rm \
            -v $(pwd)/benchmarks:/benchmarks \
            ghcr.io/${{ github.repository_owner }}/pipeline/cmd/controller:nightly-$DATE \
            /benchmarks/run-benchmarks.sh
      
      - name: Upload Results
        uses: actions/upload-artifact@v4
        with:
          name: benchmark-results-${{ github.sha }}
          path: benchmarks/results/
```

## Monitoring Integration

### 1. Slack/Teams Notifications

Get notified about nightly build status:

```yaml
# Add to your nightly-release.yaml workflow
  notify:
    runs-on: ubuntu-latest
    needs: [publish-images]
    if: always()
    steps:
      - name: Notify Teams
        if: failure()
        uses: 8398a7/action-slack@v3
        with:
          status: failure
          channel: '#tekton-ci'
          webhook_url: ${{ secrets.SLACK_WEBHOOK }}
          message: |
            🌙 Nightly build failed for ${{ github.repository }}
            Workflow: ${{ github.workflow }}
            Commit: ${{ github.sha }}
            View: ${{ github.server_url }}/${{ github.repository }}/actions/runs/${{ github.run_id }}
```

### 2. Metrics Collection

Track nightly build metrics:

```yaml
  collect-metrics:
    runs-on: ubuntu-latest
    needs: [publish-images]
    if: always()
    steps:
      - name: Send Metrics
        run: |
          # Send metrics to your monitoring system
          curl -X POST "${{ secrets.METRICS_ENDPOINT }}" \
            -H "Content-Type: application/json" \
            -d '{
              "metric": "tekton.nightly.build",
              "value": ${{ needs.publish-images.result == 'success' && 1 || 0 }},
              "tags": {
                "repository": "${{ github.repository }}",
                "branch": "${{ github.ref_name }}",
                "workflow_id": "${{ github.run_id }}"
              }
            }'
```

### 3. Health Dashboards

Create dashboards showing nightly build health:

```json
{
  "dashboard": {
    "title": "Tekton Nightly Builds",
    "panels": [
      {
        "title": "Success Rate (30 days)",
        "type": "singlestat",
        "targets": [
          {
            "expr": "rate(tekton_nightly_build_success_total[30d]) / rate(tekton_nightly_build_total[30d]) * 100"
          }
        ]
      },
      {
        "title": "Build Duration",
        "type": "graph",
        "targets": [
          {
            "expr": "tekton_nightly_build_duration_seconds"
          }
        ]
      }
    ]
  }
}
```

## Security Integration

### 1. Vulnerability Scanning

Integrate security scanning into nightly builds:

```yaml
  security-scan:
    runs-on: ubuntu-latest
    needs: [publish-images]
    steps:
      - name: Scan Images
        run: |
          DATE=$(date +%Y%m%d)
          
          # Scan all published images
          for component in controller webhook events; do
            echo "Scanning ghcr.io/${{ github.repository_owner }}/pipeline/cmd/$component:nightly-$DATE"
            
            grype ghcr.io/${{ github.repository_owner }}/pipeline/cmd/$component:nightly-$DATE \
              --output sarif \
              --file $component-vulnerabilities.sarif
          done
      
      - name: Upload SARIF
        uses: github/codeql-action/upload-sarif@v3
        with:
          sarif_file: ./*.sarif
```

### 2. Supply Chain Security

Track supply chain security:

```yaml
  supply-chain:
    runs-on: ubuntu-latest
    needs: [publish-images]
    steps:
      - name: Generate SBOM
        run: |
          DATE=$(date +%Y%m%d)
          
          # Generate Software Bill of Materials
          syft ghcr.io/${{ github.repository_owner }}/pipeline/cmd/controller:nightly-$DATE \
            --output spdx-json \
            --file controller-sbom.spdx.json
      
      - name: Sign Images
        run: |
          # Sign images with Cosign
          cosign sign --yes ghcr.io/${{ github.repository_owner }}/pipeline/cmd/controller:nightly-$DATE
```

## Advanced Scenarios

### 1. Multi-Repository Coordination

Coordinate nightly builds across multiple repositories:

```yaml
# Trigger downstream nightly builds
name: Trigger Downstream Builds
on:
  workflow_run:
    workflows: ["Tekton Nightly Release"]
    types: [completed]

jobs:
  trigger-downstream:
    if: ${{ github.event.workflow_run.conclusion == 'success' }}
    runs-on: ubuntu-latest
    strategy:
      matrix:
        repo: [triggers, chains, operator]
    steps:
      - name: Trigger ${{ matrix.repo }} Nightly
        run: |
          gh workflow run nightly-release.yaml \
            --repo tektoncd/${{ matrix.repo }} \
            -f tekton_pipeline_version=nightly-$(date +%Y%m%d)
```

### 2. Canary Deployments

Use nightly builds for canary deployments:

```yaml
name: Canary Deployment
on:
  workflow_run:
    workflows: ["Tekton Nightly Release"]
    types: [completed]

jobs:
  canary-deploy:
    runs-on: ubuntu-latest
    steps:
      - name: Deploy Canary
        run: |
          # Deploy to 5% of staging clusters
          DATE=$(date +%Y%m%d)
          kubectl patch deployment tekton-pipelines-controller \
            --patch '{
              "spec": {
                "template": {
                  "spec": {
                    "containers": [{
                      "name": "tekton-pipelines-controller",
                      "image": "ghcr.io/${{ github.repository_owner }}/pipeline/cmd/controller:nightly-'$DATE'"
                    }]
                  }
                }
              }
            }'
      
      - name: Monitor Canary
        run: |
          # Monitor canary for 30 minutes
          sleep 1800
          
          # Check error rates
          ERROR_RATE=$(kubectl logs -l app=tekton-pipelines-controller --since=30m | grep ERROR | wc -l)
          
          if [ "$ERROR_RATE" -gt 10 ]; then
            echo "High error rate detected, rolling back"
            kubectl rollout undo deployment/tekton-pipelines-controller
            exit 1
          fi
```

### 3. Cross-Platform Testing

Test nightly builds across different platforms:

```yaml
name: Cross-Platform Nightly Tests
on:
  workflow_run:
    workflows: ["Tekton Nightly Release"]
    types: [completed]

jobs:
  test-platforms:
    strategy:
      matrix:
        platform: [linux/amd64, linux/arm64, linux/s390x, linux/ppc64le]
        k8s-version: [1.26, 1.27, 1.28, 1.29]
    runs-on: ubuntu-latest
    steps:
      - name: Test Platform ${{ matrix.platform }}
        run: |
          # Set up platform-specific testing
          export PLATFORM=${{ matrix.platform }}
          export K8S_VERSION=${{ matrix.k8s-version }}
          
          # Run platform-specific tests
          ./test/cross-platform-test.sh
```

## Best Practices

1. **Isolation**: Keep nightly builds isolated from production releases
2. **Monitoring**: Implement comprehensive monitoring and alerting
3. **Security**: Regular security scanning and vulnerability assessment
4. **Documentation**: Keep integration documentation up to date
5. **Testing**: Validate integrations regularly
6. **Rollback**: Have rollback procedures for failed integrations

## Troubleshooting Integration Issues

### Common Issues

1. **Build Conflicts**: Nightly builds interfering with regular CI/CD
   - **Solution**: Use different runner pools or time slots

2. **Resource Contention**: Running out of GitHub Actions minutes
   - **Solution**: Optimize build matrix, use self-hosted runners

3. **Authentication Issues**: Problems accessing container registry
   - **Solution**: Verify token permissions and expiration

4. **Notification Fatigue**: Too many alerts from nightly builds
   - **Solution**: Implement smart alerting with thresholds

### Getting Help

- Check [nightly-releases.md](./nightly-releases.md) for general troubleshooting
- Review GitHub Actions logs for integration-specific issues
- Test integrations in a fork before implementing in production
- Use the validation scripts to verify setup

---

This integration guide provides comprehensive patterns for incorporating Tekton nightly releases into your CI/CD pipeline. Adapt these examples to your specific needs and infrastructure.
