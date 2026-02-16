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
	"strings"
	"sync"
	"time"

	"github.com/tektoncd/pipeline/pkg/dashboard"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
)

// tektonComponent describes a well-known Tekton control-plane deployment.
type tektonComponent struct {
	DisplayName string
	Deployment  string
}

// knownComponents lists the Tekton deployments we look for, ordered by
// importance.  All are expected in the tekton-pipelines namespace (or the
// operator namespace for the operator itself).
var knownComponents = []tektonComponent{
	// Core Pipelines
	{DisplayName: "Pipelines Controller", Deployment: "tekton-pipelines-controller"},
	{DisplayName: "Pipelines Webhook", Deployment: "tekton-pipelines-webhook"},
	// Events controller (optional — ships with pipelines)
	{DisplayName: "Events Controller", Deployment: "tekton-events-controller"},
	// Tekton Dashboard (self — optional)
	{DisplayName: "Dashboard", Deployment: "tekton-dashboard"},
	// Triggers (optional add-on)
	{DisplayName: "Triggers Controller", Deployment: "tekton-triggers-controller"},
	{DisplayName: "Triggers Webhook", Deployment: "tekton-triggers-webhook"},
	{DisplayName: "Triggers EventListener", Deployment: "el-tekton-triggers-eventlistener"},
	// Chains (optional add-on)
	{DisplayName: "Chains Controller", Deployment: "tekton-chains-controller"},
	// Results (optional add-on)
	{DisplayName: "Results API", Deployment: "tekton-results-api"},
	{DisplayName: "Results Watcher", Deployment: "tekton-results-watcher"},
	// Operator (optional — manages all above)
	{DisplayName: "Operator Controller", Deployment: "tekton-operator"},
}

// operatorNamespaces are the namespaces where the Tekton Operator may run.
var operatorNamespaces = []string{"tekton-operator", "openshift-operators", "tekton-pipelines"}

// ControlPlaneCollector discovers and monitors Tekton control-plane components.
type ControlPlaneCollector struct {
	ctx             context.Context
	kubeClient      kubernetes.Interface
	discoveryClient discovery.DiscoveryInterface
	logger          *zap.SugaredLogger

	mu           sync.RWMutex
	latestStatus *dashboard.ControlPlaneStatus
}

// NewControlPlaneCollector creates a new control-plane health collector.
func NewControlPlaneCollector(ctx context.Context, kubeClient kubernetes.Interface, logger *zap.SugaredLogger) *ControlPlaneCollector {
	return &ControlPlaneCollector{
		ctx:             ctx,
		kubeClient:      kubeClient,
		discoveryClient: kubeClient.Discovery(),
		logger:          logger,
	}
}

// Start begins periodic control-plane health collection.
func (c *ControlPlaneCollector) Start() {
	c.collect()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.collect()
		case <-c.ctx.Done():
			c.logger.Info("ControlPlane collector stopping")
			return
		}
	}
}

// GetStatus returns the latest control-plane health snapshot.
func (c *ControlPlaneCollector) GetStatus() *dashboard.ControlPlaneStatus {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.latestStatus == nil {
		return &dashboard.ControlPlaneStatus{
			Timestamp:     time.Now().Unix(),
			OverallHealth: "Unknown",
		}
	}
	return c.latestStatus
}

// ──────────────────────── internal ────────────────────────

func (c *ControlPlaneCollector) collect() {
	status := &dashboard.ControlPlaneStatus{
		Timestamp: time.Now().Unix(),
	}

	// 1. Detect whether the Tekton Operator is installed by probing the
	//    operator.tekton.dev API group.
	status.OperatorManaged = c.isOperatorInstalled()

	// 2. Discover components across relevant namespaces.
	namespaces := c.discoverNamespaces()
	for _, ns := range namespaces {
		c.discoverComponents(ns, status)
	}

	// 3. Try to detect the Tekton Pipelines version from the controller image
	//    tag, or from the operator CR if available.
	status.TektonVersion = c.detectVersion(status)

	// 4. Derive overall health.
	status.OverallHealth = c.deriveOverallHealth(status.Components)

	c.mu.Lock()
	c.latestStatus = status
	c.mu.Unlock()
}

// isOperatorInstalled checks whether the operator.tekton.dev API group is
// registered in the cluster.
func (c *ControlPlaneCollector) isOperatorInstalled() bool {
	groups, err := c.discoveryClient.ServerGroups()
	if err != nil {
		c.logger.Warnf("Failed to list API groups: %v", err)
		return false
	}
	for _, g := range groups.Groups {
		if g.Name == "operator.tekton.dev" {
			return true
		}
	}
	return false
}

