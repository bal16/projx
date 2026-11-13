BINARY := pjx
PKG    := github.com/bal16/projx/cmd/pjx

.PHONY: build test vet fmt clean

build:
	go build -o bin/$(BINARY) ./cmd/pjx

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -w .

clean:
	rm -rf bin/
