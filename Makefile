VERSION := $(shell git describe --always)

export PATH := $(shell pwd)/Godeps/_workspace/bin:$(PATH)
export GOPATH := $(shell pwd)/Godeps/_workspace:$(GOPATH)

build:
	go build -ldflags "-X main.version $(VERSION)"

test:
	go build -ldflags "-X main.version testversion" -o rotate-test
	go test -v ./...; rm rotate-test
