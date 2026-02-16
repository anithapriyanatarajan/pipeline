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
	"math"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/tektoncd/pipeline/pkg/dashboard"
	"go.uber.org/zap"
	"knative.dev/pkg/logging"
)

// InsightsEngine provides AI-powered analytics and recommendations
type InsightsEngine struct {
	ctx              context.Context
	metricsCollector *MetricsCollector
	costCollector    *CostCollector
	logger           *zap.SugaredLogger
	mu               sync.RWMutex
	insights         *dashboard.Insights
}

// NewInsightsEngine creates a new insights engine
func NewInsightsEngine(ctx context.Context, mc *MetricsCollector, cc *CostCollector) *InsightsEngine {
	return &InsightsEngine{
		ctx:              ctx,
		metricsCollector: mc,
		costCollector:    cc,
		logger:           logging.FromContext(ctx),
		insights: &dashboard.Insights{
			Timestamp:       time.Now().Unix(),
			Anomalies:       make([]*dashboard.Anomaly, 0),
			Recommendations: make([]*dashboard.Recommendation, 0),
			Predictions:     make([]*dashboard.Prediction, 0),
		},
	}
}

// Start begins analyzing data and generating insights
func (ie *InsightsEngine) Start() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	// Generate initial insights
	ie.generateInsights()

	for {
		select {
		case <-ticker.C:
			ie.generateInsights()
		case <-ie.ctx.Done():
			ie.logger.Info("Insights engine stopping")
			return
		}
	}
}

// generateInsights analyzes metrics and generates insights
func (ie *InsightsEngine) generateInsights() {
	ie.logger.Debug("Generating insights...")

	insights := &dashboard.Insights{
		Timestamp:       time.Now().Unix(),
		Anomalies:       ie.detectAnomalies(),
		Recommendations: ie.generateRecommendations(),
		Predictions:     ie.generatePredictions(),
	}

	ie.mu.Lock()
	ie.insights = insights
	ie.mu.Unlock()

	ie.logger.Debugf("Generated %d anomalies, %d recommendations, %d predictions",
		len(insights.Anomalies), len(insights.Recommendations), len(insights.Predictions))
}

// detectAnomalies identifies unusual patterns in metrics
func (ie *InsightsEngine) detectAnomalies() []*dashboard.Anomaly {
	anomalies := make([]*dashboard.Anomaly, 0)

	metrics := ie.metricsCollector.GetLatestMetrics()
	if metrics == nil {
		return anomalies
	}

	// Detect duration anomalies
	for _, pm := range metrics.PipelineMetrics {
		// Check if pipeline duration is significantly higher than average
		if pm.AverageDuration > 0 {
			history := ie.getPipelineHistory(pm.Namespace, pm.Name)
			if len(history) >= 10 {
				avgDuration := ie.calculateAverage(history)
				stdDev := ie.calculateStdDev(history, avgDuration)

				// If current duration is > 2 standard deviations from mean
				if pm.AverageDuration > avgDuration+2*stdDev {
					anomaly := &dashboard.Anomaly{
						ID:          uuid.New().String(),
						Type:        "duration",
						Severity:    ie.calculateSeverity((pm.AverageDuration - avgDuration) / stdDev),
						Pipeline:    pm.Name,
						Namespace:   pm.Namespace,
						Description: fmt.Sprintf("Pipeline duration (%.1fs) is significantly higher than average (%.1fs)", pm.AverageDuration, avgDuration),
						DetectedAt:  time.Now().Unix(),
						Score:       (pm.AverageDuration - avgDuration) / stdDev,
						Context: map[string]interface{}{
							"current_duration": pm.AverageDuration,
							"average_duration": avgDuration,
							"std_dev":          stdDev,
						},
					}
					anomalies = append(anomalies, anomaly)
				}
			}
		}

		// Detect failure rate anomalies
		if pm.TotalRuns >= 10 && pm.SuccessRate < 80 {
			anomaly := &dashboard.Anomaly{
				ID:          uuid.New().String(),
				Type:        "failure_rate",
				Severity:    ie.calculateFailureSeverity(pm.SuccessRate),
				Pipeline:    pm.Name,
				Namespace:   pm.Namespace,
				Description: fmt.Sprintf("Pipeline has low success rate: %.1f%% (%d/%d runs)", pm.SuccessRate, pm.SuccessfulRuns, pm.TotalRuns),
				DetectedAt:  time.Now().Unix(),
				Score:       100 - pm.SuccessRate,
				Context: map[string]interface{}{
					"success_rate":    pm.SuccessRate,
					"total_runs":      pm.TotalRuns,
					"successful_runs": pm.SuccessfulRuns,
					"failed_runs":     pm.FailedRuns,
				},
			}
			anomalies = append(anomalies, anomaly)
		}
	}

	return anomalies
}

