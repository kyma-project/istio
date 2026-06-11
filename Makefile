MODULE_NAME ?= istio

# Module Registry used for pushing the image
MODULE_REGISTRY_PORT ?= 8888
MODULE_REGISTRY ?= op-kcp-registry.localhost:$(MODULE_REGISTRY_PORT)/unsigned

# Operating system architecture
OS_ARCH ?= $(shell uname -m)

# Operating system type
OS_TYPE ?= $(shell uname)

VERSION ?= dev

# Istio install binary path for running the installation in separate process
ISTIO_INSTALL_BIN_PATH = ./bin/istio_install

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Setting SHELL to bash allows bash commands to be executed by recipes.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

# Image URL to use all building/pushing image targets
APP_NAME = istio-manager

# Image URL to use all building/pushing image targets
IMG_REGISTRY_PORT ?= $(MODULE_REGISTRY_PORT)
IMG_REGISTRY ?= op-skr-registry.localhost:$(IMG_REGISTRY_PORT)/unsigned/operator-images
IMG ?= $(IMG_REGISTRY)/$(MODULE_NAME)-operator:$(MODULE_VERSION)

COMPONENT_CLI_VERSION ?= latest

# It is required for upgrade integration test
TARGET_BRANCH ?= ""

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk command is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: manifests
manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases

.PHONY: generate-integration-test-manifest
generate-integration-test-manifest: manifests kustomize module-version
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default -o tests/integration/steps/operator_generated_manifest.yaml

.PHONY: generate-upgrade-test-manifest
generate-upgrade-test-manifest: manifests kustomize module-version
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default -o tests/e2e/tests/upgrade/operator_generated_manifest.yaml

.PHONY: generate
generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: test
test: manifests generate fmt vet setup-envtest ## Run tests.
	KUBEBUILDER_CONTROLPLANE_START_TIMEOUT=2m KUBEBUILDER_CONTROLPLANE_STOP_TIMEOUT=2m KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path)" go test $$(go list ./... | grep -v /tests/integration | grep -v /tests/performance-grpc | grep -v /tests/e2e) -coverprofile cover.out

.PHONY: test-experimental-tag
test-experimental-tag: manifests generate fmt vet setup-envtest ## Run tests.
	KUBEBUILDER_CONTROLPLANE_START_TIMEOUT=2m KUBEBUILDER_CONTROLPLANE_STOP_TIMEOUT=2m KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path)" go test -tags experimental $$(go list ./... | grep -v /tests/integration | grep -v /tests/performance-grpc | grep -v /tests/e2e) -coverprofile cover.out

##@ Build

.PHONY: build
build: generate fmt vet ## Build manager binary.
	go build -o bin/manager cmd/main.go
	go build -o $(ISTIO_INSTALL_BIN_PATH) cmd/istio-install/main.go

.PHONY: run
run: manifests install build create-kyma-system-ns ## Run a controller from your host.
	ISTIO_INSTALL_BIN_PATH=$(ISTIO_INSTALL_BIN_PATH) go run ./cmd/main.go

TARGET_OS ?= linux
TARGET_ARCH ?= amd64

.PHONY: docker-build
docker-build: ## Build docker image with the manager.
	IMG=$(IMG) docker buildx build -t ${IMG} --platform=${TARGET_OS}/${TARGET_ARCH} .

.PHONY: docker-build-experimental
docker-build-experimental: ## Build docker image with the experimental manager
	IMG=$(IMG) docker build -t ${IMG} --build-arg GO_BUILD_TAGS=experimental --platform=${TARGET_OS}/${TARGET_ARCH} .

.PHONY: docker-push
docker-push: ## Push docker image with the manager.
	docker push ${IMG}

##@ Local

.PHONY: local-run
local-run:
	make -C hack/local run

.PHONY: local-stop
local-stop:
	make -C hack/local stop

##@ Deployment

ifndef ignore-not-found
  ignore-not-found = false
endif

.PHONY: create-kyma-system-ns
create-kyma-system-ns:
	kubectl create namespace kyma-system --dry-run=client -o yaml | kubectl apply -f -
	kubectl label namespace kyma-system istio-injection=enabled --overwrite

