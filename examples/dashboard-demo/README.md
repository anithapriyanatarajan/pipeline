# Demo: Tekton Unified Observability Dashboard

This demo showcases the unified observability dashboard for Tekton Pipelines. Follow these steps to set up and demonstrate the dashboard capabilities.

## Prerequisites

- Kubernetes cluster (kind, minikube, or cloud provider)
- kubectl configured
- Tekton Pipelines installed
- Docker (for building images)

## Quick Start

### 1. Install Tekton Pipelines

```bash
kubectl apply -f https://storage.googleapis.com/tekton-releases/pipeline/latest/release.yaml
```

### 2. Deploy the Observability Dashboard

```bash
kubectl apply -f config/dashboard/
```

### 3. Verify Installation

```bash
kubectl get pods -n tekton-pipelines
kubectl logs -n tekton-pipelines deployment/tekton-dashboard
```

### 4. Access the Dashboard

```bash
kubectl port-forward -n tekton-pipelines svc/tekton-dashboard 8080:8080
```

Open http://localhost:8080 in your browser.

## Demo Scenarios

### Scenario 1: Real-Time Pipeline Monitoring

This scenario demonstrates live pipeline execution monitoring.

#### Setup

```bash
# Apply demo pipeline
kubectl apply -f examples/dashboard-demo/01-simple-pipeline.yaml

# Create multiple pipeline runs
for i in {1..5}; do
  kubectl create -f examples/dashboard-demo/01-simple-pipelinerun.yaml
  sleep 2
done
```

#### Demo Points

1. **Dashboard Overview Page**
   - Shows running pipelines count updating in real-time
   - Success rate metrics
   - Active/completed runs

2. **Pipeline Metrics Page**
   - Individual pipeline statistics
   - Duration trends
   - Success/failure breakdown

3. **Live Updates**
   - WebSocket-powered real-time metrics
   - No page refresh required

### Scenario 2: Cost Tracking & Optimization

This scenario shows cost analysis and optimization recommendations.

#### Setup

```bash
# Apply resource-intensive pipeline
kubectl apply -f examples/dashboard-demo/02-resource-intensive-pipeline.yaml

# Run multiple times to accumulate cost data
for i in {1..10}; do
  kubectl create -f examples/dashboard-demo/02-resource-intensive-pipelinerun.yaml
done
```

#### Demo Points

1. **Cost Analysis Page**
   - Total cost breakdown (CPU, Memory, Storage)
   - Cost per pipeline
   - Average cost per run

2. **Cost Trends**
   - 7-day cost trend visualization
   - Identify cost spikes
   - Compare costs across pipelines

3. **Cost Optimization**
   - AI-generated recommendations for reducing costs
   - Resource right-sizing suggestions

### Scenario 3: Performance Analytics

#### Setup

```bash
# Apply pipeline with varying performance
kubectl apply -f examples/dashboard-demo/03-performance-test-pipeline.yaml

# Create runs with different characteristics
kubectl create -f examples/dashboard-demo/03-fast-run.yaml
kubectl create -f examples/dashboard-demo/03-slow-run.yaml
kubectl create -f examples/dashboard-demo/03-normal-run.yaml
```

#### Demo Points

1. **Performance Metrics**
   - Average, P50, P95, P99 duration
   - Duration trends over time
   - Identify slow pipelines

2. **Bottleneck Detection**
   - Task-level performance breakdown
   - Identify slowest tasks
   - Compare task performance across runs

### Scenario 4: AI-Powered Insights

This scenario demonstrates anomaly detection and predictive analytics.

#### Setup

```bash
# Create baseline successful runs
for i in {1..10}; do
  kubectl create -f examples/dashboard-demo/04-baseline-run.yaml
  sleep 5
done

# Introduce anomalies
kubectl create -f examples/dashboard-demo/04-slow-anomaly-run.yaml
kubectl create -f examples/dashboard-demo/04-failing-run.yaml
```

#### Demo Points

1. **Anomaly Detection**
   - Duration anomalies (pipeline taking unusually long)
   - Failure rate anomalies (sudden increase in failures)
   - Severity scoring

2. **Recommendations**
   - Performance optimization suggestions
   - Cost reduction opportunities
   - Priority-based recommendations

3. **Predictive Analytics**
   - Failure probability prediction
   - Duration estimates
   - Confidence scoring

### Scenario 5: Distributed Tracing

#### Setup