// discoverNamespaces returns the namespaces that may contain Tekton
// control-plane deployments.
func (c *ControlPlaneCollector) discoverNamespaces() []string {
	seen := map[string]bool{}
	result := []string{"tekton-pipelines"} // always check
	seen["tekton-pipelines"] = true

	for _, ns := range operatorNamespaces {
		if !seen[ns] {
			// Only include if the namespace actually exists.
			_, err := c.kubeClient.CoreV1().Namespaces().Get(c.ctx, ns, metav1.GetOptions{})
			if err == nil {
				result = append(result, ns)
				seen[ns] = true
			}
		}
	}
	return result
}

// discoverComponents finds Tekton deployments in a given namespace and
// appends ComponentStatus entries to the status.
func (c *ControlPlaneCollector) discoverComponents(ns string, status *dashboard.ControlPlaneStatus) {
	deployments, err := c.kubeClient.AppsV1().Deployments(ns).List(c.ctx, metav1.ListOptions{})
	if err != nil {
		c.logger.Warnf("Failed to list deployments in %s: %v", ns, err)
		return
	}

	// Build a lookup of the deployments actually present.
	depMap := map[string]*appsv1.Deployment{}
	for i := range deployments.Items {
		depMap[deployments.Items[i].Name] = &deployments.Items[i]
	}

	for _, kc := range knownComponents {
		dep, ok := depMap[kc.Deployment]
		if !ok {
			continue
		}
		cs := c.buildComponentStatus(kc.DisplayName, dep, ns)
		status.Components = append(status.Components, cs)
	}

	// Also pick up any tekton-related deployments not in knownComponents
	// (custom or new add-ons).
	knownSet := map[string]bool{}
	for _, kc := range knownComponents {
		knownSet[kc.Deployment] = true
	}
	for name, dep := range depMap {
		if knownSet[name] {
			continue
		}
		if c.isTektonRelated(dep) {
			cs := c.buildComponentStatus(name, dep, ns)
			status.Components = append(status.Components, cs)
		}
	}
}

// isTektonRelated returns true if a deployment looks like a Tekton component
// based on its labels.
func (c *ControlPlaneCollector) isTektonRelated(dep *appsv1.Deployment) bool {
	for _, prefix := range []string{"tekton", "el-"} {
		if strings.HasPrefix(dep.Name, prefix) {
			return true
		}
	}
	labels := dep.Labels
	for _, key := range []string{"app.kubernetes.io/part-of", "operator.tekton.dev/operand-name"} {
		if v, ok := labels[key]; ok {
			if strings.Contains(v, "tekton") {
				return true
			}
		}
	}
	return false
}

// buildComponentStatus creates a ComponentStatus for a single Deployment.
func (c *ControlPlaneCollector) buildComponentStatus(displayName string, dep *appsv1.Deployment, ns string) *dashboard.ComponentStatus {
	cs := &dashboard.ComponentStatus{
		Name:            displayName,
		Component:       dep.Name,
		Namespace:       ns,
		Kind:            "Deployment",
		DesiredReplicas: 1,
	}

	if dep.Spec.Replicas != nil {
		cs.DesiredReplicas = *dep.Spec.Replicas
	}
	cs.ReadyReplicas = dep.Status.ReadyReplicas

	// Determine health.
	cs.Health = c.deploymentHealth(dep)

	// Extract container image (first container).
	if len(dep.Spec.Template.Spec.Containers) > 0 {
		cs.Image = dep.Spec.Template.Spec.Containers[0].Image
		cs.Version = extractVersionFromImage(cs.Image)
	}

	// Conditions.
	for _, cond := range dep.Status.Conditions {
		cs.Conditions = append(cs.Conditions, &dashboard.ComponentCondition{
			Type:    string(cond.Type),
			Status:  string(cond.Status),
			Reason:  cond.Reason,
			Message: cond.Message,
		})
		if cond.LastTransitionTime.Unix() > cs.LastTransitionTime {
			cs.LastTransitionTime = cond.LastTransitionTime.Unix()
		}
	}

	// Pods owned by this Deployment.
	cs.Pods = c.getDeploymentPods(dep, ns)

	return cs
}

