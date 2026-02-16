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

	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	tektonClient "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	"github.com/tektoncd/pipeline/pkg/dashboard"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"knative.dev/pkg/logging"
)

// CostCollector tracks resource usage and calculates costs
type CostCollector struct {
	ctx          context.Context
	kubeClient   kubernetes.Interface
	tektonClient tektonClient.Interface
	config       *dashboard.Config
	logger       *zap.SugaredLogger
	mu           sync.RWMutex
	latestCosts  *dashboard.CostBreakdown
	costHistory  []*dashboard.CostTrend
}

// NewCostCollector creates a new cost collector
func NewCostCollector(ctx context.Context, kubeClient kubernetes.Interface, tektonCl tektonClient.Interface, config *dashboard.Config) *CostCollector {
	return &CostCollector{
		ctx:          ctx,
		kubeClient:   kubeClient,
		tektonClient: tektonCl,
		config:       config,
		logger:       logging.FromContext(ctx),
		costHistory:  make([]*dashboard.CostTrend, 0),
	}
}

// Start begins collecting cost data
func (cc *CostCollector) Start() {
	if !cc.config.EnableCostTracking {
		cc.logger.Info("Cost tracking is disabled")
		return
	}

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	// Collect immediately on start
	cc.collectCosts()

	for {
		select {
		case <-ticker.C:
			cc.collectCosts()
		case <-cc.ctx.Done():
			cc.logger.Info("Cost collector stopping")
			return
		}
	}
}

// collectCosts calculates costs based on resource usage
func (cc *CostCollector) collectCosts() {
	cc.logger.Debug("Collecting cost data...")

	breakdown := &dashboard.CostBreakdown{
		Timestamp:      time.Now().Unix(),
		PipelineCosts:  make(map[string]*dashboard.PipelineCost),
		NamespaceCosts: make(map[string]float64),
		TrendData:      make([]*dashboard.CostTrend, 0),
	}

	// Get all PipelineRuns from last 24 hours
	pipelineRuns, err := cc.getPipelineRuns(24 * time.Hour)
	if err != nil {
		cc.logger.Warnf("Failed to get pipeline runs: %v", err)
		return
	}

	// Calculate costs for each pipeline run
	for _, pr := range pipelineRuns {
		cost := cc.calculatePipelineRunCost(pr)
		if cost == nil {
			continue
		}

		pipelineName := cost.PipelineName
		if pipelineName == "" {
			pipelineName = pr.Name
		}
		key := fmt.Sprintf("%s/%s", pr.Namespace, pipelineName)

		if existing, ok := breakdown.PipelineCosts[key]; ok {
			existing.TotalCost += cost.TotalCost
			existing.CPUCost += cost.CPUCost
			existing.MemoryCost += cost.MemoryCost
			existing.StorageCost += cost.StorageCost
			existing.RunCount++
			existing.CPUHours += cost.CPUHours
			existing.MemoryGBHours += cost.MemoryGBHours
			existing.StorageGBHours += cost.StorageGBHours
		} else {
			breakdown.PipelineCosts[key] = cost
		}

		breakdown.TotalCost += cost.TotalCost
		breakdown.CPUCost += cost.CPUCost
		breakdown.MemoryCost += cost.MemoryCost
		breakdown.StorageCost += cost.StorageCost

		breakdown.NamespaceCosts[pr.Namespace] += cost.TotalCost
	}

	// Calculate average cost per run
	for _, pc := range breakdown.PipelineCosts {
		if pc.RunCount > 0 {
			pc.AverageCostPerRun = pc.TotalCost / float64(pc.RunCount)
		}
	}

	// Add to trend history
	trend := &dashboard.CostTrend{
		Timestamp:   breakdown.Timestamp,
		TotalCost:   breakdown.TotalCost,
		CPUCost:     breakdown.CPUCost,
		MemoryCost:  breakdown.MemoryCost,
		StorageCost: breakdown.StorageCost,
	}

	cc.mu.Lock()
	cc.latestCosts = breakdown
	cc.costHistory = append(cc.costHistory, trend)

	// Keep only last 7 days of trends (at 5 min intervals = 2016 data points)
	if len(cc.costHistory) > 2016 {
		cc.costHistory = cc.costHistory[len(cc.costHistory)-2016:]
	}
	cc.mu.Unlock()

	cc.logger.Debugf("Collected cost data: total=$%.2f, pipelines=%d", breakdown.TotalCost, len(breakdown.PipelineCosts))
}

