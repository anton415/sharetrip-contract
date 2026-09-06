OAPI_CODEGEN_VERSION := v2.8.0
GOEXE := $(shell go env GOEXE)
OAPI_CODEGEN_BIN := bin/oapi-codegen$(GOEXE)
OAPI_CODEGEN := ./$(OAPI_CODEGEN_BIN)

export GOBIN := $(CURDIR)/bin

.PHONY: tools generate generate-api run

tools: $(OAPI_CODEGEN_BIN)

$(OAPI_CODEGEN_BIN):
	go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@$(OAPI_CODEGEN_VERSION)

generate: generate-api

generate-api: $(OAPI_CODEGEN_BIN)
	$(OAPI_CODEGEN) --config api/openapi.codegen.yaml api/contract.yaml

run: generate
	go run ./cmd
