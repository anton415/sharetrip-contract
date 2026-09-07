OAPI_CODEGEN_VERSION := v2.8.0
GOEXE := $(shell go env GOEXE)
OAPI_CODEGEN_BIN := bin/oapi-codegen$(GOEXE)
OAPI_CODEGEN := ./$(OAPI_CODEGEN_BIN)
GOLANGCI_LINT_VERSION := v2.4.0
GOLANGCI_LINT_BIN := bin/golangci-lint$(GOEXE)
GOOSE ?= goose

export GOBIN := $(CURDIR)/bin

.PHONY: tools generate generate-api run build test lint check-database-url migrate-up migrate-status

tools: $(OAPI_CODEGEN_BIN)

$(OAPI_CODEGEN_BIN):
	go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@$(OAPI_CODEGEN_VERSION)

generate: generate-api

generate-api: $(OAPI_CODEGEN_BIN)
	$(OAPI_CODEGEN) --config api/openapi.codegen.yaml api/contract.yaml

run: generate
	go run ./cmd

build:
	@mkdir -p bin
	go build -o bin/contract ./cmd

test:
	go test ./...

$(GOLANGCI_LINT_BIN):
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

lint: $(GOLANGCI_LINT_BIN)
	./$(GOLANGCI_LINT_BIN) run --enable=paralleltest ./...

check-database-url:
	@test -n "$$DATABASE_URL" || { echo "DATABASE_URL is required" >&2; exit 1; }

migrate-up: check-database-url
	$(GOOSE) -dir migrations postgres "$$DATABASE_URL" up

migrate-status: check-database-url
	$(GOOSE) -dir migrations postgres "$$DATABASE_URL" status
