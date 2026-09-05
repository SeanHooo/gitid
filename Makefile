BINARY := bin/gitid
MODULE := github.com/seanhooo/gitid
VERSION ?= dev
COMMIT ?= unknown
DATE ?= unknown
LDFLAGS := -s -w \
	-X $(MODULE)/internal/version.Version=$(VERSION) \
	-X $(MODULE)/internal/version.Commit=$(COMMIT) \
	-X $(MODULE)/internal/version.Date=$(DATE)

.PHONY: fmt test vet build check install clean

fmt:
	gofmt -w .

test:
	go test ./...

vet:
	go vet ./...

build:
	mkdir -p bin
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) ./cmd/gitid

check: fmt test vet

install:
	go install -ldflags "$(LDFLAGS)" ./cmd/gitid

clean:
	rm -rf bin coverage.out coverage.html
