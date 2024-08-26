LDFLAGSSTRING +=-X main.GitCommit=$(GITCOMMIT)
LDFLAGSSTRING +=-X main.GitDate=$(GITDATE)
LDFLAGSSTRING +=-X main.GitVersion=$(GITVERSION)
LDFLAGS := -ldflags "$(LDFLAGSSTRING)"
IMAGE_TAG := $(shell git rev-parse --short HEAD)
REGISTRY := jellydones
IMAGE_NAME := op-host-manager

# generate help info from comments: thanks to https://marmelab.com/blog/2016/02/29/auto-documented-makefile.html
.PHONY: help
help: ## help information about make commands
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: op-host-manager
op-host-manager:
	go build -v $(LDFLAGS) -o ./bin/op-host-manager ./cmd/op-host-manager/


.PHONY: docker
docker: op-host-manager
	cd .. &&  docker buildx build -f ./op-host-manager/Dockerfile -t $(REGISTRY)/$(IMAGE_NAME):$(IMAGE_TAG) . --platform linux/amd64
	docker push $(REGISTRY)/$(IMAGE_NAME):$(IMAGE_TAG)

.PHONY: deploy-local
deploy-local: docker
	@TEMP_FILE=$$(mktemp); \
	sed "s|your-registry/node-labeler:v1|$(REGISTRY)/$(IMAGE_NAME):$(IMAGE_TAG)|g" ./k8s/k8.yaml > ./k8s/k8s.tmp; \
	kubectl apply -f ./k8s/k8s.tmp; \
	# rm $$TEMP_FILE
