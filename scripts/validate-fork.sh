#!/bin/bash

# Fork Validation Script for Tekton Nightly Releases
# This script helps fork maintainers validate their setup before enabling nightly releases

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[✓]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[⚠]${NC} $1"
}

log_error() {
    echo -e "${RED}[✗]${NC} $1"
}

log_step() {
    echo -e "\n${PURPLE}▶ $1${NC}"
}

log_section() {
    echo -e "\n${BLUE}━━━ $1 ━━━${NC}"
}

# Validation results
VALIDATION_RESULTS=()
WARNINGS=()
ERRORS=()

add_result() {
    local status="$1"
    local message="$2"
    VALIDATION_RESULTS+=("$status: $message")
    case "$status" in
        "PASS") log_success "$message" ;;
        "WARN") log_warning "$message"; WARNINGS+=("$message") ;;
        "FAIL") log_error "$message"; ERRORS+=("$message") ;;
    esac
}

# Repository analysis
analyze_repository() {
    log_section "Repository Analysis"
    
    # Check if we're in a git repository
    if ! git rev-parse --git-dir &>/dev/null; then
        add_result "FAIL" "Not in a git repository"
        return 1
    fi
    
    # Get repository information
    local git_remote=$(git remote get-url origin 2>/dev/null || echo "unknown")
    local git_branch=$(git branch --show-current 2>/dev/null || echo "unknown")
    local git_commit=$(git rev-parse HEAD 2>/dev/null || echo "unknown")
    
    log_info "Git remote: $git_remote"
    log_info "Current branch: $git_branch"
    log_info "Current commit: ${git_commit:0:8}"
    
    # Determine if this is a fork
    local is_fork=false
    local repo_owner=""
    local repo_name=""
    
    if [[ "$git_remote" =~ github\.com[:/]([^/]+)/([^/.]+) ]]; then
        repo_owner="${BASH_REMATCH[1]}"
        repo_name="${BASH_REMATCH[2]}"
        
        if [ "$repo_owner" != "tektoncd" ]; then
            is_fork=true
            add_result "PASS" "Repository is a fork (owner: $repo_owner)"
        else
            add_result "PASS" "Repository is upstream tektoncd"
        fi
    else
        add_result "WARN" "Could not determine repository owner from remote URL"
    fi
    
    # Check for required branch
    if [ "$git_branch" = "nightly-pipeline-gha" ]; then
        add_result "PASS" "On correct branch for nightly releases"
    else
        add_result "WARN" "Not on nightly-pipeline-gha branch (current: $git_branch)"
    fi
    
    # Export for use in other functions
    export REPO_OWNER="$repo_owner"
    export REPO_NAME="$repo_name"
    export IS_FORK="$is_fork"
}

# Check required files
check_required_files() {
    log_section "Required Files Check"
    
    local required_files=(
        ".github/workflows/nightly-release.yaml"
        "tekton/release-nightly-pipeline.yaml"
        "tekton/publish-nightly.yaml"
        "docs/nightly-releases.md"
        "scripts/test-nightly-release.sh"
    )
    
    for file in "${required_files[@]}"; do
        local full_path="${PROJECT_ROOT}/$file"
        if [ -f "$full_path" ]; then
            add_result "PASS" "Required file exists: $file"
        else
            add_result "FAIL" "Missing required file: $file"
        fi
    done
}

# Validate GitHub Actions workflow
validate_github_actions() {
    log_section "GitHub Actions Workflow Validation"
    
    local workflow_file="${PROJECT_ROOT}/.github/workflows/nightly-release.yaml"
    
    if [ ! -f "$workflow_file" ]; then
        add_result "FAIL" "GitHub Actions workflow file not found"
        return 1
    fi
    
    # Check YAML syntax
    if command -v python3 &>/dev/null; then
        if python3 -c "import yaml; yaml.safe_load(open('$workflow_file'))" 2>/dev/null; then
            add_result "PASS" "Workflow YAML syntax is valid"
        else
            add_result "FAIL" "Workflow YAML syntax is invalid"
        fi
    else
        add_result "WARN" "Cannot validate YAML syntax (python3 not available)"
    fi
    
    # Check for required secrets
    local required_secrets=("NIGHTLY_RELEASE_TOKEN")
    for secret in "${required_secrets[@]}"; do
        if grep -q "$secret" "$workflow_file"; then
            add_result "PASS" "Workflow references required secret: $secret"
        else
            add_result "FAIL" "Workflow missing required secret: $secret"
        fi
    done
    
    # Check schedule configuration
    if grep -q "schedule:" "$workflow_file" && grep -q "cron:" "$workflow_file"; then
        add_result "PASS" "Workflow has schedule configuration"
    else
        add_result "WARN" "Workflow missing schedule configuration"
    fi
    
    # Check manual trigger
    if grep -q "workflow_dispatch:" "$workflow_file"; then
        add_result "PASS" "Workflow supports manual triggering"
    else
        add_result "WARN" "Workflow doesn't support manual triggering"
    fi
}

