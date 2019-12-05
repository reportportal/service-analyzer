.DEFAULT_GOAL := build

COMMIT_HASH = `git rev-parse --short HEAD 2>/dev/null`
BUILD_DATE = `date +%FT%T%z`

GO = go
BINARY_DIR=bin
RELEASE_DIR=release

BUILD_DEPS:= github.com/avarabyeu/releaser
GODIRS_NOVENDOR = $(shell go list ./... | grep -v /vendor/)
GOFILES_NOVENDOR = $(shell find . -type f -name '*.go' -not -path "./vendor/*")
PACKAGE_COMMONS=github.com/reportportal/commons-go
REPO_NAME=reportportal/service-analyzer

BUILD_INFO_LDFLAGS=-ldflags "-extldflags '"-static"' -X ${PACKAGE_COMMONS}/commons.repo=${REPO_NAME} -X ${PACKAGE_COMMONS}/commons.branch=${COMMIT_HASH} -X ${PACKAGE_COMMONS}/commons.buildDate=${BUILD_DATE} -X ${PACKAGE_COMMONS}/commons.version=${v}"
IMAGE_NAME=reportportal/service-analyzer$(IMAGE_POSTFIX)
IMAGE_NAME_DEV=reportportal-dev-5/service-analyzer

.PHONY: vendor test build

help:
	@echo "build      - go build"
	@echo "test       - go test"
	@echo "checkstyle - gofmt+golint+misspell"

get-build-deps:
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.21.0
	$(GO) get $(BUILD_DEPS)

test:
	$(GO) test ${GODIRS_NOVENDOR}

checkstyle:
	golangci-lint run --deadline 10m

fmt:
	gofmt -l -w -s ${GOFILES_NOVENDOR}


# Builds server
build: checkstyle test
	CGO_ENABLED=0 GOOS=linux $(GO) build ${BUILD_INFO_LDFLAGS} -o ${BINARY_DIR}/service-analyzer ./

# Builds the container
build-image-dev:
	docker build -t "$(IMAGE_NAME_DEV)" --build-arg version=${v} -f DockerfileDev .

# Builds the container
build-image:
	docker build -t "$(IMAGE_NAME)" --build-arg version=${v} -f Dockerfile .


# Builds the container and pushes to private registry
pushDev:
	echo "Registry is not provided"
	if [ -d ${REGISTRY} ] ; then echo "Provide registry"; exit 1 ; fi
	docker tag "$(IMAGE_NAME)" "$(REGISTRY)/$(IMAGE_NAME):latest"
	docker push "$(REGISTRY)/$(IMAGE_NAME):latest"

clean:
	if [ -d ${BINARY_DIR} ] ; then rm -r ${BINARY_DIR} ; fi
	if [ -d 'build' ] ; then rm -r 'build' ; fi

build-release: vendor get-build-deps fmt checkstyle test
	$(eval v := $(or $(v),$(shell releaser bump)))
	# make sure latest version is bumped to file
	releaser bump --version ${v}

	CGO_ENABLED=0 GOARCH=amd64 GOOS=linux $(GO) build ${BUILD_INFO_LDFLAGS} -o ${RELEASE_DIR}/service-analyzer_linux_amd64 ./
	CGO_ENABLED=0 GOARCH=amd64 GOOS=windows $(GO) build ${BUILD_INFO_LDFLAGS} -o ${RELEASE_DIR}/service-analyzer_win_amd64.exe ./

release: build-release
	releaser release --bintray.token ${BINTRAY_TOKEN}
