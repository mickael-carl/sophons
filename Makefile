BINS = \
	bin/executer-darwin-arm64 \
	bin/executer-darwin-x86_64 \
	bin/executer-linux-arm64 \
	bin/executer-linux-x86_64 \
	bin/dialer \
	bin/tardiff

SRCS := $(shell find cmd pkg -name '*.go')

.PHONY: clean docker-testing

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

clean:
	-rm -f $(BINS)

docker-testing: Dockerfile.testing
	docker build . -f Dockerfile.testing -t sophons-testing:latest

