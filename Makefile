.PHONY: help build test test-integration tidy lint fmt cli example pg-up pg-down migrate clean release-check \
	tag-core tag-adapter-gorm tag-plugin-username tag-cmd

help:
	@echo "Goten Makefile"
	@echo ""
	@echo "  build              Build all modules"
	@echo "  test               Run unit tests (all modules)"
	@echo "  test-integration   Run integration tests (requires Postgres)"
	@echo "  lint               Run golangci-lint on all modules"
	@echo "  fmt                Format all Go files"
	@echo "  tidy               go mod tidy across all modules"
	@echo "  cli                Build CLI binary to bin/goten"
	@echo "  example            Run example app (requires postgres up + migrations applied)"
	@echo "  pg-up              Start Postgres via docker-compose"
	@echo "  pg-down            Stop Postgres"
	@echo "  migrate            Apply all pending migrations"
	@echo "  clean              Remove bin/ and Go test cache"
	@echo "  release-check         Full check before tagging (build + test + lint)"
	@echo ""
	@echo "  tag-core              Tag core module          (VERSION=v0.x.x)"
	@echo "  tag-adapter-gorm      Tag adapters/gorm module (VERSION=v0.x.x)"
	@echo "  tag-plugin-username   Tag plugins/username     (VERSION=v0.x.x)"
	@echo "  tag-cmd               Tag cmd/goten module     (VERSION=v0.x.x)"

build:
	go build ./...
	cd adapters/gorm && go build ./...
	cd plugins/username && go build ./...
	cd cmd/goten && go build ./...
	cd test && go build ./...
	cd examples/basic && go build ./...
	cd examples/layered-gin && go build ./...

test:
	go test ./internal/...
	cd test && go test ./...

test-integration:
	cd test && go test -tags integration ./...

tidy:
	go work sync
	go mod tidy
	cd adapters/gorm && go mod tidy
	cd plugins/username && go mod tidy
	cd cmd/goten && go mod tidy
	cd test && go mod tidy
	cd examples/basic && go mod tidy
	cd examples/layered-gin && go mod tidy

lint:
	@command -v golangci-lint >/dev/null 2>&1 || { echo "Install golangci-lint: https://golangci-lint.run/usage/install/"; exit 1; }
	golangci-lint run ./...
	cd adapters/gorm && golangci-lint run ./...
	cd plugins/username && golangci-lint run ./...
	cd cmd/goten && golangci-lint run ./...
	cd test && golangci-lint run ./...

fmt:
	gofmt -w .

cli:
	cd cmd/goten && go build -o ../../bin/goten .
	@echo "✓ bin/goten built"

pg-up:
	docker-compose -f docker-compose.dev.yml up -d

pg-down:
	docker-compose -f docker-compose.dev.yml down

migrate: cli
	cd examples/basic && ../../bin/goten init && ../../bin/goten migrate up

example: cli pg-up migrate
	cd examples/basic && go run .

clean:
	rm -rf bin/
	go clean -testcache

release-check: build test lint
	@echo "✓ Ready to tag release"

# ── Per-module tagging ────────────────────────────────────────────────────────
# Usage: make tag-core VERSION=v0.1.0
tag-core:
	@test -n "$(VERSION)" || (echo "Usage: make tag-core VERSION=v0.1.0"; exit 1)
	git tag $(VERSION)
	git push origin $(VERSION)
	@echo "✓ core tagged $(VERSION)"

tag-adapter-gorm:
	@test -n "$(VERSION)" || (echo "Usage: make tag-adapter-gorm VERSION=v0.1.0"; exit 1)
	git tag adapters/gorm/$(VERSION)
	git push origin adapters/gorm/$(VERSION)
	@echo "✓ adapters/gorm tagged $(VERSION)"

tag-plugin-username:
	@test -n "$(VERSION)" || (echo "Usage: make tag-plugin-username VERSION=v0.1.0"; exit 1)
	git tag plugins/username/$(VERSION)
	git push origin plugins/username/$(VERSION)
	@echo "✓ plugins/username tagged $(VERSION)"

tag-cmd:
	@test -n "$(VERSION)" || (echo "Usage: make tag-cmd VERSION=v0.1.0"; exit 1)
	git tag cmd/goten/$(VERSION)
	git push origin cmd/goten/$(VERSION)
	@echo "✓ cmd/goten tagged $(VERSION)"
