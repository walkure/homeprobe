
# output target
BINDIR:=bin

# install prefix
PREFIX:=/usr/local/bin

# root module
ROOT_MODULE:=$(shell go list -m)
# executable 
CMD_MODULES:=$(shell go list ./cmd/...)

# exporter prefix
EXPORTER:=-exporter
# get output file path
BINARIES:=$(CMD_MODULES:$(ROOT_MODULE)/cmd/%=$(BINDIR)/%$(EXPORTER))

# lookup *.go files
GO_FILES:=$(shell find . -type f -name '*.go' -print)

# build trigger
.PHONY: build
build: $(BINARIES)

# build binary
$(BINARIES): $(GO_FILES)
	go build -o $@ $(@:$(BINDIR)/%(EXPORTER)=$(ROOT_MODULE)/cmd/%)
