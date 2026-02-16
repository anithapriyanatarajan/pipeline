# Testing Dashboard in Kind (Local Kubernetes)

This guide shows you how to test the Tekton Observability Dashboard locally using Kind (Kubernetes in Docker).

## Prerequisites

- Docker installed and running
- [kind](https://kind.sigs.k8s.io/docs/user/quick-start/) installed
- kubectl installed
- Make installed

## Quick Start (One Command!)

```bash
make test-dashboard-local
```

This single command will:
1. Create a kind cluster
2. Deploy Tekton Pipelines
3. Build the dashboard Docker image
4. Load the image into kind
5. Deploy the dashboard

Then access it:
```bash
make dashboard-port-forward
```

Open http://localhost:8080 in your browser!

## Step-by-Step Manual Testing

### 1. Create Kind Cluster

```bash
make kind-cluster
```

This creates a cluster named `tekton-test`.

### 2. Deploy Tekton Pipelines

```bash
make deploy-tekton
```

Wait for Tekton to be ready (automatically waits up to 5 minutes).

### 3. Build and Load Dashboard Image

```bash
make build-dashboard-image
make kind-load-dashboard
```

### 4. Deploy Dashboard

```bash
make deploy-dashboard
```

### 5. Check Status

```bash
make dashboard-status
```

Expected output:
```
NAME                               READY   STATUS    RESTARTS   AGE
tekton-dashboard-xxxxx-xxxxx       1/1     Running   0          30s
```

### 6. Access Dashboard

```bash
make dashboard-port-forward
```

Open http://localhost:8080

### 7. Create Demo Data

In another terminal:
```bash
make dashboard-demo-run
```

This creates 5 sample pipeline runs. Watch them appear in the dashboard!

## Available Make Commands

| Command | Description |
|---------|-------------|
| `make test-dashboard-local` | Full setup in one command |
| `make kind-cluster` | Create kind cluster |
| `make kind-delete` | Delete kind cluster |
| `make build-dashboard` | Build dashboard binary |
| `make build-dashboard-image` | Build Docker image |
| `make kind-load-dashboard` | Load image into kind |
| `make deploy-tekton` | Deploy Tekton Pipelines |
| `make deploy-dashboard` | Deploy dashboard |
| `make deploy-dashboard-demo` | Deploy demo pipelines |
| `make dashboard-logs` | Stream dashboard logs |
| `make dashboard-port-forward` | Port forward to dashboard |
| `make dashboard-demo-run` | Create sample pipeline runs |
| `make dashboard-status` | Check deployment status |
| `make dashboard-cleanup` | Remove dashboard |

## Testing Workflow

### Full Test Cycle

```bash
# Setup
make test-dashboard-local

# In Terminal 1 - Access dashboard
make dashboard-port-forward

# In Terminal 2 - Create demo data
make dashboard-demo-run

# In Terminal 3 - Watch logs
make dashboard-logs
```

### Rebuild and Redeploy

After making code changes:

```bash
# Rebuild image
make build-dashboard-image

# Load into kind
make kind-load-dashboard

# Restart deployment
kubectl rollout restart deployment/tekton-dashboard -n tekton-pipelines

# Watch it restart
kubectl rollout status deployment/tekton-dashboard -n tekton-pipelines

# Access again
make dashboard-port-forward
```

## Troubleshooting

### Dashboard pod not starting

```bash
# Check pod status
make dashboard-status

# Check logs
make dashboard-logs

# Describe pod
kubectl describe pod -n tekton-pipelines -l app.kubernetes.io/name=tekton-dashboard
```

### Image not found

Make sure you loaded the image:
```bash
make kind-load-dashboard
```

Verify it's in kind:
```bash
docker exec -it tekton-test-control-plane crictl images | grep tekton-dashboard
```

### Port already in use

If port 8080 is busy:
```bash
# Use a different port
kubectl port-forward -n tekton-pipelines svc/tekton-dashboard 9090:8080
```

### Tekton not ready

```bash
# Check Tekton status
kubectl get pods -n tekton-pipelines

# Wait for it
kubectl wait --for=condition=Ready pods --all -n tekton-pipelines --timeout=300s
```

## Development Workflow

### Watch Mode (Recommended)

For rapid development:

1. **Terminal 1** - Watch and rebuild:
```bash
# Auto-rebuild on code changes (requires entr or nodemon)
find cmd/dashboard pkg/dashboard -name "*.go" | entr -r make build-dashboard-image kind-load-dashboard
```

2. **Terminal 2** - Port forward:
```bash
make dashboard-port-forward
```

3. **Terminal 3** - Watch logs:
```bash
make dashboard-logs
```

4. **Terminal 4** - Create test data:
```bash
make dashboard-demo-run
```

### Frontend Development

If working on just the frontend:

```bash
cd web/dashboard
npm install
npm run dev
```

This runs Vite dev server with hot reload at http://localhost:5173

Configure it to proxy API calls to your kind cluster:
```bash
# In another terminal
make dashboard-port-forward
```

## Cleanup

### Remove Dashboard Only

```bash
make dashboard-cleanup
```

### Delete Entire Cluster

```bash
make kind-delete
```

## Custom Configuration

### Use Different Cluster Name

```bash
KIND_CLUSTER_NAME=my-cluster make test-dashboard-local
```

### Use Different Image Tag

```bash
DASHBOARD_IMAGE=my-dashboard:v2 make build-dashboard-image
DASHBOARD_IMAGE=my-dashboard:v2 make kind-load-dashboard
```

Update deployment to use your image:
```bash
kubectl set image deployment/tekton-dashboard \
  dashboard=my-dashboard:v2 \
  -n tekton-pipelines
```

## Performance Testing

### Create Many Runs

```bash
# Create 50 pipeline runs
for i in {1..50}; do
  kubectl create -f examples/dashboard-demo/01-simple-pipelinerun.yaml
  sleep 1
done
```

### Monitor Resource Usage

```bash
# Dashboard resource usage
kubectl top pod -n tekton-pipelines -l app.kubernetes.io/name=tekton-dashboard

# All Tekton resources
kubectl top pod -n tekton-pipelines
```

## Next Steps

- Try the [demo scenarios](examples/dashboard-demo/README.md)
- Read the [architecture docs](docs/dashboard.md)
- Check out the [API reference](docs/dashboard.md#api-reference)
- Prepare for your [conference talk](docs/conference-talk-guide.md)

## Tips

✅ **Use tmux/screen** - Easier to manage multiple terminals  
✅ **Watch logs** - `make dashboard-logs` shows what's happening  
✅ **Check metrics** - Dashboard exposes metrics at :9090/metrics  
✅ **Test failures** - Modify tasks to fail and test anomaly detection  
✅ **Vary durations** - Change sleep times to test performance analytics  

Happy testing! 🚀
