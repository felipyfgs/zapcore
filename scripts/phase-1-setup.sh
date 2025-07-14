#!/bin/bash

# =============================================================================
# FASE 1: Preparação e Estrutura Base
# =============================================================================
# Cria toda a nova estrutura de diretórios seguindo padrões idiomáticos Go
# =============================================================================

set -e

# Cores para output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

log() {
    echo -e "${BLUE}[FASE 1]${NC} $1"
}

success() {
    echo -e "${GREEN}✅ $1${NC}"
}

log "Iniciando Fase 1: Preparação e Estrutura Base"

# Criar nova estrutura de diretórios
log "Criando nova estrutura de diretórios..."

# Entry points
mkdir -p cmd/wamex

# Application layer
mkdir -p internal/app

# Domain layer
mkdir -p internal/domain/entity
mkdir -p internal/domain/service
mkdir -p internal/domain/repository

# Use case layer
mkdir -p internal/usecase/whatsapp
mkdir -p internal/usecase/media

# Infrastructure layer
mkdir -p internal/infra/database/postgres
mkdir -p internal/infra/database/migration
mkdir -p internal/infra/storage
mkdir -p internal/infra/whatsapp
mkdir -p internal/infra/config

# Transport layer
mkdir -p internal/transport/http/handler
mkdir -p internal/transport/http/middleware
mkdir -p internal/transport/http/router
mkdir -p internal/transport/grpc

# Shared internal code
mkdir -p internal/shared/errors
mkdir -p internal/shared/validator
mkdir -p internal/shared/converter

# Public packages (keep existing structure)
# pkg/ já existe, apenas garantir subdiretórios necessários
mkdir -p pkg/crypto

# API definitions
mkdir -p api/openapi
mkdir -p api/proto

# Web assets (if needed)
mkdir -p web/static
mkdir -p web/templates

# Scripts
mkdir -p scripts

# Deployment configurations
mkdir -p deployments/docker
mkdir -p deployments/k8s
mkdir -p deployments/compose

# Tests
mkdir -p test/integration
mkdir -p test/e2e
mkdir -p test/testdata

# Documentation
mkdir -p docs/api
mkdir -p docs/architecture
mkdir -p docs/deployment

# Development tools
mkdir -p tools

# Configuration files
mkdir -p configs

success "Estrutura de diretórios criada"

# Criar arquivos .gitkeep para manter diretórios vazios no git
log "Criando arquivos .gitkeep para diretórios vazios..."

find . -type d -empty -not -path "./.git/*" -exec touch {}/.gitkeep \;

success "Arquivos .gitkeep criados"

# Criar arquivo de documentação da nova estrutura
log "Criando documentação da nova estrutura..."

cat > docs/architecture/directory-structure.md << 'EOF'
# Estrutura de Diretórios - Wamex

Esta documentação descreve a nova estrutura de diretórios seguindo as melhores práticas idiomáticas do Go.

## Estrutura Geral

```
wamex/
├── cmd/                           # Entry points da aplicação
│   └── wamex/                     # Aplicação principal
├── internal/                      # Código privado da aplicação
│   ├── app/                       # Configuração e inicialização
│   ├── domain/                    # Core business logic
│   │   ├── entity/                # Entidades do domínio
│   │   ├── service/               # Interfaces de serviços
│   │   └── repository/            # Interfaces de repositórios
│   ├── usecase/                   # Casos de uso (application layer)
│   │   ├── whatsapp/              # Use cases do WhatsApp
│   │   └── media/                 # Use cases de mídia
│   ├── infra/                     # Implementações de infraestrutura
│   │   ├── database/              # Implementações de banco
│   │   ├── storage/               # Implementações de storage
│   │   ├── whatsapp/              # Cliente WhatsApp
│   │   └── config/                # Configurações
│   ├── transport/                 # Camada de transporte
│   │   ├── http/                  # HTTP transport
│   │   └── grpc/                  # gRPC transport (futuro)
│   └── shared/                    # Código compartilhado interno
├── pkg/                           # Bibliotecas públicas reutilizáveis
├── api/                           # Definições de API
├── web/                           # Assets web
├── scripts/                       # Scripts de build/deploy
├── deployments/                   # Configurações de deployment
├── test/                          # Testes de integração
├── docs/                          # Documentação
└── tools/                         # Ferramentas de desenvolvimento
```

## Princípios da Organização

