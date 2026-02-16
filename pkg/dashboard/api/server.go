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

package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/tektoncd/pipeline/pkg/dashboard/collectors"
	"go.uber.org/zap"
)

// ServerConfig holds the API server configuration
type ServerConfig struct {
	Port                  string
	MetricsCollector      *collectors.MetricsCollector
	CostCollector         *collectors.CostCollector
	TraceCollector        *collectors.TraceCollector
	InsightsEngine        *collectors.InsightsEngine
	ControlPlaneCollector *collectors.ControlPlaneCollector
	Logger                *zap.SugaredLogger
}

// Server represents the dashboard API server
type Server struct {
	config   *ServerConfig
	router   *http.ServeMux
	upgrader websocket.Upgrader
}

// NewServer creates a new API server
func NewServer(config *ServerConfig) *Server {
	s := &Server{
		config: config,
		router: http.NewServeMux(),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for demo
			},
		},
	}

	s.setupRoutes()
	return s
}

// setupRoutes configures all API routes
func (s *Server) setupRoutes() {
	// Metrics endpoints
	s.router.HandleFunc("/api/v1/metrics/overview", s.methodFilter(s.handleOverviewMetrics, "GET"))
	s.router.HandleFunc("/api/v1/metrics/pipelines", s.methodFilter(s.handlePipelineMetrics, "GET"))
	s.router.HandleFunc("/api/v1/metrics/tasks", s.methodFilter(s.handleTaskMetrics, "GET"))
	s.router.HandleFunc("/api/v1/metrics/history", s.methodFilter(s.handleMetricsHistory, "GET"))

	// Cost endpoints
	s.router.HandleFunc("/api/v1/costs/breakdown", s.methodFilter(s.handleCostBreakdown, "GET"))
	s.router.HandleFunc("/api/v1/costs/trend", s.methodFilter(s.handleCostTrend, "GET"))
	s.router.HandleFunc("/api/v1/costs/pipeline/", s.methodFilter(s.handlePipelineCost, "GET"))

	// Trace endpoints
	s.router.HandleFunc("/api/v1/traces", s.handleTraces)
	s.router.HandleFunc("/api/v1/traces/", s.handleTrace)

	// Insights endpoints
	s.router.HandleFunc("/api/v1/insights", s.methodFilter(s.handleInsights, "GET"))
	s.router.HandleFunc("/api/v1/insights/anomalies", s.methodFilter(s.handleAnomalies, "GET"))
	s.router.HandleFunc("/api/v1/insights/recommendations", s.methodFilter(s.handleRecommendations, "GET"))
	s.router.HandleFunc("/api/v1/insights/predictions", s.methodFilter(s.handlePredictions, "GET"))

	// Control plane endpoints
	s.router.HandleFunc("/api/v1/controlplane/status", s.methodFilter(s.handleControlPlaneStatus, "GET"))

	// WebSocket endpoints
	s.router.HandleFunc("/api/v1/stream/metrics", s.handleMetricsStream)
	s.router.HandleFunc("/api/v1/stream/events", s.handleEventsStream)

	// Health endpoint
	s.router.HandleFunc("/api/v1/health", s.methodFilter(s.handleHealth, "GET"))

	// Static file server for UI
	s.router.Handle("/", http.FileServer(http.Dir("./web/dashboard/build")))
}

// Handler returns the HTTP handler
func (s *Server) Handler() http.Handler {
	return s.enableCORS(s.router)
}

// Metrics handlers

func (s *Server) handleOverviewMetrics(w http.ResponseWriter, r *http.Request) {
	overview := s.config.MetricsCollector.GetOverviewMetrics()

	// Enrich with cost and insight data
	costs := s.config.CostCollector.GetLatestCosts()
	if costs != nil {
		overview.TotalCost = costs.TotalCost
	}

	insights := s.config.InsightsEngine.GetInsights()
	if insights != nil {
		overview.ActiveAnomalies = len(insights.Anomalies)
		overview.OpenRecommendations = len(insights.Recommendations)
	}

	s.respondJSON(w, overview)
}

func (s *Server) handlePipelineMetrics(w http.ResponseWriter, r *http.Request) {
	metrics := s.config.MetricsCollector.GetLatestMetrics()
	if metrics != nil {
		s.respondJSON(w, metrics.PipelineMetrics)
	} else {
		s.respondJSON(w, map[string]interface{}{})
	}
}

func (s *Server) handleTaskMetrics(w http.ResponseWriter, r *http.Request) {
	metrics := s.config.MetricsCollector.GetLatestMetrics()
	if metrics != nil {
		s.respondJSON(w, metrics.TaskMetrics)
	} else {
		s.respondJSON(w, map[string]interface{}{})
	}
}

