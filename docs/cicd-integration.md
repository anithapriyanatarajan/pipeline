# CI/CD Integration Guide for Tekton Nightly Releases

This guide provides simple patterns for integrating nightly releases into your existing CI/CD workflow.

## Overview

Nightly releases run independently and publish container images to `ghcr.io/{owner}/pipeline/`. You can integrate these into your workflow in a few simple ways:

- **Independent**: Nightly builds run on their own schedule (03:00 UTC)
- **Tested**: Each build includes basic validation before publishing
- **Available**: Images are published to GitHub Container Registry

## Integration Patterns

### Pattern 1: Use Nightly Images in Testing

Test your applications against the latest nightly build:

```yaml
name: Test with Nightly Tekton
on:
  schedule:
    - cron: '0 6 * * *'  # Run after nightly build

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Install Latest Nightly
        run: |
          # Install the latest nightly release
          kubectl apply -f https://storage.googleapis.com/tekton-releases-nightly/pipeline/nightly/latest/release.yaml
          
          # Wait for it to be ready
          kubectl wait --for=condition=Available=True deployment/tekton-pipelines-controller -n tekton-pipelines --timeout=300s
          
      - name: Run Your Tests
        run: |
          # Run your pipeline tests here
          echo "Testing with nightly Tekton..."
```

### Pattern 2: Deploy to Staging After Successful Build

Automatically deploy successful nightly builds to staging:

```yaml
name: Deploy Nightly to Staging
on:
  workflow_run:
    workflows: ["Tekton Nightly Release"]
    types: [completed]

jobs:
  deploy:
    if: ${{ github.event.workflow_run.conclusion == 'success' }}
    runs-on: ubuntu-latest
    environment: staging
    steps:
      - name: Deploy to Staging
        run: |
          # Configure kubectl for your staging cluster
          echo "${{ secrets.STAGING_KUBECONFIG }}" | base64 -d > $HOME/.kube/config
          
          # Deploy the latest nightly
          kubectl apply -f https://storage.googleapis.com/tekton-releases-nightly/pipeline/nightly/latest/release.yaml
          
          # Verify deployment
          kubectl wait --for=condition=Available=True deployment/tekton-pipelines-controller -n tekton-pipelines --timeout=300s
          echo "✅ Staging deployment complete"
```

### Pattern 3: Quality Gates

Block releases if nightly builds are failing:

```yaml
name: Release
on:
  push:
    tags: ['v*']

jobs:
  check-nightly:
    runs-on: ubuntu-latest
    steps:
      - name: Check Recent Nightly Status
        env:
          GH_TOKEN: ${{ github.token }}
        run: |
          # Check last 3 nightly runs
          FAILED_RUNS=$(gh run list --workflow="Tekton Nightly Release" --limit=3 --json=conclusion --jq '[.[] | select(.conclusion == "failure")] | length')
          
          if [ "$FAILED_RUNS" -ge 2 ]; then
            echo "❌ Multiple recent nightly failures - investigate before releasing"
            exit 1
          fi
          
          echo "✅ Recent nightly builds are healthy"
          
  release:
    needs: check-nightly
    runs-on: ubuntu-latest
    steps:
      - name: Proceed with Release
        run: echo "🚀 Releasing..."
```

## Notifications

### Slack Notifications

Get notified when nightly builds fail:

```yaml
name: Nightly Notifications
on:
  workflow_run:
    workflows: ["Tekton Nightly Release"]
    types: [completed]

jobs:
  notify:
    if: ${{ github.event.workflow_run.conclusion == 'failure' }}
    runs-on: ubuntu-latest
    steps:
      - name: Notify Slack
        env:
          SLACK_WEBHOOK: ${{ secrets.SLACK_WEBHOOK_URL }}
        run: |
          curl -X POST -H 'Content-type: application/json' \
            --data "{\"text\":\"❌ Nightly Tekton build failed: ${{ github.event.workflow_run.html_url }}\"}" \
            "$SLACK_WEBHOOK"
```

## Development Workflow

### Testing Changes Impact

Before merging changes that might affect nightly builds:

```bash
# Test your changes don't break nightly builds
gh workflow run "Tekton Nightly Release" --ref your-branch

# Wait and check the result
gh run watch
```

### Monitoring Nightly Health

Simple script to check nightly build health:

```bash
#!/bin/bash
# check-nightly-health.sh

RECENT_RUNS=$(gh run list --workflow="Tekton Nightly Release" --limit=5 --json=conclusion --jq '[.[] | select(.conclusion == "failure")] | length')

if [ "$RECENT_RUNS" -ge 3 ]; then
  echo "⚠️ Nightly builds need attention ($RECENT_RUNS recent failures)"
  exit 1
else
  echo "✅ Nightly builds are healthy"
fi
```

## Troubleshooting

### Common Issues

1. **Wrong Workflow Name**
   ```yaml
   # ❌ Wrong
   workflows: ["Tekton Nightly Release (Production Ready)"]
   
   # ✅ Correct
   workflows: ["Tekton Nightly Release"]
   ```

2. **Assuming Specific Dates**
   ```bash
   # ❌ Wrong - assumes build happened today
   :nightly-$(date +%Y%m%d)
   
   # ✅ Correct - use latest
   https://storage.googleapis.com/tekton-releases-nightly/pipeline/nightly/latest/release.yaml
   ```

3. **Missing Error Handling**
   ```bash
   # ❌ Wrong
   kubectl apply -f some-url
   
   # ✅ Correct
   if ! kubectl apply -f some-url; then
     echo "Failed to apply"
     exit 1
   fi
   ```

### Getting Help

- Check the nightly workflow logs in your repository's Actions tab
- Look at recent runs to see if there's a pattern in failures
- Issues are typically related to:
  - GitHub token permissions (`packages:write`, `contents:read`)
  - Container registry connectivity
  - Kubernetes version compatibility

---

This simplified integration approach focuses on practical, everyday use cases without unnecessary complexity. The nightly release system is designed to "just work" - these patterns help you make the most of it without overengineering.
