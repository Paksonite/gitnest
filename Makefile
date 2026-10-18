VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
PREFIX  ?= $(HOME)/.local
LDFLAGS := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)

.PHONY: build install dev test vet tidy snapshot clean

build:
	CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o bin/gitnest ./cmd/gitnest

install: build
	mkdir -p $(PREFIX)/bin
	install -m 755 bin/gitnest $(PREFIX)/bin/gitnest
	@echo "installed $(PREFIX)/bin/gitnest — run: gitnest init"

# A preview build of the current checkout as ~/.local/bin/gitnest-dev, next
# to the released gitnest; gitnest-dev update fetches the latest dev build.
dev:
	CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=dev-$(COMMIT) -X main.commit=$(COMMIT) -X main.date=$(DATE)" -o bin/gitnest-dev ./cmd/gitnest
	mkdir -p $(PREFIX)/bin
	install -m 755 bin/gitnest-dev $(PREFIX)/bin/gitnest-dev
	@echo "installed $(PREFIX)/bin/gitnest-dev"

test:
	go test ./...

vet:
	go vet ./...

tidy:
	go mod tidy

# Build release artifacts locally without publishing anything.
snapshot:
	goreleaser release --snapshot --clean

clean:
	rm -rf bin dist
