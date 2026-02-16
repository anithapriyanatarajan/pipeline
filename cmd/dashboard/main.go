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

package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	tektonClient "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	"github.com/tektoncd/pipeline/pkg/dashboard"
	"github.com/tektoncd/pipeline/pkg/dashboard/api"
	"github.com/tektoncd/pipeline/pkg/dashboard/collectors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/signals"
)

var (
	masterURL       = flag.String("master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig.")
	kubeconfig      = flag.String("kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	port            = flag.String("port", "8080", "Port to run the dashboard server on")
	metricsEndpoint = flag.String("metrics-endpoint", "http://tekton-pipelines-controller:9090/metrics", "Prometheus metrics endpoint")
)

func main() {
	flag.Parse()

	ctx := signals.NewContext()
	logger := logging.FromContext(ctx)

	// Create Kubernetes client
	cfg, err := clientcmd.BuildConfigFromFlags(*masterURL, *kubeconfig)
	if err != nil {
		logger.Fatalf("Error building kubeconfig: %v", err)
	}

	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		logger.Fatalf("Error building kubernetes clientset: %v", err)
	}

	tektonCl, err := tektonClient.NewForConfig(cfg)
	if err != nil {
		logger.Fatalf("Error building tekton clientset: %v", err)
	}

	// Initialize dashboard configuration
	dashboardConfig := &dashboard.Config{
		MetricsEndpoint:      *metricsEndpoint,
		EnableCostTracking:   getEnvOrDefault("ENABLE_COST_TRACKING", "true") == "true",
		EnableAIInsights:     getEnvOrDefault("ENABLE_AI_INSIGHTS", "true") == "true",
		CPUCostPerHour:       getEnvFloat("CPU_COST_PER_HOUR", 0.05),
		MemoryCostPerGBHour:  getEnvFloat("MEMORY_COST_PER_GB_HOUR", 0.01),
		StorageCostPerGBHour: getEnvFloat("STORAGE_COST_PER_GB_HOUR", 0.001),
	}

	// Initialize collectors
	metricsCollector := collectors.NewMetricsCollector(ctx, kubeClient, dashboardConfig)
	costCollector := collectors.NewCostCollector(ctx, kubeClient, tektonCl, dashboardConfig)
	traceCollector := collectors.NewTraceCollector(ctx, kubeClient, tektonCl)
	insightsEngine := collectors.NewInsightsEngine(ctx, metricsCollector, costCollector)
	controlPlaneCollector := collectors.NewControlPlaneCollector(ctx, kubeClient, logger)

	// Start collectors
	go metricsCollector.Start()
	go costCollector.Start()
	go traceCollector.Start()
	go insightsEngine.Start()
	go controlPlaneCollector.Start()

	// Initialize API server
	apiServer := api.NewServer(&api.ServerConfig{
		Port:                  *port,
		MetricsCollector:      metricsCollector,
		CostCollector:         costCollector,
		TraceCollector:        traceCollector,
		InsightsEngine:        insightsEngine,
		ControlPlaneCollector: controlPlaneCollector,
		Logger:                logger,
	})

	// Setup HTTP server
	srv := &http.Server{
		Addr:         ":" + *port,
		Handler:      apiServer.Handler(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		logger.Infof("Starting Tekton Dashboard on port %s", *port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-quit:
		logger.Info("Shutting down server...")
	case <-ctx.Done():
		logger.Info("Context cancelled, shutting down...")
	}

	// Graceful shutdown with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Fatalf("Server forced to shutdown: %v", err)
	}

	logger.Info("Server exited")
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if f, err := strconv.ParseFloat(value, 64); err == nil {
			return f
		}
	}
	return defaultValue
}
