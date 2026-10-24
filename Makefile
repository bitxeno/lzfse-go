.PHONY: build test vet build-libs clean

# Build the native shared library for this platform into libs/ and embed it.
build-libs:
	go run ./internal/libbuild -host

# Cross-compile the native library for every supported target (needs zig).
build-libs-all:
	go run ./internal/libbuild -all

build:
	go build ./...

test:
	CGO_ENABLED=0 go test -count=1 ./...

vet:
	CGO_ENABLED=0 go vet ./...

clean:
	rm -rf libs/*/ build/
