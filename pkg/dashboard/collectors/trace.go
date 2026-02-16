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
	"sync"
	"time"

	tektonClient "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	"github.com/tektoncd/pipeline/pkg/dashboard"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"knative.dev/pkg/logging"
)

// TraceCollector collects and aggregates distributed traces
type TraceCollector struct {
	ctx          context.Context
	kubeClient   kubernetes.Interface
	tektonClient tektonClient.Interface
	logger       *zap.SugaredLogger
	mu           sync.RWMutex
	traces       map[string]*dashboard.Trace
}

// NewTraceCollector creates a new trace collector
func NewTraceCollector(ctx context.Context, kubeClient kubernetes.Interface, tektonCl tektonClient.Interface) *TraceCollector {
	return &TraceCollector{
		ctx:          ctx,
		kubeClient:   kubeClient,
		tektonClient: tektonCl,
		logger:       logging.FromContext(ctx),
		traces:       make(map[string]*dashboard.Trace),
	}
}

// Start begins collecting traces
func (tc *TraceCollector) Start() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	tc.collectTraces()

	for {
		select {
		case <-ticker.C:
			tc.collectTraces()
		case <-tc.ctx.Done():
			tc.logger.Info("Trace collector stopping")
			return
		}
	}
}

// collectTraces builds trace data from PipelineRuns and TaskRuns
func (tc *TraceCollector) collectTraces() {
	tc.logger.Debug("Collecting trace data...")

	if tc.tektonClient == nil {
		return
	}

	// List recent PipelineRuns
	prList, err := tc.tektonClient.TektonV1().PipelineRuns("").List(tc.ctx, metav1.ListOptions{})
	if err != nil {
		tc.logger.Warnf("Failed to list pipeline runs for traces: %v", err)
		return
	}

	// List all TaskRuns
	trList, err := tc.tektonClient.TektonV1().TaskRuns("").List(tc.ctx, metav1.ListOptions{})
	if err != nil {
		tc.logger.Warnf("Failed to list task runs for traces: %v", err)
		return
	}

	// Build a map of TaskRuns by owner PipelineRun
	taskRunsByPR := make(map[string][]metav1.Object)
	for i := range trList.Items {
		tr := &trList.Items[i]
		for _, owner := range tr.OwnerReferences {
			if owner.Kind == "PipelineRun" {
				taskRunsByPR[owner.Name] = append(taskRunsByPR[owner.Name], tr)
			}
		}
	}

	tc.mu.Lock()
	defer tc.mu.Unlock()

	// Clean up old traces (older than 1 hour)
	cutoff := time.Now().Add(-1 * time.Hour).Unix()
	for traceID, trace := range tc.traces {
		if trace.EndTime > 0 && trace.EndTime < cutoff {
			delete(tc.traces, traceID)
		}
	}

	// Build traces from PipelineRuns
	for i := range prList.Items {
		pr := &prList.Items[i]
		traceID := fmt.Sprintf("pr-%s-%s", pr.Namespace, pr.Name)

		var startTime, endTime int64
		var duration float64
		status := "Unknown"

		if pr.Status.StartTime != nil {
			startTime = pr.Status.StartTime.Time.Unix()
		} else {
			startTime = pr.CreationTimestamp.Unix()
		}
		if pr.Status.CompletionTime != nil {
			endTime = pr.Status.CompletionTime.Time.Unix()
			duration = float64(endTime - startTime)
		} else if startTime > 0 {
			endTime = time.Now().Unix()
			duration = float64(endTime - startTime)
		}

		if len(pr.Status.Conditions) > 0 {
			cond := pr.Status.Conditions[0]
			if cond.IsTrue() {
				status = "Succeeded"
			} else if cond.IsFalse() {
				status = "Failed"
			} else {
				status = "Running"
			}
		}

		pipelineName := pr.Name
		if pr.Spec.PipelineRef != nil {
			pipelineName = pr.Spec.PipelineRef.Name
		}

		trace := &dashboard.Trace{
			TraceID:     traceID,
			PipelineRun: pr.Name,
			Pipeline:    pipelineName,
			Namespace:   pr.Namespace,
			StartTime:   startTime,
			EndTime:     endTime,
			Duration:    duration,
			Status:      status,
			Spans:       make([]*dashboard.Span, 0),
		}

		// Build spans from child TaskRuns
		for _, childTR := range trList.Items {
			owned := false
			for _, owner := range childTR.OwnerReferences {
				if owner.Kind == "PipelineRun" && owner.Name == pr.Name {
					owned = true
					break
				}
			}
			if !owned {
				continue
			}

			var trStart, trEnd int64
			var trDuration float64
			trStatus := "Unknown"

			if childTR.Status.StartTime != nil {
				trStart = childTR.Status.StartTime.Time.Unix()
			}
			if childTR.Status.CompletionTime != nil {
				trEnd = childTR.Status.CompletionTime.Time.Unix()
				if trStart > 0 {
					trDuration = float64(trEnd - trStart)
				}
			}
			if len(childTR.Status.Conditions) > 0 {
				cond := childTR.Status.Conditions[0]
				if cond.IsTrue() {
					trStatus = "Succeeded"
				} else if cond.IsFalse() {
					trStatus = "Failed"
				} else {
					trStatus = "Running"
				}
			}

			taskName := childTR.Name
			if childTR.Spec.TaskRef != nil {
				taskName = childTR.Spec.TaskRef.Name
			}

			span := &dashboard.Span{
				SpanID:       fmt.Sprintf("tr-%s", childTR.Name),
				ParentSpanID: traceID,
				Name:         taskName,
				TaskRun:      childTR.Name,
				Task:         taskName,
				StartTime:    trStart,
				EndTime:      trEnd,
				Duration:     trDuration,
				Status:       trStatus,
				Tags: map[string]string{
					"namespace": childTR.Namespace,
				},
			}
			trace.Spans = append(trace.Spans, span)
		}

		tc.traces[traceID] = trace
	}
}

// GetTraces returns all active traces
func (tc *TraceCollector) GetTraces() *dashboard.TraceData {
	tc.mu.RLock()
	defer tc.mu.RUnlock()

	traces := make([]*dashboard.Trace, 0, len(tc.traces))
	for _, trace := range tc.traces {
		traces = append(traces, trace)
	}

	return &dashboard.TraceData{
		Traces: traces,
	}
}

// GetTrace returns a specific trace by ID
func (tc *TraceCollector) GetTrace(traceID string) *dashboard.Trace {
	tc.mu.RLock()
	defer tc.mu.RUnlock()

	return tc.traces[traceID]
}
