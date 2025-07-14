#!/bin/bash

# =============================================================================
# WAMEX - Script de Validação da Migração
# =============================================================================
# Este script valida se a migração foi realizada corretamente, testando
# compilação, testes, estrutura de diretórios e funcionalidade da aplicação.
#
# Uso: ./scripts/validate-migration.sh [--detailed]
# =============================================================================

set -e

# Cores para output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
NC='\033[0m'

# Configurações
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
LOG_FILE="${PROJECT_ROOT}/validation.log"
DETAILED=${1:-""}

# Contadores
TESTS_PASSED=0
TESTS_FAILED=0
WARNINGS=0

# Funções utilitárias
log() {
    echo -e "${BLUE}[$(date +'%Y-%m-%d %H:%M:%S')]${NC} $1" | tee -a "$LOG_FILE"
}

success() {
    echo -e "${GREEN}✅ $1${NC}" | tee -a "$LOG_FILE"
    ((TESTS_PASSED++))
}

warning() {
    echo -e "${YELLOW}⚠️  $1${NC}" | tee -a "$LOG_FILE"
    ((WARNINGS++))
}

error() {
    echo -e "${RED}❌ $1${NC}" | tee -a "$LOG_FILE"
    ((TESTS_FAILED++))
}

info() {
    echo -e "${PURPLE}ℹ️  $1${NC}" | tee -a "$LOG_FILE"
}

# Função para validar estrutura de diretórios
validate_directory_structure() {
    log "Validando estrutura de diretórios..."
    
    # Estrutura esperada da nova organização
    expected_dirs=(
        "cmd/wamex"
        "internal/app"
        "internal/domain/entity"
        "internal/domain/service"
        "internal/domain/repository"
        "internal/usecase/whatsapp"
        "internal/usecase/media"
        "internal/infra/database"
        "internal/infra/storage"
        "internal/infra/config"
        "internal/transport/http/handler"
        "internal/transport/http/middleware"
        "internal/transport/http/router"
        "internal/shared"
        "pkg/logger"
        "api"
        "scripts"
        "test"
        "docs"
    )
    
    for dir in "${expected_dirs[@]}"; do
        if [[ -d "$dir" ]]; then
            success "Diretório existe: $dir"
        else
            error "Diretório ausente: $dir"
        fi
    done
    
    # Verificar se diretórios antigos foram removidos
    old_dirs=(
        "internal/handler"
        "internal/middleware"
        "internal/routes"
        "configs"
    )
    
    for dir in "${old_dirs[@]}"; do
        if [[ -d "$dir" ]]; then
            warning "Diretório antigo ainda existe: $dir"
        else
            success "Diretório antigo removido: $dir"
        fi
    done
}

# Função para validar arquivos essenciais
validate_essential_files() {
    log "Validando arquivos essenciais..."
    
    essential_files=(
        "go.mod"
        "go.sum"
        "cmd/wamex/main.go"
        "internal/domain/entity/session.go"
        "internal/domain/entity/message.go"
        "internal/domain/entity/media.go"
        "pkg/logger/logger.go"
        "Makefile"
    )
    
    for file in "${essential_files[@]}"; do
        if [[ -f "$file" ]]; then
            success "Arquivo existe: $file"
        else
            error "Arquivo ausente: $file"
        fi
    done
}

# Função para validar imports
validate_imports() {
    log "Validando imports nos arquivos Go..."
    
    # Verificar se não há imports da estrutura antiga
    old_imports=(
        "wamex/internal/handler"
        "wamex/internal/middleware"
        "wamex/internal/routes"
        "wamex/configs"
    )
    
    for import in "${old_imports[@]}"; do
        if grep -r "$import" --include="*.go" . >/dev/null 2>&1; then
            error "Import antigo encontrado: $import"
            if [[ "$DETAILED" == "--detailed" ]]; then
                grep -r "$import" --include="*.go" . | head -5
            fi
        else
            success "Import antigo não encontrado: $import"
        fi
    done
    
    # Verificar se imports da nova estrutura estão presentes
    new_imports=(
        "wamex/internal/domain/entity"
        "wamex/internal/transport/http"
        "wamex/internal/infra/config"
    )
    
    for import in "${new_imports[@]}"; do
        if grep -r "$import" --include="*.go" . >/dev/null 2>&1; then
            success "Import da nova estrutura encontrado: $import"
        else
            warning "Import da nova estrutura não encontrado: $import"
        fi
    done
}

# Função para validar compilação
validate_compilation() {
    log "Validando compilação..."
    
    cd "$PROJECT_ROOT"
    
    # Limpar módulos
    if go mod tidy; then
        success "go mod tidy executado com sucesso"
    else
        error "Falha em go mod tidy"
        return
    fi
    
    # Compilar todos os pacotes
    if go build ./...; then
        success "Compilação de todos os pacotes bem-sucedida"
    else
        error "Falha na compilação"
        return
    fi
    
    # Compilar aplicação principal
    if go build -o build/wamex cmd/wamex/main.go; then
        success "Compilação da aplicação principal bem-sucedida"
        
        # Verificar se binário foi criado
        if [[ -f "build/wamex" ]]; then
            success "Binário criado: build/wamex"
        else
            error "Binário não foi criado"
        fi
    else
        error "Falha na compilação da aplicação principal"
    fi
}

