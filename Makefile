MODULE   := github.com/0377/m3u8
BINARY   := m3u8
GO       := go
GOFLAGS  ?= -mod=vendor

export GOFLAGS

.PHONY: all build test vendor tidy clean

all: build

build:
	$(GO) build -o $(BINARY) .

test:
	$(GO) test ./...

vendor: tidy
	@$(GO) mod vendor 2>/dev/null || { \
		mkdir -p vendor; \
		printf '# %s\n## explicit; go 1.22\n%s\n' $(MODULE) $(MODULE) > vendor/modules.txt; \
	}

tidy:
	$(GO) mod tidy

clean:
	rm -f $(BINARY) $(BINARY).exe

# 交叉编译
build-linux:
	GOOS=linux GOARCH=amd64 $(GO) build -o $(BINARY)-linux-amd64 .

build-darwin-arm64:
	GOOS=darwin GOARCH=arm64 $(GO) build -o $(BINARY)-darwin-arm64 .

build-windows:
	GOOS=windows GOARCH=amd64 $(GO) build -o $(BINARY).exe .
