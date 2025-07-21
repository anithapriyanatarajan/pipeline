#!/bin/bash

# Tekton Nightly Release Testing Script
# This script validates the complete nightly release setup for any fork

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
TEST_RESULTS_DIR="${PROJECT_ROOT}/test-results"
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")

# Test configuration
TEST_CLUSTER_NAME="tekton-nightly-test"
TEST_NAMESPACE="tekton-pipelines-test"
KUBERNETES_VERSION="${KUBERNETES_VERSION:-v1.31.0}"
REGISTRY_PREFIX="${GITHUB_REPOSITORY_OWNER:-testuser}"

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_section() {
    echo -e "\n${BLUE}=== $1 ===${NC}"
}

# Setup test environment
setup_test_environment() {
    log_section "Setting up test environment"
    
    # Create test results directory
    mkdir -p "${TEST_RESULTS_DIR}"
    
    # Save test run metadata
    cat > "${TEST_RESULTS_DIR}/test-run-${TIMESTAMP}.json" <<EOF
{
    "timestamp": "${TIMESTAMP}",
    "kubernetes_version": "${KUBERNETES_VERSION}",
    "registry_prefix": "${REGISTRY_PREFIX}",
    "git_commit": "$(git rev-parse HEAD 2>/dev/null || echo 'unknown')",
    "git_branch": "$(git branch --show-current 2>/dev/null || echo 'unknown')"
}
EOF
    
    log_success "Test environment setup complete"
}

