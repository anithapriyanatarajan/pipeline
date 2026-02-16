# 🚀 Quick Start: Test Dashboard in Kind

## One Command Setup

```bash
make test-dashboard-local
```

Then in another terminal:

```bash
make dashboard-port-forward
```

Open **http://localhost:8080** 🎉

## Create Demo Data

```bash
make dashboard-demo-run
```

Watch the dashboard populate with metrics!

## All Commands

```bash
# Full setup
make test-dashboard-local          # Create cluster + deploy everything

# Access
make dashboard-port-forward        # Access at http://localhost:8080
make dashboard-logs                # View logs

# Demo
make dashboard-demo-run           # Create sample pipeline runs

# Status
make dashboard-status             # Check if it's running

# Cleanup
make dashboard-cleanup            # Remove dashboard
make kind-delete                  # Delete cluster
```

## What Gets Installed

1. Kind cluster named `tekton-test`
2. Tekton Pipelines (latest version)
3. Dashboard (built from source)
4. Demo pipelines (optional)

## Requirements

- Docker
- kind (`brew install kind` or see https://kind.sigs.k8s.io)
- kubectl
- make
- Go 1.25+ (for building)
- Node.js 18+ (for frontend)

## Troubleshooting

**Dashboard not starting?**
```bash
make dashboard-logs
```

**Port 8080 busy?**
```bash
kubectl port-forward -n tekton-pipelines svc/tekton-dashboard 9090:8080
```

**Want to rebuild?**
```bash
make build-dashboard-image
make kind-load-dashboard
kubectl rollout restart deployment/tekton-dashboard -n tekton-pipelines
```

## Full Documentation

See [TESTING_IN_KIND.md](TESTING_IN_KIND.md) for detailed instructions.

---

**Ready?** Run `make test-dashboard-local` and start testing! 🎯
