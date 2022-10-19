LASTCOMMIT := $(shell git rev-parse --short HEAD)
BUILDTIME := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
BUILDUSER := $(shell id -u -n)
GOLDFLAGS += -X main.version=$(WEBVERSION) -X main.commit=$(LASTCOMMIT)
GOLDFLAGS += -X main.buildTime=$(BUILDTIME) -X main.buildUser=$(BUILDUSER)
GOFLAGS = -ldflags "$(GOLDFLAGS)"

#export GO_RUN=env GO111MODULE=on go run $(GO_BUILD_ARGS)

help:
	@echo 'Targets:'
	@echo '  test         - run short unit tests'
	@echo '  tidy         - tidy go modules'
	@echo '  clear        - clear test cache'

clear:
	go clean -testcache

test:
	go test -short -race ./...

tidy:
	go mod verify
	go mod tidy
	@if ! git diff --quiet go.mod go.sum; then \
		echo "please run go mod tidy and check in changes, you might have to use the same version of Go as the CI"; \
		exit 1; \
	fi
	
.PHONY: help clear test tidy