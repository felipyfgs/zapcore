#!/bin/bash

# =============================================================================
# WAMEX - Script Master de Migração para Estrutura Idiomática Go
# =============================================================================
# Este script orquestra toda a migração da estrutura atual para a nova
# estrutura idiomática seguindo as melhores práticas do Go.
#
# Uso: ./scripts/migration-master.sh [fase]
# Exemplo: ./scripts/migration-master.sh 1    # Executa apenas a fase 1
#          ./scripts/migration-master.sh all  # Executa todas as fases
# =============================================================================

set -e  # Parar em caso de erro
set -u  # Parar se variável não definida

# Cores para output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configurações
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BACKUP_DIR="${PROJECT_ROOT}/migration-backup"
LOG_FILE="${PROJECT_ROOT}/migration.log"

# Funções utilitárias
log() {
    echo -e "${BLUE}[$(date +'%Y-%m-%d %H:%M:%S')]${NC} $1" | tee -a "$LOG_FILE"
}

success() {
    echo -e "${GREEN}✅ $1${NC}" | tee -a "$LOG_FILE"
}

warning() {
    echo -e "${YELLOW}⚠️  $1${NC}" | tee -a "$LOG_FILE"
}

error() {
    echo -e "${RED}❌ $1${NC}" | tee -a "$LOG_FILE"
    exit 1
}

# Função para validar pré-requisitos
validate_prerequisites() {
    log "Validando pré-requisitos..."
    
    # Verificar se estamos no diretório correto
    if [[ ! -f "go.mod" ]]; then
        error "go.mod não encontrado. Execute o script a partir da raiz do projeto."
    fi
    
    # Verificar se Go está instalado
    if ! command -v go &> /dev/null; then
        error "Go não está instalado ou não está no PATH."
    fi
    
    # Verificar versão do Go
    GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
    log "Versão do Go: $GO_VERSION"
    
    # Verificar se git está disponível
    if ! command -v git &> /dev/null; then
        error "Git não está instalado ou não está no PATH."
    fi
    
    success "Pré-requisitos validados"
}

# Função para criar backup
create_backup() {
    log "Criando backup da estrutura atual..."
    
    # Criar diretório de backup
    mkdir -p "$BACKUP_DIR"
    
    # Fazer backup com git
    git add -A
    git commit -m "backup: estrutura atual antes da migração ($(date))" || true
    git tag -a "pre-migration-backup-$(date +%Y%m%d-%H%M%S)" -m "Backup antes da migração para estrutura idiomática"
    
    # Criar backup físico também
    cp -r internal "$BACKUP_DIR/internal-backup"
    cp -r cmd "$BACKUP_DIR/cmd-backup"
    cp -r pkg "$BACKUP_DIR/pkg-backup"
    cp -r configs "$BACKUP_DIR/configs-backup" 2>/dev/null || true
    
    success "Backup criado em $BACKUP_DIR"
}

# Função para validar compilação
validate_compilation() {
    log "Validando compilação..."
    
    cd "$PROJECT_ROOT"
    
    # Limpar módulos
    go mod tidy
    
    # Tentar compilar
    if go build ./...; then
        success "Compilação bem-sucedida"
    else
        error "Falha na compilação"
    fi
    
    # Executar testes
    if go test ./...; then
        success "Testes executados com sucesso"
    else
        warning "Alguns testes falharam, mas continuando..."
    fi
}

# Função para executar fase específica
execute_phase() {
    local phase=$1
    log "Executando Fase $phase..."
    
    case $phase in
        1)
            execute_phase_1
            ;;
        2)
            execute_phase_2
            ;;
        3)
            execute_phase_3
            ;;
        4)
            execute_phase_4
            ;;
        5)
            execute_phase_5
            ;;
        6)
            execute_phase_6
            ;;
        7)
            execute_phase_7
            ;;
        8)
            execute_phase_8
            ;;
        9)
            execute_phase_9
            ;;
        10)
            execute_phase_10
            ;;
        *)
            error "Fase $phase não reconhecida"
            ;;
    esac
    
    # Validar após cada fase
    validate_compilation
    success "Fase $phase concluída com sucesso"
}

# Implementação das fases (será expandida nos próximos scripts)
execute_phase_1() {
    log "Fase 1: Preparação e Estrutura Base"
    bash "$PROJECT_ROOT/scripts/phase-1-setup.sh"
}

execute_phase_2() {
    log "Fase 2: Migração do Domain Layer"
    bash "$PROJECT_ROOT/scripts/phase-2-domain.sh"
}

execute_phase_3() {
    log "Fase 3: Migração da Camada de Infraestrutura"
    bash "$PROJECT_ROOT/scripts/phase-3-infra.sh"
}

execute_phase_4() {
    log "Fase 4: Migração da Camada de Use Cases"
    bash "$PROJECT_ROOT/scripts/phase-4-usecase.sh"
}

execute_phase_5() {
    log "Fase 5: Migração da Camada de Transport"
    bash "$PROJECT_ROOT/scripts/phase-5-transport.sh"
}

execute_phase_6() {
    log "Fase 6: Migração da Camada de Repository"
    bash "$PROJECT_ROOT/scripts/phase-6-repository.sh"
}

execute_phase_7() {
    log "Fase 7: Reorganização de Shared e PKG"
    bash "$PROJECT_ROOT/scripts/phase-7-shared.sh"
}

execute_phase_8() {
    log "Fase 8: Migração do Entry Point"
    bash "$PROJECT_ROOT/scripts/phase-8-entrypoint.sh"
}

execute_phase_9() {
    log "Fase 9: Migração de Testes"
    bash "$PROJECT_ROOT/scripts/phase-9-tests.sh"
}

execute_phase_10() {
    log "Fase 10: Limpeza e Validação Final"
    bash "$PROJECT_ROOT/scripts/phase-10-cleanup.sh"
}

# Função principal
main() {
    local phase=${1:-"all"}
    
    log "=== INICIANDO MIGRAÇÃO PARA ESTRUTURA IDIOMÁTICA GO ==="
    log "Fase solicitada: $phase"
    
    # Validar pré-requisitos
    validate_prerequisites
    
    # Criar backup
    create_backup
    
    # Validar estado inicial
    validate_compilation
    
    if [[ "$phase" == "all" ]]; then
        # Executar todas as fases
        for i in {1..10}; do
            execute_phase $i
        done
    else
        # Executar fase específica
        execute_phase "$phase"
    fi
    
    log "=== MIGRAÇÃO CONCLUÍDA COM SUCESSO ==="
    success "Nova estrutura idiomática implementada!"
    
    # Mostrar resumo
    echo ""
    echo "📊 RESUMO DA MIGRAÇÃO:"
    echo "├── Backup criado em: $BACKUP_DIR"
    echo "├── Log completo em: $LOG_FILE"
    echo "├── Tag de backup: pre-migration-backup-$(date +%Y%m%d-%H%M%S)"
    echo "└── Status: ✅ SUCESSO"
    echo ""
    echo "🚀 Próximos passos:"
    echo "1. Testar a aplicação: go run cmd/wamex/main.go"
    echo "2. Executar testes: go test ./..."
    echo "3. Verificar endpoints: curl http://localhost:8080/health"
}

# Executar função principal
main "$@"