.PHONY: install
install: manifests kustomize module-version ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	@out="$$( $(KUSTOMIZE) build config/crd 2>/dev/null || true )"; \
	if [ -n "$$out" ]; then echo "$$out" | kubectl apply -f -; else echo "No CRDs to install; skipping."; fi

.PHONY: uninstall
uninstall: manifests kustomize module-version ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	@out="$$( $(KUSTOMIZE) build config/crd 2>/dev/null || true )"; \
	if [ -n "$$out" ]; then echo "$$out" | kubectl delete --ignore-not-found=$(ignore-not-found) -f -; else echo "No CRDs to delete; skipping."; fi

.PHONY: deploy
deploy: create-kyma-system-ns manifests kustomize module-version ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
ifeq (,$(findstring experimental,$(VERSION)))
	$(KUSTOMIZE) build config/regular | kubectl apply -f -
else
	$(KUSTOMIZE) build config/default | kubectl apply -f -
endif


.PHONY: undeploy
undeploy: kustomize ## Undeploy controller from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/default | kubectl delete --ignore-not-found=$(ignore-not-found) -f -

##@ Dependencies

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
KUSTOMIZE ?= $(LOCALBIN)/kustomize
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
ENVTEST ?= $(LOCALBIN)/setup-envtest
GOTESTSUM ?= $(LOCALBIN)/gotestsum

## Tool Versions
KUSTOMIZE_VERSION ?= v5.7.1
CONTROLLER_TOOLS_VERSION ?= v0.19.0
ENVTEST_VERSION ?= latest
#ENVTEST_K8S_VERSION is the version of Kubernetes to use for setting up ENVTEST binaries (i.e. 1.31)
ENVTEST_K8S_VERSION ?= $(shell go list -m -f "{{ .Version }}" k8s.io/api | awk -F'[v.]' '{printf "1.%d", $$3}')

.PHONY: kustomize
kustomize: $(KUSTOMIZE) ## Download kustomize locally if necessary.
$(KUSTOMIZE): $(LOCALBIN)
	$(call go-install-tool,$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v5,$(KUSTOMIZE_VERSION))

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary.
$(CONTROLLER_GEN): $(LOCALBIN)
	$(call go-install-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen,$(CONTROLLER_TOOLS_VERSION))

.PHONY: gotestsum
gotestsum:
	$(call go-install-tool,$(GOTESTSUM),gotest.tools/gotestsum,latest)  

