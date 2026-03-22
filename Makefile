.PHONY: build test vet clean

BINARY := yatz
BUILD_DIR := ./cmd/yatz

build:
	go build -o $(BINARY) $(BUILD_DIR)

test:
	go test ./...

vet:
	go vet ./...

clean:
	rm -f $(BINARY)