```bash
# Apply pipeline with multiple tasks
kubectl apply -f examples/dashboard-demo/05-complex-pipeline.yaml
kubectl create -f examples/dashboard-demo/05-complex-pipelinerun.yaml
```

#### Demo Points

1. **Trace Visualization**
   - End-to-end pipeline execution flow
   - Task dependencies
   - Execution timeline

2. **Span Analysis**
   - Individual task spans
   - Latency breakdown
   - Critical path identification

## Demo Script for Conference Talk

### Introduction (2 minutes)

"Today, platform engineers using Tekton face a common challenge: observability is fragmented. They need Prometheus for metrics, Grafana for visualization, Jaeger for tracing, and custom scripts to tie it all together. What if we could provide world-class observability built right into Tekton?"

### Live Demo (15 minutes)

#### Part 1: Dashboard Overview (3 min)

1. Open dashboard homepage
2. Show real-time metrics
3. Explain key metrics (running pipelines, success rate, costs)

#### Part 2: Pipeline Monitoring (4 min)

1. Create new pipeline runs live
2. Show real-time updates
3. Navigate to pipeline details
4. Show duration trends

#### Part 3: Cost Analysis (3 min)

1. Navigate to cost page
2. Show total cost breakdown
3. Explain cost per pipeline
4. Show 7-day trend

#### Part 4: AI Insights (5 min)

1. Navigate to insights page
2. Show detected anomalies (triggered by demo pipelines)
3. Explain anomaly scoring
4. Show recommendations
5. Discuss predictive analytics

### Architecture Deep Dive (5 minutes)

Show architecture diagram and explain:
- Dashboard backend (Go service)
- Data collectors (metrics, cost, traces)
- Insights engine (ML-powered)
- Frontend (React SPA)
- Integration points

### vs. External Tools (3 minutes)

Comparison table:
| Feature | Built-in Dashboard | External Tools |
|---------|-------------------|----------------|
| Setup Time | < 5 minutes | Hours/Days |
| Tekton-Specific | ✅ Native | ❌ Generic |
| Cost Tracking | ✅ Built-in | ❌ Custom |
| AI Insights | ✅ Included | ❌ Separate |
| Maintenance | ✅ Part of Tekton | ❌ Additional |

### Roadmap & Community (2 minutes)

- Current: Core functionality
- Next: Advanced trace visualization, custom dashboards
- Future: Multi-cluster support, plugin system
- Call for contributors

## Troubleshooting

### Dashboard not starting

```bash
kubectl logs -n tekton-pipelines deployment/tekton-dashboard
kubectl describe pod -n tekton-pipelines -l app.kubernetes.io/name=tekton-dashboard
```

### No metrics showing

```bash
# Verify controller is exposing metrics
kubectl port-forward -n tekton-pipelines svc/tekton-pipelines-controller 9090:9090
curl http://localhost:9090/metrics
```

### Cost tracking not working

```bash
# Check ConfigMap
kubectl get configmap -n tekton-pipelines config-dashboard -o yaml
```

## Cleanup

```bash
kubectl delete -f config/dashboard/
kubectl delete -f examples/dashboard-demo/
```

## Tips for Live Demo

1. **Pre-create some pipeline runs** before the demo for historical data
2. **Have backup screenshots** in case of connectivity issues
3. **Use a large font** in terminal and browser
4. **Zoom in** on dashboard UI for audience visibility
5. **Practice transitions** between demo scenarios
6. **Have timing** - know what to skip if running short on time

## Questions & Answers Preparation

**Q: How does this compare to Tekton Dashboard?**
A: The existing Tekton Dashboard focuses on resource management and log viewing. This observability dashboard focuses on metrics, analytics, cost tracking, and AI-powered insights. They complement each other.

**Q: Can I use this with my existing Grafana setup?**
A: Absolutely! The dashboard exposes standard Prometheus metrics, so you can continue using Grafana while benefiting from the built-in dashboard for quick insights.

**Q: What's the performance impact?**
A: Minimal. The dashboard runs as a lightweight service and polls metrics every 15 seconds. Resource usage is typically < 100MB memory and < 100m CPU.

**Q: Is this production-ready?**
A: This is currently a proof-of-concept for the conference talk. We're gathering feedback from the community before stabilizing for production use.

**Q: How can I contribute?**
A: Check out the GitHub repository! We need help with UI improvements, additional ML models, trace visualization, and documentation.
