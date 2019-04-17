.PHONY: linux macos

BINARY ?= elb-inject
LDFLAGS ?= -X=main.version=$(VERSION) -w -s
#VERSION ?= $(shell git describe --tags --always --dirty)
VERSION ?= '0.0.1'
BUILD_FLAGS ?= -v
CGO_ENABLED ?= 0


macos:
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=${CGO_ENABLED} go build -o build/macos/${BINARY} ${BUILD_FLAGS} -ldflags "$(LDFLAGS)" $^

linux:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=${CGO_ENABLED} go build -o build/linux/${BINARY} ${BUILD_FLAGS} -ldflags "$(LDFLAGS)" $^

run:
	./build/macos/${BINARY} -kubeconfig=./minikube.config
