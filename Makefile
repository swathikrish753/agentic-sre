# ---- config ----
BIN_DIR := bin
CMDS    := copilot mcp-k8s
GO      := go

# ---- meta ----
.PHONY: all build test lint tidy clean $(CMDS)

all: build

# build every binary in cmd/ into ./bin
build: $(CMDS)

$(CMDS):
	$(GO) build -o $(BIN_DIR)/$@ ./cmd/$@

# run the whole test suite with race detector on
test:
	$(GO) test -race ./...

# static checks (installs nothing; uses go vet for now, swap in golangci-lint later)
lint:
	$(GO) vet ./...

# keep go.mod / go.sum honest
tidy:
	$(GO) mod tidy

clean:
	rm -rf $(BIN_DIR)
