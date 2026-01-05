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

# start docker services
docker-up:
	docker compose up -d

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
	@echo "  fmt            - format code"
	@echo "  check          - verify compilation"
	@echo "  clean          - clean build artifacts"
