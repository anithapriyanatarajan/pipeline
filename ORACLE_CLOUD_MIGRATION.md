# Oracle Cloud Storage Migration Guide

This document describes the migration from Google Cloud Storage (GCS) to Oracle Cloud Infrastructure (OCI) Object Storage for the Tekton release pipeline.

## Changes Made

### 1. GitHub Actions Workflow (`/.github/workflows/nightly-builds.yaml`)

#### Added Temporary Testing Trigger
- Added `on.push` trigger for the `update-release-pipeline` branch
- Workflow now triggers on changes to:
  - `.github/workflows/nightly-builds.yaml`
  - `tekton/release-pipeline.yaml`
  - `tekton/publish.yaml`
- **Note**: Remove this `push` trigger before merging to main, keeping only the `schedule` and `workflow_dispatch` triggers

#### Replaced GCS Credentials with OCI Credentials
Changed from GCS service account to OCI API key authentication:

**Old secrets required:**
- `GCS_SERVICE_ACCOUNT_KEY`

**New secrets required:**
- `OCI_API_KEY` - Your OCI API private key (PEM format)
- `OCI_FINGERPRINT` - Your API key fingerprint
- `OCI_TENANCY_OCID` - Your OCI tenancy OCID
- `OCI_USER_OCID` - Your OCI user OCID
- `OCI_REGION` - OCI region (e.g., `us-ashburn-1`)
- `OCI_NAMESPACE` - OCI Object Storage namespace (optional, can be auto-detected)

#### Updated Secret Creation
The workflow now creates a Kubernetes secret with OCI credentials instead of GCS credentials:
- Secret name remains `release-secret` for compatibility
- Contains OCI configuration files instead of GCS service account JSON

#### Removed Parameter
- Removed `serviceAccountPath` parameter from pipeline invocation (not needed for OCI)

### 2. Release Pipeline (`/tekton/release-pipeline.yaml`)

#### Removed Parameter
- Removed `serviceAccountPath` parameter from pipeline spec (it was GCS-specific)

#### Task Replacements
Replaced both GCS upload tasks with Oracle Cloud Storage upload tasks:

**Task 1: `publish-to-bucket`**
- Old: `ghcr.io/tektoncd/catalog/upstream/tasks/gcs-upload:0.3`
- New: `ghcr.io/tektoncd/catalog/upstream/tasks/oracle-cloud-storage-upload:0.1`
- Uploads release artifacts to `<bucket>/previous/<versionTag>/`

**Task 2: `publish-to-bucket-latest`**
- Old: `ghcr.io/tektoncd/catalog/upstream/tasks/gcs-upload:0.3`
- New: `ghcr.io/tektoncd/catalog/upstream/tasks/oracle-cloud-storage-upload:0.1`
- Uploads release artifacts to `<bucket>/latest/`
- Uses `deleteExtraFiles: "true"` to sync (similar to GCS rsync behavior)

#### Parameter Mapping
The Oracle Cloud Storage task uses different but compatible parameters:

| GCS Parameter | OCI Parameter | Notes |
|---------------|---------------|-------|
| `location` | `bucketName` + `objectPrefix` | Split into bucket name and path within bucket |
| `path` | `path` | Same - relative path within source workspace |
| `serviceAccountPath` | N/A | Credentials handled differently in OCI secret |
| `deleteExtraFiles` | `deleteExtraFiles` | Same behavior - sync with delete |
| N/A | `replaceExistingFiles` | New - set to "true" for compatibility |
| N/A | `recursive` | New - set to "true" for directory uploads |

## Setup Instructions

### 1. Create OCI Credentials

Generate API keys in OCI Console:
1. Go to User Settings → API Keys
2. Generate API Key pair
3. Download the private key (PEM format)
4. Copy the fingerprint, tenancy OCID, user OCID, and region

### 2. Add GitHub Secrets

Add the following secrets to your GitHub repository (Settings → Secrets and variables → Actions):