### 1. Separação de Responsabilidades
- **cmd/**: Apenas entry points, lógica mínima
- **internal/domain/**: Core business, sem dependências externas
- **internal/usecase/**: Orquestração de casos de uso
- **internal/infra/**: Implementações concretas
- **internal/transport/**: Interfaces de comunicação

### 2. Dependency Rule
- Dependências apontam sempre para dentro (domain)
- Camadas externas dependem de camadas internas
- Domain não depende de nada

### 3. Testabilidade
- Interfaces bem definidas
- Injeção de dependência
- Mocks e stubs facilitados

## Migração

Esta estrutura foi criada durante a migração da estrutura anterior.
Consulte o plano de migração completo em `migration-plan.md`.
EOF

success "Documentação da estrutura criada"

# Validar que a estrutura foi criada corretamente
log "Validando estrutura criada..."

# Verificar se diretórios principais existem
required_dirs=(
    "cmd/wamex"
    "internal/app"
    "internal/domain/entity"
    "internal/usecase/whatsapp"
    "internal/infra/database"
    "internal/transport/http"
    "pkg"
    "api"
    "scripts"
    "test"
    "docs"
)

for dir in "${required_dirs[@]}"; do
    if [[ -d "$dir" ]]; then
        success "✓ $dir"
    else
        echo "❌ $dir não encontrado"
        exit 1
    fi
done

log "Criando Makefile para automação..."

cat > Makefile << 'EOF'
# Makefile para Wamex

.PHONY: build test clean run dev migrate-up migrate-down docker-build docker-run

# Configurações
APP_NAME=wamex
MAIN_PATH=./cmd/wamex
BUILD_DIR=./build

# Build
build:
	@echo "🔨 Building $(APP_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/$(APP_NAME) $(MAIN_PATH)
	@echo "✅ Build completed: $(BUILD_DIR)/$(APP_NAME)"

# Test
test:
	@echo "🧪 Running tests..."
	@go test -v ./...

test-coverage:
	@echo "📊 Running tests with coverage..."
	@go test -v -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "✅ Coverage report: coverage.html"

# Clean
clean:
	@echo "🧹 Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html
	@echo "✅ Cleaned"

# Run
run:
	@echo "🚀 Running $(APP_NAME)..."
	@go run $(MAIN_PATH)

# Development with hot reload (requires air)
dev:
	@echo "🔥 Starting development server..."
	@air

# Database migrations (placeholder)
migrate-up:
	@echo "⬆️  Running migrations up..."
	# TODO: Implement migration up

migrate-down:
	@echo "⬇️  Running migrations down..."
	# TODO: Implement migration down

# Docker
docker-build:
	@echo "🐳 Building Docker image..."
	@docker build -t $(APP_NAME) .

docker-run:
	@echo "🐳 Running Docker container..."
	@docker run -p 8080:8080 $(APP_NAME)

# Help
help:
	@echo "Available commands:"
	@echo "  build         - Build the application"
	@echo "  test          - Run tests"
	@echo "  test-coverage - Run tests with coverage"
	@echo "  clean         - Clean build artifacts"
	@echo "  run           - Run the application"
	@echo "  dev           - Run with hot reload (requires air)"
	@echo "  migrate-up    - Run database migrations up"
	@echo "  migrate-down  - Run database migrations down"
	@echo "  docker-build  - Build Docker image"
	@echo "  docker-run    - Run Docker container"
EOF

success "Makefile criado"

# Criar .air.toml para desenvolvimento com hot reload
log "Criando configuração do Air para hot reload..."

cat > .air.toml << 'EOF'
root = "."
testdata_dir = "testdata"
tmp_dir = "tmp"

[build]
  args_bin = []
  bin = "./tmp/main"
  cmd = "go build -o ./tmp/main ./cmd/wamex"
  delay = 1000
  exclude_dir = ["assets", "tmp", "vendor", "testdata", "build", "docs"]
  exclude_file = []
  exclude_regex = ["_test.go"]
  exclude_unchanged = false
  follow_symlink = false
  full_bin = ""
  include_dir = []
  include_ext = ["go", "tpl", "tmpl", "html"]
  kill_delay = "0s"
  log = "build-errors.log"
  send_interrupt = false
  stop_on_root = false

[color]
  app = ""
  build = "yellow"
  main = "magenta"
  runner = "green"
  watcher = "cyan"

[log]
  time = false

[misc]
  clean_on_exit = false

[screen]
  clear_on_rebuild = false
EOF

success "Configuração do Air criada"

success "Fase 1 concluída com sucesso!"
log "Nova estrutura de diretórios criada e pronta para migração"
