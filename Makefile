BINARY := tustat
PKG := ./cmd/tustat

# CGO_ENABLED=0 produces a fully static binary that runs on any Linux host
# regardless of its C library — no interpreter or linker at runtime.
.PHONY: build test vet fmt tidy clean

build:
	CGO_ENABLED=0 go build -o $(BINARY) $(PKG)

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -w cmd internal

tidy:
	go mod tidy

clean:
	rm -f $(BINARY)
