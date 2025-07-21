#!/bin/bash

# Quick Setup Script for Tekton Nightly Releases
# This script helps new fork maintainers get started quickly

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
log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[✓]${NC} $1"; }
log_warning() { echo -e "${YELLOW}[⚠]${NC} $1"; }
log_error() { echo -e "${RED}[✗]${NC} $1"; }
log_step() { echo -e "\n${PURPLE}▶ $1${NC}"; }

# Welcome message
show_welcome() {
    cat << "EOF"
🌙 Tekton Nightly Release Quick Setup
====================================

This script will help you set up nightly releases for your Tekton Pipeline fork.

What this script does:
• Validates your repository setup
• Checks for required files
• Provides GitHub secrets configuration guidance
• Tests your setup (optional)
• Gives you next steps

EOF
}

# Check if we're in the right directory
check_repository() {
    log_step "Checking repository setup"
    
    if ! git rev-parse --git-dir &>/dev/null; then
        log_error "Not in a git repository"
        exit 1
    fi
    
    local remote_url=$(git remote get-url origin 2>/dev/null || echo "")
    if [[ ! "$remote_url" =~ github\.com.*pipeline ]]; then
        log_warning "This doesn't appear to be a Tekton Pipeline repository"
        read -p "Continue anyway? (y/N): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            exit 1
        fi
    fi
    
    local repo_owner=""
    if [[ "$remote_url" =~ github\.com[:/]([^/]+)/([^/.]+) ]]; then
        repo_owner="${BASH_REMATCH[1]}"
        log_info "Repository owner: $repo_owner"
        
        if [[ "$repo_owner" == "tektoncd" ]]; then
            log_info "This is the upstream repository"
        else
            log_success "This is a fork (owner: $repo_owner)"
        fi
    fi
    
    export REPO_OWNER="$repo_owner"
}

# Check for required files
check_files() {
    log_step "Checking required files"
    
    local required_files=(
        ".github/workflows/nightly-release.yaml:GitHub Actions workflow"
        "tekton/release-nightly-pipeline.yaml:Tekton pipeline definition"
        "tekton/publish-nightly.yaml:Publishing task"
        "docs/nightly-releases.md:Documentation"
        "scripts/validate-fork.sh:Fork validation script"
    )
    
    local missing_files=()
    
    for file_info in "${required_files[@]}"; do
        local file="${file_info%%:*}"
        local description="${file_info##*:}"
        
        if [[ -f "${PROJECT_ROOT}/$file" ]]; then
            log_success "$description: $file"
        else
            log_error "Missing: $file ($description)"
            missing_files+=("$file")
        fi
    done
    
    if [[ ${#missing_files[@]} -gt 0 ]]; then
        log_error "Missing ${#missing_files[@]} required file(s)"
        log_info "Please ensure you have the complete nightly release setup"
        exit 1
    fi
}

# Guide user through GitHub secrets setup
setup_github_secrets() {
    log_step "GitHub Secrets Configuration"
    
    cat << EOF
To enable nightly releases, you need to configure GitHub secrets:

1. Go to your repository on GitHub
2. Navigate to: Settings → Secrets and Variables → Actions
3. Click "New repository secret"
4. Add the following secret:

   Name: NIGHTLY_RELEASE_TOKEN
   Value: [Your GitHub Personal Access Token]

To create a Personal Access Token:
1. Go to GitHub Settings → Developer Settings → Personal Access Tokens
2. Click "Generate new token (classic)"
3. Select these scopes:
   ✓ packages:write (required for pushing to GHCR)
   ✓ contents:read (required for accessing repository)
4. Copy the token and use it as the secret value

EOF

    read -p "Have you configured the GitHub secret? (y/N): " -n 1 -r
    echo
    
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        log_success "GitHub secrets configured"
        return 0
    else
        log_warning "You can configure secrets later and re-run this script"
        return 1
    fi
}

# Test the setup
test_setup() {
    log_step "Testing your setup"
    
    if [[ ! -f "${PROJECT_ROOT}/scripts/validate-fork.sh" ]]; then
        log_error "Validation script not found"
        return 1
    fi
    
    log_info "Running fork validation..."
    if "${PROJECT_ROOT}/scripts/validate-fork.sh"; then
        log_success "Validation passed!"
        return 0
    else
        log_error "Validation failed - please check the output above"
        return 1
    fi
}

# Show next steps
show_next_steps() {
    log_step "Next Steps"
    
    cat << EOF
🎉 Setup is ready! Here's what to do next:

1. Enable GitHub Actions (if not already enabled):
   • Go to your repository's Actions tab
   • Click "I understand my workflows, go ahead and enable them"

2. Test with a manual run:
   • Go to Actions → "Tekton Nightly Release"
   • Click "Run workflow"
   • Select branch: nightly-pipeline-gha
   • Click "Run workflow"

3. Monitor the results:
   • Check workflow logs for any errors
   • Verify images are published to: ghcr.io/${REPO_OWNER}/pipeline/

4. Enable scheduled runs:
   • If manual run succeeds, nightly releases will run automatically at 03:00 UTC

📚 Additional Resources:
• Complete documentation: docs/nightly-releases.md
• Production checklist: docs/production-checklist.md
• Troubleshooting: docs/nightly-releases.md#troubleshooting

🆘 Need help?
• Run validation: ./scripts/validate-fork.sh
• Full testing: ./scripts/test-nightly-release.sh
• Open an issue: https://github.com/tektoncd/pipeline/issues

EOF
}

# Interactive mode
interactive_setup() {
    log_info "Starting interactive setup..."
    
    check_repository
    check_files
    
    local secrets_configured=false
    if setup_github_secrets; then
        secrets_configured=true
    fi
    
    if [[ "$secrets_configured" == "true" ]]; then
        read -p "Run validation test now? (Y/n): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Nn]$ ]]; then
            if test_setup; then
                log_success "🎉 Your fork is ready for nightly releases!"
            else
                log_warning "Setup needs attention - check validation output"
            fi
        fi
    fi
    
    show_next_steps
}

# Quick validation mode
quick_validation() {
    log_info "Running quick validation..."
    
    check_repository
    check_files
    
    if [[ -f "${PROJECT_ROOT}/scripts/validate-fork.sh" ]]; then
        "${PROJECT_ROOT}/scripts/validate-fork.sh"
    else
        log_error "Validation script not found"
        exit 1
    fi
}

# Show help
show_help() {
    cat << EOF
Tekton Nightly Release Quick Setup Script

USAGE:
    $0 [OPTIONS]

OPTIONS:
    -h, --help              Show this help message
    -q, --quick             Run quick validation only (no interactive setup)
    -v, --validate          Run validation script only
    --check-files           Check required files only

EXAMPLES:
    # Interactive setup (recommended for first time)
    $0
    
    # Quick validation
    $0 --quick
    
    # Just validate configuration
    $0 --validate

EOF
}

# Main function
main() {
    case "${1:-}" in
        -h|--help)
            show_help
            exit 0
            ;;
        -q|--quick)
            show_welcome
            quick_validation
            ;;
        -v|--validate)
            if [[ -f "${PROJECT_ROOT}/scripts/validate-fork.sh" ]]; then
                "${PROJECT_ROOT}/scripts/validate-fork.sh"
            else
                log_error "Validation script not found"
                exit 1
            fi
            ;;
        --check-files)
            check_repository
            check_files
            log_success "All required files present"
            ;;
        "")
            show_welcome
            interactive_setup
            ;;
        *)
            log_error "Unknown option: $1"
            show_help
            exit 1
            ;;
    esac
}

# Run main function
main "$@"
