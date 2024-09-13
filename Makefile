.PHONY: help
help:
	@echo 'make help:'
	@echo "\t\`make all_up\`: start containers"
	@echo "\t\`make db_layout\`: re-launch the db container, then import roles, rules and starting data"

.PHONY: db_layout
db_layout:
	docker compose up db -d
	docker compose exec db psql -U app_user -d app_db  -f /psql_boot.sql

.PHONY: all_up
all_up:
	docker compose up -d

.PHONY: test
test:
	go test -v ./...