# Validate Tekton pipeline
validate_tekton_pipeline() {
    log_section "Tekton Pipeline Validation"
    
    local pipeline_file="${PROJECT_ROOT}/tekton/release-nightly-pipeline.yaml"
    local task_file="${PROJECT_ROOT}/tekton/publish-nightly.yaml"
    
    # Check pipeline file
    if [ -f "$pipeline_file" ]; then
        # Basic YAML validation
        if command -v python3 &>/dev/null; then
            if python3 -c "import yaml; yaml.safe_load(open('$pipeline_file'))" 2>/dev/null; then
                add_result "PASS" "Pipeline YAML syntax is valid"
            else
                add_result "FAIL" "Pipeline YAML syntax is invalid"
            fi
        fi
        
        # Check for bundle resolvers
        if grep -q "resolver: bundles" "$pipeline_file"; then
            add_result "PASS" "Pipeline uses bundle resolvers"
        else
            add_result "WARN" "Pipeline may not be using bundle resolvers"
        fi
        
        # Check for required tasks
        local required_tasks=("git-clone" "golang-test" "golang-build")
        for task in "${required_tasks[@]}"; do
            if grep -q "$task" "$pipeline_file"; then
                add_result "PASS" "Pipeline includes task: $task"
            else
                add_result "WARN" "Pipeline may be missing task: $task"
            fi
        done
    fi
    
    # Check task file
    if [ -f "$task_file" ]; then
        # Basic YAML validation
        if command -v python3 &>/dev/null; then
            if python3 -c "import yaml; yaml.safe_load(open('$task_file'))" 2>/dev/null; then
                add_result "PASS" "Task YAML syntax is valid"
            else
                add_result "FAIL" "Task YAML syntax is invalid"
            fi
        fi
        
        # Check for koparse step
        if grep -q "koparse" "$task_file"; then
            add_result "PASS" "Task includes koparse step"
        else
            add_result "WARN" "Task may be missing koparse step"
        fi
    fi
}

# Check Docker and container registry access
check_container_registry() {
    log_section "Container Registry Access"
    
    # Check Docker
    if command -v docker &>/dev/null; then
        add_result "PASS" "Docker is available"
        
        if docker info &>/dev/null; then
            add_result "PASS" "Docker daemon is running"
        else
            add_result "FAIL" "Docker daemon is not running"
        fi
    else
        add_result "FAIL" "Docker is not installed"
    fi
    
    # Test GHCR access if token is available
    if [ -n "${NIGHTLY_RELEASE_TOKEN:-}" ]; then
        log_info "Testing GitHub Container Registry access..."
        
        if echo "${NIGHTLY_RELEASE_TOKEN}" | docker login ghcr.io -u "${GITHUB_ACTOR:-${REPO_OWNER}}" --password-stdin &>/dev/null; then
            add_result "PASS" "Successfully authenticated with GHCR"
            docker logout ghcr.io &>/dev/null
        else
            add_result "FAIL" "Failed to authenticate with GHCR"
        fi
    else
        add_result "WARN" "NIGHTLY_RELEASE_TOKEN not set - cannot test GHCR access"
    fi
}

# Check Go environment
check_go_environment() {
    log_section "Go Environment"
    
    if command -v go &>/dev/null; then
        local go_version=$(go version)
        add_result "PASS" "Go is available: $go_version"
        
        # Check if we can build
        cd "$PROJECT_ROOT"
        if go mod download &>/dev/null; then
            add_result "PASS" "Go modules are accessible"
        else
            add_result "WARN" "Go modules may have issues"
        fi
        
        # Check vendor directory
        if [ -d "vendor" ]; then
            add_result "PASS" "Vendor directory exists"
        else
            add_result "WARN" "Vendor directory not found - may need 'go mod vendor'"
        fi
    else
        add_result "FAIL" "Go is not installed"
    fi
    
    # Check Ko
    if command -v ko &>/dev/null; then
        local ko_version=$(ko version 2>/dev/null || echo "unknown")
        add_result "PASS" "Ko is available: $ko_version"
    else
        add_result "WARN" "Ko is not installed - will be installed during workflow"
    fi
}

