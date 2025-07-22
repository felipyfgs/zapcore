# 📱 ZapCore - WhatsApp API

Uma API REST moderna e robusta para integração com WhatsApp usando Clean Architecture em Go.

## ✨ Características

- 🚀 **Clean Architecture** - Código organizado e manutenível
- 📱 **WhatsApp Multi-Device** - Protocolo oficial do WhatsApp
- 🔄 **Múltiplas Sessões** - Gerencie várias contas simultaneamente
- 📎 **Envio de Mídia** - Suporte completo para documentos, imagens, vídeos e áudios
- 🔐 **Autenticação** - API Key para segurança
- 📊 **Logs Detalhados** - Monitoramento completo
- 🐳 **Docker Ready** - Containerização incluída

## 🚀 Funcionalidades

### 📞 Gerenciamento de Sessões
- ✅ Criar e gerenciar sessões
- ✅ Conectar/desconectar WhatsApp
- ✅ Gerar QR Code para autenticação
- ✅ Verificar status de conexão
- ✅ Listar sessões ativas

### 💬 Envio de Mensagens
- ✅ **Texto** - Mensagens simples e com reply
- ✅ **Documentos** - PDF, DOC, XLSX, etc.
- ✅ **Imagens** - JPG, PNG, GIF, etc.
- ✅ **Vídeos** - MP4, AVI, MOV, etc.
- ✅ **Áudios** - MP3, WAV, OGG, etc.

### 📤 Formatos de Envio
- 📁 **Upload direto** - Form-data multipart
- 🌐 **URL pública** - Links externos
- 📋 **Base64** - Dados codificados

## 🛠️ Tecnologias

- **Go 1.23+** - Linguagem principal
- **Gin** - Framework HTTP
- **Bun ORM** - Banco de dados
- **PostgreSQL** - Armazenamento
- **WhatsApp Multi-Device** - Protocolo oficial
- **MinIO** - Storage de mídia
- **Docker** - Containerização

## 📋 Pré-requisitos

- Go 1.23 ou superior
- PostgreSQL 13+
- Docker e Docker Compose (opcional)

## 🚀 Instalação

### 1. Clone o repositório
```bash
git clone https://github.com/felipyfgs/zapcore.git
cd zapcore
```

### 2. Configure as variáveis de ambiente
```bash
cp .env.example .env
# Edite o arquivo .env com suas configurações
```

### 3. Execute com Docker (Recomendado)
```bash
docker-compose up -d
```

### 4. Ou execute manualmente
```bash
# Instale as dependências
go mod download

# Execute a aplicação
go run cmd/server/main.go
```

## 📚 Uso Rápido

### Criar uma sessão
```bash
curl -X POST "http://localhost:8080/sessions" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-api-key" \
  -d '{"name": "Minha Sessão"}'
```

### Enviar mensagem de texto
```bash
curl -X POST "http://localhost:8080/messages/{sessionID}/send/text" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-api-key" \
  -d '{
    "to": "5511999999999@s.whatsapp.net",
    "text": "Olá! Esta é uma mensagem de teste."
  }'
```

### Enviar imagem
```bash
curl -X POST "http://localhost:8080/messages/{sessionID}/send/image" \
  -H "X-API-Key: your-api-key" \
  -F "to=5511999999999@s.whatsapp.net" \
  -F "caption=🖼️ Imagem de exemplo" \
  -F "media=@/caminho/para/imagem.jpg"
```

## 📖 Documentação Completa

Para documentação detalhada da API, consulte:
- 📄 [API.md](API.md) - Documentação completa dos endpoints
- 🌐 **Swagger** - `http://localhost:8080/docs` (quando rodando)

## 🏗️ Arquitetura

```
zapcore/
├── cmd/                    # Pontos de entrada da aplicação
│   └── server/            # Servidor HTTP
├── internal/              # Código interno da aplicação
│   ├── app/              # Configuração da aplicação
│   ├── domain/           # Entidades e regras de negócio
│   │   ├── chat/         # Domínio de chats
│   │   ├── contact/      # Domínio de contatos
│   │   ├── message/      # Domínio de mensagens
│   │   ├── session/      # Domínio de sessões
│   │   └── webhook/      # Domínio de webhooks
│   ├── http/             # Camada HTTP
│   │   ├── handlers/     # Controladores
│   │   ├── middleware/   # Middlewares
│   │   └── router/       # Roteamento
│   ├── infra/            # Infraestrutura
│   │   ├── database/     # Banco de dados
│   │   ├── repository/   # Repositórios
│   │   ├── storage/      # Armazenamento
│   │   └── whatsapp/     # Cliente WhatsApp
│   ├── shared/           # Código compartilhado
│   └── usecases/         # Casos de uso
├── pkg/                   # Bibliotecas públicas
└── assets/               # Arquivos de exemplo
```

## 🔧 Configuração

### Variáveis de Ambiente
```env
# Banco de dados
DB_HOST=localhost
DB_PORT=5432
DB_NAME=zapcore
DB_USER=postgres
DB_PASSWORD=password

# API
API_KEY=your-api-key-for-authentication
PORT=8080

# Storage (MinIO)
MINIO_ENDPOINT=localhost:9000
MINIO_ACCESS_KEY=minioadmin
MINIO_SECRET_KEY=minioadmin

# Logs
LOG_LEVEL=info
```

## 🧪 Testes

```bash
# Executar todos os testes
go test ./...

# Testes com coverage
go test -cover ./...

# Testes verbosos
go test -v ./...
```

## 🐳 Docker

### Desenvolvimento
```bash
# Subir todos os serviços
docker-compose up -d

# Ver logs
docker-compose logs -f zapcore

# Parar serviços
docker-compose down
```

### Produção
```bash
# Build da imagem
docker build -t zapcore:latest .

# Executar
docker run -d \
  --name zapcore \
  -p 8080:8080 \
  --env-file .env \
  zapcore:latest
```

## 📊 Monitoramento

### Health Check
```bash
curl http://localhost:8080/health
```

### Logs
```bash
# Docker
docker-compose logs -f zapcore

# Local
tail -f logs/app.log
```

## 🤝 Contribuição

1. Faça um fork do projeto
2. Crie uma branch para sua feature (`git checkout -b feature/AmazingFeature`)
3. Commit suas mudanças (`git commit -m 'Add some AmazingFeature'`)
4. Push para a branch (`git push origin feature/AmazingFeature`)
5. Abra um Pull Request

## 📝 Licença

Este projeto está sob a licença MIT. Veja o arquivo [LICENSE](LICENSE) para mais detalhes.

## 📞 Suporte

- 🐛 **Issues**: [GitHub Issues](https://github.com/felipyfgs/zapcore/issues)
- 📧 **Email**: suporte@zapcore.com
- 📖 **Documentação**: [API.md](API.md)

---

<div align="center">
  <p>Feito com ❤️ em Go</p>
  <p>⭐ Se este projeto te ajudou, considere dar uma estrela!</p>
</div>