# 🏗️ Plano de Implementação - Funcionalidades Wuzapi para WAMEX

## Contexto

### Problema
O projeto WAMEX possui uma base sólida de funcionalidades WhatsApp, mas carece de recursos avançados presentes no projeto Wuzapi, como gerenciamento de usuários, grupos, webhooks e funcionalidades de chat avançadas.

### Objetivo
Implementar funcionalidades do Wuzapi no WAMEX mantendo a arquitetura Clean Architecture existente, padrões de nomenclatura e estrutura organizacional já estabelecidos.

### Restrições
- **Arquitetura**: Manter Clean Architecture (Domain → UseCase → Infrastructure → Transport)
- **Padrões**: Seguir convenções WAMEX existentes
- **Compatibilidade**: Não quebrar APIs existentes
- **Qualidade**: Cada fase deve compilar e funcionar antes da próxima
- **Testes**: Implementar testes unitários para cada componente

## Análise Comparativa

### ✅ Funcionalidades já implementadas no WAMEX
- Session management (connect, disconnect, status, qr, pairphone)
- Basic messaging (text, image, audio, document, video)
- Advanced messaging (sticker, location, contact, react, poll, list, edit)
- Media management (upload, download, list, delete)
- Middleware de resolução de sessão
- Logging estruturado
- Tratamento de erros padronizado

### 🆕 Funcionalidades do Wuzapi para implementar
- User management (check, info, avatar, contacts, presence)
- Chat management (delete, markread, presence)
- Group management (create, list, info, participants, etc.)
- Webhook management
- Advanced features (proxy, S3, newsletter)

## Alternativas Avaliadas

### Opção 1: Implementação Monolítica
- **Prós**: Implementação rápida, menos complexidade inicial
- **Contras**: Viola princípios SOLID, dificulta manutenção futura
- **Complexidade**: Baixa

### Opção 2: Implementação por Módulos Separados
- **Prós**: Separação clara de responsabilidades, facilita testes
- **Contras**: Maior complexidade inicial, mais arquivos
- **Complexidade**: Média

### Opção 3: Implementação Incremental seguindo Clean Architecture
- **Prós**: Mantém padrões existentes, facilita evolução, testável
- **Contras**: Mais tempo de implementação inicial
- **Complexidade**: Média-Alta

## Recomendação

**Solução escolhida**: Opção 3 - Implementação Incremental seguindo Clean Architecture

**Justificativa**:
- Mantém consistência com arquitetura existente
- Facilita manutenção e evolução futura
- Permite implementação e validação por fases
- Reutiliza infraestrutura existente (middleware, logging, etc.)
- Segue princípios SOLID e dependency rule

## Plano de Implementação

### 🎯 Fase 1: User Management (Prioridade Alta)
**Duração estimada**: 2-3 dias
**Objetivo**: Implementar funcionalidades básicas de gerenciamento de usuários

#### Funcionalidades
```
POST /user/{sessionID}/check     - Verificar se número está no WhatsApp
POST /user/{sessionID}/info      - Obter informações detalhadas do usuário
POST /user/{sessionID}/avatar    - Obter avatar do usuário
GET  /user/{sessionID}/contacts  - Listar contatos da sessão
POST /user/{sessionID}/presence  - Definir status de presença
```

#### Estrutura de Implementação
1. **Domain Layer**
   - `internal/domain/entity/user.go` - Entidades UserInfo, Contact, UserPresence
   - `internal/domain/service/user.go` - Interface UserService

2. **UseCase Layer**
   - `internal/usecase/user/check_user.go`
   - `internal/usecase/user/get_user_info.go`
   - `internal/usecase/user/get_user_avatar.go`
   - `internal/usecase/user/get_contacts.go`
   - `internal/usecase/user/set_presence.go`

3. **Infrastructure Layer**
   - Extensão de `internal/infra/whatsapp/whatsapp_service.go`
   - Implementação dos métodos UserService

4. **Transport Layer**
   - `internal/transport/http/handler/user.go`
   - Extensão de `internal/transport/http/router/router.go`

