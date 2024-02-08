MODULE_NAME ?= istio

# Module Registry used for pushing the image
MODULE_REGISTRY_PORT ?= 8888
MODULE_REGISTRY ?= op-kcp-registry.localhost:$(MODULE_REGISTRY_PORT)/unsigned

# Operating system architecture
OS_ARCH ?= $(shell uname -m)

# Operating system type
OS_TYPE ?= $(shell uname)

# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION = 1.24.2

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
# target descriptions by '##'. The awk commands is responsible for reading the
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
generate-integration-test-manifest: manifests kustomize
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default -o tests/integration/manifests/generated-operator-manifest.yaml

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
test: manifests generate fmt vet envtest ## Run tests.
	KUBEBUILDER_CONTROLPLANE_START_TIMEOUT=2m KUBEBUILDER_CONTROLPLANE_STOP_TIMEOUT=2m KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path)" go test $(shell go list ./... | grep -v /tests/integration) -coverprofile cover.out

##@ Build

.PHONY: build
build: generate fmt vet ## Build manager binary.
	go build -o bin/manager main.go

.PHONY: run
run: manifests install create-kyma-system-ns build ## Run a controller from your host.
	go run ./main.go

.PHONY: docker-build
docker-build: ## Build docker image with the manager.
	IMG=$(IMG) docker build -t ${IMG} --build-arg TARGETOS=${TARGETOS} --build-arg TARGETARCH=${TARGETARCH} .

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
install: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

.PHONY: uninstall
uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/crd | kubectl delete --ignore-not-found=$(ignore-not-found) -f -

.PHONY: deploy
deploy: create-kyma-system-ns manifests kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default | kubectl apply -f -

.PHONY: undeploy
undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/default | kubectl delete --ignore-not-found=$(ignore-not-found) -f -

##@ Build Dependencies

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
KUSTOMIZE ?= $(LOCALBIN)/kustomize
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
ENVTEST ?= $(LOCALBIN)/setup-envtest
YQUERY ?= $(LOCALBIN)/yq

## Tool Versions
KUSTOMIZE_VERSION ?= v4.5.5
CONTROLLER_TOOLS_VERSION ?= v0.10.0
YQ_VERSION ?= v4

KUSTOMIZE_INSTALL_SCRIPT ?= "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh"
.PHONY: kustomize
kustomize: $(KUSTOMIZE) ## Download kustomize locally if necessary.
$(KUSTOMIZE): $(LOCALBIN)
	test -s $(LOCALBIN)/kustomize || { curl -s $(KUSTOMIZE_INSTALL_SCRIPT) | bash -s -- $(subst v,,$(KUSTOMIZE_VERSION)) $(LOCALBIN); }

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary.
$(CONTROLLER_GEN): $(LOCALBIN)
	test -s $(LOCALBIN)/controller-gen || GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION)

.PHONY: yq
yq: $(YQUERY) ## Download yq locally if necessary.
$(YQUERY): $(LOCALBIN)
	test -s $(LOCALBIN)/yq || { go get github.com/mikefarah/yq/$(YQ_VERSION) ; GOBIN=$(LOCALBIN) go install github.com/mikefarah/yq/$(YQ_VERSION) ; }

.PHONY: envtest
envtest: $(ENVTEST) ## Download envtest-setup locally if necessary.
$(ENVTEST): $(LOCALBIN)
	test -s $(LOCALBIN)/setup-envtest || GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest

##@ Module

.PHONY: module-image
module-image: docker-build docker-push ## Build the Module Image and push it to a registry defined in IMG_REGISTRY
	echo "built and pushed module image $(IMG)"

.PHONY: generate-manifests
generate-manifests: kustomize
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default > istio-manager.yaml

########## Grafana Dashboard ###########
.PHONY: grafana-dashboard
grafana-dashboard: ## Generating Grafana manifests to visualize controller status.
	cd operator && kubebuilder edit --plugins grafana.kubebuilder.io/v1-alpha

########## Integration Tests ###########
PULL_IMAGE_VERSION=PR-${PULL_NUMBER}
POST_IMAGE_VERSION=v$(shell date '+%Y%m%d')-$(shell printf %.8s ${PULL_BASE_SHA})

.PHONY: istio-integration-test
istio-integration-test: install deploy
	# Increased TEST_REQUEST_TIMEOUT to 300s to avoid timeouts on newly created k3s clusters on Prow
	cd tests/integration && TEST_REQUEST_TIMEOUT=300s && EXPORT_RESULT=true go test -v -timeout 35m -run TestIstioMain

.PHONY: aws-integration-test
aws-integration-test: install deploy
	# Increased TEST_REQUEST_TIMEOUT to 600s to avoid timeouts on Gardener clusters
	cd tests/integration && TEST_REQUEST_TIMEOUT=600s && EXPORT_RESULT=true go test -v -timeout 35m -run TestAws

.PHONY: deploy-latest-release
deploy-latest-release: create-kyma-system-ns
	cd tests/integration && ./scripts/deploy-latest-release-to-cluster.sh $(TARGET_BRANCH)

# Latest release deployed on cluster is a prerequisite, it is handled by deploy-latest-release target
.PHONY: istio-upgrade-integration-test
istio-upgrade-integration-test: deploy-latest-release generate-integration-test-manifest
	# Increased TEST_REQUEST_TIMEOUT to 300s to avoid timeouts on newly created k3s clusters on Prow
	cd tests/integration &&  TEST_REQUEST_TIMEOUT=300s && EXPORT_RESULT=true go test -v -timeout 10m -run TestIstioUpgrade

########## Gardener specific ###########

.PHONY: gardener-istio-integration-test
gardener-istio-integration-test:
	./hack/ci/gardener-integration.sh

.PHONY: gardener-perf-test
gardener-perf-test:
	./hack/ci/gardener-perf-test.sh

.PHONY: gardener-aws-integration-test
gardener-aws-integration-test:
	./hack/ci/gardener-integration-aws-specific.sh