# Check Kubernetes tools
check_kubernetes_tools() {
    log_section "Kubernetes Tools"
    
    local tools=(
        "kubectl:Kubernetes CLI"
        "kind:Kubernetes in Docker"
        "tkn:Tekton CLI"
    )
    
    for tool_info in "${tools[@]}"; do
        local tool="${tool_info%%:*}"
        local description="${tool_info##*:}"
        
        if command -v "$tool" &>/dev/null; then
            local version=$($tool version --client 2>/dev/null | head -n1 || echo "unknown")
            add_result "PASS" "$description is available: $version"
        else
            add_result "WARN" "$description is not installed - workflow will install if needed"
        fi
    done
}

# Validate bundle resolver configuration
validate_bundle_resolvers() {
    log_section "Bundle Resolver Configuration"
    
    local pipeline_file="${PROJECT_ROOT}/tekton/release-nightly-pipeline.yaml"
    
    if [ ! -f "$pipeline_file" ]; then
        add_result "FAIL" "Pipeline file not found for bundle resolver validation"
        return 1
    fi
    
    # Check for correct bundle references
    local expected_bundles=(
        "ghcr.io/tektoncd/catalog/upstream/tasks/git-clone:0.10"
        "ghcr.io/tektoncd/catalog/upstream/tasks/golang-test:0.2"
        "ghcr.io/tektoncd/catalog/upstream/tasks/golang-build:0.3"
    )
    
    for bundle in "${expected_bundles[@]}"; do
        if grep -q "$bundle" "$pipeline_file"; then
            add_result "PASS" "Correct bundle reference: $bundle"
        else
            add_result "WARN" "May be missing or outdated bundle: $bundle"
        fi
    done
    
    # Test bundle accessibility
    log_info "Testing bundle accessibility..."
    
    for bundle in "${expected_bundles[@]}"; do
        if command -v crane &>/dev/null; then
            if crane manifest "$bundle" &>/dev/null; then
                add_result "PASS" "Bundle accessible: $bundle"
            else
                add_result "WARN" "Bundle may not be accessible: $bundle"
            fi
        else
            add_result "WARN" "Cannot test bundle accessibility (crane not available)"
            break
        fi
    done
}

# Check GitHub repository settings
check_github_settings() {
    log_section "GitHub Repository Settings"
    
    if [ -n "${GITHUB_TOKEN:-}" ] && [ -n "${REPO_OWNER}" ] && [ -n "${REPO_NAME}" ]; then
        log_info "Checking GitHub repository settings via API..."
        
        # Check if Actions are enabled
        local actions_enabled=$(curl -s -H "Authorization: token ${GITHUB_TOKEN}" \
            "https://api.github.com/repos/${REPO_OWNER}/${REPO_NAME}" | \
            jq -r '.has_actions // false' 2>/dev/null || echo "unknown")
        
        if [ "$actions_enabled" = "true" ]; then
            add_result "PASS" "GitHub Actions are enabled"
        elif [ "$actions_enabled" = "false" ]; then
            add_result "FAIL" "GitHub Actions are disabled"
        else
            add_result "WARN" "Cannot determine GitHub Actions status"
        fi
        
        # Check if packages are enabled
        local packages_enabled=$(curl -s -H "Authorization: token ${GITHUB_TOKEN}" \
            "https://api.github.com/repos/${REPO_OWNER}/${REPO_NAME}" | \
            jq -r '.has_packages // false' 2>/dev/null || echo "unknown")
        
        if [ "$packages_enabled" = "true" ]; then
            add_result "PASS" "GitHub Packages are enabled"
        elif [ "$packages_enabled" = "false" ]; then
            add_result "WARN" "GitHub Packages may be disabled"
        else
            add_result "WARN" "Cannot determine GitHub Packages status"
        fi
    else
        add_result "WARN" "Cannot check GitHub settings (missing GITHUB_TOKEN or repository info)"
    fi
}

