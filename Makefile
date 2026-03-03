BINARY := colosseum
PKG := github.com/ramtinj/colosseum
CMD := ./cmd/colosseum

.PHONY: build test lint vet clean install

build:
	go build -o $(BINARY) $(CMD)

test:
	go test ./... -count=1

lint: vet
	@command -v golangci-lint >/dev/null 2>&1 && golangci-lint run || echo "golangci-lint not installed, skipping"

vet:
	go vet ./...

clean:
	rm -f $(BINARY)

install:
	go install $(CMD)