#### Critérios de Aceitação
- [ ] Todas as rotas respondem corretamente
- [ ] Validação de entrada implementada
- [ ] Tratamento de erros padronizado
- [ ] Logging estruturado
- [ ] Testes unitários implementados
- [ ] Build compila sem erros

### 🎯 Fase 2: Chat Management (Prioridade Alta)
**Duração estimada**: 2 dias
**Objetivo**: Implementar funcionalidades avançadas de chat

#### Funcionalidades
```
POST /chat/{sessionID}/delete     - Deletar mensagens enviadas
POST /chat/{sessionID}/markread   - Marcar mensagens como lidas
POST /chat/{sessionID}/presence   - Definir presença em chat (typing, recording)
```

#### Estrutura de Implementação
1. **Domain Layer**
   - `internal/domain/entity/chat.go` - Entidades ChatPresence, DeleteMessage
   - Extensão de interfaces existentes

2. **UseCase Layer**
   - `internal/usecase/chat/delete_message.go`
   - `internal/usecase/chat/mark_read.go`
   - `internal/usecase/chat/set_chat_presence.go`

3. **Infrastructure Layer**
   - Extensão do WhatsApp service existente

4. **Transport Layer**
   - `internal/transport/http/handler/chat.go`
   - Extensão do router

### 🎯 Fase 3: Group Management (Prioridade Média)
**Duração estimada**: 4-5 dias
**Objetivo**: Implementar gerenciamento completo de grupos

#### Funcionalidades
```
POST /group/{sessionID}/create           - Criar grupos
GET  /group/{sessionID}/list             - Listar grupos
POST /group/{sessionID}/info             - Obter informações do grupo
GET  /group/{sessionID}/invitelink       - Gerar/obter link de convite
POST /group/{sessionID}/photo            - Definir foto do grupo
POST /group/{sessionID}/leave            - Sair do grupo
POST /group/{sessionID}/join             - Entrar no grupo via link
POST /group/{sessionID}/participants     - Gerenciar participantes
```

#### Estrutura de Implementação
1. **Domain Layer**
   - `internal/domain/entity/group.go`
   - `internal/domain/service/group.go`

2. **UseCase Layer**
   - `internal/usecase/group/` - Múltiplos use cases

3. **Infrastructure Layer**
   - Implementação GroupService

4. **Transport Layer**
   - `internal/transport/http/handler/group.go`

### 🎯 Fase 4: Webhook Management (Prioridade Média)
**Duração estimada**: 2-3 dias
**Objetivo**: Implementar sistema de webhooks

#### Funcionalidades
```
POST   /webhook/{sessionID}        - Configurar webhook
GET    /webhook/{sessionID}        - Obter configuração webhook
PUT    /webhook/{sessionID}        - Atualizar webhook
DELETE /webhook/{sessionID}        - Remover webhook
```

### 🎯 Fase 5: Advanced Features (Prioridade Baixa)
**Duração estimada**: 3-4 dias
**Objetivo**: Implementar funcionalidades avançadas

#### Funcionalidades
- Configuração de proxy
- Integração S3
- Gerenciamento de newsletters
- Downloads específicos por tipo de mídia

## Riscos e Mitigações

### Risco 1: Complexidade da API WhatsApp
**Descrição**: Algumas funcionalidades podem não estar disponíveis na versão atual do whatsmeow
**Mitigação**: 
- Verificar documentação do whatsmeow antes da implementação
- Implementar fallbacks ou mensagens de "não implementado"
- Manter interfaces preparadas para implementação futura

### Risco 2: Breaking Changes
**Descrição**: Alterações podem quebrar funcionalidades existentes
**Mitigação**:
- Implementar testes de regressão
- Validar build a cada fase
- Manter versionamento de API

### Risco 3: Performance
**Descrição**: Novas funcionalidades podem impactar performance
**Mitigação**:
- Implementar cache onde apropriado
- Monitorar métricas de performance
- Implementar rate limiting

