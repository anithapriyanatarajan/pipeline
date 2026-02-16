# Tekton Observability Dashboard - Quick Start Guide

## Overview

The Tekton Observability Dashboard provides a comprehensive, built-in solution for monitoring Tekton Pipelines with:

- 📊 **Real-time Metrics** - Live pipeline and task monitoring
- 💰 **Cost Tracking** - Resource usage and cost analysis
- 🎯 **Performance Analytics** - Duration trends and bottleneck detection
- 🤖 **AI Insights** - Anomaly detection and optimization recommendations
- 🔍 **Distributed Tracing** - End-to-end execution visualization

## Quick Start (5 minutes)

### Prerequisites

- Kubernetes cluster (1.24+)
- kubectl configured
- Tekton Pipelines v0.50.0+

### 1. Install Tekton Pipelines (if not already installed)

```bash
kubectl apply -f https://storage.googleapis.com/tekton-releases/pipeline/latest/release.yaml
```

Wait for Tekton to be ready:

```bash
kubectl wait --for=condition=Ready pods --all -n tekton-pipelines --timeout=300s
```

### 2. Deploy the Observability Dashboard

```bash
kubectl apply -f config/dashboard/
```

### 3. Verify Installation

```bash
kubectl get pods -n tekton-pipelines -l app.kubernetes.io/name=tekton-dashboard
```

Expected output:
```
NAME                               READY   STATUS    RESTARTS   AGE
tekton-dashboard-xxxxx-xxxxx       1/1     Running   0          30s
```

### 4. Access the Dashboard

```bash
kubectl port-forward -n tekton-pipelines svc/tekton-dashboard 8080:8080
```

Open your browser to: **http://localhost:8080**

## Features

### Dashboard Overview
- Real-time pipeline and task counts
- Success rate tracking
- Active anomaly alerts
- Total cost (last 24 hours)

### Pipeline Metrics
- Detailed per-pipeline statistics
- Success/failure breakdown
- Average duration tracking
- Historical trends

### Cost Analysis
- Cost breakdown by resource type (CPU, Memory, Storage)
- Per-pipeline cost tracking
- 7-day cost trends
- Optimization recommendations

### AI-Powered Insights
- **Anomaly Detection**: Identifies unusual patterns in pipeline duration and failure rates
- **Recommendations**: Actionable suggestions for cost reduction and performance optimization
- **Predictions**: Failure probability and duration estimates

### Distributed Tracing
- End-to-end pipeline execution flows
- Task dependency visualization
- Latency analysis

## Configuration

### Cost Tracking

Edit the ConfigMap to set your infrastructure costs:

```bash
kubectl edit configmap -n tekton-pipelines config-dashboard
```

```yaml
data:
  cost-tracking.enabled: "true"
  cost-tracking.cpu-cost-per-hour: "0.05"        # USD per CPU core hour
  cost-tracking.memory-cost-per-gb-hour: "0.01"  # USD per GB hour
  cost-tracking.storage-cost-per-gb-hour: "0.001" # USD per GB hour
```

### AI Insights

Configure AI-powered features:

```yaml
data:
  ai-insights.enabled: "true"
  ai-insights.anomaly-detection: "true"
  ai-insights.predictive-analysis: "true"
  ai-insights.min-samples-for-prediction: "10"
```

Changes take effect immediately (no restart required).

## Demo

Run the included demo pipelines:

```bash
# Deploy demo pipelines
kubectl apply -f examples/dashboard-demo/01-simple-pipeline.yaml

# Create some pipeline runs
for i in {1..5}; do
  kubectl create -f examples/dashboard-demo/01-simple-pipelinerun.yaml
  sleep 2
done
```

Watch the dashboard update in real-time!

## Architecture

```
┌─────────────────────────────────────────────────┐
│            Dashboard Frontend (React)            │
│  - Real-time charts                             │
│  - WebSocket updates                            │
└─────────────────┬───────────────────────────────┘
                  │ HTTP/WebSocket
┌─────────────────▼───────────────────────────────┐
│          Dashboard Backend (Go)                  │
│  - REST API                                      │
│  - Data aggregation                              │
└─────────┬───────────────┬────────────┬──────────┘
          │               │            │
┌─────────▼─────┐ ┌──────▼──────┐ ┌──▼─────────┐
│   Metrics     │ │    Cost     │ │   Trace    │
│   Collector   │ │  Collector  │ │ Collector  │
└─────┬─────────┘ └──────┬──────┘ └──┬─────────┘
      │                  │            │
┌─────▼──────────────────▼────────────▼─────────┐
│          AI Insights Engine                    │
│  - Anomaly detection                           │
│  - Predictive analytics                        │
│  - Recommendations                             │
└────────────────────────────────────────────────┘
```

## API Reference

### REST Endpoints

```
GET  /api/v1/metrics/overview          - High-level summary
GET  /api/v1/metrics/pipelines         - Pipeline-level metrics
GET  /api/v1/metrics/tasks             - Task-level metrics
GET  /api/v1/metrics/history           - Historical metrics
GET  /api/v1/costs/breakdown           - Cost analysis
GET  /api/v1/costs/trend               - Cost trends
GET  /api/v1/traces                    - Distributed traces
GET  /api/v1/insights                  - AI insights
GET  /api/v1/insights/anomalies        - Detected anomalies
GET  /api/v1/insights/recommendations  - Optimization recommendations
```

### WebSocket Endpoints

