# Oliver - run Windows programs with the best runtime installed.
# Zip Transform - turn an MCreator project ZIP into a mod jar.

BIN     := oliver
PKG     := ./cmd/oliver
DIST    := dist
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)

# CGO is disabled everywhere: a static binary is what lets one AppImage run on
# any distribution regardless of its glibc version.
GOBUILD := CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)"

.PHONY: all
all: build

.PHONY: build
build: ## Build the binary for this machine
	$(GOBUILD) -o $(DIST)/$(BIN) $(PKG)
	@echo "built $(DIST)/$(BIN) ($(VERSION))"

.PHONY: ziptransform
ziptransform: ## Build Zip Transform for this machine
	$(GOBUILD) -o $(DIST)/ziptransform ./cmd/ziptransform
	@echo "built $(DIST)/ziptransform ($(VERSION))"

.PHONY: install
install: ## Install into GOPATH/bin
	CGO_ENABLED=0 go install -trimpath -ldflags "$(LDFLAGS)" $(PKG)

.PHONY: test
test: ## Run the test suite
	go test ./...

.PHONY: test-race
test-race: ## Run the test suite under the race detector
	go test -race ./...

.PHONY: cover
cover: ## Report test coverage per package
	go test -coverprofile=$(DIST)/coverage.out ./...
	go tool cover -func=$(DIST)/coverage.out | tail -1

.PHONY: check
check: ## Format check, vet and test
	@test -z "$$(gofmt -l . | tee /dev/stderr)" || (echo "run: gofmt -w ." && false)
	go vet ./...
	go test ./...

.PHONY: appimage
appimage: ## Build the AppImage
	./packaging/build-appimage.sh

.PHONY: ziptransform-appimage
ziptransform-appimage: ## Build the Zip Transform AppImage
	APP=ziptransform ./packaging/build-appimage.sh

.PHONY: appdir
appdir: ## Build the AppDir only, without packaging it
	APPDIR_ONLY=1 ./packaging/build-appimage.sh

.PHONY: release
release: ## Build AppImages for every supported architecture
	@for app in oliver ziptransform; do \
		for arch in x86_64 aarch64; do \
			echo "--- $$app $$arch ---"; \
			APP=$$app ARCH=$$arch ./packaging/build-appimage.sh || exit 1; \
		done; \
	done

.PHONY: clean
clean: ## Remove build output
	rm -rf $(DIST)

.PHONY: help
help: ## List the available targets
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'
