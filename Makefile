.PHONY: build test test-short test-e2e test-all test-coverage vet lint check clean

BINARY := yatz
BUILD_DIR := ./cmd/yatz

build:
	go build -o $(BINARY) $(BUILD_DIR)

test:
	go test ./...

test-short:
	go test -short ./... -count=1

test-e2e:
	go test ./... -count=1 -run TestE2E -timeout 120s

test-all:
	go test ./... -v -count=1 -timeout 120s

test-coverage:
	go test -short ./... -coverprofile=coverage.out
	go tool cover -func=coverage.out

vet:
	go vet ./...

lint: vet

check: vet test-short build

clean:
	rm -f $(BINARY) coverage.out
