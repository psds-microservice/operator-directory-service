.PHONY: help build run run-dev migrate reindex-search clean tidy vet fmt health-check proto proto-build proto-generate proto-generate-local proto-generate-docker proto-openapi install-deps update docker-build docker-compose-up docker-compose-down

APP_NAME = operator-directory-service
CMD_PATH = ./cmd/operator-directory-service
BIN_DIR = bin
PORT = 8098
PROTOC_IMAGE = local/protoc-go:latest
PROTO_ROOT = pkg/operator_directory_service
GEN_DIR = pkg/gen/operator_directory_service
GO_MODULE = github.com/psds-microservice/operator-directory-service
OPENAPI_OUT = api
.DEFAULT_GOAL := help

help:
	@echo "operator-directory-service"
	@echo "  make build run run-dev migrate reindex-search clean tidy vet fmt health-check docker-build docker-compose-up"
	@echo "  make proto              - Build protoc image (if infra) and generate Go from .proto"
	@echo "  make proto-generate     - Generate Go + gRPC + grpc-gateway from pkg/operator_directory_service/*.proto"
	@echo "  make proto-openapi      - Generate OpenAPI (api/openapi.json) from .proto"
	@echo "  make install-deps       - go mod download + install protoc plugins"
	@echo "  make update             - go get -u, tidy, vendor, proto (as in user-service)"
	@echo "  Port: $(PORT)  Health: http://localhost:$(PORT)/health  Swagger: http://localhost:$(PORT)/swagger"

build:
	@mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/$(APP_NAME) $(CMD_PATH)
	@echo "OK: $(BIN_DIR)/$(APP_NAME)"

run: build
	@cd $(BIN_DIR) && ./$(APP_NAME) api

run-dev:
	go run $(CMD_PATH) api

migrate: build
	@cd $(BIN_DIR) && ./$(APP_NAME) migrate up

search: build
	@cd $(BIN_DIR) && ./$(APP_NAME) reindex-search

health-check:
	@curl -sf http://localhost:$(PORT)/health && echo " OK" || echo " FAIL"

vet:
	go vet ./...

fmt:
	go fmt ./...

clean:
	rm -rf $(BIN_DIR)
	go clean

tidy:
	go mod tidy

install-deps:
	@echo "Installing dependencies..."
	go mod download
	go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest
	go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@latest
	@echo "Dependencies installed"

update:
	@echo "Updating dependencies..."
	go get -u ./... github.com/psds-microservice/helpy
	go mod tidy
	go mod vendor
	$(MAKE) proto
	@$(MAKE) proto-openapi 2>/dev/null || true
	@echo "Dependencies updated"

## Proto: образ из vendored infra (go mod vendor)
INFRA_VENDOR := vendor/github.com/psds-microservice/infra
proto: proto-build proto-generate

proto-build:
	@echo "Building protoc-go image (from $(INFRA_VENDOR))..."
	@docker build -t $(PROTOC_IMAGE) -f $(INFRA_VENDOR)/protoc-go.Dockerfile $(INFRA_VENDOR)
	@echo "Docker image built"

proto-generate:
	@PATH="$$(go env GOPATH 2>/dev/null)/bin:$$PATH"; \
	if command -v protoc >/dev/null 2>&1 && command -v protoc-gen-go >/dev/null 2>&1 && command -v protoc-gen-go-grpc >/dev/null 2>&1; then \
		$(MAKE) proto-generate-local; \
	else \
		$(MAKE) proto-generate-docker; \
	fi

proto-generate-local:
	@echo "Generating Go code (local protoc)..."
	@mkdir -p $(GEN_DIR) $(OPENAPI_OUT)
	@command -v protoc-gen-grpc-gateway >/dev/null 2>&1 || (echo "Install: go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest" && exit 1)
	@PATH="$$(go env GOPATH)/bin:$$PATH"; \
	protoc -I $(PROTO_ROOT) -I third_party \
		--go_out=. --go_opt=module=$(GO_MODULE) \
		--go-grpc_out=. --go-grpc_opt=module=$(GO_MODULE) \
		--grpc-gateway_out=. --grpc-gateway_opt=module=$(GO_MODULE) \
		$(PROTO_ROOT)/operator_directory.proto
	@echo "OK: $(GEN_DIR)"

proto-generate-docker:
	@echo "Generating Go code (Docker)..."
	@mkdir -p $(GEN_DIR)
	@docker run --rm -v "$(CURDIR):/workspace" -w /workspace --entrypoint sh $(PROTOC_IMAGE) -c '\
		protoc -I $(PROTO_ROOT) -I third_party -I /include \
		--go_out=. --go_opt=module=$(GO_MODULE) \
		--go-grpc_out=. --go-grpc_opt=module=$(GO_MODULE) \
		--grpc-gateway_out=. --grpc-gateway_opt=module=$(GO_MODULE) \
		$(PROTO_ROOT)/operator_directory.proto' || (echo "Run: make proto-build (vendor infra) or use local protoc + plugins" && exit 1)
	@echo "OK: $(GEN_DIR)"

proto-openapi:
	@command -v protoc >/dev/null 2>&1 || (echo "Install protoc" && exit 1); \
	command -v protoc-gen-openapiv2 >/dev/null 2>&1 || (echo "Install: go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@latest" && exit 1)
	@mkdir -p $(OPENAPI_OUT)
	@PATH="$$(go env GOPATH)/bin:$$PATH"; \
	protoc -I $(PROTO_ROOT) -I third_party \
		--openapiv2_out=$(OPENAPI_OUT) \
		--openapiv2_opt=logtostderr=true \
		--openapiv2_opt=allow_merge=true \
		--openapiv2_opt=merge_file_name=openapi \
		$(PROTO_ROOT)/operator_directory.proto
	@if [ -f $(OPENAPI_OUT)/openapi.swagger.json ]; then cp $(OPENAPI_OUT)/openapi.swagger.json $(OPENAPI_OUT)/openapi.json; echo "OK: $(OPENAPI_OUT)/openapi.json"; fi

docker-build:
	docker build -f deployments/Dockerfile -t $(APP_NAME):latest ..

docker-compose-up:
	docker compose -f deployments/docker-compose.yml up -d

docker-compose-down:
	docker compose -f deployments/docker-compose.yml down