# Generate setup instructions
generate_setup_instructions() {
    log_section "Setup Instructions"
    
    echo ""
    log_info "Based on the validation results, here are your next steps:"
    echo ""
    
    if [ ${#ERRORS[@]} -eq 0 ]; then
        log_success "🎉 Validation passed! Your fork is ready for nightly releases."
        echo ""
        echo "To enable nightly releases:"
        echo "1. Set up the required GitHub secret:"
        echo "   - Go to Settings → Secrets and Variables → Actions"
        echo "   - Add NIGHTLY_RELEASE_TOKEN with a GitHub Personal Access Token"
        echo "   - Token needs 'packages:write' and 'contents:read' scopes"
        echo ""
        echo "2. Enable the workflow:"
        echo "   - Go to Actions tab in your repository"
        echo "   - Find 'Nightly Tekton Release' workflow"
        echo "   - Enable it if disabled"
        echo ""
        echo "3. Test the setup:"
        echo "   - Run the workflow manually first"
        echo "   - Monitor the logs for any issues"
        echo "   - Check that images are published to ghcr.io/${REPO_OWNER}/pipeline"
        echo ""
    else
        log_error "❌ Validation failed. Please fix the following issues:"
        echo ""
        for error in "${ERRORS[@]}"; do
            echo "   • $error"
        done
        echo ""
    fi
    
    if [ ${#WARNINGS[@]} -gt 0 ]; then
        log_warning "⚠️  Warnings (recommended to address):"
        echo ""
        for warning in "${WARNINGS[@]}"; do
            echo "   • $warning"
        done
        echo ""
    fi
    
    echo "Additional resources:"
    echo "• Documentation: docs/nightly-releases.md"
    echo "• Test script: scripts/test-nightly-release.sh"
    echo "• Example workflow: .github/workflows/nightly-release.yaml"
}

# Generate validation report
generate_validation_report() {
    local report_file="${PROJECT_ROOT}/fork-validation-report.md"
    
    cat > "$report_file" <<EOF
# Fork Validation Report

**Generated**: $(date)
**Repository**: ${REPO_OWNER}/${REPO_NAME}
**Branch**: $(git branch --show-current 2>/dev/null || echo 'unknown')
**Commit**: $(git rev-parse HEAD 2>/dev/null || echo 'unknown')

## Validation Results

EOF
    
    for result in "${VALIDATION_RESULTS[@]}"; do
        local status="${result%%:*}"
        local message="${result#*: }"
        
        case "$status" in
            "PASS") echo "✅ $message" >> "$report_file" ;;
            "WARN") echo "⚠️  $message" >> "$report_file" ;;
            "FAIL") echo "❌ $message" >> "$report_file" ;;
        esac
    done
    
    cat >> "$report_file" <<EOF

## Summary

- **Passed**: $(printf '%s\n' "${VALIDATION_RESULTS[@]}" | grep -c "PASS:")
- **Warnings**: $(printf '%s\n' "${VALIDATION_RESULTS[@]}" | grep -c "WARN:")
- **Failures**: $(printf '%s\n' "${VALIDATION_RESULTS[@]}" | grep -c "FAIL:")

## Next Steps

EOF
    
    if [ ${#ERRORS[@]} -eq 0 ]; then
        cat >> "$report_file" <<EOF
🎉 **Validation Passed!** Your fork is ready for nightly releases.

1. Configure GitHub secrets (NIGHTLY_RELEASE_TOKEN)
2. Enable GitHub Actions workflow
3. Test with manual workflow run
4. Monitor first few nightly releases

EOF
    else
        cat >> "$report_file" <<EOF
❌ **Validation Failed.** Please address the following issues:

EOF
        for error in "${ERRORS[@]}"; do
            echo "- $error" >> "$report_file"
        done
    fi
    
    if [ ${#WARNINGS[@]} -gt 0 ]; then
        cat >> "$report_file" <<EOF

⚠️  **Warnings to address:**

EOF
        for warning in "${WARNINGS[@]}"; do
            echo "- $warning" >> "$report_file"
        done
    fi
    
    log_info "Validation report saved to: $report_file"
}

# Main function
main() {
    echo "🔍 Fork Validation for Tekton Nightly Releases"
    echo "=============================================="
    echo ""
    
    # Run all validations
    analyze_repository
    check_required_files
    validate_github_actions
    validate_tekton_pipeline
    check_container_registry
    check_go_environment
    check_kubernetes_tools
    validate_bundle_resolvers
    check_github_settings
    
    # Generate outputs
    generate_setup_instructions
    generate_validation_report
    
    # Return appropriate exit code
    if [ ${#ERRORS[@]} -eq 0 ]; then
        echo ""
        log_success "✅ Fork validation completed successfully!"
        return 0
    else
        echo ""
        log_error "❌ Fork validation failed with ${#ERRORS[@]} error(s)"
        return 1
    fi
}

# Help function
show_help() {
    cat <<EOF
Fork Validation Script for Tekton Nightly Releases

This script validates that your fork is properly configured for nightly releases.

USAGE:
    $0 [OPTIONS]

OPTIONS:
    -h, --help              Show this help message

ENVIRONMENT VARIABLES:
    NIGHTLY_RELEASE_TOKEN   GitHub token for testing container registry access
    GITHUB_TOKEN           GitHub token for checking repository settings
    GITHUB_ACTOR           GitHub username for authentication

EXAMPLES:
    # Basic validation
    $0
    
    # With token for full validation
    NIGHTLY_RELEASE_TOKEN=ghp_xxx $0

EOF
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            show_help
            exit 0
            ;;
        *)
            log_error "Unknown option: $1"
            show_help
            exit 1
            ;;
    esac
done

# Run main function
main "$@"
