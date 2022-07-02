
# Image URL to use all building/pushing image targets
IMG ?= mcr.microsoft.com/aks/canipull:v0.1.0

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Run tests
test: fmt vet
	go test ./... -coverprofile=cover.out -covermode=atomic

fmt:
	go fmt ./...

# Run go vet against code
vet:
	go vet ./...

# Build the docker image
docker-build: test
	docker build . -t ${IMG}

# Push the docker image
docker-push:
	docker push ${IMG}