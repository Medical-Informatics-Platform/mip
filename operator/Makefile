LOCALBIN ?= $(shell pwd)/bin
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
ENVTEST ?= $(LOCALBIN)/setup-envtest
CONTROLLER_TOOLS_VERSION ?= v0.23.0
ENVTEST_VERSION ?= $(shell go list -m -f "{{ .Version }}" sigs.k8s.io/controller-runtime | awk -F'[v.]' '{printf "release-%d.%d", $$2, $$3}')
ENVTEST_K8S_VERSION ?= $(shell go list -m -f "{{ .Version }}" k8s.io/api | awk -F'[v.]' '{printf "1.%d", $$3}')

.PHONY: manifests
manifests: $(CONTROLLER_GEN) ## CRDs (config/crd/bases) and RBAC (config/rbac/role.yaml).
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd paths="./..." output:crd:artifacts:config=config/crd/bases

.PHONY: generate
generate: $(CONTROLLER_GEN) ## DeepCopy methods.
	$(CONTROLLER_GEN) object paths="./..."

.PHONY: test
test: manifests generate $(ENVTEST) ## envtest controller tests.
	go fmt ./...
	go vet ./...
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" go test ./... -coverprofile cover.out

$(LOCALBIN):
	mkdir -p $(LOCALBIN)

$(CONTROLLER_GEN): | $(LOCALBIN)
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION)

$(ENVTEST): | $(LOCALBIN)
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@$(ENVTEST_VERSION)