// generateRecommendations creates optimization suggestions
func (ie *InsightsEngine) generateRecommendations() []*dashboard.Recommendation {
	recommendations := make([]*dashboard.Recommendation, 0)

	metrics := ie.metricsCollector.GetLatestMetrics()
	costs := ie.costCollector.GetLatestCosts()

	if metrics == nil || costs == nil {
		return recommendations
	}

	// Resource optimization recommendations
	for key, pc := range costs.PipelineCosts {
		// High cost pipeline
		if pc.TotalCost > 10.0 { // More than $10
			pm := metrics.PipelineMetrics[key]
			if pm != nil && pm.TotalRuns > 0 {
				rec := &dashboard.Recommendation{
					ID:          uuid.New().String(),
					Type:        "cost_reduction",
					Priority:    "high",
					Pipeline:    pc.PipelineName,
					Namespace:   pc.Namespace,
					Title:       "High Cost Pipeline",
					Description: fmt.Sprintf("This pipeline has cost $%.2f (avg $%.2f/run). Consider optimizing resource requests or caching dependencies.", pc.TotalCost, pc.AverageCostPerRun),
					Impact:      fmt.Sprintf("Potential savings: $%.2f/week", pc.TotalCost*0.3), // Estimate 30% savings
					Effort:      "medium",
					Savings:     pc.TotalCost * 0.3,
					CreatedAt:   time.Now().Unix(),
				}
				recommendations = append(recommendations, rec)
			}
		}
	}

	// Performance optimization recommendations
	for _, pm := range metrics.PipelineMetrics {
		if pm.AverageDuration > 600 { // More than 10 minutes
			rec := &dashboard.Recommendation{
				ID:          uuid.New().String(),
				Type:        "performance",
				Priority:    "medium",
				Pipeline:    pm.Name,
				Namespace:   pm.Namespace,
				Title:       "Long Pipeline Duration",
				Description: fmt.Sprintf("Pipeline takes average of %.1f seconds. Consider parallelizing tasks or optimizing slow steps.", pm.AverageDuration),
				Impact:      "Faster feedback cycles, improved developer productivity",
				Effort:      "medium",
				CreatedAt:   time.Now().Unix(),
			}
			recommendations = append(recommendations, rec)
		}
	}

	return recommendations
}

// generatePredictions creates predictive analytics
func (ie *InsightsEngine) generatePredictions() []*dashboard.Prediction {
	predictions := make([]*dashboard.Prediction, 0)

	metrics := ie.metricsCollector.GetLatestMetrics()
	if metrics == nil {
		return predictions
	}

	// Predict failure likelihood based on recent trends
	for _, pm := range metrics.PipelineMetrics {
		if pm.TotalRuns >= 5 {
			// Simple prediction based on recent success rate
			failureProbability := (100 - pm.SuccessRate) / 100

			if failureProbability > 0.2 { // More than 20% failure rate
				pred := &dashboard.Prediction{
					ID:          uuid.New().String(),
					Type:        "failure_prediction",
					Pipeline:    pm.Name,
					Namespace:   pm.Namespace,
					Description: fmt.Sprintf("High probability (%.0f%%) of failure in next run based on recent trends", failureProbability*100),
					Confidence:  ie.calculateConfidence(pm.TotalRuns),
					Value:       failureProbability,
					CreatedAt:   time.Now().Unix(),
				}
				predictions = append(predictions, pred)
			}
		}
	}

	return predictions
}

// Helper functions

func (ie *InsightsEngine) getPipelineHistory(namespace, pipeline string) []float64 {
	history := ie.metricsCollector.GetMetricsHistory(time.Now().Add(-24 * time.Hour))
	durations := make([]float64, 0)

	key := fmt.Sprintf("%s/%s", namespace, pipeline)
	for _, snapshot := range history {
		if pm, ok := snapshot.PipelineMetrics[key]; ok && pm.AverageDuration > 0 {
			durations = append(durations, pm.AverageDuration)
		}
	}

	return durations
}

func (ie *InsightsEngine) calculateAverage(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func (ie *InsightsEngine) calculateStdDev(values []float64, mean float64) float64 {
	if len(values) == 0 {
		return 0
	}

	variance := 0.0
	for _, v := range values {
		variance += math.Pow(v-mean, 2)
	}
	variance /= float64(len(values))

	return math.Sqrt(variance)
}

func (ie *InsightsEngine) calculateSeverity(score float64) string {
	if score > 3 {
		return "critical"
	} else if score > 2 {
		return "high"
	} else if score > 1 {
		return "medium"
	}
	return "low"
}

func (ie *InsightsEngine) calculateFailureSeverity(successRate float64) string {
	if successRate < 50 {
		return "critical"
	} else if successRate < 70 {
		return "high"
	} else if successRate < 85 {
		return "medium"
	}
	return "low"
}

func (ie *InsightsEngine) calculateConfidence(sampleSize int) float64 {
	// More samples = higher confidence (max 0.95)
	confidence := math.Min(0.95, float64(sampleSize)/50)
	return confidence
}

// GetInsights returns the latest insights
func (ie *InsightsEngine) GetInsights() *dashboard.Insights {
	ie.mu.RLock()
	defer ie.mu.RUnlock()
	return ie.insights
}

// GetAnomalies returns all detected anomalies
func (ie *InsightsEngine) GetAnomalies() []*dashboard.Anomaly {
	ie.mu.RLock()
	defer ie.mu.RUnlock()
	return ie.insights.Anomalies
}

// GetRecommendations returns all recommendations
func (ie *InsightsEngine) GetRecommendations() []*dashboard.Recommendation {
	ie.mu.RLock()
	defer ie.mu.RUnlock()
	return ie.insights.Recommendations
}
