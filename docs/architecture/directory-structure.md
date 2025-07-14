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
