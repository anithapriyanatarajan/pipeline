MODULE   = $(shell env GO111MODULE=on $(GO) list -m)
DATE    ?= $(shell date +%FT%T%z)
VERSION ?= $(shell git describe --tags --always --dirty --match=v* 2> /dev/null || \
			cat $(CURDIR)/.version 2> /dev/null || echo v0)
PKGS     = $(or $(PKG),$(shell env GO111MODULE=on $(GO) list ./... ))
TESTPKGS = $(shell env GO111MODULE=on $(GO) list -f \
			'{{ if or .TestGoFiles .XTestGoFiles }}{{ .ImportPath }}{{ end }}' \
			$(PKGS))
BIN      = $(CURDIR)/.bin
WOKE 	?= go run -modfile go.mod github.com/get-woke/woke

GOLANGCI_VERSION := $(shell yq '.jobs.linting.steps[] | select(.name == "golangci-lint") | .with.version' .github/workflows/ci.yaml)
WOKE_VERSION     = v0.19.0

GO           = go
TIMEOUT_UNIT = 5m
TIMEOUT_E2E  = 20m
V = 0
Q = $(if $(filter 1,$V),,@)
M = $(shell printf "\033[34;1m🐱\033[0m")

export GO111MODULE=on

COMMANDS=$(patsubst cmd/%,%,$(wildcard cmd/*))
BINARIES=$(addprefix bin/,$(COMMANDS))

.PHONY: all
all: fmt $(BINARIES) | $(BIN) ; $(info $(M) building executable…) @ ## Build program binary

$(BIN):
	@mkdir -p $@
$(BIN)/%: | $(BIN) ; $(info $(M) building $(PACKAGE)…)
	$Q tmp=$$(mktemp -d); \
	   env GO111MODULE=off GOPATH=$$tmp GOBIN=$(BIN) $(GO) get $(PACKAGE) \
		|| ret=$$?; \
	   rm -rf $$tmp ; exit $$ret

FORCE:

bin/%: cmd/% FORCE
	$Q $(GO) build -mod=vendor $(LDFLAGS) -v -o $@ ./$<

.PHONY: cross
cross: amd64 arm arm64 s390x ppc64le ## build cross platform binaries

.PHONY: amd64
amd64:
	GOOS=linux GOARCH=amd64 go build -mod=vendor $(LDFLAGS) ./cmd/...

.PHONY: arm
arm:
	GOOS=linux GOARCH=arm go build -mod=vendor $(LDFLAGS) ./cmd/...

.PHONY: arm64
arm64:
	GOOS=linux GOARCH=arm64 go build -mod=vendor $(LDFLAGS) ./cmd/...

.PHONY: s390x
s390x:
	GOOS=linux GOARCH=s390x go build -mod=vendor $(LDFLAGS) ./cmd/...

.PHONY: ppc64le
ppc64le:
	GOOS=linux GOARCH=ppc64le go build -mod=vendor $(LDFLAGS) ./cmd/...

KO = $(or ${KO_BIN},${KO_BIN},$(BIN)/ko)
$(BIN)/ko: PACKAGE=github.com/google/ko

.PHONY: apply
apply: | $(KO) ; $(info $(M) ko apply -R -f config/) @ ## Apply config to the current cluster
	$Q $(KO) apply -R -f config

.PHONY: resolve
resolve: | $(KO) ; $(info $(M) ko resolve -R -f config/) @ ## Resolve config to the current cluster
	$Q $(KO) resolve --push=false --oci-layout-path=$(BIN)/oci -R -f config

.PHONY: generated
generated: | vendor ; $(info $(M) update generated files) ## Update generated files
	$Q ./hack/update-codegen.sh

.PHONY: vendor
vendor:
	$Q ./hack/update-deps.sh

## Tests
TEST_UNIT_TARGETS := test-unit-verbose test-unit-race test-unit-verbose-and-race
test-unit-verbose:          ARGS=-v
test-unit-race:             ARGS=-race
test-unit-verbose-and-race: ARGS=-v -race
$(TEST_UNIT_TARGETS): test-unit
.PHONY: $(TEST_UNIT_TARGETS) test-unit
test-unit: ## Run unit tests
	$(GO) test -timeout $(TIMEOUT_UNIT) $(ARGS) ./...

TEST_E2E_TARGETS := test-e2e-short test-e2e-verbose test-e2e-race
test-e2e-short:   ARGS=-short
test-e2e-verbose: ARGS=-v
test-e2e-race:    ARGS=-race
$(TEST_E2E_TARGETS): test-e2e
.PHONY: $(TEST_E2E_TARGETS) test-e2e
test-e2e:  ## Run end-to-end tests
	$(GO) test -timeout $(TIMEOUT_E2E) -tags e2e $(ARGS) ./test/...

.PHONY: test-yamls
test-yamls: ## Run yaml tests
	./test/e2e-tests-yaml.sh --run-tests

.PHONY: check tests
check tests: test-unit test-e2e test-yamls

RAM = $(BIN)/ram
$(BIN)/ram: PACKAGE=go.sbr.pm/ram

.PHONY: watch-test
watch-test: | $(RAM) ; $(info $(M) watch and run tests) @ ## Watch and run tests
	$Q $(RAM) -- -failfast

.PHONY: watch-resolve
watch-resolve: | $(KO) ; $(info $(M) watch and resolve config) @ ## Watch and build to the current cluster
	$Q $(KO) resolve -W --push=false --oci-layout-path=$(BIN)/oci -f config 1>/dev/null

.PHONY: watch-config
watch-config: | $(KO) ; $(info $(M) watch and apply config) @ ## Watch and apply to the current cluster
	$Q $(KO) apply -W -f config

## Linters configuration and targets
# TODO(vdemeester) gofmt and goimports checks (run them with -w and make a diff)

GOLINT = $(BIN)/golint
$(BIN)/golint: PACKAGE=golang.org/x/lint/golint

.PHONY: golint
golint: | $(GOLINT) ; $(info $(M) running golint…) @ ## Run golint
	$Q $(GOLINT) -set_exit_status $(PKGS)

.PHONY: vet
vet: | ; $(info $(M) running go vet…) @ ## Run go vet
	$Q go vet ./...

INEFFASSIGN = $(BIN)/ineffassign
$(BIN)/ineffassign: PACKAGE=github.com/gordonklaus/ineffassign

.PHONY: ineffassign
ineffassign: | $(INEFFASSIGN) ; $(info $(M) running static ineffassign…) @ ## Run ineffassign
	$Q $(INEFFASSIGN) .

STATICCHECK = $(BIN)/staticcheck
$(BIN)/staticcheck: PACKAGE=honnef.co/go/tools/cmd/staticcheck

.PHONY: staticcheck
staticcheck: | $(STATICCHECK) ; $(info $(M) running static check…) @ ## Run staticcheck
	$Q $(STATICCHECK) ./...

DUPL = $(BIN)/dupl
$(BIN)/dupl: PACKAGE=github.com/mibk/dupl

.PHONY: dupl
dupl: | $(DUPL) ; $(info $(M) running dupl…) ## Run dupl
	$Q $(DUPL)

ERRCHECK = $(BIN)/errcheck
$(BIN)/errcheck: PACKAGE=github.com/kisielk/errcheck

.PHONY: errcheck
errcheck: | $(ERRCHECK) ; $(info $(M) running errcheck…) ## Run errcheck
	$Q $(ERRCHECK) ./...

GOLANGCILINT = $(BIN)/golangci-lint-$(GOLANGCI_VERSION)
$(BIN)/golangci-lint-$(GOLANGCI_VERSION): ; $(info $(M) getting golangci-lint $(GOLANGCI_VERSION))
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(BIN) $(GOLANGCI_VERSION)
	mv $(BIN)/golangci-lint $(BIN)/golangci-lint-$(GOLANGCI_VERSION)

.PHONY: golangci-lint
golangci-lint: | $(GOLANGCILINT) ; $(info $(M) running golangci-lint…) @ ## Run golangci-lint
	$Q $(GOLANGCILINT) config verify
	$Q $(GOLANGCILINT) run --modules-download-mode=vendor --max-issues-per-linter=0 --max-same-issues=0 --timeout 5m

.PHONY: golangci-lint-check
golangci-lint-check: | $(GOLANGCILINT) ; $(info $(M) Testing if golint has been done…) @ ## Run golangci-lint for build tests CI job
	$Q $(GOLANGCILINT) run -j 1 --color=never

GOIMPORTS = $(BIN)/goimports
$(BIN)/goimports: | $(BIN) ; $(info $(M) building goimports…)
	GOBIN=$(BIN) go install golang.org/x/tools/cmd/goimports@latest

.PHONY: goimports
goimports: | $(GOIMPORTS) ; $(info $(M) running goimports…) ## Run goimports
	$Q $(GOIMPORTS) -l -e -w pkg cmd test

.PHONY: fmt
fmt: ; $(info $(M) running gofmt…) @ ## Run gofmt on all source files
	$Q $(GO) fmt $(PKGS)

WOKE = $(BIN)/woke
$(BIN)/woke: ; $(info $(M) getting woke $(WOKE_VERSION))
	cd tools; GOBIN=$(BIN) go install github.com/get-woke/woke@$(WOKE_VERSION)

.PHONY: woke 
woke: | $(WOKE) ; $(info $(M) running woke...) @ ## Run woke
	$Q $(WOKE) -c https://github.com/canonical/Inclusive-naming/raw/main/config.yml

.PHONY: yamlint
YAMLLINT := $(shell find . -path ./vendor -prune -o -type f -regex ".*y[a]ml" -print) 
yamllint: | $(BIN) ; $(info $(M) running yamlint…)
	yamllint -c .yamllint $(YAMLLINT)

# Misc

.PHONY: clean
clean: ; $(info $(M) cleaning…)	@ ## Cleanup everything
	@rm -rf $(BIN)
	@rm -rf bin
	@rm -rf test/tests.* test/coverage.*

.PHONY: help
help:
	@grep -hE '^[ a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-17s\033[0m %s\n", $$1, $$2}'

.PHONY: version
version:
	@echo $(VERSION)

## Dashboard targets

DASHBOARD_IMAGE ?= tekton-dashboard:latest
KIND_CLUSTER_NAME ?= tekton-test

.PHONY: build-dashboard
build-dashboard: ## Build dashboard binary
	$(info $(M) building dashboard binary…)
	$Q $(GO) build -mod=vendor -v -o bin/dashboard ./cmd/dashboard

.PHONY: build-dashboard-image
build-dashboard-image: ## Build dashboard Docker image
	$(info $(M) building dashboard Docker image…)
	$Q docker build -t $(DASHBOARD_IMAGE) -f Dockerfile.dashboard .

.PHONY: kind-cluster
kind-cluster: ## Create a kind cluster for testing
	$(info $(M) creating kind cluster…)
	$Q kind create cluster --name $(KIND_CLUSTER_NAME) || true
	$Q kubectl config use-context kind-$(KIND_CLUSTER_NAME)

.PHONY: kind-delete
kind-delete: ## Delete the kind cluster
	$(info $(M) deleting kind cluster…)
	$Q kind delete cluster --name $(KIND_CLUSTER_NAME)

.PHONY: kind-load-dashboard
kind-load-dashboard: build-dashboard-image ## Load dashboard image into kind
	$(info $(M) loading dashboard image into kind…)
	$Q kind load docker-image $(DASHBOARD_IMAGE) --name $(KIND_CLUSTER_NAME)

.PHONY: deploy-tekton
deploy-tekton: ## Deploy Tekton Pipelines to the cluster
	$(info $(M) deploying Tekton Pipelines…)
	$Q kubectl apply -f https://storage.googleapis.com/tekton-releases/pipeline/latest/release.yaml
	$Q kubectl wait --for=condition=Ready pods --all -n tekton-pipelines --timeout=300s

.PHONY: deploy-dashboard
deploy-dashboard: ## Deploy the dashboard to the cluster
	$(info $(M) deploying dashboard…)
	$Q kubectl apply -f config/dashboard/

.PHONY: deploy-dashboard-demo
deploy-dashboard-demo: ## Deploy demo pipelines
	$(info $(M) deploying demo pipelines…)
	$Q kubectl apply -f examples/dashboard-demo/

.PHONY: dashboard-logs
dashboard-logs: ## Show dashboard logs
	$Q kubectl logs -n tekton-pipelines -l app.kubernetes.io/name=tekton-dashboard --tail=100 -f

.PHONY: dashboard-port-forward
dashboard-port-forward: ## Port forward to access dashboard (http://localhost:8080)
	$(info $(M) port forwarding to dashboard on http://localhost:8080…)
	$Q kubectl port-forward -n tekton-pipelines svc/tekton-dashboard 8080:8080

.PHONY: test-dashboard-local
test-dashboard-local: kind-cluster deploy-tekton kind-load-dashboard deploy-dashboard ## Full local dashboard test in kind
	$(info $(M) ========================================)
	$(info $(M) Dashboard deployed successfully!)
	$(info $(M) Run 'make dashboard-port-forward' to access it)
	$(info $(M) Or run 'make dashboard-demo-run' to create demo data)
	$(info $(M) ========================================)

.PHONY: dashboard-demo-run
dashboard-demo-run: deploy-dashboard-demo ## Run demo pipeline runs
	$(info $(M) creating demo pipeline runs…)
	$Q for i in 1 2 3 4 5; do \
		kubectl create -f examples/dashboard-demo/01-simple-pipelinerun.yaml 2>/dev/null || true; \
		sleep 2; \
	done
	$(info $(M) demo runs created! Access dashboard with: make dashboard-port-forward)

.PHONY: dashboard-status
dashboard-status: ## Check dashboard deployment status
	$(info $(M) checking dashboard status…)
	$Q kubectl get pods -n tekton-pipelines -l app.kubernetes.io/name=tekton-dashboard
	$Q kubectl get svc -n tekton-pipelines tekton-dashboard

.PHONY: dashboard-cleanup
dashboard-cleanup: ## Remove dashboard from cluster
	$(info $(M) cleaning up dashboard…)
	$Q kubectl delete -f config/dashboard/ --ignore-not-found=true
	$Q kubectl delete -f examples/dashboard-demo/ --ignore-not-found=true
