# PrimeDrive on Lambda durable functions.
#
# `make` lists what you can do. The targets under "Check" and "Run locally"
# need no AWS account; the ones under "AWS" use the credentials and the
# region of your AWS CLI.

MODEL_DIR := local/models/resnet/1
MODEL_URL := https://tfhub.dev/tensorflow/resnet_50/classification/1?tf-hub-format=compressed
COMPOSE   := docker compose -f local/compose.yaml
DURABLELINT := go tool -modfile=tools/go.mod durablelint
LOCAL_API  := localhost:8080
LOCAL_USER := alice

# The region the stack is deployed to and looked up in: make deploy AWS_REGION=eu-central-1
AWS_REGION ?= eu-west-1
export AWS_REGION
export AWS_DEFAULT_REGION := $(AWS_REGION)
export LOCAL_API LOCAL_USER

# NAME=<username> counts only when given on the command line. Some shells
# (WSL for one) already have a NAME in the environment, which is not a user.
DRIVE_USER := $(if $(filter command line,$(origin NAME)),$(NAME))

.DEFAULT_GOAL := help

.PHONY: help
help: ## Show this help
	@awk -F':.*## ' '/^##@/ {printf "\n%s\n", substr($$0, 5)} /^[a-z-]+:.*## / {printf "  %-14s %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@echo

##@ Check

.PHONY: check
check: lint lint-web test test-infra synth ## Everything below in one go; run it before you push
	@echo "All checks passed."

.PHONY: lint
lint: ## Check readability (golangci-lint) and replay safety (durablelint)
	golangci-lint run ./...
	$(DURABLELINT) ./...

.PHONY: lint-web
lint-web: frontend/node_modules ## Lint the frontend
	cd frontend && npm run lint

.PHONY: test
test: ## Run all Go tests, including the durable workflow on the local runner
	go test ./...

.PHONY: test-workflow
test-workflow: ## Run the workflow tests and print each execution's operations
	go test ./internal/pipeline/ -v -count=1

.PHONY: test-infra
test-infra: infra/node_modules ## Test the CloudFormation template the CDK app produces
	cd infra && npm test

.PHONY: synth
synth: infra/node_modules ## Build the Go functions and synthesize the CloudFormation template
	cd infra && npx cdk synth --quiet

.PHONY: fmt
fmt: ## Format the Go code
	golangci-lint fmt ./...

##@ Run locally, without AWS

.PHONY: local
local: local-up frontend/node_modules ## The whole app on your machine: containers, API and frontend
	@tools/local.sh

.PHONY: local-seed
local-seed: ## Upload sample photos to the local drive, some of which need a review
	@tools/seed.sh local $(LOCAL_USER)

.PHONY: local-run
local-run: ## Run the pipeline once on the terminal: make local-run IMAGE=~/Pictures/cat.jpg
	go run ./cmd/local -image "$(IMAGE)"

.PHONY: local-up
local-up: local-model ## Start the containers: SeaweedFS, DynamoDB Local, TensorFlow Serving
	$(COMPOSE) up -d --wait

.PHONY: local-down
local-down: ## Stop the containers and delete their data
	$(COMPOSE) down -v

.PHONY: local-api
local-api: ## Only the API, on localhost:8080, in place of API Gateway and Cognito
	@cat local/NOTICE.txt
	go run ./cmd/local -serve $(LOCAL_API)

.PHONY: local-web
local-web: frontend/node_modules ## Only the frontend, against the local API, signed in as alice
	cd frontend && NEXT_PUBLIC_ENDPOINT=http://$(LOCAL_API) NEXT_PUBLIC_LOCAL_USER=$(LOCAL_USER) npm run dev

.PHONY: local-model
local-model: $(MODEL_DIR)/saved_model.pb ## Download the image classifier (about 100 MB, once)

$(MODEL_DIR)/saved_model.pb:
	mkdir -p $(MODEL_DIR)
	curl -sSL "$(MODEL_URL)" | tar -xz -C $(MODEL_DIR)

##@ AWS

.PHONY: bootstrap
bootstrap: infra/node_modules ## Prepare an account and region for the CDK (once per account and region)
	cd infra && npx cdk bootstrap

.PHONY: deploy
deploy: infra/node_modules ## Deploy the stack with the CDK, to AWS_REGION (eu-west-1 unless you set it)
	cd infra && npx cdk deploy
	@echo
	@echo "Deployed to $(AWS_REGION). Next: make user NAME=<username>, make seed NAME=<username>, make web"

.PHONY: outputs
outputs: ## Show the outputs of the deployed stack: API URL, bucket, table
	@aws cloudformation describe-stacks --stack-name PrimeDriveStack \
		--query "Stacks[0].Outputs[].[OutputKey,OutputValue]" --output table

.PHONY: user
user: ## Create a user and print a generated password: make user NAME=alice
	@test -n "$(DRIVE_USER)" || { echo "usage: make user NAME=<username> [EMAIL=<address>]" >&2; exit 2; }
	@tools/create-user.sh "$(DRIVE_USER)" $(EMAIL)

.PHONY: seed
seed: ## Upload sample photos, some of which need a review: make seed NAME=alice
	@test -n "$(DRIVE_USER)" || { echo "usage: make seed NAME=<username>" >&2; exit 2; }
	@tools/seed.sh aws "$(DRIVE_USER)"

.PHONY: empty
empty: ## Delete a user's photos and catalog records for good: make empty NAME=alice
	@test -n "$(DRIVE_USER)" || { echo "usage: make empty NAME=<username>" >&2; exit 2; }
	@tools/empty-drive.sh "$(DRIVE_USER)"

.PHONY: empty-all
empty-all: ## Start from scratch: empty every drive and stop the open executions
	@tools/empty-drive.sh --all

.PHONY: web
web: frontend/node_modules ## Run the frontend on localhost:3000 against the deployed API
	@tools/web.sh

.PHONY: destroy
destroy: infra/node_modules ## Delete the deployed stack, with its users, uploads and catalog
	cd infra && npx cdk destroy

frontend/node_modules:
	cd frontend && npm install

infra/node_modules:
	cd infra && npm install
