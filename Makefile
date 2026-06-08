GO_PATH := $(shell go env GOPATH)

# The repository is a multi-module workspace: the shared contract module plus
# one module per microservice. Most Go targets iterate over all of them.
GO_MODULES := pkg/contract services/api services/notifications

dependencies:
	@for m in $(GO_MODULES); do \
		echo "==> $$m"; \
		(cd $$m && go mod tidy && go mod download) || exit $$?; \
	done

lint: check-lint
	@for m in services/api services/notifications; do \
		echo "==> lint $$m"; \
		(cd $$m && $(GO_PATH)/bin/golangci-lint run --timeout=1m -c $(CURDIR)/.golangci.yml ./...) || exit $$?; \
	done

check-lint:
	@which golangci-lint || curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(GO_PATH)/bin latest

swagger:
	@cd services/api && swag init -g cmd/main.go -o docs/generated

test-unit:
	cd services/api && go test -race -count=1 ./internal/...
	cd services/notifications && go test -race -count=1 ./...
	cd pkg/contract && go test -race -count=1 ./...

test-integration:
	@docker compose -f docker-compose.test.yml up --build --abort-on-container-exit --exit-code-from tests; \
	rc=$$?; \
	docker compose -f docker-compose.test.yml down -v; \
	exit $$rc

test-integration-down:
	docker compose -f docker-compose.test.yml down -v

test-e2e:
	@docker compose --env-file .env.e2e -f docker-compose.e2e.yml up --build --abort-on-container-exit --exit-code-from tests; \
	rc=$$?; \
	docker compose --env-file .env.e2e -f docker-compose.e2e.yml down -v; \
	exit $$rc

test-e2e-down:
	docker compose --env-file .env.e2e -f docker-compose.e2e.yml down -v

# -- Logging pipeline (Elasticsearch + Kibana + Filebeat) ----------------------

LOGGING_COMPOSE := -f docker-compose.yml -f docker-compose.logging.yml

# Bring up the app together with the Elasticsearch/Kibana/Filebeat stack.
logging-up:
	docker compose $(LOGGING_COMPOSE) up --build -d

# Tear the logging stack down (add `-v` manually to also drop the ES volume).
logging-down:
	docker compose $(LOGGING_COMPOSE) down

# Follow the logs of the pipeline components.
logging-logs:
	docker compose $(LOGGING_COMPOSE) logs -f filebeat elasticsearch kibana
