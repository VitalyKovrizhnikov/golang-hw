BIN := "./bin/calendar"
SCHEDULER_BIN := "./bin/calendar_scheduler"
STORER_BIN := "./bin/calendar_storer"
DOCKER_IMG="calendar:develop"

GIT_HASH := $(shell git log --format="%h" -n 1)
LDFLAGS := -X main.release="develop" -X main.buildDate=$(shell date -u +%Y-%m-%dT%H:%M:%S) -X main.gitHash=$(GIT_HASH)

STORAGE_DSN ?= postgres://calendar:calendar@localhost:5432/calendar?sslmode=disable

build:
	mkdir -p ./bin
	go build -v -o $(BIN) -ldflags "$(LDFLAGS)" ./cmd/calendar
	go build -v -o $(SCHEDULER_BIN) -ldflags "$(LDFLAGS)" ./cmd/calendar_scheduler
	go build -v -o $(STORER_BIN) -ldflags "$(LDFLAGS)" ./cmd/calendar_storer

run: build
	$(BIN) -config ./configs/config.toml

run-scheduler: build
	$(SCHEDULER_BIN) -config ./configs/scheduler.toml

run-storer: build
	$(STORER_BIN) -config ./configs/storer.toml

build-img:
	docker build \
		--build-arg=LDFLAGS="$(LDFLAGS)" \
		-t $(DOCKER_IMG) \
		-f build/Dockerfile .

run-img: build-img
	docker run $(DOCKER_IMG)

version: build
	$(BIN) version

generate:
	go run github.com/deepmap/oapi-codegen/cmd/oapi-codegen@v1.12.4 -generate chi-server,types -package openapi -o internal/server/http/openapi/api.gen.go api/openapi.yaml

test:
	go test -race ./...

install-lint-deps:
	(which golangci-lint > /dev/null) || curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin v1.50.1

lint: install-lint-deps
	golangci-lint run ./...

migrate:
	@for file in migrations/*.sql; do \
		echo "Applying $$file"; \
		psql "$(STORAGE_DSN)" -f "$$file" || exit 1; \
	done

up:
	docker compose -f deployments/docker-compose.yaml up --build -d

down:
	docker compose -f deployments/docker-compose.yaml down -v --remove-orphans

integration-tests:
	go run ./scripts/integration-tests

.PHONY: build run run-scheduler run-storer build-img run-img version generate test lint migrate up down integration-tests
