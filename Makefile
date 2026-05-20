build:
	go build ./...

test:
	go test ./...

vet:
	go vet ./...

tidy:
	go work sync
	go mod tidy
	cd test && go mod tidy

cli:
	go build -o bin/goten ./cmd/goten

run-example:
	cd examples/basic && docker-compose up -d && go run .

.PHONY: build test vet tidy cli run-example