# Check prerequisites
check_prerequisites() {
    log_section "Checking prerequisites"
    
    local missing_tools=()
    
    # Required tools
    local tools=(
        "kind"
        "kubectl"
        "tkn"
        "docker"
        "git"
        "jq"
        "curl"
    )
    
    for tool in "${tools[@]}"; do
        if ! command -v "$tool" &> /dev/null; then
            missing_tools+=("$tool")
        else
            log_info "$tool: $(command -v "$tool")"
        fi
    done
    
    if [ ${#missing_tools[@]} -ne 0 ]; then
        log_error "Missing required tools: ${missing_tools[*]}"
        log_info "Please install missing tools and re-run the test"
        return 1
    fi
    
    # Check Docker daemon
    if ! docker info &> /dev/null; then
        log_error "Docker daemon is not running"
        return 1
    fi
    
    log_success "All prerequisites satisfied"
}

# Test Kind cluster creation
test_cluster_creation() {
    log_section "Testing Kind cluster creation"
    
    # Clean up any existing test cluster
    if kind get clusters | grep -q "^${TEST_CLUSTER_NAME}$"; then
        log_info "Cleaning up existing test cluster"
        kind delete cluster --name="${TEST_CLUSTER_NAME}"
    fi
    
    # Create Kind cluster
    log_info "Creating Kind cluster with Kubernetes ${KUBERNETES_VERSION}"
    
    cat <<EOF | kind create cluster --name="${TEST_CLUSTER_NAME}" --config=-
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
  image: kindest/node:${KUBERNETES_VERSION}
  extraPortMappings:
  - containerPort: 8080
    hostPort: 8080
    protocol: TCP
EOF
    
    # Verify cluster is ready
    local retries=30
    while [ $retries -gt 0 ]; do
        if kubectl cluster-info --context="kind-${TEST_CLUSTER_NAME}" &> /dev/null; then
            break
        fi
        log_info "Waiting for cluster to be ready... ($retries attempts remaining)"
        sleep 10
        ((retries--))
    done
    
    if [ $retries -eq 0 ]; then
        log_error "Cluster failed to become ready"
        return 1
    fi
    
    # Set kubectl context
    kubectl config use-context "kind-${TEST_CLUSTER_NAME}"
    
    log_success "Kind cluster created and ready"
}

# Test Tekton installation
test_tekton_installation() {
    log_section "Testing Tekton installation"
    
    # Install Tekton Pipelines
    log_info "Installing Tekton Pipelines"
    kubectl apply --filename https://storage.googleapis.com/tekton-releases/pipeline/latest/release.yaml
    
    # Wait for Tekton to be ready
    log_info "Waiting for Tekton Pipelines to be ready"
    kubectl wait --for=condition=ready pod -l app.kubernetes.io/part-of=tekton-pipelines -n tekton-pipelines --timeout=300s
    
    # Install bundle resolver
    log_info "Installing Tekton bundle resolver"
    kubectl apply -f https://storage.googleapis.com/tekton-releases/pipeline/latest/resolvers.yaml
    
    # Wait for resolvers to be ready
    log_info "Waiting for resolvers to be ready"
    kubectl wait --for=condition=ready pod -l app.kubernetes.io/part-of=tekton-pipelines-resolvers -n tekton-pipelines-resolvers --timeout=300s
    
    # Create test namespace
    kubectl create namespace "${TEST_NAMESPACE}" --dry-run=client -o yaml | kubectl apply -f -
    
    log_success "Tekton installation complete"
}

# Test bundle resolver functionality
test_bundle_resolvers() {
    log_section "Testing bundle resolver functionality"
    
    # Test git-clone bundle
    log_info "Testing git-clone bundle resolver"
    
    cat <<EOF | kubectl apply -f -
apiVersion: tekton.dev/v1beta1
kind: TaskRun
metadata:
  name: test-git-clone-bundle-${TIMESTAMP}
  namespace: ${TEST_NAMESPACE}
spec:
  taskRef:
    resolver: bundles
    params:
    - name: bundle
      value: ghcr.io/tektoncd/catalog/upstream/tasks/git-clone:0.10
    - name: name
      value: git-clone
    - name: kind
      value: task
  params:
  - name: url
    value: https://github.com/tektoncd/pipeline.git
  - name: revision
    value: main
  workspaces:
  - name: output
    emptyDir: {}
EOF
    
    # Wait for TaskRun to complete
    local retries=30
    while [ $retries -gt 0 ]; do
        local status=$(kubectl get taskrun "test-git-clone-bundle-${TIMESTAMP}" -n "${TEST_NAMESPACE}" -o jsonpath='{.status.conditions[0].status}' 2>/dev/null || echo "Unknown")
        if [ "$status" = "True" ]; then
            log_success "Git-clone bundle resolver test passed"
            break
        elif [ "$status" = "False" ]; then
            log_error "Git-clone bundle resolver test failed"
            kubectl describe taskrun "test-git-clone-bundle-${TIMESTAMP}" -n "${TEST_NAMESPACE}"
            return 1
        fi
        sleep 10
        ((retries--))
    done
    
    if [ $retries -eq 0 ]; then
        log_error "Git-clone bundle resolver test timed out"
        return 1
    fi
}

# Test authentication setup
test_authentication() {
    log_section "Testing authentication setup"
    
    # Check for required secrets
    if [ -z "${NIGHTLY_RELEASE_TOKEN:-}" ]; then
        log_warning "NIGHTLY_RELEASE_TOKEN not set - skipping authentication test"
        return 0
    fi
    
    # Test GitHub Container Registry authentication
    log_info "Testing GHCR authentication"
    
    if echo "${NIGHTLY_RELEASE_TOKEN}" | docker login ghcr.io -u "${GITHUB_ACTOR:-testuser}" --password-stdin; then
        log_success "GHCR authentication successful"
    else
        log_error "GHCR authentication failed"
        return 1
    fi
}

# Test pipeline configuration
test_pipeline_configuration() {
    log_section "Testing pipeline configuration"
    
    # Apply pipeline configuration
    log_info "Applying nightly release pipeline"
    kubectl apply -f "${PROJECT_ROOT}/tekton/release-nightly-pipeline.yaml" -n "${TEST_NAMESPACE}"
    
    # Apply publish task
    log_info "Applying publish task"
    kubectl apply -f "${PROJECT_ROOT}/tekton/publish-nightly.yaml" -n "${TEST_NAMESPACE}"
    
    # Validate pipeline
    if kubectl get pipeline release-nightly-pipeline -n "${TEST_NAMESPACE}" &> /dev/null; then
        log_success "Pipeline configuration valid"
    else
        log_error "Pipeline configuration invalid"
        return 1
    fi
    
    # Validate task
    if kubectl get task publish-release-nightly -n "${TEST_NAMESPACE}" &> /dev/null; then
        log_success "Task configuration valid"
    else
        log_error "Task configuration invalid"
        return 1
    fi
}

# Test image building (dry run)
test_image_building() {
    log_section "Testing image building (dry run)"
    
    # Check if Ko is available
    if ! command -v ko &> /dev/null; then
        log_warning "Ko not available - installing"
        go install github.com/ko-build/ko@latest
    fi
    
    # Test Ko configuration
    log_info "Testing Ko configuration"
    
    cd "${PROJECT_ROOT}"
    
    # Set required environment variables
    export KO_DOCKER_REPO="ghcr.io/${REGISTRY_PREFIX}/pipeline"
    export GOFLAGS="-mod=vendor"
    
    # Test Ko resolve (dry run)
    if ko resolve --dry-run -f config/ > /dev/null; then
        log_success "Ko configuration valid"
    else
        log_error "Ko configuration invalid"
        return 1
    fi
}

# Test GitHub Actions workflow
test_github_actions_workflow() {
    log_section "Testing GitHub Actions workflow configuration"
    
    local workflow_file="${PROJECT_ROOT}/.github/workflows/nightly-release.yaml"
    
    if [ ! -f "$workflow_file" ]; then
        log_error "GitHub Actions workflow file not found"
        return 1
    fi
    
    # Validate YAML syntax
    if python3 -c "import yaml; yaml.safe_load(open('$workflow_file'))" 2>/dev/null; then
        log_success "Workflow YAML syntax valid"
    else
        log_error "Workflow YAML syntax invalid"
        return 1
    fi
    
    # Check required secrets
    local required_secrets=("NIGHTLY_RELEASE_TOKEN")
    for secret in "${required_secrets[@]}"; do
        if grep -q "$secret" "$workflow_file"; then
            log_info "Required secret '$secret' referenced in workflow"
        else
            log_warning "Required secret '$secret' not found in workflow"
        fi
    done
}

# Run end-to-end test
test_end_to_end() {
    log_section "Running end-to-end test"
    
    if [ -z "${NIGHTLY_RELEASE_TOKEN:-}" ]; then
        log_warning "Skipping end-to-end test - NIGHTLY_RELEASE_TOKEN not set"
        return 0
    fi
    
    # Create test secret
    kubectl create secret generic release-secret \
        --from-literal=token="${NIGHTLY_RELEASE_TOKEN}" \
        -n "${TEST_NAMESPACE}" \
        --dry-run=client -o yaml | kubectl apply -f -
    
    # Create test workspace
    kubectl apply -f - <<EOF
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: test-workspace
  namespace: ${TEST_NAMESPACE}
spec:
  accessModes:
  - ReadWriteOnce
  resources:
    requests:
      storage: 1Gi
---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: test-output
  namespace: ${TEST_NAMESPACE}
spec:
  accessModes:
  - ReadWriteOnce
  resources:
    requests:
      storage: 500Mi
EOF
    
    # Run pipeline
    log_info "Starting pipeline run"
    
    cat <<EOF | kubectl apply -f -
apiVersion: tekton.dev/v1beta1
kind: PipelineRun
metadata:
  name: test-nightly-release-${TIMESTAMP}
  namespace: ${TEST_NAMESPACE}
spec:
  pipelineRef:
    name: release-nightly-pipeline
  params:
  - name: package
    value: github.com/${GITHUB_REPOSITORY:-tektoncd/pipeline}
  - name: koExtraArgs
    value: ""
  - name: versionTag
    value: v$(date +"%Y%m%d")-test
  - name: imageRegistry
    value: ghcr.io
  - name: imageRegistryPath
    value: ${REGISTRY_PREFIX}/pipeline
  - name: serviceAccountPath
    value: token
  workspaces:
  - name: source
    persistentVolumeClaim:
      claimName: test-workspace
  - name: release-secret
    secret:
      secretName: release-secret
  - name: output
    persistentVolumeClaim:
      claimName: test-output
EOF
    
    # Wait for pipeline to complete (with timeout)
    local timeout=1800  # 30 minutes
    local elapsed=0
    
    while [ $elapsed -lt $timeout ]; do
        local status=$(kubectl get pipelinerun "test-nightly-release-${TIMESTAMP}" -n "${TEST_NAMESPACE}" -o jsonpath='{.status.conditions[0].status}' 2>/dev/null || echo "Unknown")
        local reason=$(kubectl get pipelinerun "test-nightly-release-${TIMESTAMP}" -n "${TEST_NAMESPACE}" -o jsonpath='{.status.conditions[0].reason}' 2>/dev/null || echo "Unknown")
        
        if [ "$status" = "True" ]; then
            log_success "End-to-end pipeline test passed"
            return 0
        elif [ "$status" = "False" ]; then
            log_error "End-to-end pipeline test failed: $reason"
            kubectl describe pipelinerun "test-nightly-release-${TIMESTAMP}" -n "${TEST_NAMESPACE}"
            return 1
        fi
        
        log_info "Pipeline running... (${elapsed}s elapsed)"
        sleep 30
        ((elapsed+=30))
    done
    
    log_error "End-to-end pipeline test timed out"
    return 1
}

# Generate test report
generate_test_report() {
    log_section "Generating test report"
    
    local report_file="${TEST_RESULTS_DIR}/test-report-${TIMESTAMP}.md"
    
    cat > "$report_file" <<EOF
# Tekton Nightly Release Test Report

**Generated**: $(date)
**Kubernetes Version**: ${KUBERNETES_VERSION}
**Registry Prefix**: ${REGISTRY_PREFIX}
**Git Commit**: $(git rev-parse HEAD 2>/dev/null || echo 'unknown')

## Test Results

$([ -f "${TEST_RESULTS_DIR}/test-results-${TIMESTAMP}.txt" ] && cat "${TEST_RESULTS_DIR}/test-results-${TIMESTAMP}.txt" || echo "No detailed results available")

## Recommendations

$([ -f "${TEST_RESULTS_DIR}/recommendations-${TIMESTAMP}.txt" ] && cat "${TEST_RESULTS_DIR}/recommendations-${TIMESTAMP}.txt" || echo "No recommendations available")

## Next Steps

1. Review any failed tests and address issues
2. Update documentation if configuration changes are needed
3. Test in your fork before enabling production releases
4. Monitor the first few nightly releases closely

EOF
    
    log_success "Test report generated: $report_file"
}

# Cleanup
cleanup() {
    log_section "Cleaning up test resources"
    
    # Delete test cluster
    if kind get clusters | grep -q "^${TEST_CLUSTER_NAME}$"; then
        log_info "Deleting test cluster"
        kind delete cluster --name="${TEST_CLUSTER_NAME}"
    fi
    
    # Docker logout
    docker logout ghcr.io &> /dev/null || true
    
    log_success "Cleanup complete"
}

# Main test execution
main() {
    local exit_code=0
    
    echo "🧪 Tekton Nightly Release Testing Suite"
    echo "========================================"
    
    # Trap cleanup on exit
    trap cleanup EXIT
    
    # Run tests
    {
        setup_test_environment &&
        check_prerequisites &&
        test_cluster_creation &&
        test_tekton_installation &&
        test_bundle_resolvers &&
        test_authentication &&
        test_pipeline_configuration &&
        test_image_building &&
        test_github_actions_workflow &&
        test_end_to_end
    } || exit_code=$?
    
    # Generate report
    generate_test_report
    
    if [ $exit_code -eq 0 ]; then
        log_success "🎉 All tests passed! Your nightly release setup is ready for production."
        echo ""
        echo "Next steps:"
        echo "1. Review the test report in ${TEST_RESULTS_DIR}/"
        echo "2. Configure GitHub secrets in your repository"
        echo "3. Enable the nightly release workflow"
        echo "4. Monitor the first few releases"
    else
        log_error "❌ Some tests failed. Please review the output and fix issues before proceeding."
        echo ""
        echo "Common solutions:"
        echo "1. Check GitHub token permissions"
        echo "2. Verify container registry access"
        echo "3. Review bundle resolver connectivity"
        echo "4. Check Tekton installation"
    fi
    
    return $exit_code
}

# Help function
show_help() {
    cat <<EOF
Tekton Nightly Release Testing Script

USAGE:
    $0 [OPTIONS]

OPTIONS:
    -h, --help              Show this help message
    --kubernetes-version    Kubernetes version to test with (default: v1.31.0)
    --registry-prefix       Container registry prefix (default: from GITHUB_REPOSITORY_OWNER)
    --skip-e2e              Skip end-to-end testing (useful for quick validation)

ENVIRONMENT VARIABLES:
    NIGHTLY_RELEASE_TOKEN   GitHub token for container registry authentication
    GITHUB_REPOSITORY_OWNER Repository owner (for registry prefix)
    GITHUB_ACTOR           GitHub username (for authentication)

EXAMPLES:
    # Basic test run
    $0
    
    # Test with specific Kubernetes version
    $0 --kubernetes-version v1.30.0
    
    # Test with custom registry prefix
    $0 --registry-prefix myusername
    
    # Skip end-to-end testing for faster validation
    $0 --skip-e2e

EOF
}

# Parse command line arguments
SKIP_E2E=false

while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            show_help
            exit 0
            ;;
        --kubernetes-version)
            KUBERNETES_VERSION="$2"
            shift 2
            ;;
        --registry-prefix)
            REGISTRY_PREFIX="$2"
            shift 2
            ;;
        --skip-e2e)
            SKIP_E2E=true
            shift
            ;;
        *)
            log_error "Unknown option: $1"
            show_help
            exit 1
            ;;
    esac
done

# Override end-to-end test if requested
if [ "$SKIP_E2E" = true ]; then
    test_end_to_end() {
        log_section "Skipping end-to-end test (--skip-e2e specified)"
        log_warning "End-to-end test skipped - manual testing recommended"
    }
fi

# Run main function
main "$@"
