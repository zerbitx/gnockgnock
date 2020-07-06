GOOS?=linux
GOARCH?=amd64
CGO_ENABLED?=0

REPO_BASE?=zerbitx
REPO_NAME?=gnockgnock
BUILD_TAG?=latest

all: test linux

linux: CGO_ENABLED=0
linux: test build

mac: GOOS = darwin
mac: test build
mac_testless: GOOS = darwin
mac_testless: build

docker: all
	docker build -t $(REPO_BASE)/$(REPO_NAME):$(BUILD_TAG) .

docker_publish: docker
	docker push $(REPO_BASE)/$(REPO_NAME):$(BUILD_TAG)

test:
	go test -v ./...

clean:
	rm ./bin/gnockgnock

build:
	GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=$(CGO_ENABLED) go build -o bin/gnockgnock
