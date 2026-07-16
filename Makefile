.PHONY: all build build-web test clean run lint lintmax gosec govulncheck goreleaser proto test-proto test-web

BINARY := almanach-render-service
GORELEASER_ARGS ?= --skip=sign --snapshot --clean
GORELEASER_TARGET ?= --single-target

all: build

build-web:
	GOWORK=off go run ./cmd/build-web

test:
	GOWORK=off go test ./...

# Regenerate the Layout DSL v2 code (Go -> gen/, TypeScript -> web/src/pb/) from
# proto/almanach/layout/v1/layout.proto. Local plugins: protoc-gen-go on PATH,
# protoc-gen-es from web/node_modules (run `cd web && pnpm install` first).
proto:
	buf generate

# Round-trip decode tests locking the layout wire contract on both sides.
test-proto:
	GOWORK=off go test ./internal/layoutpb/...
	node web/test/layout.roundtrip.test.mjs

# Runner-free web unit tests (proto round-trip + block registry).
test-web:
	node web/test/layout.roundtrip.test.mjs
	node web/src/blocks/registry.test.mjs

build: build-web
	GOWORK=off go build -tags embed -o ./dist/$(BINARY) ./cmd/almanach-render-service

run:
	GOWORK=off go run ./cmd/almanach-render-service serve

clean:
	rm -rf dist web/dist

docker-lint:
	docker run --rm -v $(shell pwd):/app -w /app golangci/golangci-lint:latest golangci-lint run -v

lint:
	GOWORK=off golangci-lint run -v

lintmax:
	GOWORK=off golangci-lint run -v --max-same-issues=100

gosec:
	GOWORK=off go install github.com/securego/gosec/v2/cmd/gosec@latest
	gosec -exclude-generated -exclude=G101,G304,G301,G306,G204,G404,G703,G122 -exclude-dir=.history ./...

govulncheck:
	GOWORK=off go install golang.org/x/vuln/cmd/govulncheck@latest
	govulncheck ./...

goreleaser:
	GOWORK=off goreleaser release $(GORELEASER_ARGS) $(GORELEASER_TARGET)

tag-major:
	git tag $(shell svu major)

tag-minor:
	git tag $(shell svu minor)

tag-patch:
	git tag $(shell svu patch)

release:
	git push origin --tags
	GOWORK=off GOPROXY=proxy.golang.org go list -m github.com/go-go-golems/almanach@$(shell svu current)

bump-glazed:
	GOWORK=off go get github.com/go-go-golems/glazed@latest
	GOWORK=off go mod tidy

.PHONY: logcopter-generate
logcopter-generate:
	GOWORK=off go tool logcopter-gen -include-main -var zlog -area-prefix go-go-golems.almanach -strip-prefix github.com/go-go-golems/almanach ./cmd/... ./pkg/...

.PHONY: logcopter-check
logcopter-check:
	GOWORK=off go tool logcopter-gen -include-main -var zlog -area-prefix go-go-golems.almanach -strip-prefix github.com/go-go-golems/almanach -check ./cmd/... ./pkg/...

GLAZED_LINT_BIN ?= /tmp/glazed-lint
GLAZED_LINT_PKG ?= github.com/go-go-golems/glazed/cmd/tools/glazed-lint
GLAZED_VERSION ?= v1.3.6

.PHONY: glazed-lint-build glazed-lint

glazed-lint-build:
	@echo "Building glazed-lint from Glazed module..."
	@if [ -n "$(GLAZED_VERSION)" ]; then \
		echo "Installing $(GLAZED_LINT_PKG)@$(GLAZED_VERSION)"; \
		GOBIN=$(dir $(GLAZED_LINT_BIN)) GOWORK=off go install $(GLAZED_LINT_PKG)@$(GLAZED_VERSION); \
	else \
		echo "Installing $(GLAZED_LINT_PKG) from workspace/module"; \
		GOBIN=$(dir $(GLAZED_LINT_BIN)) go install $(GLAZED_LINT_PKG); \
	fi

glazed-lint: glazed-lint-build
	GOWORK=off go vet -vettool=$(GLAZED_LINT_BIN) ./cmd/... ./pkg/...
