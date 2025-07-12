# WAMEX - WhatsApp API 🚀

Sistema de API REST para integração com WhatsApp usando Go, PostgreSQL e whatsmeow.

## ✨ Funcionalidades

- 🔄 **Auto-reconexão** - Reconecta sessões automaticamente quando o servidor reinicia
- 📱 **QR Code no Terminal** - Exibe QR code diretamente no terminal para pareamento
- 💬 **Envio de Mensagens** - API REST para envio de mensagens de texto
- 🗄️ **PostgreSQL** - Persistência de sessões e configurações
- 📊 **Logs Estruturados** - Sistema de logging com contexto de sessão
- 🐳 **Docker Ready** - PostgreSQL via Docker Compose
- 🔒 **Sessões Múltiplas** - Suporte a múltiplas sessões WhatsApp simultâneas

## 🚀 Início Rápido

### Pré-requisitos
- Docker e Docker Compose
- Go 1.21+ (para desenvolvimento)

### 1. Configurar Banco de Dados

```bash
# Subir PostgreSQL com Docker
docker-compose up -d postgres

# Verificar se está rodando
docker-compose ps
```

### 2. Configurar Variáveis de Ambiente

Copie o arquivo `.env` e ajuste as configurações se necessário:
```bash
cp .env.example .env
```

### 3. Executar a Aplicação

```bash
# Compilar e executar
go run ./cmd/server

# Ou compilar primeiro
go build ./cmd/server
./server.exe
```

## 📋 Endpoints da API

### Sessões WhatsApp

| Método | Endpoint | Descrição |
|--------|----------|-----------|
| POST | `/sessions/add` | Criar nova sessão |
| GET | `/sessions/list` | Listar todas as sessões |
| GET | `/sessions/list/{session}` | Obter sessão específica |
| DELETE | `/sessions/del/{session}` | Remover sessão |
| POST | `/sessions/connect/{session}` | Conectar sessão |
| POST | `/sessions/disconnect/{session}` | Desconectar sessão |
| GET | `/sessions/status/{session}` | Status da sessão |
| GET | `/sessions/qr/{session}` | Obter QR Code |
| POST | `/sessions/pairphone/{session}` | Emparelhar telefone |

### Health Check
| Método | Endpoint | Descrição |
|--------|----------|-----------|
| GET | `/health` | Verificar status da API |

### Mensagens
| Método | Endpoint | Descrição |
|--------|----------|-----------|
| POST | `/sessions/send/{session}` | Enviar mensagem de texto |

## 📝 Exemplos de Uso

### 1. Criar uma nova sessão
```bash
curl -X POST http://localhost:8080/sessions/add \
  -H "Content-Type: application/json" \
  -d '{"session": "minha-sessao", "webhook_url": "https://webhook.site/test"}'
```

### 2. Conectar sessão (gera QR Code)
```bash
curl -X POST http://localhost:8080/sessions/connect/minha-sessao
```

### 3. Enviar mensagem de texto
```bash
curl -X POST http://localhost:8080/sessions/send/minha-sessao \
  -H "Content-Type: application/json" \
  -d '{"to": "5511999999999", "message": "Olá! Mensagem via WAMEX API 🚀"}'
```

### 4. Verificar status da sessão
```bash
curl -X GET http://localhost:8080/sessions/status/minha-sessao
```

## 🛠️ Tecnologias Utilizadas

- **[Go](https://golang.org/)** - Linguagem de programação
- **[whatsmeow](https://github.com/tulir/whatsmeow)** - Biblioteca WhatsApp Web API
- **[PostgreSQL](https://www.postgresql.org/)** - Banco de dados
- **[Bun ORM](https://bun.uptrace.dev/)** - ORM para Go
- **[Chi Router](https://github.com/go-chi/chi)** - Router HTTP
- **[Zerolog](https://github.com/rs/zerolog)** - Logging estruturado
- **[Docker](https://www.docker.com/)** - Containerização
- **[QR Terminal](https://github.com/mdp/qrterminal)** - QR Code no terminal

## 🔧 Desenvolvimento

### Estrutura do Projeto
```
wamex/
├── cmd/server/          # Aplicação principal
├── internal/
│   ├── domain/          # Entidades de negócio
│   ├── service/         # Lógica de negócio
│   ├── handler/         # Handlers HTTP
│   ├── repository/      # Acesso a dados
│   └── routes/          # Configuração de rotas
├── pkg/
│   └── logger/          # Sistema de logging
├── configs/             # Configurações
└── docker-compose.yml   # Docker services
```

### Comandos Úteis

```bash
# Parar todos os serviços
docker-compose down

# Parar e remover volumes (CUIDADO: apaga dados)
docker-compose down -v

# Ver logs do PostgreSQL
docker-compose logs postgres

# Acessar PostgreSQL
docker-compose exec postgres psql -U wamex -d wamex
```

## 🐳 Docker

O projeto inclui um `docker-compose.yml` que configura:
- PostgreSQL 15 com dados persistentes
- Configurações baseadas no arquivo `.env`
- Health checks automáticos

## 📝 Logs

O sistema usa logging estruturado com diferentes níveis:
- `DEBUG`: Informações detalhadas para desenvolvimento
- `INFO`: Informações gerais de operação
- `WARN`: Avisos que não impedem a operação
- `ERROR`: Erros que precisam de atenção
- `FATAL`: Erros críticos que param a aplicação

## 🔐 Configuração

Todas as configurações são feitas através de variáveis de ambiente no arquivo `.env`:

```env
# Servidor
PORT=8080
ENVIRONMENT=development

# Banco de Dados
DB_HOST=localhost
DB_PORT=5432
DB_USER=wamex
DB_PASSWORD=wamex123
DB_NAME=wamex

# Logging
LOG_LEVEL=info
LOG_FORMAT=json
```
