.PHONY: linux macos docker run test clean

NAME ?= elb-inject
LDFLAGS ?= -X=main.version=$(VERSION) -w -s
VERSION ?= 1.19
BUILD_FLAGS ?= -v
CGO_ENABLED ?= 0


macos:
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=${CGO_ENABLED} go build -o build/macos/${NAME} ${BUILD_FLAGS} -ldflags "$(LDFLAGS)" $^

linux:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=${CGO_ENABLED} go build -o build/linux/${NAME} ${BUILD_FLAGS} -ldflags "$(LDFLAGS)" $^

docker: linux
	docker build --no-cache --squash --rm -t ${NAME}:${VERSION} .
	docker tag ${NAME}:${VERSION} duym/${NAME}:${VERSION}
	docker push duym/${NAME}:${VERSION}

run:
	./build/macos/${NAME} -kubeconfig=/Users/dmai/.kube/staging -aws.creds=/Users/dmai/.aws/credentials

test:
	go test -v -race $(shell go list ./... )

clean:
	- rm -fr ./build/*
	- docker rmi `docker images -f "dangling=true" -q --no-trunc`
