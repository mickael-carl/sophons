BINS = \
	bin/executer-darwin-arm64 \
	bin/executer-darwin-x86_64 \
	bin/executer-linux-arm64 \
	bin/executer-linux-x86_64 \
	bin/dialer \
	bin/tardiff \
	bin/docgen

SRCS := $(shell find cmd pkg -name '*.go') go.mod go.sum

.PHONY: clean docker-testing docs fmt lint unit-tests mocks

all: $(BINS)

bin/executer-darwin-arm64: $(SRCS)
	GOOS=darwin GOARCH=arm64 go build -o $@ ./cmd/executer

bin/executer-darwin-x86_64: $(SRCS)
	GOOS=darwin GOARCH=amd64 go build -o $@ ./cmd/executer

bin/executer-linux-arm64: $(SRCS)
	GOOS=linux GOARCH=arm64 go build -o $@ ./cmd/executer

bin/executer-linux-x86_64: $(SRCS)
	GOOS=linux GOARCH=amd64 go build -o $@ ./cmd/executer

bin/dialer: $(SRCS)
	go build -o $@ ./cmd/dialer

bin/tardiff: $(SRCS)
	go build -o $@ ./cmd/tardiff

bin/docgen: $(SRCS)
	go build -o $@ ./cmd/docgen

clean:
	-rm -f $(BINS)

docker-testing: Dockerfile.testing
	docker build . -f Dockerfile.testing -t sophons-testing:latest

docs: $(SRCS) bin/docgen
	./bin/docgen pkg/exec docs/builtins

fmt:
	golangci-lint fmt

lint:
	golangci-lint run

unit-tests:
	go test ./...

mocks:
	go generate ./...
