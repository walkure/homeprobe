
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
	go build \
	-ldflags="\
	-X github.com/walkure/homeprobe/pkg/revision.commit=$(shell git rev-parse --verify --short HEAD)$(shell test -z `git status --porcelain` || echo '-dirty') \
	-X 'github.com/walkure/homeprobe/pkg/revision.tag=$(shell git describe --exact-match --tags 2>/dev/null || echo NO_TAG)' \
	-X main.binName=$(shell basename $@) \
	" \
	-o $@ $(@:$(BINDIR)/%$(EXPORTER)=$(ROOT_MODULE)/cmd/%) 

clean:
	rm -rf $(BINDIR)
