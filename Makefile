SERVICE = log-processor
ARCH ?= $(shell uname -m)
VERSION_SCRIPT = ./assets/scripts/get-version.sh
VERSION ?= $(shell git rev-parse --short HEAD)
SHORT_SHA ?= $(shell git rev-parse --short HEAD)
GIT_TAG ?= $(shell git describe --tags --abbrev=0)

GO = CGO_ENABLED=$(CGO_ENABLED) GOFLAGS=-mod=vendor go
CGO_ENABLED ?= 0
GO_BUILD_FLAGS = -ldflags "-X main.version=${GIT_TAG}-${VERSION}"

# Utility functions
check_defined = \
	$(strip $(foreach 1,$1, \
		$(call __check_defined,$1,$(strip $(value 2)))))
__check_defined = $(if $(value $1),, \
	$(error undefined '$1' variable: $2))

# Pattern #1 example: "example : description = Description for example target"
# Pattern #2 example: "### Example separator text
help: HELP_SCRIPT = \
	if (/^([a-zA-Z0-9-\.\/]+).*?: description\s*=\s*(.+)/) { \
		printf "\033[34m%-40s\033[0m %s\n", $$1, $$2 \
	} elsif(/^\#\#\#\s*(.+)/) { \
		printf "\033[33m>> %s\033[0m\n", $$1 \
	}

.PHONY: help
help:
	@perl -ne '$(HELP_SCRIPT)' $(MAKEFILE_LIST)

### Test

.PHONY: test
test: description = Run Go tests (make sure to bring up deps first; tests are ran non-parallel)
test: GOFLAGS=
test:
	TEST=true $(GO) test ./apis/grpcapi/... -v -count=1

### Docker

.PHONY: docker/build/local
docker/build/local: description = Build docker image locally (needed for M1+)
docker/build/local:
	docker build --load --build-arg TARGETOS=linux --build-arg TARGETARCH=arm64 \
	-t streamdal/$(SERVICE):$(VERSION) \
	-t streamdal/$(SERVICE):latest \
	-f ./Dockerfile .

.PHONY: docker/build
docker/build: description = Build docker image
docker/build:
	docker buildx build --push --platform=linux/amd64,linux/arm64 \
	-t streamdal/$(SERVICE):$(VERSION) \
	-t streamdal/$(SERVICE):$(SHORT_SHA) \
	-t streamdal/$(SERVICE):latest \
	-f ./Dockerfile .

.PHONY: docker/push
docker/push: description = Push local docker image
docker/push:
	docker push streamdal/$(SERVICE):$(VERSION) && \
	docker push streamdal/$(SERVICE):latest