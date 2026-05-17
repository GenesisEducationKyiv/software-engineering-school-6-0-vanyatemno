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
