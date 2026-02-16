/*
Copyright 2026 The Tekton Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package dashboard

// Config holds the dashboard configuration
type Config struct {
	MetricsEndpoint      string
	EnableCostTracking   bool
	EnableAIInsights     bool
	CPUCostPerHour       float64
	MemoryCostPerGBHour  float64
	StorageCostPerGBHour float64
}

// MetricsSnapshot represents a point-in-time view of pipeline metrics
type MetricsSnapshot struct {
	Timestamp               int64                      `json:"timestamp"`
	RunningPipelines        int                        `json:"running_pipelines"`
	RunningTasks            int                        `json:"running_tasks"`
	SuccessfulPipelines     int                        `json:"successful_pipelines"`
	FailedPipelines         int                        `json:"failed_pipelines"`
	TotalPipelines          int                        `json:"total_pipelines"`
	TotalTasks              int                        `json:"total_tasks"`
	SuccessRate             float64                    `json:"success_rate"`
	AveragePipelineDuration float64                    `json:"average_pipeline_duration"`
	AverageTaskDuration     float64                    `json:"average_task_duration"`
	PipelineMetrics         map[string]*PipelineMetric `json:"pipeline_metrics"`
	TaskMetrics             map[string]*TaskMetric     `json:"task_metrics"`
}

// PipelineMetric contains metrics for a specific pipeline
type PipelineMetric struct {
	Name            string  `json:"name"`
	Namespace       string  `json:"namespace"`
	TotalRuns       int     `json:"total_runs"`
	SuccessfulRuns  int     `json:"successful_runs"`
	FailedRuns      int     `json:"failed_runs"`
	RunningRuns     int     `json:"running_runs"`
	AverageDuration float64 `json:"average_duration"`
	P50Duration     float64 `json:"p50_duration"`
	P95Duration     float64 `json:"p95_duration"`
	P99Duration     float64 `json:"p99_duration"`
	LastRunTime     int64   `json:"last_run_time"`
	SuccessRate     float64 `json:"success_rate"`
}

// TaskMetric contains metrics for a specific task
type TaskMetric struct {
	Name            string  `json:"name"`
	Namespace       string  `json:"namespace"`
	TotalRuns       int     `json:"total_runs"`
	SuccessfulRuns  int     `json:"successful_runs"`
	FailedRuns      int     `json:"failed_runs"`
	RunningRuns     int     `json:"running_runs"`
	AverageDuration float64 `json:"average_duration"`
	SuccessRate     float64 `json:"success_rate"`
}

// CostBreakdown represents cost analysis data
type CostBreakdown struct {
	Timestamp      int64                    `json:"timestamp"`
	TotalCost      float64                  `json:"total_cost"`
	CPUCost        float64                  `json:"cpu_cost"`
	MemoryCost     float64                  `json:"memory_cost"`
	StorageCost    float64                  `json:"storage_cost"`
	PipelineCosts  map[string]*PipelineCost `json:"pipeline_costs"`
	NamespaceCosts map[string]float64       `json:"namespace_costs"`
	TrendData      []*CostTrend             `json:"trend_data"`
}

// PipelineCost represents cost data for a specific pipeline
type PipelineCost struct {
	PipelineName      string  `json:"pipeline_name"`
	Namespace         string  `json:"namespace"`
	TotalCost         float64 `json:"total_cost"`
	CPUCost           float64 `json:"cpu_cost"`
	MemoryCost        float64 `json:"memory_cost"`
	StorageCost       float64 `json:"storage_cost"`
	RunCount          int     `json:"run_count"`
	AverageCostPerRun float64 `json:"average_cost_per_run"`
	CPUHours          float64 `json:"cpu_hours"`
	MemoryGBHours     float64 `json:"memory_gb_hours"`
	StorageGBHours    float64 `json:"storage_gb_hours"`
}

// CostTrend represents cost data over time
type CostTrend struct {
	Timestamp   int64   `json:"timestamp"`
	TotalCost   float64 `json:"total_cost"`
	CPUCost     float64 `json:"cpu_cost"`
	MemoryCost  float64 `json:"memory_cost"`
	StorageCost float64 `json:"storage_cost"`
}

// TraceData represents distributed tracing information
type TraceData struct {
	Traces []*Trace `json:"traces"`
}

// Trace represents a single distributed trace
type Trace struct {
	TraceID     string  `json:"trace_id"`
	PipelineRun string  `json:"pipeline_run"`
	Pipeline    string  `json:"pipeline"`
	Namespace   string  `json:"namespace"`
	StartTime   int64   `json:"start_time"`
	EndTime     int64   `json:"end_time"`
	Duration    float64 `json:"duration"`
	Status      string  `json:"status"`
	Spans       []*Span `json:"spans"`
}

// Span represents a single span in a trace
type Span struct {
	SpanID       string            `json:"span_id"`
	ParentSpanID string            `json:"parent_span_id"`
	Name         string            `json:"name"`
	TaskRun      string            `json:"task_run"`
	Task         string            `json:"task"`
	StartTime    int64             `json:"start_time"`
	EndTime      int64             `json:"end_time"`
	Duration     float64           `json:"duration"`
	Status       string            `json:"status"`
	Tags         map[string]string `json:"tags"`
}

// Insights represents AI-powered analytics
type Insights struct {
	Timestamp       int64             `json:"timestamp"`
	Anomalies       []*Anomaly        `json:"anomalies"`
	Recommendations []*Recommendation `json:"recommendations"`
	Predictions     []*Prediction     `json:"predictions"`
}

// Anomaly represents a detected anomaly
type Anomaly struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`     // duration, failure_rate, resource_usage
	Severity    string                 `json:"severity"` // low, medium, high, critical
	Pipeline    string                 `json:"pipeline"`
	Namespace   string                 `json:"namespace"`
	Description string                 `json:"description"`
	DetectedAt  int64                  `json:"detected_at"`
	Score       float64                `json:"score"` // Anomaly score
	Context     map[string]interface{} `json:"context"`
}

// Recommendation represents an optimization recommendation
type Recommendation struct {
	ID          string  `json:"id"`
	Type        string  `json:"type"`     // resource_optimization, cost_reduction, performance
	Priority    string  `json:"priority"` // low, medium, high
	Pipeline    string  `json:"pipeline"`
	Namespace   string  `json:"namespace"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Impact      string  `json:"impact"`  // Estimated impact
	Effort      string  `json:"effort"`  // Implementation effort
	Savings     float64 `json:"savings"` // Estimated cost savings (if applicable)
	CreatedAt   int64   `json:"created_at"`
}

// Prediction represents a predictive analysis result
type Prediction struct {
	ID          string      `json:"id"`
	Type        string      `json:"type"` // failure_prediction, duration_prediction
	Pipeline    string      `json:"pipeline"`
	Namespace   string      `json:"namespace"`
	Description string      `json:"description"`
	Confidence  float64     `json:"confidence"` // 0-1 confidence score
	Value       interface{} `json:"value"`      // Predicted value
	CreatedAt   int64       `json:"created_at"`
}

// OverviewMetrics provides a high-level summary
type OverviewMetrics struct {
	Timestamp           int64   `json:"timestamp"`
	TotalPipelines      int     `json:"total_pipelines"`
	RunningPipelines    int     `json:"running_pipelines"`
	SuccessfulPipelines int     `json:"successful_pipelines"`
	FailedPipelines     int     `json:"failed_pipelines"`
	TotalTasks          int     `json:"total_tasks"`
	RunningTasks        int     `json:"running_tasks"`
	SuccessRate         float64 `json:"success_rate"`
	AverageDuration     float64 `json:"average_duration"`
	TotalCost           float64 `json:"total_cost"`
	CostTrend           string  `json:"cost_trend"` // up, down, stable
	ActiveAnomalies     int     `json:"active_anomalies"`
	OpenRecommendations int     `json:"open_recommendations"`
}

// ControlPlaneStatus represents the overall Tekton control plane health
type ControlPlaneStatus struct {
	Timestamp       int64              `json:"timestamp"`
	OverallHealth   string             `json:"overall_health"` // Healthy, Degraded, Unhealthy
	Components      []*ComponentStatus `json:"components"`
	OperatorManaged bool               `json:"operator_managed"` // True if Tekton Operator is present
	TektonVersion   string             `json:"tekton_version"`
}

// ComponentStatus represents status of one Tekton control plane component
type ComponentStatus struct {
	Name               string                `json:"name"`      // e.g. "Pipelines Controller"
	Component          string                `json:"component"` // e.g. "tekton-pipelines-controller"
	Namespace          string                `json:"namespace"`
	Kind               string                `json:"kind"`   // Deployment, StatefulSet
	Health             string                `json:"health"` // Healthy, Degraded, Unhealthy, Unknown
	ReadyReplicas      int32                 `json:"ready_replicas"`
	DesiredReplicas    int32                 `json:"desired_replicas"`
	Image              string                `json:"image"`
	Version            string                `json:"version"` // extracted from image tag
	Pods               []*PodStatus          `json:"pods"`
	Conditions         []*ComponentCondition `json:"conditions"`
	MetricsEndpoint    string                `json:"metrics_endpoint,omitempty"`
	LastTransitionTime int64                 `json:"last_transition_time"`
}

// PodStatus represents the status of a single pod
type PodStatus struct {
	Name       string           `json:"name"`
	Phase      string           `json:"phase"` // Running, Pending, Succeeded, Failed, Unknown
	Ready      bool             `json:"ready"`
	Restarts   int32            `json:"restarts"`
	Age        int64            `json:"age"` // seconds since creation
	Node       string           `json:"node"`
	IP         string           `json:"ip"`
	Containers []*ContainerInfo `json:"containers"`
}

// ContainerInfo represents a container within a pod
type ContainerInfo struct {
	Name   string `json:"name"`
	Image  string `json:"image"`
	Ready  bool   `json:"ready"`
	State  string `json:"state"` // running, waiting, terminated
	Reason string `json:"reason,omitempty"`
}

// ComponentCondition represents a deployment condition
type ComponentCondition struct {
	Type    string `json:"type"`
	Status  string `json:"status"`
	Reason  string `json:"reason,omitempty"`
	Message string `json:"message,omitempty"`
}