### Risco 4: Segurança
**Descrição**: Novas endpoints podem introduzir vulnerabilidades
**Mitigação**:
- Validação rigorosa de entrada
- Autenticação/autorização adequada
- Logging de segurança

## Critérios de Sucesso

### Técnicos
- [ ] Todas as funcionalidades implementadas conforme especificação
- [ ] Cobertura de testes > 80%
- [ ] Build pipeline verde
- [ ] Performance mantida ou melhorada
- [ ] Zero breaking changes

### Arquiteturais
- [ ] Clean Architecture mantida
- [ ] Padrões WAMEX seguidos
- [ ] Dependency rule respeitada
- [ ] Interfaces bem definidas
- [ ] Separação de responsabilidades clara

### Operacionais
- [ ] Documentação atualizada
- [ ] Logs estruturados implementados
- [ ] Métricas de monitoramento
- [ ] Tratamento de erros padronizado

## Próximos Passos

1. **Aprovação do plano** pela equipe
2. **Setup do ambiente** de desenvolvimento
3. **Implementação Fase 1** (User Management)
4. **Validação e testes** da Fase 1
5. **Iteração** para próximas fases

## Diagramas de Arquitetura

### Fluxo de Implementação - Fase 1 (User Management)

```mermaid
graph TD
    A[HTTP Request] --> B[UserHandler]
    B --> C[UserUseCase]
    C --> D[WhatsAppService]
    D --> E[whatsmeow Client]
    E --> F[WhatsApp API]

    C --> G[SessionRepository]
    G --> H[PostgreSQL]

    B --> I[Response JSON]

    subgraph "Transport Layer"
        B
    end

    subgraph "UseCase Layer"
        C
    end

    subgraph "Infrastructure Layer"
        D
        G
        H
        E
    end

    subgraph "External"
        F
    end
```

### Estrutura de Diretórios - Após Implementação

```
internal/
├── domain/
│   ├── entity/
│   │   ├── session.go
│   │   ├── message.go
│   │   ├── media.go
│   │   ├── user.go          # 🆕 Fase 1
│   │   ├── chat.go          # 🆕 Fase 2
│   │   ├── group.go         # 🆕 Fase 3
│   │   └── webhook.go       # 🆕 Fase 4
│   ├── service/
│   │   ├── whatsapp.go
│   │   ├── media.go
│   │   ├── user.go          # 🆕 Fase 1
│   │   ├── chat.go          # 🆕 Fase 2
│   │   ├── group.go         # 🆕 Fase 3
│   │   └── webhook.go       # 🆕 Fase 4
│   └── repository/
│       ├── session.go
│       ├── media.go
│       └── webhook.go       # 🆕 Fase 4
├── usecase/
│   ├── whatsapp/
│   ├── media/
│   ├── user/               # 🆕 Fase 1
│   ├── chat/               # 🆕 Fase 2
│   ├── group/              # 🆕 Fase 3
│   └── webhook/            # 🆕 Fase 4
├── infra/
│   ├── whatsapp/
│   │   ├── whatsapp_service.go
│   │   ├── user_service.go  # 🆕 Fase 1
│   │   ├── chat_service.go  # 🆕 Fase 2
│   │   └── group_service.go # 🆕 Fase 3
│   └── database/
└── transport/
    └── http/
        ├── handler/
        │   ├── session.go
        │   ├── message.go
        │   ├── media.go
        │   ├── user.go      # 🆕 Fase 1
        │   ├── chat.go      # 🆕 Fase 2
        │   ├── group.go     # 🆕 Fase 3
        │   └── webhook.go   # 🆕 Fase 4
        └── router/
            └── router.go
```

## Especificações Técnicas Detalhadas

### Fase 1: User Management - Especificação de APIs

#### POST /user/{sessionID}/check
```json
// Request
{
  "phone": "5511999999999"
}

// Response
{
  "success": true,
  "message": "User check completed",
  "phone": "5511999999999",
  "is_on_whatsapp": true,
  "jid": "5511999999999@s.whatsapp.net",
  "timestamp": "2025-01-14T10:30:00Z"
}
```

