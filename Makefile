.PHONY: help build test test-integration tidy lint fmt cli example pg-up pg-down generate clean release-check \
	release tag-all tag-core tag-adapter-gorm tag-plugin-username tag-plugin-oauth tag-plugin-admin tag-cmd

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
	@echo "  example            Run example app (requires postgres up)"
	@echo "  pg-up              Start Postgres via docker-compose"
	@echo "  pg-down            Stop Postgres"
	@echo "  generate           Generate ORM models for the bundled example"
	@echo "  clean              Remove bin/ and Go test cache"
	@echo "  release-check         Full check before tagging (build + test + lint)"
	@echo ""
	@echo "  release               Lockstep release: check + tag ALL modules (VERSION=v0.x.x)"
	@echo "  tag-all               Tag + push every module at one version  (VERSION=v0.x.x)"
	@echo "  tag-core              Tag core module          (VERSION=v0.x.x)"
	@echo "  tag-adapter-gorm      Tag adapters/gorm module (VERSION=v0.x.x)"
	@echo "  tag-plugin-username   Tag plugins/username     (VERSION=v0.x.x)"
	@echo "  tag-plugin-oauth      Tag plugins/oauth        (VERSION=v0.x.x)"
	@echo "  tag-plugin-admin      Tag plugins/admin        (VERSION=v0.x.x)"
	@echo "  tag-cmd               Tag cmd/goten module     (VERSION=v0.x.x)"

build:
	go build ./...
	cd adapters/gorm && go build ./...
	cd plugins/username && go build ./...
	cd plugins/oauth && go build ./...
	cd plugins/admin && go build ./...
	cd cmd/goten && go build ./...
	cd test && go build ./...
	cd examples/basic && go build ./...
	cd examples/layered-gin && go build ./...
	cd examples/oauth-google && go build ./...

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
	cd plugins/oauth && go mod tidy
	cd plugins/admin && go mod tidy
	cd examples/basic && go mod tidy
	cd examples/layered-gin && go mod tidy
	cd examples/oauth-google && go mod tidy

lint:
	@command -v golangci-lint >/dev/null 2>&1 || { echo "Install golangci-lint: https://golangci-lint.run/usage/install/"; exit 1; }
	golangci-lint run ./...
	cd adapters/gorm && golangci-lint run ./...
	cd plugins/username && golangci-lint run ./...
	cd plugins/oauth && golangci-lint run ./...
	cd plugins/admin && golangci-lint run ./...
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

generate: cli
	cd examples/basic && ../../bin/goten generate -y

example: cli pg-up generate
	cd examples/basic && go run .

clean:
	rm -rf bin/
	go clean -testcache

release-check: build test lint
	@echo "✓ Ready to tag release"

# ── Lockstep release (better-auth style: every module shares one version) ──────
# Usage: make release VERSION=v0.4.0
#
# Tags ALL modules at the same VERSION so "goten vX.Y.Z" means every module.
# Run this from a clean `main` that is already pushed. tag-all pushes every tag
# in one shot, so the inter-module requires resolve (the tags co-exist at HEAD).
#
# When a module starts depending on a NEWER sibling, first bump its require —
#   cd <module> && go mod edit -require=github.com/dnahilman/goten@$(VERSION)
# then `go work sync` + commit before tagging. Otherwise requires may stay at an
# older compatible version; that is fine for Go.
release: release-check tag-all
	@echo "✓ Released $(VERSION) (all modules)"

tag-all:
	@test -n "$(VERSION)" || (echo "Usage: make tag-all VERSION=v0.4.0"; exit 1)
	git tag $(VERSION)
	git tag adapters/gorm/$(VERSION)
	git tag plugins/username/$(VERSION)
	git tag plugins/oauth/$(VERSION)
	git tag plugins/admin/$(VERSION)
	git tag cmd/goten/$(VERSION)
	git push origin $(VERSION) \
		adapters/gorm/$(VERSION) \
		plugins/username/$(VERSION) \
		plugins/oauth/$(VERSION) \
		plugins/admin/$(VERSION) \
		cmd/goten/$(VERSION)
	@echo "✓ Tagged all modules at $(VERSION)"

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

tag-plugin-oauth:
	@test -n "$(VERSION)" || (echo "Usage: make tag-plugin-oauth VERSION=v0.1.0"; exit 1)
	git tag plugins/oauth/$(VERSION)
	git push origin plugins/oauth/$(VERSION)
	@echo "✓ plugins/oauth tagged $(VERSION)"

tag-plugin-admin:
	@test -n "$(VERSION)" || (echo "Usage: make tag-plugin-admin VERSION=v0.1.0"; exit 1)
	git tag plugins/admin/$(VERSION)
	git push origin plugins/admin/$(VERSION)
	@echo "✓ plugins/admin tagged $(VERSION)"

tag-cmd:
	@test -n "$(VERSION)" || (echo "Usage: make tag-cmd VERSION=v0.1.0"; exit 1)
	git tag cmd/goten/$(VERSION)
	git push origin cmd/goten/$(VERSION)
	@echo "✓ cmd/goten tagged $(VERSION)"
