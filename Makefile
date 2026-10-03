RELEASE_VERSION ?= 2.0.0-dev
RELEASE_ROOT ?= dist/release
RELEASE_DIR := $(RELEASE_ROOT)/$(RELEASE_VERSION)
RELEASE_BINARIES := aidi-control aidi-node-agent aidi-installer aidi-admin
GO_RELEASE_BUILD_FLAGS := -trimpath -buildvcs=false -ldflags=-buildid=

.PHONY: check test build run release verify-release-reproducible

check:
	@test -z "$$(gofmt -l .)"
	go vet ./...
	go test -race ./...
	cd web && npm run build
	PYTHONPATH=workers/src python -c "from aidi_worker import health; assert health()['status'] == 'ok'"
	PYTHONPATH=workers/src python -m unittest discover -s workers/tests -v

test:
	go test -race ./...
	PYTHONPATH=workers/src python -m unittest discover -s workers/tests -v

build:
	mkdir -p bin
	go build -trimpath -o bin/aidi ./cmd/aidi
	cd web && npm run build
	python -m compileall -q workers/src

release:
	@case "$(RELEASE_VERSION)" in \
		""|*[!A-Za-z0-9._+-]*) echo "RELEASE_VERSION must use only A-Z, a-z, 0-9, '.', '_', '+', or '-'" >&2; exit 2 ;; \
	esac
	@case "$(RELEASE_ROOT)" in \
		""|"/"|".") echo "RELEASE_ROOT must name a dedicated output directory" >&2; exit 2 ;; \
	esac
	rm -rf -- "$(RELEASE_DIR)"
	mkdir -p "$(RELEASE_DIR)"
	@set -eu; \
	for name in $(RELEASE_BINARIES); do \
		CGO_ENABLED=0 go build $(GO_RELEASE_BUILD_FLAGS) -o "$(RELEASE_DIR)/$$name" "./cmd/$$name"; \
	done
	@cd "$(RELEASE_DIR)" && sha256sum $(RELEASE_BINARIES) > SHA256SUMS

verify-release-reproducible:
	@set -eu; \
	tmp="$$(mktemp -d)"; \
	trap 'rm -rf "$$tmp"' 0 1 2 15; \
	$(MAKE) --no-print-directory release RELEASE_VERSION="$(RELEASE_VERSION)" RELEASE_ROOT="$$tmp/first"; \
	$(MAKE) --no-print-directory release RELEASE_VERSION="$(RELEASE_VERSION)" RELEASE_ROOT="$$tmp/second"; \
	for name in $(RELEASE_BINARIES); do \
		cmp "$$tmp/first/$(RELEASE_VERSION)/$$name" "$$tmp/second/$(RELEASE_VERSION)/$$name"; \
	done; \
	cmp "$$tmp/first/$(RELEASE_VERSION)/SHA256SUMS" "$$tmp/second/$(RELEASE_VERSION)/SHA256SUMS"

run:
	go run ./cmd/aidi