#### POST /user/{sessionID}/info
```json
// Request
{
  "phone": "5511999999999"
}

// Response
{
  "success": true,
  "message": "User info retrieved",
  "user": {
    "jid": "5511999999999@s.whatsapp.net",
    "phone_number": "5511999999999",
    "name": "João Silva",
    "business_name": "",
    "is_on_whatsapp": true,
    "is_business": false,
    "status": "Disponível",
    "avatar": "https://..."
  },
  "timestamp": "2025-01-14T10:30:00Z"
}
```

#### GET /user/{sessionID}/contacts
```json
// Response
{
  "success": true,
  "message": "Contacts retrieved",
  "contacts": [
    {
      "jid": "5511999999999@s.whatsapp.net",
      "name": "João Silva",
      "phone_number": "5511999999999",
      "is_business": false,
      "is_my_contact": true,
      "avatar": "https://..."
    }
  ],
  "total": 150,
  "timestamp": "2025-01-14T10:30:00Z"
}
```

### Padrões de Implementação

#### Estrutura de UseCase
```go
type CheckUserUseCase struct {
    sessionRepo domainRepo.SessionRepository
    userService domainService.UserService
}

func NewCheckUserUseCase(
    sessionRepo domainRepo.SessionRepository,
    userService domainService.UserService,
) *CheckUserUseCase {
    return &CheckUserUseCase{
        sessionRepo: sessionRepo,
        userService: userService,
    }
}

func (uc *CheckUserUseCase) Execute(sessionName, phone string) (*entity.UserInfo, error) {
    // 1. Validar sessão
    session, err := uc.sessionRepo.GetBySession(sessionName)
    if err != nil {
        return nil, fmt.Errorf("failed to get session: %w", err)
    }

    if session.Status != entity.StatusConnected {
        return nil, fmt.Errorf("session %s is not connected", sessionName)
    }

    // 2. Verificar usuário
    userInfo, err := uc.userService.CheckUser(sessionName, phone)
    if err != nil {
        return nil, fmt.Errorf("failed to check user: %w", err)
    }

    return userInfo, nil
}
```

#### Estrutura de Handler
```go
func (h *UserHandler) CheckUser(w http.ResponseWriter, r *http.Request) {
    sessionID := chi.URLParam(r, "sessionID")

    var req entity.UserCheckRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        h.writeErrorResponse(w, http.StatusBadRequest, "Invalid request body", err)
        return
    }

    // Validação
    if req.Phone == "" {
        h.writeErrorResponse(w, http.StatusBadRequest, "Phone is required", nil)
        return
    }

    // Executar use case
    userInfo, err := h.checkUserUseCase.Execute(sessionID, req.Phone)
    if err != nil {
        h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to check user", err)
        return
    }

    // Resposta
    response := entity.UserCheckResponse{
        Success:      true,
        Message:      "User check completed",
        Phone:        req.Phone,
        IsOnWhatsApp: userInfo.IsOnWhatsApp,
        JID:          userInfo.JID,
        Timestamp:    time.Now(),
    }

    h.writeJSONResponse(w, http.StatusOK, response)
}
```

## Cronograma Detalhado

### Semana 1: Fase 1 - User Management
- **Dia 1**: Domain entities e interfaces
- **Dia 2**: Use cases e testes unitários
- **Dia 3**: Infrastructure implementation
- **Dia 4**: Transport layer e integração
- **Dia 5**: Testes de integração e documentação

### Semana 2: Fase 2 - Chat Management
- **Dia 1-2**: Implementação completa
- **Dia 3**: Testes e validação
- **Dia 4-5**: Fase 3 início (Group Management)

### Semana 3-4: Fase 3 - Group Management
- Implementação completa de todas as funcionalidades de grupo

### Semana 5: Fase 4 - Webhook Management
- Sistema completo de webhooks

### Semana 6: Fase 5 - Advanced Features
- Funcionalidades avançadas e polimento

---

**Observação**: Este plano segue os princípios arquiteturais do WAMEX e pode ser ajustado conforme necessidades específicas ou descobertas durante a implementação.