// getPipelineRuns retrieves pipeline runs from the specified duration
func (cc *CostCollector) getPipelineRuns(duration time.Duration) ([]*v1.PipelineRun, error) {
	if cc.tektonClient == nil {
		cc.logger.Debug("Tekton client not available, skipping pipeline run collection")
		return []*v1.PipelineRun{}, nil
	}

	prList, err := cc.tektonClient.TektonV1().PipelineRuns("").List(cc.ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list pipeline runs: %w", err)
	}

	cutoff := time.Now().Add(-duration)
	result := make([]*v1.PipelineRun, 0)
	for i := range prList.Items {
		pr := &prList.Items[i]
		if pr.Status.StartTime != nil && pr.Status.StartTime.Time.After(cutoff) {
			result = append(result, pr)
		} else if pr.CreationTimestamp.Time.After(cutoff) {
			result = append(result, pr)
		}
	}

	return result, nil
}

// calculatePipelineRunCost calculates the cost of a single pipeline run
func (cc *CostCollector) calculatePipelineRunCost(pr *v1.PipelineRun) *dashboard.PipelineCost {
	if pr.Status.StartTime == nil {
		return nil
	}

	var endTime time.Time
	if pr.Status.CompletionTime != nil {
		endTime = pr.Status.CompletionTime.Time
	} else {
		endTime = time.Now()
	}

	durationHours := endTime.Sub(pr.Status.StartTime.Time).Hours()

	// Estimate resource usage (in real impl, would get from pod metrics)
	// For demo purposes, using conservative estimates
	avgCPUCores := 1.0   // Average CPU cores used
	avgMemoryGB := 2.0   // Average memory in GB
	avgStorageGB := 10.0 // Average storage in GB

	cpuHours := avgCPUCores * durationHours
	memoryGBHours := avgMemoryGB * durationHours
	storageGBHours := avgStorageGB * durationHours

	cpuCost := cpuHours * cc.config.CPUCostPerHour
	memoryCost := memoryGBHours * cc.config.MemoryCostPerGBHour
	storageCost := storageGBHours * cc.config.StorageCostPerGBHour

	pipelineName := ""
	if pr.Spec.PipelineRef != nil {
		pipelineName = pr.Spec.PipelineRef.Name
	}

	return &dashboard.PipelineCost{
		PipelineName:      pipelineName,
		Namespace:         pr.Namespace,
		TotalCost:         cpuCost + memoryCost + storageCost,
		CPUCost:           cpuCost,
		MemoryCost:        memoryCost,
		StorageCost:       storageCost,
		RunCount:          1,
		AverageCostPerRun: cpuCost + memoryCost + storageCost,
		CPUHours:          cpuHours,
		MemoryGBHours:     memoryGBHours,
		StorageGBHours:    storageGBHours,
	}
}

// GetLatestCosts returns the most recent cost breakdown
func (cc *CostCollector) GetLatestCosts() *dashboard.CostBreakdown {
	cc.mu.RLock()
	defer cc.mu.RUnlock()

	if cc.latestCosts == nil {
		return &dashboard.CostBreakdown{
			Timestamp:      time.Now().Unix(),
			PipelineCosts:  make(map[string]*dashboard.PipelineCost),
			NamespaceCosts: make(map[string]float64),
			TrendData:      make([]*dashboard.CostTrend, 0),
		}
	}

	// Add trend history to the breakdown
	result := *cc.latestCosts
	result.TrendData = cc.costHistory

	return &result
}

// GetCostTrend returns cost trend data for the specified duration
func (cc *CostCollector) GetCostTrend(duration time.Duration) []*dashboard.CostTrend {
	cc.mu.RLock()
	defer cc.mu.RUnlock()

	since := time.Now().Add(-duration).Unix()
	result := make([]*dashboard.CostTrend, 0)

	for _, trend := range cc.costHistory {
		if trend.Timestamp >= since {
			result = append(result, trend)
		}
	}

	return result
}

// GetPipelineCostBreakdown returns detailed cost breakdown for a specific pipeline
func (cc *CostCollector) GetPipelineCostBreakdown(namespace, pipeline string) *dashboard.PipelineCost {
	cc.mu.RLock()
	defer cc.mu.RUnlock()

	if cc.latestCosts == nil {
		return nil
	}

	key := fmt.Sprintf("%s/%s", namespace, pipeline)
	return cc.latestCosts.PipelineCosts[key]
}
