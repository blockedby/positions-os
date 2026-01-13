.PHONY: migrate-up migrate-down migrate-create deps docker-up docker-down test build

# database url for migrations
DB_URL ?= postgres://jhos:jhos_secret@localhost:5432/jhos?sslmode=disable

# install go dependencies
deps:
	go mod tidy

# run all migrations
migrate-up:
	migrate -path migrations -database "$(DB_URL)" up

# rollback last migration
migrate-down:
	migrate -path migrations -database "$(DB_URL)" down 1

# rollback all migrations
migrate-reset:
	migrate -path migrations -database "$(DB_URL)" down -all

# create new migration (usage: make migrate-create name=create_users)
migrate-create:
	migrate create -ext sql -dir migrations -seq $(name)

# start docker services (infrastructure only)
docker-up:
	docker compose up -d postgres nats

# start full application in docker
docker-app:
	docker compose --profile app up -d

# stop docker services
docker-down:
	docker compose down

# run tests
test:
	go test -v ./...

# build all binaries
build:
	go build -o bin/ ./cmd/...

# run linter
lint:
	golangci-lint run

# format code
fmt:
	go fmt ./...
	goimports -w .

# check if everything compiles
check:
	go build ./...

# clean build artifacts
clean:
	rm -rf bin/
	go clean

# show help
help:
	@echo "Available targets:"
	@echo "  deps           - install go dependencies"
	@echo "  migrate-up     - run all migrations"
	@echo "  migrate-down   - rollback last migration"  
	@echo "  migrate-reset  - rollback all migrations"
	@echo "  migrate-create - create new migration (name=xxx)"
	@echo "  docker-up      - start docker services"
	@echo "  docker-down    - stop docker services"
	@echo "  test           - run tests"
	@echo "  build          - build binaries"
	@echo "  lint           - run linter"
	@echo "  lint-install   - install golangci-lint (Windows)"
	@echo "  fmt            - format code"
	@echo "  check          - verify compilation"
	@echo "  clean          - clean build artifacts"
	@echo "  collector      - run collector service"
	@echo "  tg-auth        - run telegram auth tool"
	@echo "  tg-topics      - run topics lister"

# run collector service
collector:
	go run ./cmd/collector/main.go

# Determine OS
ifeq ($(OS),Windows_NT)
    DETECTED_OS := Windows
else
    DETECTED_OS := $(shell uname -s)
endif

# install linter and git hooks (Universal)
lint-install:
ifeq ($(DETECTED_OS),Windows)
	powershell -ExecutionPolicy ByPass -Command "iwr -Uri https://raw.githubusercontent.com/golangci/golangci-lint/master/install.ps1 -OutFile install.ps1; .\install.ps1 -b (go env GOPATH)/bin v2.8.0; rm install.ps1"
else
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin v2.8.0
endif
	go install github.com/evilmartians/lefthook@latest
	lefthook install

# run telegram auth tool
tg-auth:
	go run ./cmd/tg-auth/main.go

# run telegram topics lister (usage: make tg-topics channel=@forum)
tg-topics:
	go run ./cmd/tg-topics/main.go $(channel)

# run unit tests only
test-unit:
	go test -v -race ./internal/...

# run all tests with coverage
test-coverage:
	go test -cover ./...
