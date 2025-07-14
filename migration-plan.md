# 🏗️ Plano de Migração para Estrutura Idiomática Go - Wamex

> **Data de Criação**: 2025-01-14  
> **Status**: Pronto para Execução  
> **Versão**: 1.0  

## 📋 Índice

1. [Resumo Executivo](#resumo-executivo)
2. [Análise da Estrutura Atual](#análise-da-estrutura-atual)
3. [Pesquisa de Boas Práticas](#pesquisa-de-boas-práticas)
4. [Nova Estrutura Proposta](#nova-estrutura-proposta)
5. [Mapeamento de Dependências](#mapeamento-de-dependências)
6. [Plano de Migração Detalhado](#plano-de-migração-detalhado)
7. [Scripts de Automação](#scripts-de-automação)
8. [Validação e Rollback](#validação-e-rollback)
9. [Cronograma e Recursos](#cronograma-e-recursos)

---

## 🎯 Resumo Executivo

### Objetivo
Migrar a aplicação Wamex da estrutura atual para uma estrutura idiomática Go moderna, seguindo as melhores práticas da comunidade e mantendo a funcionalidade existente.

### Benefícios Esperados
- ✅ **Manutenibilidade**: Código mais organizado e fácil de encontrar
- ✅ **Testabilidade**: Separação clara facilita testes unitários e de integração  
- ✅ **Escalabilidade**: Estrutura preparada para crescimento
- ✅ **Padrões Go**: Alinhada com as melhores práticas da comunidade Go
- ✅ **Clean Architecture**: Mantém os princípios de arquitetura limpa
- ✅ **DDD**: Suporte a Domain-Driven Design

### Riscos Identificados
- 🔄 Atualização de imports em todos os arquivos
- 🔄 Possível quebra temporária durante migração
- 🔄 Necessidade de atualizar testes

### Mitigação de Riscos
- ✅ Migração incremental por fases
- ✅ Validação após cada etapa
- ✅ Scripts de rollback automático
- ✅ Backup completo antes da migração

---

## 🔍 Análise da Estrutura Atual

### Estrutura Atual Identificada

```
wamex/
├── cmd/
│   └── main.go                    # Entry point da aplicação
├── internal/
│   ├── domain/                    # Entidades e regras de negócio
│   │   ├── interfaces.go          # Interfaces principais
│   │   ├── message.go             # Estruturas de mensagens
│   │   ├── media.go               # Estruturas de mídia
│   │   └── session.go             # Entidades de sessão
│   ├── service/                   # Lógica de negócio
│   │   ├── whatsapp_service.go    # Serviço principal WhatsApp
│   │   ├── unified_service.go     # Serviço unificado de mídia
│   │   ├── media_processor.go     # Processamento de mídia
│   │   ├── media_service.go       # Serviços de mídia
│   │   ├── media_validator.go     # Validação de mídia
│   │   ├── security_validator.go  # Validação de segurança
│   │   └── type_detector.go       # Detecção de tipos
│   ├── handler/                   # HTTP handlers
│   │   ├── session_handler.go     # Handler de sessões
│   │   ├── message_handler.go     # Handler de mensagens
│   │   └── media_handler.go       # Handler de mídia
│   ├── repository/                # Data access
│   │   ├── session_repository.go  # Repositório de sessões
│   │   └── media_repository.go    # Repositório de mídia
│   ├── middleware/                # HTTP middleware
│   │   └── session_resolver.go    # Resolução de sessões
│   ├── routes/                    # Route configuration
│   │   └── routes.go              # Configuração de rotas
│   └── infra/                     # Infrastructure
│       ├── infra.go               # Orquestrador
│       ├── database.go            # Abstração DB
│       └── storage.go             # Abstração Storage
├── pkg/                           # Shared utilities
│   ├── logger/                    # Logging system
│   ├── storage/                   # MinIO client
│   ├── utils/                     # Utilities
│   └── validator/                 # Validators
├── configs/
│   └── config.go                  # Configurações
└── tests/                         # Testes
```

### Pontos Positivos Identificados
- ✅ Já segue Clean Architecture
- ✅ Usa `internal/` para código privado
- ✅ Separação clara entre domínio, serviços, handlers
- ✅ Usa `cmd/` para entry points
- ✅ Tem `pkg/` para utilitários reutilizáveis

### Pontos de Melhoria Identificados
- 🔄 Estrutura pode ser mais idiomática
- 🔄 Separação de use cases pode ser melhorada
- 🔄 Organização de testes pode ser otimizada
- 🔄 Configurações podem ser movidas para infra

---

## 📚 Pesquisa de Boas Práticas

### Fontes Consultadas

1. **golang-standards/project-layout** (Padrão de Facto)
   - Estrutura mais aceita pela comunidade Go
   - Foco em separação clara entre código público (`pkg/`) e privado (`internal/`)
   - Uso de `cmd/` para múltiplos binários

2. **Google Go Style Guide** (2024)
   - Ênfase em nomes de pacotes claros e concisos
   - Evitar pacotes "util" genéricos
   - Preferir estruturas que facilitem testes

3. **Tendências Modernas 2024/2025**
   - **Clean Architecture** continua sendo preferida
   - **Domain-Driven Design** para projetos complexos
   - **Nomes curtos** mas descritivos para diretórios
   - **Modularidade** e **testabilidade** como prioridades

### Princípios Aplicados

- **Dependency Rule**: Dependências apontam sempre para dentro (domain)
- **Single Responsibility**: Cada pacote tem uma responsabilidade clara
- **Interface Segregation**: Interfaces pequenas e específicas
- **Dependency Inversion**: Depender de abstrações, não implementações

---

## 🏗️ Nova Estrutura Proposta

### Estrutura Completa

```
wamex/
├── cmd/
│   └── wamex/                     # Nome específico da aplicação
│       └── main.go                # Entry point principal
├── internal/
│   ├── app/                       # Configuração e inicialização da aplicação
│   │   ├── app.go                 # Estrutura principal da aplicação
│   │   └── wire.go                # Dependency injection (se usar wire)
│   ├── domain/                    # Entidades e regras de negócio (core)
│   │   ├── entity/                # Entidades do domínio
│   │   │   ├── session.go
│   │   │   ├── message.go
│   │   │   └── media.go
│   │   ├── service/               # Interfaces de serviços (contratos)
│   │   │   ├── whatsapp.go
│   │   │   ├── media.go
│   │   │   └── session.go
│   │   └── repository/            # Interfaces de repositórios
│   │       ├── session.go
│   │       └── media.go
│   ├── usecase/                   # Casos de uso (application layer)
│   │   ├── whatsapp/
│   │   │   ├── send_message.go
│   │   │   ├── manage_session.go
│   │   │   └── process_media.go
│   │   └── media/
│   │       ├── upload.go
│   │       ├── validate.go
│   │       └── transform.go
│   ├── infra/                     # Infraestrutura (implementações)
│   │   ├── database/              # Implementações de repositório
│   │   │   ├── postgres/
│   │   │   │   ├── session.go
│   │   │   │   └── media.go
│   │   │   └── migration/
│   │   ├── storage/               # Storage externo (MinIO, S3, etc.)
│   │   │   └── minio.go
│   │   ├── whatsapp/              # Implementação do cliente WhatsApp
│   │   │   ├── client.go
│   │   │   └── handler.go
│   │   └── config/                # Configurações
│   │       └── config.go
│   ├── transport/                 # Camada de transporte
│   │   ├── http/                  # HTTP handlers
│   │   │   ├── handler/
│   │   │   │   ├── session.go
│   │   │   │   ├── message.go
│   │   │   │   └── media.go
│   │   │   ├── middleware/
│   │   │   │   ├── cors.go
│   │   │   │   ├── auth.go
│   │   │   │   └── session.go
│   │   │   └── router/
│   │   │       └── router.go
│   │   └── grpc/                  # gRPC handlers (futuro)
│   └── shared/                    # Código compartilhado interno
│       ├── errors/
│       ├── validator/
│       └── converter/
├── pkg/                           # Bibliotecas públicas reutilizáveis
│   ├── logger/                    # Sistema de logging
│   ├── crypto/                    # Utilitários de criptografia
│   └── validator/                 # Validadores genéricos
├── api/                           # Definições de API
│   ├── openapi/
│   │   └── spec.yaml
│   └── proto/                     # Protocol Buffers (futuro)
├── web/                           # Assets web (se houver frontend)
├── scripts/                       # Scripts de build, deploy, etc.
├── deployments/                   # Configurações de deployment
├── test/                          # Testes de integração e dados de teste
├── docs/                          # Documentação
├── tools/                         # Ferramentas de desenvolvimento
├── configs/                       # Configurações de ambiente
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

### Principais Melhorias

1. **Separação mais clara de responsabilidades**
2. **Estrutura mais idiomática**
3. **Melhor organização de testes**
4. **Preparação para crescimento**

---

## 🗺️ Mapeamento de Dependências

### Análise de Dependências Circulares
✅ **NENHUMA DEPENDÊNCIA CIRCULAR DETECTADA**

### Fluxo de Dependências Atual
```
cmd/main.go → configs → infra → repository ← service ← handler ← routes
                ↓         ↓        ↓         ↓         ↓
              pkg/    domain/   domain/   domain/   middleware/
```

### Pontos de Maior Acoplamento Identificados

1. **Handler Layer com Multiple Dependencies**
   - Handlers importam: `domain`, `repository`, `service`, `pkg/storage`

2. **Service Layer Interdependencies**
   - `unified_service.go` agrega múltiplos services internos
   - Services dependem diretamente de `repository`

3. **Infrastructure Layer Centralization**
   - `infra/infra.go` orquestra tudo
   - Ponto único de falha potencial

---

## 📋 Plano de Migração Detalhado

### Estratégia de Migração
- **Princípio**: Cada etapa deve compilar e funcionar antes de prosseguir
- **Abordagem**: Migração incremental com validação contínua
- **Tempo Total Estimado**: 6-8 horas

### Fases da Migração

#### **FASE 1: Preparação e Estrutura Base** (30-45 min)
- Backup e validação inicial
- Criação da nova estrutura de diretórios
- Validação da estrutura

#### **FASE 2: Migração do Domain Layer** (45-60 min)
- Migração de entidades
- Separação de interfaces por contexto
- Atualização de imports

#### **FASE 3: Migração da Camada de Infraestrutura** (60-90 min)
- Reorganização de database
- Reorganização de storage
- Criação de config infrastructure

#### **FASE 4: Migração da Camada de Use Cases** (90-120 min)
- Criação de use cases de WhatsApp
- Criação de use cases de media
- Refatoração de services

#### **FASE 5: Migração da Camada de Transport** (60-90 min)
- Reorganização de handlers
- Reorganização de middleware
- Reorganização de routes

#### **FASE 6: Migração da Camada de Repository** (45-60 min)
- Manutenção de repositories
- Atualização de interfaces

#### **FASE 7: Reorganização de Shared e PKG** (30-45 min)
- Separação de PKG e Shared
- Atualização de imports

#### **FASE 8: Migração do Entry Point** (30-45 min)
- Reorganização de CMD
- Criação de App Layer

#### **FASE 9: Migração de Testes** (45-60 min)
- Reorganização de testes
- Atualização de imports

#### **FASE 10: Limpeza e Validação Final** (30-45 min)
- Remoção de diretórios antigos
- Validação completa
- Commit da nova estrutura

---

## 🔧 Scripts de Automação

### Scripts Criados

1. **`scripts/migration-master.sh`**
   - Script principal que orquestra toda a migração
   - Executa todas as fases ou fases específicas
   - Inclui validação e logging

2. **`scripts/phase-1-setup.sh`**
   - Cria toda a estrutura de diretórios
   - Configura arquivos base (Makefile, .air.toml)
   - Valida estrutura criada

3. **`scripts/rollback.sh`**
   - Permite rollback completo da migração
   - Suporte a backup via git e backup físico
   - Validação pós-rollback

4. **`scripts/validate-migration.sh`**
   - Validação completa da migração
   - Testa compilação, testes, estrutura
   - Gera relatório detalhado

### Como Usar os Scripts

```bash
# Executar migração completa
./scripts/migration-master.sh all

# Executar fase específica
./scripts/migration-master.sh 1

# Validar migração
./scripts/validate-migration.sh --detailed

# Fazer rollback se necessário
./scripts/rollback.sh latest
```

---

## ✅ Validação e Rollback

### Critérios de Sucesso
- ✅ Aplicação compila sem erros
- ✅ Todos os testes passam
- ✅ Aplicação executa normalmente
- ✅ Endpoints respondem corretamente
- ✅ Estrutura segue padrões idiomáticos Go
- ✅ Clean Architecture mantida
- ✅ Nenhuma dependência circular introduzida

### Processo de Rollback
1. Backup automático antes da migração
2. Tags git para pontos de restauração
3. Script de rollback automatizado
4. Validação pós-rollback

### Pontos de Atenção
- **Imports**: Cada mudança requer atualização
- **Interfaces**: Manter compatibilidade
- **Testes**: Validar após cada fase
- **Dependências**: Verificar go.mod
- **Build**: Compilar após cada fase

---

## 📅 Cronograma e Recursos

### Cronograma Estimado
- **Preparação**: 30 minutos
- **Execução**: 6-7 horas
- **Validação**: 1 hora
- **Total**: 7-8 horas

### Recursos Necessários
- Desenvolvedor experiente em Go
- Acesso ao repositório
- Ambiente de desenvolvimento configurado
- Backup do estado atual

### Próximos Passos
1. Revisar este plano com a equipe
2. Agendar janela de manutenção
3. Executar migração usando os scripts
4. Validar funcionamento completo
5. Atualizar documentação

---

## 🚀 Execução

Para executar a migração, use:

```bash
# 1. Revisar o plano
cat migration-plan.md

# 2. Executar migração
./scripts/migration-master.sh all

# 3. Validar resultado
./scripts/validate-migration.sh --detailed

# 4. Em caso de problemas
./scripts/rollback.sh latest
```

---

## 📋 Tarefas Detalhadas Criadas

As seguintes tarefas foram criadas no sistema de gerenciamento, ordenadas por dependências:

### 📦 SAÍDA: Migração para Estrutura Idiomática Go - Wamex
```
├── [SETUP] Preparar ambiente e criar backup
├── [SETUP] Criar nova estrutura de diretórios
├── [FEAT] Migrar entidades do domain
├── [FEAT] Separar interfaces por contexto
├── [REFACTOR] Atualizar imports do domain layer
├── [FEAT] Migrar configurações para infra
├── [FEAT] Reorganizar database infrastructure
├── [FEAT] Reorganizar storage infrastructure
├── [REFACTOR] Atualizar imports da infraestrutura
├── [FEAT] Criar use cases de WhatsApp
├── [FEAT] Criar use cases de media
├── [REFACTOR] Refatorar services para use cases
├── [FEAT] Migrar handlers para transport
├── [FEAT] Migrar middleware para transport
├── [FEAT] Migrar routes para transport
├── [REFACTOR] Atualizar imports da transport layer
├── [FEAT] Atualizar interfaces dos repositories
├── [FEAT] Separar utilitários PKG e Shared
├── [REFACTOR] Atualizar imports de shared e pkg
├── [FEAT] Reorganizar entry point
├── [FEAT] Simplificar main.go
├── [FEAT] Reorganizar testes de integração
├── [REFACTOR] Atualizar imports dos testes
├── [SETUP] Remover diretórios antigos
├── [TEST] Validação completa da migração
└── [DOCS] Commit da nova estrutura
```

**Regra**: Cada subtarefa entrega um pedaço de valor testável e deve compilar antes de prosseguir para a próxima.

### 🎯 Critérios de Sucesso por Fase
- **SETUP**: Estrutura criada e backup realizado
- **FEAT**: Funcionalidade migrada e compilando
- **REFACTOR**: Imports atualizados e sem erros
- **TEST**: Todos os testes passando
- **DOCS**: Documentação atualizada

---

**Documento criado em**: 2025-01-14
**Última atualização**: 2025-01-14
**Versão**: 1.0
**Status**: ✅ Pronto para Execução