```
WS   /api/v1/stream/metrics            - Real-time metrics stream
WS   /api/v1/stream/events             - Real-time event stream
```

## Development

### Building from Source

```bash
# Build backend
make build-dashboard

# Build frontend
cd web/dashboard
npm install
npm run build

# Run locally
./bin/dashboard --kubeconfig ~/.kube/config
```

### Running Tests

```bash
# Backend tests
go test ./pkg/dashboard/... ./cmd/dashboard/...

# Frontend tests
cd web/dashboard && npm test
```

## Troubleshooting

### Dashboard not starting

```bash
# Check logs
kubectl logs -n tekton-pipelines deployment/tekton-dashboard

# Check pod status
kubectl describe pod -n tekton-pipelines -l app.kubernetes.io/name=tekton-dashboard
```

### No metrics showing

Verify Tekton controller is exposing metrics:

```bash
kubectl port-forward -n tekton-pipelines svc/tekton-pipelines-controller 9090:9090
curl http://localhost:9090/metrics | grep tekton_pipelines
```

### Cost tracking not working

Check configuration:

```bash
kubectl get configmap -n tekton-pipelines config-dashboard -o yaml
```

## Comparison with Other Tools

| Feature | Built-in Dashboard | External Tools (Grafana/Jaeger) |
|---------|-------------------|---------------------------------|
| Setup Time | < 5 minutes | Hours to days |
| Tekton-Specific Metrics | ✅ Native | ❌ Generic |
| Cost Tracking | ✅ Built-in | ❌ Custom implementation |
| AI Insights | ✅ Included | ❌ Separate tools required |
| Maintenance | ✅ Part of Tekton | ❌ Additional overhead |
| Custom Dashboards | ❌ Limited | ✅ Highly flexible |
| Long-term Storage | ❌ 24-48 hours | ✅ Unlimited |

**Recommendation**: Use both approaches complementarily:
- **Built-in Dashboard**: Day-to-day monitoring, quick insights
- **External Tools**: Organization-wide dashboards, long-term analysis

## Performance

The dashboard is designed to be lightweight:

- **Memory**: ~100-200 MB
- **CPU**: < 100m under normal load
- **Storage**: In-memory only (24h metrics, 7d costs)
- **Network**: ~15KB/s during active monitoring

## Security

The dashboard requires the following permissions:

- **Read**: Tekton resources (Pipelines, PipelineRuns, Tasks, TaskRuns)
- **Read**: Pods and logs
- **Read**: Metrics API

No write permissions are required.

## Roadmap

### Current (v1.0 - Alpha)
- ✅ Real-time metrics aggregation
- ✅ Cost tracking
- ✅ Basic AI insights
- ✅ REST API
- ✅ WebSocket streaming

### Next (v1.1 - Q2 2026)
- 🔄 Advanced trace visualization
- 🔄 Custom dashboard layouts
- 🔄 Alert management
- 🔄 Export/import configurations

### Future (v2.0 - Q3 2026)
- 📋 Multi-cluster support
- 📋 RBAC integration
- 📋 Plugin system
- 📋 Advanced ML models

## Contributing

We welcome contributions! Areas where we need help:

- 🎨 UI/UX improvements
- 📊 New visualization types
- 🤖 Enhanced ML models
- 📚 Documentation
- 🧪 Testing
- 🌍 Internationalization

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## Community

- **Slack**: [#tekton-dashboard](https://tektoncd.slack.com/messages/tekton-dashboard)
- **GitHub**: [tektoncd/pipeline](https://github.com/tektoncd/pipeline)
- **Community Meetings**: Tuesdays 9:00 AM PT

## License

Apache License 2.0 - See [LICENSE](LICENSE) for details.

## Frequently Asked Questions

**Q: How does this relate to the existing Tekton Dashboard?**

A: The existing Tekton Dashboard focuses on resource management and log viewing. This observability dashboard focuses on metrics, analytics, cost tracking, and AI-powered insights. They are complementary.

**Q: Can I use this in production?**

A: This is currently in alpha. We recommend using it in staging/development environments while we gather feedback and stabilize the API.

**Q: Does this work with Tekton Triggers?**

A: Yes! The dashboard monitors all PipelineRuns and TaskRuns, regardless of how they were triggered.

**Q: Can I export data for external analysis?**

A: Currently, data can be accessed via the REST API. Export functionality is planned for v1.1.

**Q: What's the performance impact on my cluster?**

A: Minimal. The dashboard polls the Tekton controller's metrics endpoint every 15 seconds and maintains data in memory. CPU usage is typically < 100m and memory < 200MB.

**Q: How accurate is the cost tracking?**

A: Cost calculation is based on pod resource requests multiplied by duration and your configured rates. It provides estimates rather than exact billing data. For precise costs, integrate with your cloud provider's billing APIs.

**Q: Can I customize the cost rates per namespace/pipeline?**

A: Not currently, but this is on the roadmap for v1.1.

**Q: How long is data retained?**

A: Metrics: 24 hours, Costs: 7 days, Traces: 1 hour. These are configurable via the ConfigMap.

## Support

For issues, questions, or feature requests:

1. Check the [documentation](docs/dashboard.md)
2. Search [existing issues](https://github.com/tektoncd/pipeline/issues)
3. Ask in [Slack #tekton-dashboard](https://tektoncd.slack.com)
4. [Open a new issue](https://github.com/tektoncd/pipeline/issues/new)

---

**Ready to get started?** Jump to [Quick Start](#quick-start-5-minutes) above! 🚀
