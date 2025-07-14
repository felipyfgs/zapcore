#!/bin/bash

# =============================================================================
# WAMEX - Script de Rollback da Migração
# =============================================================================
# Este script permite fazer rollback da migração para a estrutura anterior
# em caso de problemas durante a migração.
#
# Uso: ./scripts/rollback.sh [backup-tag]
# Exemplo: ./scripts/rollback.sh pre-migration-backup-20250114-143000
#          ./scripts/rollback.sh latest  # Usa o backup mais recente
# =============================================================================

set -e

# Cores para output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Configurações
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BACKUP_DIR="${PROJECT_ROOT}/migration-backup"
LOG_FILE="${PROJECT_ROOT}/rollback.log"

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

# Função para listar backups disponíveis
list_backups() {
    log "Backups disponíveis:"
    
    # Listar tags de backup do git
    echo "📋 Tags de backup no Git:"
    git tag -l "pre-migration-backup-*" | sort -r | head -10
    
    echo ""
    
    # Listar backups físicos
    if [[ -d "$BACKUP_DIR" ]]; then
        echo "📁 Backups físicos em $BACKUP_DIR:"
        ls -la "$BACKUP_DIR" | grep -E "(internal|cmd|pkg|configs)-backup" || echo "Nenhum backup físico encontrado"
    else
        echo "📁 Diretório de backup não encontrado: $BACKUP_DIR"
    fi
}

# Função para validar backup
validate_backup() {
    local backup_tag=$1
    
    log "Validando backup: $backup_tag"
    
    if [[ "$backup_tag" == "latest" ]]; then
        # Encontrar o backup mais recente
        backup_tag=$(git tag -l "pre-migration-backup-*" | sort -r | head -1)
        if [[ -z "$backup_tag" ]]; then
            error "Nenhum backup encontrado"
        fi
        log "Usando backup mais recente: $backup_tag"
    fi
    
    # Verificar se a tag existe
    if ! git tag -l | grep -q "^$backup_tag$"; then
        error "Tag de backup não encontrada: $backup_tag"
    fi
    
    success "Backup validado: $backup_tag"
    echo "$backup_tag"
}

# Função para criar backup do estado atual antes do rollback
create_pre_rollback_backup() {
    log "Criando backup do estado atual antes do rollback..."
    
    git add -A
    git commit -m "backup: estado antes do rollback ($(date))" || true
    git tag -a "pre-rollback-backup-$(date +%Y%m%d-%H%M%S)" -m "Backup antes do rollback"
    
    success "Backup pré-rollback criado"
}

# Função para fazer rollback usando git
rollback_with_git() {
    local backup_tag=$1
    
    log "Fazendo rollback usando Git para tag: $backup_tag"
    
    # Verificar se há mudanças não commitadas
    if ! git diff --quiet || ! git diff --cached --quiet; then
        warning "Há mudanças não commitadas. Fazendo stash..."
        git stash push -m "stash antes do rollback $(date)"
    fi
    
    # Fazer checkout para o backup
    git checkout "$backup_tag"
    
    # Criar nova branch a partir do backup
    local rollback_branch="rollback-from-$backup_tag-$(date +%Y%m%d-%H%M%S)"
    git checkout -b "$rollback_branch"
    
    success "Rollback realizado. Nova branch: $rollback_branch"
}

# Função para fazer rollback usando backup físico
rollback_with_physical_backup() {
    log "Fazendo rollback usando backup físico..."
    
    if [[ ! -d "$BACKUP_DIR" ]]; then
        error "Diretório de backup não encontrado: $BACKUP_DIR"
    fi
    
    # Remover estrutura atual
    log "Removendo estrutura atual..."
    rm -rf internal cmd pkg configs 2>/dev/null || true
    
    # Restaurar do backup
    log "Restaurando do backup físico..."
    
    if [[ -d "$BACKUP_DIR/internal-backup" ]]; then
        cp -r "$BACKUP_DIR/internal-backup" internal
        success "internal/ restaurado"
    fi
    
    if [[ -d "$BACKUP_DIR/cmd-backup" ]]; then
        cp -r "$BACKUP_DIR/cmd-backup" cmd
        success "cmd/ restaurado"
    fi
    
    if [[ -d "$BACKUP_DIR/pkg-backup" ]]; then
        cp -r "$BACKUP_DIR/pkg-backup" pkg
        success "pkg/ restaurado"
    fi
    
    if [[ -d "$BACKUP_DIR/configs-backup" ]]; then
        cp -r "$BACKUP_DIR/configs-backup" configs
        success "configs/ restaurado"
    fi
    
    success "Rollback físico concluído"
}

