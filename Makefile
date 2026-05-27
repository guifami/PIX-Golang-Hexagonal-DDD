# ==============================
# CONFIG
# ==============================

APP_NAME=go-api
MAIN_PATH=cmd/api/main.go

# ==============================
# DOCKER
# ==============================

up:
	docker compose up --build

down:
	docker compose down

reset-db:
	docker compose down -v
	docker compose up --build

logs:
	docker compose logs -f

ps:
	docker compose ps

# ==============================
# LOCAL RUN
# ==============================

run:
	go run $(MAIN_PATH)

# ==============================
# SWAGGER
# ==============================

swag:
	rm -rf docs
	swag init -g $(MAIN_PATH) --parseInternal --output docs

# ==============================
# GO UTILS
# ==============================

tidy:
	go mod tidy

fmt:
	go fmt ./...

test:
	go test ./internal/...

cover:
	go test -coverprofile=coverage.out ./internal/domain/... ./internal/application/usecase/...
	go tool cover -html=coverage.out -o coverage.html
	@echo "--- Cobertura por função ---"
	@go tool cover -func=coverage.out
	open coverage.html

# ==============================
# SETUP (ONBOARDING)
# ==============================

setup: tidy swag up

# ==============================
# REBUILD TOTAL
# ==============================

rebuild:
	docker compose down -v
	docker compose up --build --force-recreate