# Função para validar testes
validate_tests() {
    log "Validando testes..."
    
    # Executar todos os testes
    if go test ./...; then
        success "Todos os testes passaram"
    else
        error "Alguns testes falharam"
    fi
    
    # Executar testes com verbose se detalhado
    if [[ "$DETAILED" == "--detailed" ]]; then
        log "Executando testes detalhados..."
        go test -v ./... | tee -a "$LOG_FILE"
    fi
    
    # Verificar cobertura de testes
    if go test -coverprofile=coverage.out ./... >/dev/null 2>&1; then
        coverage=$(go tool cover -func=coverage.out | tail -1 | awk '{print $3}')
        info "Cobertura de testes: $coverage"
        rm -f coverage.out
    fi
}

# Função para validar dependências
validate_dependencies() {
    log "Validando dependências..."
    
    # Verificar se go.mod está correto
    if go mod verify; then
        success "Módulos verificados com sucesso"
    else
        error "Falha na verificação de módulos"
    fi
    
    # Verificar dependências não utilizadas
    if command -v go-mod-outdated >/dev/null 2>&1; then
        info "Verificando dependências desatualizadas..."
        go list -u -m all | grep '\[' || info "Todas as dependências estão atualizadas"
    fi
    
    # Verificar vulnerabilidades (se govulncheck estiver disponível)
    if command -v govulncheck >/dev/null 2>&1; then
        log "Verificando vulnerabilidades..."
        if govulncheck ./...; then
            success "Nenhuma vulnerabilidade encontrada"
        else
            warning "Vulnerabilidades encontradas"
        fi
    fi
}

# Função para validar funcionalidade básica
validate_basic_functionality() {
    log "Validando funcionalidade básica..."
    
    # Tentar executar aplicação por alguns segundos
    if [[ -f "build/wamex" ]]; then
        log "Testando execução da aplicação..."
        
        # Executar aplicação em background
        timeout 10s ./build/wamex &
        APP_PID=$!
        
        sleep 3
        
        # Verificar se aplicação está rodando
        if kill -0 $APP_PID 2>/dev/null; then
            success "Aplicação executou com sucesso"
            
            # Testar endpoint de health se disponível
            if curl -s http://localhost:8080/health >/dev/null 2>&1; then
                success "Endpoint /health respondeu"
            else
                warning "Endpoint /health não respondeu (pode ser normal)"
            fi
            
            # Parar aplicação
            kill $APP_PID 2>/dev/null || true
            wait $APP_PID 2>/dev/null || true
        else
            error "Aplicação não conseguiu executar"
        fi
    else
        warning "Binário não encontrado, pulando teste de funcionalidade"
    fi
}

# Função para validar performance
validate_performance() {
    log "Validando performance da compilação..."
    
    # Medir tempo de compilação
    start_time=$(date +%s)
    go build ./... >/dev/null 2>&1
    end_time=$(date +%s)
    
    compile_time=$((end_time - start_time))
    info "Tempo de compilação: ${compile_time}s"
    
    if [[ $compile_time -lt 30 ]]; then
        success "Tempo de compilação aceitável"
    else
        warning "Tempo de compilação alto: ${compile_time}s"
    fi
}

# Função para gerar relatório
generate_report() {
    log "Gerando relatório de validação..."
    
    echo ""
    echo "📊 RELATÓRIO DE VALIDAÇÃO DA MIGRAÇÃO"
    echo "======================================"
    echo ""
    echo "📈 Estatísticas:"
    echo "├── Testes Passaram: $TESTS_PASSED"
    echo "├── Testes Falharam: $TESTS_FAILED"
    echo "├── Avisos: $WARNINGS"
    echo "└── Total: $((TESTS_PASSED + TESTS_FAILED + WARNINGS))"
    echo ""
    
    if [[ $TESTS_FAILED -eq 0 ]]; then
        echo -e "${GREEN}🎉 MIGRAÇÃO VALIDADA COM SUCESSO!${NC}"
        echo "A nova estrutura idiomática está funcionando corretamente."
    else
        echo -e "${RED}⚠️  PROBLEMAS ENCONTRADOS NA MIGRAÇÃO${NC}"
        echo "Verifique os erros acima e corrija antes de prosseguir."
    fi
    
    echo ""
    echo "📋 Próximos passos recomendados:"
    echo "1. Revisar logs detalhados em: $LOG_FILE"
    echo "2. Executar testes manuais da aplicação"
    echo "3. Verificar funcionalidades específicas do negócio"
    echo "4. Atualizar documentação se necessário"
    
    if [[ $WARNINGS -gt 0 ]]; then
        echo "5. Revisar e resolver avisos encontrados"
    fi
}

# Função principal
main() {
    log "=== INICIANDO VALIDAÇÃO DA MIGRAÇÃO ==="
    
    cd "$PROJECT_ROOT"
    
    # Executar validações
    validate_directory_structure
    validate_essential_files
    validate_imports
    validate_compilation
    validate_tests
    validate_dependencies
    validate_basic_functionality
    validate_performance
    
    # Gerar relatório
    generate_report
    
    # Retornar código de saída apropriado
    if [[ $TESTS_FAILED -eq 0 ]]; then
        exit 0
    else
        exit 1
    fi
}

# Executar função principal
main "$@"
