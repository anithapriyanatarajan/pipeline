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

package collectors

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/common/model"
	"github.com/tektoncd/pipeline/pkg/dashboard"
	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes"
	"knative.dev/pkg/logging"
)

// MetricsCollector collects and aggregates pipeline metrics
type MetricsCollector struct {
	ctx            context.Context
	kubeClient     kubernetes.Interface
	config         *dashboard.Config
	logger         *zap.SugaredLogger
	mu             sync.RWMutex
	latestMetrics  *dashboard.MetricsSnapshot
	metricsHistory []*dashboard.MetricsSnapshot
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector(ctx context.Context, kubeClient kubernetes.Interface, config *dashboard.Config) *MetricsCollector {
	return &MetricsCollector{
		ctx:            ctx,
		kubeClient:     kubeClient,
		config:         config,
		logger:         logging.FromContext(ctx),
		metricsHistory: make([]*dashboard.MetricsSnapshot, 0, 1000),
	}
}

// Start begins collecting metrics
func (mc *MetricsCollector) Start() {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	// Collect immediately on start
	mc.collectMetrics()

	for {
		select {
		case <-ticker.C:
			mc.collectMetrics()
		case <-mc.ctx.Done():
			mc.logger.Info("Metrics collector stopping")
			return
		}
	}
}

// collectMetrics fetches and processes metrics from Prometheus endpoint
func (mc *MetricsCollector) collectMetrics() {
	resp, err := http.Get(mc.config.MetricsEndpoint)
	if err != nil {
		mc.logger.Warnf("Failed to fetch metrics: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		mc.logger.Warnf("Metrics endpoint returned status %d", resp.StatusCode)
		return
	}

	metrics, err := mc.parsePrometheusMetrics(resp.Body)
	if err != nil {
		mc.logger.Warnf("Failed to parse metrics: %v", err)
		return
	}

	snapshot := mc.aggregateMetrics(metrics)

	mc.mu.Lock()
	mc.latestMetrics = snapshot
	mc.metricsHistory = append(mc.metricsHistory, snapshot)

	// Keep only last 24 hours of data (at 15s intervals = 5760 snapshots)
	if len(mc.metricsHistory) > 5760 {
		mc.metricsHistory = mc.metricsHistory[len(mc.metricsHistory)-5760:]
	}
	mc.mu.Unlock()
}

// parsePrometheusMetrics parses Prometheus text format
func (mc *MetricsCollector) parsePrometheusMetrics(r io.Reader) (map[string][]*model.Sample, error) {
	var parser expfmt.TextParser
	metricFamilies, err := parser.TextToMetricFamilies(r)
	if err != nil {
		return nil, fmt.Errorf("failed to parse metrics: %w", err)
	}

	metrics := make(map[string][]*model.Sample)

	for name, mf := range metricFamilies {
		samples := make([]*model.Sample, 0)

		for _, m := range mf.Metric {
			labels := make(model.LabelSet)
			for _, l := range m.Label {
				labels[model.LabelName(l.GetName())] = model.LabelValue(l.GetValue())
			}

			switch mf.GetType() {
			case dto.MetricType_COUNTER:
				samples = append(samples, &model.Sample{
					Metric:    model.Metric(labels),
					Value:     model.SampleValue(m.Counter.GetValue()),
					Timestamp: model.Now(),
				})
			case dto.MetricType_GAUGE:
				samples = append(samples, &model.Sample{
					Metric:    model.Metric(labels),
					Value:     model.SampleValue(m.Gauge.GetValue()),
					Timestamp: model.Now(),
				})
			case dto.MetricType_HISTOGRAM:
				if m.Histogram != nil {
					// Emit _count and _sum as separate synthetic metrics
					// so aggregation can look them up by key.
					countKey := name + "_count"
					sumKey := name + "_sum"
					if _, ok := metrics[countKey]; !ok {
						metrics[countKey] = make([]*model.Sample, 0)
					}
					if _, ok := metrics[sumKey]; !ok {
						metrics[sumKey] = make([]*model.Sample, 0)
					}
					metrics[countKey] = append(metrics[countKey], &model.Sample{
						Metric:    model.Metric(labels),
						Value:     model.SampleValue(m.Histogram.GetSampleCount()),
						Timestamp: model.Now(),
					})
					metrics[sumKey] = append(metrics[sumKey], &model.Sample{
						Metric:    model.Metric(labels),
						Value:     model.SampleValue(m.Histogram.GetSampleSum()),
						Timestamp: model.Now(),
					})
					// Also add a sample for the histogram family itself
					samples = append(samples, &model.Sample{
						Metric:    model.Metric(labels),
						Value:     model.SampleValue(m.Histogram.GetSampleSum()),
						Timestamp: model.Now(),
					})
				}
			case dto.MetricType_SUMMARY:
				if m.Summary != nil {
					samples = append(samples, &model.Sample{
						Metric:    model.Metric(labels),
						Value:     model.SampleValue(m.Summary.GetSampleSum()),
						Timestamp: model.Now(),
					})
				}
			case dto.MetricType_UNTYPED:
				samples = append(samples, &model.Sample{
					Metric:    model.Metric(labels),
					Value:     model.SampleValue(m.Untyped.GetValue()),
					Timestamp: model.Now(),
				})
			}
		}

		metrics[name] = samples
	}

	return metrics, nil
}

// aggregateMetrics processes raw metrics into structured snapshot
func (mc *MetricsCollector) aggregateMetrics(rawMetrics map[string][]*model.Sample) *dashboard.MetricsSnapshot {
	snapshot := &dashboard.MetricsSnapshot{
		Timestamp:       time.Now().Unix(),
		PipelineMetrics: make(map[string]*dashboard.PipelineMetric),
		TaskMetrics:     make(map[string]*dashboard.TaskMetric),
	}

	// Aggregate pipeline metrics from gauges
	if samples, ok := rawMetrics["tekton_pipelines_controller_running_pipelineruns"]; ok {
		for _, s := range samples {
			snapshot.RunningPipelines += int(s.Value)
		}
	}

	if samples, ok := rawMetrics["tekton_pipelines_controller_running_taskruns"]; ok {
		for _, s := range samples {
			snapshot.RunningTasks += int(s.Value)
		}
	}

	// Process pipeline duration histogram metrics.
	// expfmt parses "pipelinerun_duration_seconds_count" as part of the
	// histogram family "pipelinerun_duration_seconds", so we need to build
	// per-label-set counts from the Histogram.SampleCount field.
	mc.aggregateHistogramCounts(rawMetrics,
		"tekton_pipelines_controller_pipelinerun_duration_seconds",
		func(labels model.Metric, count uint64, sumSeconds float64) {
			pipeline := string(labels["pipeline"])
			namespace := string(labels["namespace"])
			status := string(labels["status"])

			key := fmt.Sprintf("%s/%s", namespace, pipeline)
			if _, exists := snapshot.PipelineMetrics[key]; !exists {
				snapshot.PipelineMetrics[key] = &dashboard.PipelineMetric{
					Name:      pipeline,
					Namespace: namespace,
				}
			}

			pm := snapshot.PipelineMetrics[key]
			pm.TotalRuns += int(count)

			if status == "success" {
				pm.SuccessfulRuns += int(count)
			} else if status == "failed" {
				pm.FailedRuns += int(count)
			}

			// Accumulate durations for average calculation
			if count > 0 {
				pm.AverageDuration = sumSeconds / float64(count)
			}
		})

	// Calculate success rates and totals
	for _, pm := range snapshot.PipelineMetrics {
		if pm.TotalRuns > 0 {
			pm.SuccessRate = float64(pm.SuccessfulRuns) / float64(pm.TotalRuns) * 100
		}
		snapshot.TotalPipelines += pm.TotalRuns
		snapshot.SuccessfulPipelines += pm.SuccessfulRuns
		snapshot.FailedPipelines += pm.FailedRuns
	}

	// Compute overall success rate
	totalFinished := snapshot.SuccessfulPipelines + snapshot.FailedPipelines
	if totalFinished > 0 {
		snapshot.SuccessRate = float64(snapshot.SuccessfulPipelines) / float64(totalFinished) * 100
	}

	// Process task duration histogram metrics
	mc.aggregateHistogramCounts(rawMetrics,
		"tekton_pipelines_controller_pipelinerun_taskrun_duration_seconds",
		func(labels model.Metric, count uint64, sumSeconds float64) {
			task := string(labels["task"])
			namespace := string(labels["namespace"])
			status := string(labels["status"])

			key := fmt.Sprintf("%s/%s", namespace, task)
			if _, exists := snapshot.TaskMetrics[key]; !exists {
				snapshot.TaskMetrics[key] = &dashboard.TaskMetric{
					Name:      task,
					Namespace: namespace,
				}
			}

			tm := snapshot.TaskMetrics[key]
			tm.TotalRuns += int(count)

			if status == "success" {
				tm.SuccessfulRuns += int(count)
			} else if status == "failed" {
				tm.FailedRuns += int(count)
			}

			if count > 0 {
				tm.AverageDuration = sumSeconds / float64(count)
			}
		})

	for _, tm := range snapshot.TaskMetrics {
		if tm.TotalRuns > 0 {
			tm.SuccessRate = float64(tm.SuccessfulRuns) / float64(tm.TotalRuns) * 100
		}
		snapshot.TotalTasks += tm.TotalRuns
	}

	return snapshot
}

// aggregateHistogramCounts iterates samples for a histogram metric and calls fn
// with the per-label-set count and sum values that expfmt inlines from the
// _count and _sum sub-metrics of the histogram family.
func (mc *MetricsCollector) aggregateHistogramCounts(
	rawMetrics map[string][]*model.Sample,
	familyName string,
	fn func(labels model.Metric, count uint64, sumSeconds float64),
) {
	samples, ok := rawMetrics[familyName]
	if !ok {
		return
	}
	for _, s := range samples {
		// The sample value for histograms is the _sum. But we stored it that
		// way in parsePrometheusMetrics. We need the count too. Unfortunately
		// our current parsing flattens histograms into a single sample with
		// value = SampleSum. We should fix the parser to emit count as well.
		// For now, let's store count separately.
		// Actually the samples here have value = SampleSum. We need a different
		// approach — read count from the raw metric families directly.
		_ = s
	}
	// Histogram data in our current model only has the sum, not the count.
	// We need to fix the parser. For now, fall back to looking for synthetic
	// _count keys, or reparse differently.
	// Let's try the synthetic keys that expfmt sometimes puts into the map:
	if countSamples, ok2 := rawMetrics[familyName+"_count"]; ok2 {
		for _, cs := range countSamples {
			count := uint64(cs.Value)
			// Find matching sum
			var sum float64
			if sumSamples, ok3 := rawMetrics[familyName+"_sum"]; ok3 {
				for _, ss := range sumSamples {
					if ss.Metric.Equal(cs.Metric) {
						sum = float64(ss.Value)
						break
					}
				}
			}
			fn(cs.Metric, count, sum)
		}
	}
}

// GetLatestMetrics returns the most recent metrics snapshot
func (mc *MetricsCollector) GetLatestMetrics() *dashboard.MetricsSnapshot {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	return mc.latestMetrics
}

// GetMetricsHistory returns historical metrics
func (mc *MetricsCollector) GetMetricsHistory(since time.Time) []*dashboard.MetricsSnapshot {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	result := make([]*dashboard.MetricsSnapshot, 0)
	sinceUnix := since.Unix()

	for _, snapshot := range mc.metricsHistory {
		if snapshot.Timestamp >= sinceUnix {
			result = append(result, snapshot)
		}
	}

	return result
}

// GetOverviewMetrics returns high-level summary metrics
func (mc *MetricsCollector) GetOverviewMetrics() *dashboard.OverviewMetrics {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	if mc.latestMetrics == nil {
		return &dashboard.OverviewMetrics{
			Timestamp: time.Now().Unix(),
		}
	}

	snapshot := mc.latestMetrics

	overview := &dashboard.OverviewMetrics{
		Timestamp:           snapshot.Timestamp,
		TotalPipelines:      snapshot.TotalPipelines,
		RunningPipelines:    snapshot.RunningPipelines,
		SuccessfulPipelines: snapshot.SuccessfulPipelines,
		FailedPipelines:     snapshot.FailedPipelines,
		TotalTasks:          snapshot.TotalTasks,
		RunningTasks:        snapshot.RunningTasks,
		AverageDuration:     snapshot.AveragePipelineDuration,
	}

	if snapshot.TotalPipelines > 0 {
		overview.SuccessRate = float64(snapshot.SuccessfulPipelines) / float64(snapshot.TotalPipelines) * 100
	}

	return overview
}
