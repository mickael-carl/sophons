BINS = \
	bin/executer-darwin-arm64 \
	bin/executer-darwin-x86_64 \
	bin/executer-linux-arm64 \
	bin/executer-linux-x86_64 \
	bin/dialer \
	bin/tardiff \
	bin/docgen \
	bin/conformance

SRCS := $(shell find cmd pkg -name '*.go') go.mod go.sum

.PHONY: clean docker-testing docs fmt lint unit-tests conformance-tests mocks proto install-protoc-plugins

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

bin/conformance: $(SRCS)
	go build -o $@ ./cmd/conformance

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

conformance-tests: bin/conformance
	./bin/conformance $(ARGS)

mocks:
	go generate ./...

PROTOC_GEN_GO = $(shell go env GOPATH)/bin/protoc-gen-go
PROTOC_GEN_GO_GRPC = $(shell go env GOPATH)/bin/protoc-gen-go-grpc
PROTOC_GO_INJECT_TAG = $(shell go env GOPATH)/bin/protoc-go-inject-tag
PROTO_SRCS = $(shell find proto/ -name '*.proto')

install-protoc-plugins:
	@echo "Installing protoc plugins..."
	@go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	@go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	@go install github.com/favadi/protoc-go-inject-tag@latest
	@go mod tidy # Ensure tools.go dependencies are in go.mod

proto: $(PROTO_SRCS)
	@echo "Generating protobuf Go code..."
	@protoc \
		--go_out=pkg/ --go_opt=paths=source_relative \
		--go-grpc_out=pkg/ --go-grpc_opt=paths=source_relative \
		$(PROTO_SRCS)
	@echo "Injecting YAML tags into generated Go code..."
	@$(PROTOC_GO_INJECT_TAG) -input="pkg/proto/*.pb.go"