.PHONY: setup-envtest
setup-envtest: envtest ## Download the binaries required for ENVTEST in the local bin directory.
	@echo "Setting up envtest binaries for Kubernetes version $(ENVTEST_K8S_VERSION)..."
	@$(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path || { \
		echo "Error: Failed to set up envtest binaries for version $(ENVTEST_K8S_VERSION)."; \
		exit 1; \
	}

.PHONY: envtest
envtest: $(ENVTEST) ## Download setup-envtest locally if necessary.
$(ENVTEST): $(LOCALBIN)
	$(call go-install-tool,$(ENVTEST),sigs.k8s.io/controller-runtime/tools/setup-envtest,$(ENVTEST_VERSION))

# go-install-tool will 'go install' any package with custom target and name of binary, if it doesn't exist
# $1 - target path with name of binary
# $2 - package url which can be installed
# $3 - specific version of package
define go-install-tool
@[ -f "$(1)-$(3)" ] && [ "$$(readlink -- "$(1)" 2>/dev/null)" = "$(1)-$(3)" ] || { \
set -e; \
package=$(2)@$(3) ;\
echo "Downloading $${package}" ;\
rm -f $(1) ;\
GOBIN=$(LOCALBIN) go install $${package} ;\
mv $(1) $(1)-$(3) ;\
} ;\
ln -sf $$(realpath $(1)-$(3)) $(1)
endef

##@ E2E Tests
.PHONY: %-e2e-test
%-e2e-test: gotestsum deploy
	@echo "Running E2E test: $*"
	go clean -testcache
	$(GOTESTSUM) --format testname --rerun-fails --packages="./tests/e2e/tests/$*/..." --junitfile "./tests/e2e/tests/$*/report.xml" -- -timeout 20m
	@echo "Finished E2E test: $*"

##@ Module

.PHONY: module-image
module-image: docker-build docker-push ## Build the Module Image and push it to a registry defined in IMG_REGISTRY
	echo "built and pushed module image $(IMG)"

.PHONY: generate-manifests
generate-manifests: kustomize module-version
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
ifeq (,$(findstring experimental,$(VERSION)))
	echo "Generating manifest for regular and fast channel releases"
	$(KUSTOMIZE) build config/regular > istio-manager.yaml
else
	echo "Generating manifest for experimental channel releases"
	$(KUSTOMIZE) build config/default > istio-manager.yaml
endif
	cat config/namespace/istio_system_namespace.yaml >> istio-manager.yaml

########## Grafana Dashboard ###########
.PHONY: grafana-dashboard
grafana-dashboard: ## Generating Grafana manifests to visualize controller status.
	cd operator && kubebuilder edit --plugins grafana.kubebuilder.io/v1-alpha

########## Integration Tests ###########
PULL_IMAGE_VERSION=PR-${PULL_NUMBER}
POST_IMAGE_VERSION=v$(shell date '+%Y%m%d')-$(shell printf %.8s ${PULL_BASE_SHA})

.PHONY: grpc-performance-test
grpc-performance-test:
	make -c tests/performance-grpc deploy-helm
	make -c tests/performance-grpc grpc-load-test
	make -c tests/performance-grpc export-results

.PHONY: deploy-latest-release
deploy-latest-release: create-kyma-system-ns
	./hack/ci/deploy-latest-release-to-cluster.sh $(TARGET_BRANCH)

########## Gardener specific ###########

.PHONY: module-version
module-version:
	sed 's/VERSION/$(VERSION)/g' config/default/kustomization.template.yaml > config/default/kustomization.yaml

########## Docs generation ###########
bin/crd-ref-docs:
	mkdir -p bin
	wget "https://github.com/elastic/crd-ref-docs/releases/download/v0.2.0/crd-ref-docs_0.2.0_${OS_TYPE}_${OS_ARCH}.tar.gz" -O bin/crd-ref-docs.tar.gz
	mkdir -p bin/crd-ref-docs-x
	tar -xzf bin/crd-ref-docs.tar.gz -C bin/crd-ref-docs-x
	rm bin/crd-ref-docs.tar.gz
	mv bin/crd-ref-docs-x/crd-ref-docs bin/crd-ref-docs
	rm -r bin/crd-ref-docs-x

.PHONY: generate-crd-docs
generate-crd-docs: bin/crd-ref-docs ## Generate CRD reference docs
	./bin/crd-ref-docs \
	--output-path=docs/user/04-00-istio-custom-resource.md \
	--source-path=api/v1alpha2 \
	--renderer=markdown \
	--config=crd-ref-docs/config.yaml \
	--templates-dir=crd-ref-docs/templates \
	--max-depth=25
	sed -i'' -e 's/Optional: \\{\\}/Optional/g' docs/user/04-00-istio-custom-resource.md
	sed -i'' -e 's/Required: \\{\\}/Required/g' docs/user/04-00-istio-custom-resource.md
	sed -i'' -e 's/XIntOrString: \\{\\}/XIntOrString/g' docs/user/04-00-istio-custom-resource.md
	sed -i'' -e '1N;$$!N;/\n.*ReasonWithMessage/!P;D' docs/user/04-00-istio-custom-resource.md
	sed -i'' -e '/ReasonWithMessage/d' docs/user/04-00-istio-custom-resource.md
	sed -i'' -e 's/\\}/\}/g' docs/user/04-00-istio-custom-resource.md
	sed -i'' -e 's/\\{/\{/g' docs/user/04-00-istio-custom-resource.md
	rm -f docs/user/04-00-istio-custom-resource.md-e