```bash
# Required secrets
OCI_API_KEY=<contents of private key PEM file>
OCI_FINGERPRINT=<your key fingerprint>
OCI_TENANCY_OCID=<your tenancy OCID>
OCI_USER_OCID=<your user OCID>
OCI_REGION=<your OCI region, e.g., us-ashburn-1>

# Optional secret (auto-detected if not provided)
OCI_NAMESPACE=<your Object Storage namespace>
```

### 3. Create OCI Object Storage Bucket

1. Go to OCI Console → Storage → Buckets
2. Create a new bucket for your releases (e.g., `tekton-releases-nightly`)
3. Set appropriate access policies
4. Note the bucket name for the next step

### 4. Update Bucket Parameter

When running the workflow, provide the OCI bucket name instead of a GCS bucket path:

**Old format:**
```
gs://tekton-releases-nightly/pipeline
```

**New format:**
```
tekton-releases-nightly
```

The bucket name should be just the bucket name, not a full path. The `objectPrefix` parameter in the tasks will handle the subdirectories (`previous/`, `latest/`, etc.).

Alternatively, if you want to maintain compatibility with the path structure, you can update the workflow input default:

```yaml
nightly_bucket:
  description: 'Nightly bucket for builds'
  required: false
  default: 'tekton-releases-nightly'  # Just the bucket name
  type: string
```

### 5. IAM Policies

Ensure your OCI user has the following policies:

```
Allow group <your-group> to manage objects in compartment <your-compartment>
Allow group <your-group> to manage buckets in compartment <your-compartment>
```

Or more restrictive:

```
Allow group <your-group> to manage objects in compartment <your-compartment> where target.bucket.name='<bucket-name>'
```

## Verification

### Test the Workflow

1. Push changes to the `update-release-pipeline` branch
2. Workflow will automatically trigger
3. Check the workflow run logs for successful OCI uploads
4. Verify files appear in your OCI bucket under `previous/<versionTag>/` and `latest/` directories

### Expected Output Structure

After a successful run, your OCI bucket should contain:

```
<bucket-name>/
  ├── previous/
  │   └── v20231031-abc1234/
  │       ├── release.yaml
  │       ├── release.notag.yaml
  │       └── ... (other release artifacts)
  └── latest/
      ├── release.yaml
      ├── release.notag.yaml
      └── ... (other release artifacts)
```

## Compatibility Notes

1. **Bucket Path Format**: The Oracle Cloud Storage task expects a bucket name, not a URL. The subdirectories are specified via `objectPrefix`.

2. **Credentials Format**: OCI uses API key authentication (multiple files) instead of a single JSON service account file.

3. **Sync Behavior**: The `deleteExtraFiles: "true"` parameter provides similar behavior to GCS's rsync functionality.

4. **URL Generation**: The `report-bucket` task may need additional updates if you need to generate public URLs for OCI Object Storage. Currently, it returns the bucket path format.

## Rollback

To rollback to GCS:

1. Revert changes in `.github/workflows/nightly-builds.yaml`
2. Revert changes in `tekton/release-pipeline.yaml`
3. Re-add the `serviceAccountPath` parameter
4. Replace Oracle Cloud Storage tasks with GCS upload tasks

## Support

For issues with the Oracle Cloud Storage task, refer to:
- Task documentation: https://artifacthub.io/packages/tekton-task/tekton-catalog-tasks/oracle-cloud-storage-upload
- Source code: https://github.com/tektoncd/catalog/tree/main/task/oracle-cloud-storage-upload/0.1
- OCI CLI documentation: https://docs.oracle.com/en-us/iaas/tools/oci-cli/latest/

## Before PR Submission

Remember to remove the temporary testing trigger from `.github/workflows/nightly-builds.yaml`:

```yaml
# Remove this section before PR:
on:
  push:
    branches:
      - update-release-pipeline
    paths:
      - '.github/workflows/nightly-builds.yaml'
      - 'tekton/release-pipeline.yaml'
      - 'tekton/publish.yaml'
```

Keep only:
```yaml
on:
  schedule:
    - cron: "0 3 * * *"
  workflow_dispatch:
    ...
```
