GO_PATH := $(shell go env GOPATH)

dependencies:
	@go mod tidy
	@go mod download

lint: check-lint dependencies
	$(GO_PATH)/bin/golangci-lint run --timeout=1m -c .golangci.yml

check-lint:
	@which golangci-lint || curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(GO_PATH)/bin latest

swagger:
	@swag init -g cmd/main.go -o docs/generated

test-unit:
	go test -race -count=1 ./internal/...

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

# -- Metrics pipeline (Prometheus + Grafana) -----------------------------------

METRICS_COMPOSE := -f docker-compose.yml -f docker-compose.metrics.yml

# Bring up the app together with the Prometheus/Grafana stack.
# Prometheus UI: http://localhost:9090  Grafana: http://localhost:3000 (admin/admin).
metrics-up:
	docker compose $(METRICS_COMPOSE) up --build -d

# Tear the metrics stack down (add `-v` manually to also drop the TSDB volume).
metrics-down:
	docker compose $(METRICS_COMPOSE) down

# Follow the logs of the pipeline components.
metrics-logs:
	docker compose $(METRICS_COMPOSE) logs -f prometheus grafana
