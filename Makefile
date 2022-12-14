# VERSION defines the project version for the bundle.
# Update this value when you upgrade the version of your project.
# To re-generate a bundle for another specific version without changing the standard setup, you can:
# - use the VERSION as arg of the bundle target (e.g make bundle VERSION=0.0.2)
# - use environment variables to overwrite this value (e.g export VERSION=0.0.2)
VERSION ?= 0.0.1

# Image URL to use all building/pushing image targets
REGISTRY ?= quay.io
ORG ?= fmuntean
APPSTUDIO_IMG ?= $(REGISTRY)/$(ORG)/apimanager:latest-appstudio
HACBS_IMG ?= $(REGISTRY)/$(ORG)/apimanager:latest-hacbs
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

.PHONY: all
all: build

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

# kcp specific
APIEXPORT_PREFIX ?= apimanger
APIEXPORT_WORKSPACE ?= root:my-org:api-manager-ws

##@ Development

.PHONY: manifests
manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases

.PHONY: apiresourceschemas
apiresourceschemas: $(KUSTOMIZE) ## Convert CRDs from config/crds to APIResourceSchemas. Specify APIEXPORT_PREFIX as needed.
	$(KUSTOMIZE) build config/crd | kubectl kcp crd snapshot -f - --prefix $(APIEXPORT_PREFIX) > config/kcp/$(APIEXPORT_PREFIX).apiresourceschemas.yaml


#
.PHONY: spapiexports
spapiexports: ## Create service providers APIExports for testing purposes
	kubectl ws $(APIEXPORT_WORKSPACE)
	kubectl apply -f ./test/spApiExports/


#
.PHONY: apibinding
apibinding: ## Generate ApiBinding in the current namespace for the ApiExport, used for testing the controller.
	$( eval WORKSPACE = $(shell kubectl kcp workspace . --short))
	sed 's/WORKSPACE/$(APIEXPORT_WORKSPACE)/' ./test/apibinding.yaml | kubectl apply -f -
	kubectl wait --for=condition=Ready apibinding/api-manager-binding


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
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path)" go test ./... -coverprofile cover.out

##@ Build

.PHONY: build
build: generate fmt vet ## Build manager binary.
	go build -o bin/manager main.go

.PHONY: run
run: manifests generate fmt vet ## Run a controller from your host.
	go run ./main.go

.PHONY: docker-build
docker-build: test ## Build docker image with the manager.
	docker build --build-arg CHART_PATH=helm/appstudio -t ${APPSTUDIO_IMG} .
	docker build --build-arg CHART_PATH=helm/hacbs -t ${HACBS_IMG} .

.PHONY: docker-push
docker-push: ## Push docker image with the manager.
	docker push ${APPSTUDIO_IMG}
	docker push ${HACBS_IMG}

.PHONY: podman-build
podman-build: test ## Build docker image with the manager.
	podman build --build-arg CHART_PATH=helm/appstudio -t ${APPSTUDIO_IMG} .
	podman build --build-arg CHART_PATH=helm/hacbs -t ${HACBS_IMG} .

.PHONY: podman-push
podman-push: ## Push docker image with the manager.
	podman push ${APPSTUDIO_IMG}
	podman push ${HACBS_IMG}

##@ Deployment

ifndef ignore-not-found
  ignore-not-found = false
endif

.PHONY: install
install: manifests $(KUSTOMIZE) ## Install APIResourceSchemas and APIExport into kcp (using $KUBECONFIG or ~/.kube/config).
	$(KUSTOMIZE) build config/kcp | kubectl --kubeconfig $(KUBECONFIG) apply -f -

.PHONY: uninstall
uninstall: manifests $(KUSTOMIZE) ## Uninstall APIResourceSchemas and APIExport from kcp (using $KUBECONFIG or ~/.kube/config). Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/kcp | kubectl --kubeconfig $(KUBECONFIG) delete --ignore-not-found=$(ignore-not-found) -f -

.PHONY: deploy_appstudio
deploy_appstudio: manifests kustomize ## Deploy the appstudio controller to the K8s cluster specified in ~/.kube/config.
	cd config/manager && $(KUSTOMIZE) edit set image controller=${APPSTUDIO_IMG}
	$(KUSTOMIZE) build config/default | kubectl apply -f -

.PHONY: deploy_hacbs
deploy_hacbs: manifests kustomize ## Deploy the hacbs controller to the K8s cluster specified in ~/.kube/config.
	cd config/manager && $(KUSTOMIZE) edit set image controller=${HACBS_IMG}
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

## Tool Versions
KUSTOMIZE_VERSION ?= v4.5.5
CONTROLLER_TOOLS_VERSION ?= v0.9.2

KUSTOMIZE_INSTALL_SCRIPT ?= "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh"
.PHONY: kustomize
kustomize: $(KUSTOMIZE) ## Download kustomize locally if necessary.
$(KUSTOMIZE): $(LOCALBIN)
	test -s $(LOCALBIN)/kustomize || { curl -s $(KUSTOMIZE_INSTALL_SCRIPT) | bash -s -- $(subst v,,$(KUSTOMIZE_VERSION)) $(LOCALBIN); }

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary.
$(CONTROLLER_GEN): $(LOCALBIN)
	test -s $(LOCALBIN)/controller-gen || GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION)

.PHONY: envtest
envtest: $(ENVTEST) ## Download envtest-setup locally if necessary.
$(ENVTEST): $(LOCALBIN)
	test -s $(LOCALBIN)/setup-envtest || GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest