# fletway-backend — tareas de desarrollo
# Uso: make <target>   (en Windows: usar `make` de Git Bash / MSYS, o los scripts en scripts/)

.DEFAULT_GOAL := help
BIN := bin/api
PKG := ./...

.PHONY: help
help: ## Lista los targets disponibles
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}'

.PHONY: run
run: ## Corre el servidor (go run ./cmd/api)
	go run ./cmd/api

.PHONY: build
build: ## Compila el binario en bin/api
	go build -o $(BIN) ./cmd/api

.PHONY: test
test: ## Corre los tests con race detector
	go test -race -count=1 $(PKG)

.PHONY: cover
cover: ## Corre tests con reporte de cobertura
	go test -race -coverprofile=coverage.out $(PKG)
	go tool cover -func=coverage.out | tail -1

.PHONY: lint
lint: ## Corre golangci-lint (requiere instalarlo)
	golangci-lint run

.PHONY: fmt
fmt: ## Formatea el código
	gofmt -w .
	@command -v goimports >/dev/null 2>&1 && goimports -w . || true

.PHONY: vet
vet: ## go vet
	go vet $(PKG)

.PHONY: tidy
tidy: ## go mod tidy
	go mod tidy

.PHONY: check
check: fmt vet lint test ## Corrida completa pre-commit

.PHONY: hooks
hooks: ## Instala el hook de pre-commit
	cp scripts/pre-commit .git/hooks/pre-commit && chmod +x .git/hooks/pre-commit
	@echo "hook de pre-commit instalado"
