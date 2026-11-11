# PrimeDrive on Lambda durable functions.
# Everything except `synth` works offline, without AWS credentials.

MODEL_DIR := local/models/resnet/1
MODEL_URL := https://tfhub.dev/tensorflow/resnet_50/classification/1?tf-hub-format=compressed
COMPOSE   := docker compose -f local/compose.yaml
DURABLELINT := go tool -modfile=tools/go.mod durablelint
LOCAL_API  := localhost:8080
LOCAL_USER := alice

.DEFAULT_GOAL := help

.PHONY: help
help: ## Show this help
	@grep -E '^[a-z-]+:.*##' $(MAKEFILE_LIST) | awk -F':.*## ' '{printf "  %-14s %s\n", $$1, $$2}'

.PHONY: test
test: ## Run all Go tests, including the durable workflow on the local runner
	go test ./...

.PHONY: test-workflow
test-workflow: ## Run the workflow tests and print each execution's operations
	go test ./internal/pipeline/ -v -count=1

.PHONY: lint
lint: ## Check readability (golangci-lint) and replay safety (durablelint)
	golangci-lint run ./...
	$(DURABLELINT) ./...

.PHONY: fmt
fmt: ## Format the Go code
	golangci-lint fmt ./...

.PHONY: synth
synth: ## Build the Go functions and synthesize the CloudFormation template
	cd infra && npx cdk synth --quiet

.PHONY: local-model
local-model: $(MODEL_DIR)/saved_model.pb ## Download the image classifier (about 100 MB, once)

$(MODEL_DIR)/saved_model.pb:
	mkdir -p $(MODEL_DIR)
	curl -sSL "$(MODEL_URL)" | tar -xz -C $(MODEL_DIR)

.PHONY: local-up
local-up: local-model ## Start SeaweedFS, DynamoDB Local and TensorFlow Serving
	$(COMPOSE) up -d --wait

.PHONY: local-down
local-down: ## Stop the local stack and delete its data
	$(COMPOSE) down -v

.PHONY: local-run
local-run: ## Run the pipeline locally: make local-run IMAGE=~/Pictures/cat.jpg
	go run ./cmd/local -image "$(IMAGE)"

.PHONY: local-api
local-api: ## Serve the REST API on localhost:8080, in place of API Gateway and Cognito
	go run ./cmd/local -serve $(LOCAL_API)

.PHONY: local-web
local-web: frontend/node_modules ## Start the frontend against the local API, signed in as alice
	cd frontend && NEXT_PUBLIC_ENDPOINT=http://$(LOCAL_API) NEXT_PUBLIC_LOCAL_USER=$(LOCAL_USER) npm run dev

frontend/node_modules:
	cd frontend && npm install