func (s *Server) handleMetricsHistory(w http.ResponseWriter, r *http.Request) {
	// Default to last hour
	duration := time.Hour
	if durationParam := r.URL.Query().Get("duration"); durationParam != "" {
		if d, err := time.ParseDuration(durationParam); err == nil {
			duration = d
		}
	}

	history := s.config.MetricsCollector.GetMetricsHistory(time.Now().Add(-duration))
	s.respondJSON(w, history)
}

// Cost handlers

func (s *Server) handleCostBreakdown(w http.ResponseWriter, r *http.Request) {
	costs := s.config.CostCollector.GetLatestCosts()
	s.respondJSON(w, costs)
}

func (s *Server) handleCostTrend(w http.ResponseWriter, r *http.Request) {
	duration := 24 * time.Hour
	if durationParam := r.URL.Query().Get("duration"); durationParam != "" {
		if d, err := time.ParseDuration(durationParam); err == nil {
			duration = d
		}
	}

	trend := s.config.CostCollector.GetCostTrend(duration)
	s.respondJSON(w, trend)
}

func (s *Server) handlePipelineCost(w http.ResponseWriter, r *http.Request) {
	// Parse namespace and pipeline from URL path
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/costs/pipeline/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	namespace := parts[0]
	pipeline := parts[1]
	cost := s.config.CostCollector.GetPipelineCostBreakdown(namespace, pipeline)
	if cost != nil {
		s.respondJSON(w, cost)
	} else {
		http.NotFound(w, r)
	}
}

// Trace handlers

func (s *Server) handleTraces(w http.ResponseWriter, r *http.Request) {
	traces := s.config.TraceCollector.GetTraces()
	s.respondJSON(w, traces)
}

func (s *Server) handleTrace(w http.ResponseWriter, r *http.Request) {
	// Parse traceId from URL path
	traceID := strings.TrimPrefix(r.URL.Path, "/api/v1/traces/")
	if traceID == "" {
		http.Error(w, "Trace ID required", http.StatusBadRequest)
		return
	}
	trace := s.config.TraceCollector.GetTrace(traceID)
	if trace != nil {
		s.respondJSON(w, trace)
	} else {
		http.NotFound(w, r)
	}
}

// Insights handlers

func (s *Server) handleInsights(w http.ResponseWriter, r *http.Request) {
	insights := s.config.InsightsEngine.GetInsights()
	s.respondJSON(w, insights)
}

func (s *Server) handleAnomalies(w http.ResponseWriter, r *http.Request) {
	anomalies := s.config.InsightsEngine.GetAnomalies()
	s.respondJSON(w, anomalies)
}

func (s *Server) handleRecommendations(w http.ResponseWriter, r *http.Request) {
	recommendations := s.config.InsightsEngine.GetRecommendations()
	s.respondJSON(w, recommendations)
}

func (s *Server) handlePredictions(w http.ResponseWriter, r *http.Request) {
	insights := s.config.InsightsEngine.GetInsights()
	s.respondJSON(w, insights.Predictions)
}

// WebSocket handlers

func (s *Server) handleMetricsStream(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.config.Logger.Errorf("Failed to upgrade connection: %v", err)
		return
	}
	defer conn.Close()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			metrics := s.config.MetricsCollector.GetLatestMetrics()
			if err := conn.WriteJSON(metrics); err != nil {
				return
			}
		case <-r.Context().Done():
			return
		}
	}
}

func (s *Server) handleEventsStream(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.config.Logger.Errorf("Failed to upgrade connection: %v", err)
		return
	}
	defer conn.Close()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			insights := s.config.InsightsEngine.GetInsights()
			event := map[string]interface{}{
				"timestamp":       time.Now().Unix(),
				"anomalies":       len(insights.Anomalies),
				"recommendations": len(insights.Recommendations),
			}
			if err := conn.WriteJSON(event); err != nil {
				return
			}
		case <-r.Context().Done():
			return
		}
	}
}

// Control plane handler

func (s *Server) handleControlPlaneStatus(w http.ResponseWriter, r *http.Request) {
	status := s.config.ControlPlaneCollector.GetStatus()
	s.respondJSON(w, status)
}

// Health handler

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
		"version":   "1.0.0",
	}
	s.respondJSON(w, health)
}

// Helper methods

func (s *Server) respondJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func (s *Server) enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// methodFilter ensures only specified HTTP methods are allowed
func (s *Server) methodFilter(handler http.HandlerFunc, methods ...string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		for _, method := range methods {
			if r.Method == method {
				handler(w, r)
				return
			}
		}
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
