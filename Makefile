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

# -- gRPC / protobuf codegen ---------------------------------------------------
#
# The notifier's synchronous delivery API is defined in pkg/contract/notifierpb.
# `buf` is fetched on demand via `go run` (it bundles its own proto compiler, so
# no system protoc is needed); it invokes the protoc-gen-go / protoc-gen-go-grpc
# plugins, which we install to GOPATH/bin first. Generated *.pb.go files are
# checked in, so day-to-day builds and CI do not run this target.
PROTOC_GEN_GO_VERSION := v1.36.6
PROTOC_GEN_GO_GRPC_VERSION := v1.5.1
BUF_VERSION := v1.50.0

proto:
	@echo "==> installing protoc plugins"
	@GOTOOLCHAIN=local go install google.golang.org/protobuf/cmd/protoc-gen-go@$(PROTOC_GEN_GO_VERSION)
	@GOTOOLCHAIN=local go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@$(PROTOC_GEN_GO_GRPC_VERSION)
	@echo "==> buf generate (pkg/contract)"
	@cd pkg/contract && PATH="$(GO_PATH)/bin:$$PATH" GOTOOLCHAIN=local \
		go run github.com/bufbuild/buf/cmd/buf@$(BUF_VERSION) generate

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

# -- Full observability stack (app + logging + metrics) ------------------------
#
# Everything lives in the single docker-compose.yml: backend, Postgres, Redis,
# Elasticsearch + Kibana + Filebeat, and Prometheus + Grafana.
#   Swagger:    http://localhost:8080/swagger/index.html
#   Kibana:     http://localhost:5601
#   Prometheus: http://localhost:9090
#   Grafana:    http://localhost:3000 (admin/admin)

# Bring up the whole stack.
up:
	docker compose up --build -d

# Tear it down (append `-v` manually to also drop the ES/TSDB/db volumes).
down:
	docker compose down

# Follow logs (pass SVC=... to scope, e.g. `make logs SVC="prometheus grafana"`).
logs:
	docker compose logs -f $(SVC)
