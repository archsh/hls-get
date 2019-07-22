###### Makefile for hls-get

export CGO_ENABLED=0
GOPATHSRC := $(GOPATH)/src
GOHOSTARCH:=$(shell go env GOHOSTARCH)
GOHOSTOS:=$(shell go env GOHOSTOS)
GOARCH ?= $(GOHOSTARCH)
GOOS ?= $(GOHOSTOS)
GO111MODULE ?= on
ifneq ($(GOOS)_$(GOARCH),$(GOHOSTOS)_$(GOHOSTARCH))
  GOPATHBIN := $(GOPATH)/bin/$(GOOS)_$(GOARCH)
else
  GOPATHBIN := $(GOPATH)/bin
endif
GOSRCS := $(shell find . -name "*.go")
.PHONY=version.go

VERSION=$(subst heads/,,$(shell git describe --all))
TAG=$(shell git rev-parse --short HEAD)
DATE=`date +'%F %T'`
MACHINE=$(shell uname -a)

all: $(GOPATHBIN)/hls-get

version.go: go.mod $(GOSRCS) Makefile
	@echo "Generating Version $@ ..."
	@echo "/* AUTO GENERATED */" > $@
	@echo "package main" >> $@
	@echo "const (" >> $@
	@echo "    VERSION=\"$(VERSION)\"" >> $@
	@echo "    TAG=\"$(TAG)\"" >> $@
	@echo "    BUILD_TIME=\"$(DATE)\"" >> $@
	@echo ")" >> $@

htmldocs/assets_vfsdata.go:
	@make -C htmldocs/

$(GOPATHBIN)/hls-get: version.go go.mod $(GOSRCS) htmldocs/assets_vfsdata.go
	@echo "Building $@ ..."
	@go install -tags release hls-get

run: $(GOPATHBIN)/hls-get
	@echo "Starting test ..."
	@$(GOPATHBIN)/hls-get --config example.yaml serve --debug --combine --remove --format=mp4

clean:
	@rm -f $(GOPATHBIN)/hls-get
	@make -C htmldocs/ clean