# Função para validar estado após rollback
validate_rollback() {
    log "Validando estado após rollback..."
    
    # Verificar se arquivos essenciais existem
    essential_files=(
        "go.mod"
        "cmd/main.go"
        "internal/domain/interfaces.go"
    )
    
    for file in "${essential_files[@]}"; do
        if [[ -f "$file" ]]; then
            success "✓ $file existe"
        else
            warning "⚠ $file não encontrado"
        fi
    done
    
    # Tentar compilar
    log "Testando compilação..."
    if go mod tidy && go build ./...; then
        success "Compilação bem-sucedida"
    else
        error "Falha na compilação após rollback"
    fi
    
    # Tentar executar testes
    log "Executando testes..."
    if go test ./...; then
        success "Testes executados com sucesso"
    else
        warning "Alguns testes falharam, mas rollback foi concluído"
    fi
}

# Função para limpeza pós-rollback
cleanup_post_rollback() {
    log "Executando limpeza pós-rollback..."
    
    # Remover diretórios da nova estrutura que podem ter sobrado
    new_structure_dirs=(
        "internal/app"
        "internal/domain/entity"
        "internal/usecase"
        "internal/transport"
        "internal/shared"
        "api"
        "web"
        "test/integration"
        "docs/architecture"
        "deployments"
    )
    
    for dir in "${new_structure_dirs[@]}"; do
        if [[ -d "$dir" ]]; then
            rm -rf "$dir"
            log "Removido: $dir"
        fi
    done
    
    # Remover arquivos da nova estrutura
    new_structure_files=(
        ".air.toml"
        "Makefile"
        "docs/architecture/directory-structure.md"
    )
    
    for file in "${new_structure_files[@]}"; do
        if [[ -f "$file" ]]; then
            rm -f "$file"
            log "Removido: $file"
        fi
    done
    
    success "Limpeza concluída"
}

# Função principal
main() {
    local backup_tag=${1:-""}
    
    log "=== INICIANDO ROLLBACK DA MIGRAÇÃO ==="
    
    if [[ -z "$backup_tag" ]]; then
        echo "Uso: $0 [backup-tag|latest]"
        echo ""
        list_backups
        exit 1
    fi
    
    # Confirmar rollback
    echo ""
    warning "⚠️  ATENÇÃO: Esta operação fará rollback da migração!"
    warning "Isso reverterá todas as mudanças da nova estrutura idiomática."
    echo ""
    read -p "Tem certeza que deseja continuar? (y/N): " -n 1 -r
    echo ""
    
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        log "Rollback cancelado pelo usuário"
        exit 0
    fi
    
    # Validar backup
    backup_tag=$(validate_backup "$backup_tag")
    
    # Criar backup do estado atual
    create_pre_rollback_backup
    
    # Tentar rollback com git primeiro
    if rollback_with_git "$backup_tag"; then
        success "Rollback com Git realizado"
    else
        warning "Rollback com Git falhou, tentando backup físico..."
        rollback_with_physical_backup
    fi
    
    # Limpeza
    cleanup_post_rollback
    
    # Validar resultado
    validate_rollback
    
    log "=== ROLLBACK CONCLUÍDO COM SUCESSO ==="
    success "Estrutura anterior restaurada!"
    
    # Mostrar resumo
    echo ""
    echo "📊 RESUMO DO ROLLBACK:"
    echo "├── Backup usado: $backup_tag"
    echo "├── Log completo em: $LOG_FILE"
    echo "├── Backup pré-rollback criado"
    echo "└── Status: ✅ SUCESSO"
    echo ""
    echo "🚀 Próximos passos:"
    echo "1. Testar a aplicação: go run cmd/main.go"
    echo "2. Executar testes: go test ./..."
    echo "3. Verificar se tudo funciona normalmente"
    echo ""
    echo "💡 Para ver mudanças feitas:"
    echo "   git log --oneline -10"
}

# Executar função principal
main "$@"
