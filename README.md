
# ZapCore - WhatsApp API

API REST para integração com WhatsApp usando Clean Architecture em Go.

## 🚀 Funcionalidades

### Gerenciamento de Sessões
- ✅ Criar nova sessão
- ✅ Listar sessões ativas
- ✅ Conectar/desconectar sessão
- ✅ Obter status da sessão
- ✅ Gerar QR Code para autenticação
- ✅ Emparelhar telefone
- ✅ Configurar proxy

### Envio de Mensagens
- ✅ Mensagens de texto
- ✅ Imagens, áudios, vídeos
- ✅ Documentos e stickers
- ✅ Localização e contatos
- ✅ Botões interativos
- ✅ Listas interativas
- ✅ Enquetes
- ✅ Edição de mensagens

## 🏗️ Arquitetura

Projeto estruturado seguindo **Clean Architecture**:

```
├── internal/
│   ├── domain/          # Entidades e regras de negócio
│   ├── usecases/        # Casos de uso
│   ├── infra/           # Implementações externas
│   ├── interfaces/      # Controllers HTTP
│   └── app/             # Configuração da aplicação
└── pkg/                 # Bibliotecas públicas
```

## 🛠️ Tecnologias

- **Go 1.21+**
- **Gin** - Framework HTTP
- **PostgreSQL** - Banco de dados
- **WhatsApp Web Multi-Device** - Protocolo WhatsApp
- **Docker** - Containerização

## 📋 Pré-requisitos

- Go 1.21+
- PostgreSQL 13+
- Docker (opcional)

## 🚀 Instalação

1. Clone o repositório:
```bash
git clone https://github.com/felipe/zapcore.git
cd zapcore
```

2. Configure as variáveis de ambiente:
```bash
cp .env.example .env
# Edite o arquivo .env com suas configurações
```

3. Execute as migrações do banco:
```bash
go run cmd/migrate/main.go up
```

4. Inicie a aplicação:
```bash
go run cmd/server/main.go
```

## 📚 Documentação da API

### Sessões

| Método | Endpoint | Descrição |
|--------|----------|-----------|
| POST | `/sessions/add` | Criar nova sessão |
| GET | `/sessions/list` | Listar sessões |
| GET | `/sessions/{id}` | Obter sessão |
| DELETE | `/sessions/{id}` | Remover sessão |
| POST | `/sessions/{id}/connect` | Conectar sessão |
| POST | `/sessions/{id}/logout` | Desconectar sessão |
| GET | `/sessions/{id}/status` | Status da sessão |
| GET | `/sessions/{id}/qr` | Gerar QR Code |

### Mensagens

| Método | Endpoint | Descrição |
|--------|----------|-----------|
| POST | `/messages/{sessionID}/send/text` | Enviar texto |
| POST | `/messages/{sessionID}/send/image` | Enviar imagem |
| POST | `/messages/{sessionID}/send/audio` | Enviar áudio |
| POST | `/messages/{sessionID}/send/video` | Enviar vídeo |
| POST | `/messages/{sessionID}/send/document` | Enviar documento |

## 🐳 Docker

```bash
# Build da imagem
docker build -t zapcore .

# Executar com docker-compose
docker-compose up -d
```

## 🧪 Testes

```bash
# Executar todos os testes
go test ./...

# Executar testes com coverage
go test -cover ./...
```

## 📄 Licença

Este projeto está sob a licença MIT. Veja o arquivo [LICENSE](LICENSE) para mais detalhes.
