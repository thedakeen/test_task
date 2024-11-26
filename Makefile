include .env


.PHONY: db/migrations/new
db/migrations/new:
	@echo 'Creating migration files for ${name}...'
	migrate create -seq -ext=sql -dir=D:/ProgramData/workspacego/test_task/migrations ${name}

.PHONY: db/migrations/up
db/migrations/up:
	@echo 'Running up migrations for auth service...'
	migrate -path=D:/ProgramData/workspacego/test_task/migrations -database=${DB_DSN} up

.PHONY: app/run
app/run:
	@migrate -path=migrations -database=${DB_DSN} up
	@go run ./cmd/api -db-dsn=${DB_DSN}