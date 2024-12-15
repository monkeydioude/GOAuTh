.PHONY: help
help:
	@echo 'make help:'
	@echo "\t\`make all_up\`: start containers"
	@echo "\t\`make db_layout\`: re-launch the db container, then import roles, rules and starting data"

.PHONY: install
install:
	cp scripts/git/pre-commit .git/hooks

.PHONY: db_layout
db_layout:
	docker compose up db -d
	docker compose exec db psql -U app_user -d app_db  -f /psql_boot.sql

.PHONY: all_up
all_up:
	docker compose up -d

.PHONY: run-test-db
run-test-db:
	@echo "[INFO] Starting 'goauth-tests-db-manual' PGSQL test container!"
	docker run -d --rm -p 5445:5432 --name goauth-tests-db-manual --env POSTGRES_USER=test --env POSTGRES_PASSWORD=test --env POSTGRES_DB=test_db postgres

.PHONY: stop-test-db
stop-test-db:
	@echo "[INFO] Stopping 'goauth-tests-db-manual' PGSQL test container!"
	docker stop "goauth-tests-db-manual"

.PHONY: unit-test
unit-test:
	go test -count=1 -v ./internal/... ./pkg/... ./plugins/...

.PHONY: test
test:
	@sh scripts/tests.sh

.PHONY: proto-go
proto-go:
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	protoc --go_out=. --go-grpc_out=. -I proto proto/rpc_v1.proto

.PHONY: proto-rust
proto-rust:
	cd proto/rust && cargo build

.PHONY: proto
proto: proto-go proto-rust

.PHONY: dev
dev:
	@docker compose up -d
	cd bin/GOAuTh && go run .