// getDeploymentPods lists pods that belong to the given Deployment.
func (c *ControlPlaneCollector) getDeploymentPods(dep *appsv1.Deployment, ns string) []*dashboard.PodStatus {
	// Use the deployment's matchLabels selector.
	sel := dep.Spec.Selector
	if sel == nil {
		return nil
	}
	var parts []string
	for k, v := range sel.MatchLabels {
		parts = append(parts, k+"="+v)
	}
	labelSelector := strings.Join(parts, ",")

	pods, err := c.kubeClient.CoreV1().Pods(ns).List(c.ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		c.logger.Warnf("Failed to list pods for %s/%s: %v", ns, dep.Name, err)
		return nil
	}

	var result []*dashboard.PodStatus
	now := time.Now()
	for i := range pods.Items {
		p := &pods.Items[i]
		ps := &dashboard.PodStatus{
			Name:  p.Name,
			Phase: string(p.Status.Phase),
			Ready: isPodReady(p),
			Node:  p.Spec.NodeName,
			IP:    p.Status.PodIP,
			Age:   int64(now.Sub(p.CreationTimestamp.Time).Seconds()),
		}

		// Sum restarts.
		for _, cs := range p.Status.ContainerStatuses {
			ps.Restarts += cs.RestartCount
		}

		// Container details.
		for _, cs := range p.Status.ContainerStatuses {
			ci := &dashboard.ContainerInfo{
				Name:  cs.Name,
				Image: cs.Image,
				Ready: cs.Ready,
			}
			if cs.State.Running != nil {
				ci.State = "running"
			} else if cs.State.Waiting != nil {
				ci.State = "waiting"
				ci.Reason = cs.State.Waiting.Reason
			} else if cs.State.Terminated != nil {
				ci.State = "terminated"
				ci.Reason = cs.State.Terminated.Reason
			}
			ps.Containers = append(ps.Containers, ci)
		}

		result = append(result, ps)
	}
	return result
}

// deploymentHealth returns a health string based on the Deployment status.
func (c *ControlPlaneCollector) deploymentHealth(dep *appsv1.Deployment) string {
	desired := int32(1)
	if dep.Spec.Replicas != nil {
		desired = *dep.Spec.Replicas
	}
	if desired == 0 {
		return "Scaled Down"
	}

	ready := dep.Status.ReadyReplicas
	if ready >= desired {
		return "Healthy"
	}
	if ready > 0 {
		return "Degraded"
	}
	return "Unhealthy"
}

// detectVersion tries to extract the Tekton version from the pipelines
// controller image tag.
func (c *ControlPlaneCollector) detectVersion(status *dashboard.ControlPlaneStatus) string {
	// First try to get version from operator TektonConfig CR.
	if status.OperatorManaged {
		if v := c.getOperatorVersion(); v != "" {
			return v
		}
	}
	// Fall back to extracting from the controller image tag.
	for _, comp := range status.Components {
		if comp.Component == "tekton-pipelines-controller" && comp.Version != "" {
			return comp.Version
		}
	}
	return "unknown"
}

// getOperatorVersion tries to read the Tekton version from the TektonConfig CR.
func (c *ControlPlaneCollector) getOperatorVersion() string {
	// Use the dynamic client via the REST interface to read the TektonConfig
	// status, which contains the installed version.
	gvr := schema.GroupVersionResource{
		Group:    "operator.tekton.dev",
		Version:  "v1alpha1",
		Resource: "tektonconfigs",
	}

	// Use the discovery client to confirm the resource exists.
	resources, err := c.discoveryClient.ServerResourcesForGroupVersion(gvr.GroupVersion().String())
	if err != nil {
		return "" // CRD not available
	}

	found := false
	for _, r := range resources.APIResources {
		if r.Name == "tektonconfigs" {
			found = true
			break
		}
	}
	if !found {
		return ""
	}

	// We don't vendor the operator types, so read via REST.
	// For now just return empty — the image-tag fallback works.
	return ""
}

// deriveOverallHealth returns a summary health from all components.
func (c *ControlPlaneCollector) deriveOverallHealth(components []*dashboard.ComponentStatus) string {
	if len(components) == 0 {
		return "Unknown"
	}
	hasUnhealthy := false
	hasDegraded := false
	for _, comp := range components {
		switch comp.Health {
		case "Unhealthy":
			hasUnhealthy = true
		case "Degraded":
			hasDegraded = true
		}
	}
	if hasUnhealthy {
		return "Unhealthy"
	}
	if hasDegraded {
		return "Degraded"
	}
	return "Healthy"
}

// ──────────────────────── helpers ────────────────────────

// extractVersionFromImage extracts a semver-like tag from a container image
// reference. Examples:
//
//	gcr.io/tekton-releases/controller:v0.51.0  → v0.51.0
//	ghcr.io/tektoncd/operator/cmd/operator:v0.72.0@sha256:abc → v0.72.0
func extractVersionFromImage(image string) string {
	// Strip digest
	if idx := strings.Index(image, "@"); idx != -1 {
		image = image[:idx]
	}
	// Get tag
	if idx := strings.LastIndex(image, ":"); idx != -1 {
		tag := image[idx+1:]
		if tag != "latest" && tag != "" {
			return tag
		}
	}
	return ""
}

// isPodReady returns true if all containers in the pod are ready.
func isPodReady(pod *corev1.Pod) bool {
	for _, cond := range pod.Status.Conditions {
		if cond.Type == corev1.PodReady {
			return cond.Status == corev1.ConditionTrue
		}
	}
	return false
}
