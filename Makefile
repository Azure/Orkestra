
# Image URL to use all building/pushing image targets
IMG ?= azureorkestra/orkestra:latest
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true"
DEBUG_LEVEL ?= 1
CI_VALUES ?= "chart/orkestra/values-ci.yaml"

# Directories
ROOT_DIR := $(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))
BIN_DIR := $(abspath $(ROOT_DIR)/bin)

reg_name='kind-registry'

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

all: manager

## Create kind cluster with local registry and prepopulate the local registry.
setup: kind-create prepopulate-kind-registry

## Deploy the Orkestra helm chart (values-ci.yaml) with Orkestra controller disabled.
dev:
	helm upgrade --install orkestra chart/orkestra --wait --atomic -n orkestra --create-namespace --values ${CI_VALUES}

## Deploy the Orkestra helm chart (values-ci.yaml) with Orkestra controller enabled (in debug mode).
debug: dev
	go run main.go --debug --log-level ${DEBUG_LEVEL}

## Cleanup any Orkestra installation and the kind cluster with registry.
clean: kind-delete
	@rm -rf $(BIN_DIR)

## Cleanup any previous Orkestra helm chart installation.
clean-chart:
	helm delete orkestra -n orkestra 2>&1 || true
	@rm -rf $(BIN_DIR)

## Run tests.
test: install
	go test -v ./... -coverprofile coverage.txt -timeout 35m

## Run tests using Ginkgo.
ginkgo-test: install
	go get github.com/onsi/ginkgo/ginkgo
	ginkgo ./... -cover -coverprofile coverage.txt

## Prepare code for PR.
prepare-for-pr: vet fmt api-docs
	@echo "\n> ❗️ Remember to run the tests"

## Build manager binary.
manager: generate fmt vet
	go build -o bin/manager main.go

## Run against the configured Kubernetes cluster in ~/.kube/config.
run: generate fmt vet manifests
	go run ./main.go

## Install CRDs into a cluster.
install: manifests
	kustomize build config/crd | kubectl apply -f -

## Uninstall CRDs from a cluster.
uninstall: manifests
	kustomize build config/crd | kubectl delete -f -

# #Deploy controller in the configured Kubernetes cluster in ~/.kube/config.
deploy: manifests
	cd config/manager && kustomize edit set image controller=${IMG}
	kustomize build config/default | kubectl apply -f -

## Generate code.
generate: controller-gen
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

## Generate manifests e.g. CRD, RBAC etc.
manifests: generate
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=chart/orkestra/crds

## Generate API reference documentation.
api-docs: gen-crd-api-reference-docs
	$(API_REF_GEN) -api-dir=./api/v1alpha1 -config=./hack/api-docs/config.json -template-dir=./hack/api-docs/template -out-file=./docs/api.md

## Create kind cluster with the local registry enabled in containerd.
kind-create:
	@echo "> 🔨 Creating kind cluster with local registry...\n"
	bash ./hack/create-kind-with-registry.sh
	@echo "> 👍 Done\n"

## Delete kind cluster with the local registry enabled in containerd.
kind-delete: 
	@echo "> 🔨 Deleting kind cluster with local registry...\n"
	bash ./hack/teardown-kind-with-registry.sh
	@echo "> 👍 Done\n"

## Prepopulate kind registry with required docker images.
prepopulate-kind-registry:
	@echo "> 🔨 Prepopulating kind registry...\n"
	bash ./hack/prepopulate-kind-registry.sh
	@echo "> 👍 Done\n"

## Run go fmt against code.
fmt:
	go fmt ./...

## Run go vet against code.
vet:
	go vet ./...

## Build the docker image.
docker-build: test
	docker build . -t ${IMG}

## Push the docker image.
docker-push:
	docker push ${IMG}

## setup kubebuilder.
setup-kubebuilder:
	bash hack/setup-envtest.sh;
	bash hack/setup-kubebuilder.sh

## Find or download controller-gen.
controller-gen:
ifeq (, $(shell which controller-gen))
	@{ \
	set -e ;\
	CONTROLLER_GEN_TMP_DIR=$$(mktemp -d) ;\
	cd $$CONTROLLER_GEN_TMP_DIR ;\
	go mod init tmp ;\
	go get sigs.k8s.io/controller-tools/cmd/controller-gen@v0.5.0 ;\
	rm -rf $$CONTROLLER_GEN_TMP_DIR ;\
	}
CONTROLLER_GEN=$(GOBIN)/controller-gen
else
CONTROLLER_GEN=$(shell which controller-gen)
endif

## Find or download gen-crd-api-reference-docs.
gen-crd-api-reference-docs:
ifeq (, $(shell which gen-crd-api-reference-docs))
	@{ \
	set -e ;\
	API_REF_GEN_TMP_DIR=$$(mktemp -d) ;\
	cd $$API_REF_GEN_TMP_DIR ;\
	go mod init tmp ;\
	go get github.com/ahmetb/gen-crd-api-reference-docs@v0.2.0 ;\
	rm -rf $$API_REF_GEN_TMP_DIR ;\
	}
API_REF_GEN=$(GOBIN)/gen-crd-api-reference-docs
else
API_REF_GEN=$(shell which gen-crd-api-reference-docs)
endif
