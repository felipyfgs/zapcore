# ZapCore WhatsApp API - Documentação Completa

## 📋 Índice
- [Visão Geral](#visão-geral)
- [Autenticação](#autenticação)
- [Gerenciamento de Sessões](#gerenciamento-de-sessões)
- [Envio de Mensagens](#envio-de-mensagens)
- [Códigos de Status](#códigos-de-status)
- [Exemplos Práticos](#exemplos-práticos)

## 🚀 Visão Geral

A ZapCore WhatsApp API é uma API REST completa para integração com WhatsApp usando o protocolo Multi-Device. Permite gerenciar múltiplas sessões e enviar diversos tipos de mensagens.

**Base URL:** `http://localhost:8080`

## 🔐 Autenticação

Todas as rotas protegidas requerem autenticação via API Key no header:

```bash
-H "X-API-Key: your-api-key-for-authentication"
```

## 📱 Gerenciamento de Sessões

### Criar Nova Sessão
```bash
curl -X POST "http://localhost:8080/sessions/add" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-api-key-for-authentication" \
  -d '{
    "name": "Minha Sessão",
    "description": "Sessão para testes"
  }'
```

### Listar Sessões
```bash
curl -X GET "http://localhost:8080/sessions/list" \
  -H "X-API-Key: your-api-key-for-authentication"
```

### Obter Status da Sessão
```bash
curl -X GET "http://localhost:8080/sessions/{sessionID}/status" \
  -H "X-API-Key: your-api-key-for-authentication"
```

### Conectar Sessão
```bash
curl -X POST "http://localhost:8080/sessions/{sessionID}/connect" \
  -H "X-API-Key: your-api-key-for-authentication"
```

### Desconectar Sessão
```bash
curl -X POST "http://localhost:8080/sessions/{sessionID}/logout" \
  -H "X-API-Key: your-api-key-for-authentication"
```

### Obter QR Code
```bash
curl -X GET "http://localhost:8080/sessions/{sessionID}/qr" \
  -H "X-API-Key: your-api-key-for-authentication"
```

## 💬 Envio de Mensagens

### Mensagem de Texto
```bash
curl -X POST "http://localhost:8080/messages/{sessionID}/send/text" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-api-key-for-authentication" \
  -d '{
    "to_jid": "5511999999999@s.whatsapp.net",
    "content": "Olá! Esta é uma mensagem de teste.",
    "reply_to_id": "optional_message_id"
  }'
```

### Envio de Imagem

**Padrão 1: Via Base64**
```bash
curl -X POST "http://localhost:8080/messages/{sessionID}/send/image" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-api-key-for-authentication" \
  -d '{
    "to": "5511999999999@s.whatsapp.net",
    "base64": "data:image/jpeg;base64,/9j/4AAQSkZJRgABAQAAAQ...",
    "caption": "Legenda da imagem",
    "replyId": "optional_message_id"
  }'
```

**Padrão 2: Via URL**
```bash
curl -X POST "http://localhost:8080/messages/{sessionID}/send/image" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-api-key-for-authentication" \
  -d '{
    "to": "5511999999999@s.whatsapp.net",
    "url": "https://exemplo.com/imagem.jpg",
    "caption": "Legenda da imagem",
    "replyId": "optional_message_id"
  }'
```

### Envio de Áudio

**Padrão 1: Via Base64**
```bash
curl -X POST "http://localhost:8080/messages/{sessionID}/send/audio" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-api-key-for-authentication" \
  -d '{
    "to": "5511999999999@s.whatsapp.net",
    "base64": "data:audio/mpeg;base64,SUQzAwAAAAAfdlBSSVYAAAAOAAABWE1QMwAAAAAAAAA...",
    "replyId": "optional_message_id"
  }'
```

**Padrão 2: Via URL**
```bash
curl -X POST "http://localhost:8080/messages/{sessionID}/send/audio" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-api-key-for-authentication" \
  -d '{
    "to": "5511999999999@s.whatsapp.net",
    "url": "https://exemplo.com/audio.mp3",
    "replyId": "optional_message_id"
  }'
```

### Envio de Vídeo

**Padrão 1: Via Base64**
```bash
curl -X POST "http://localhost:8080/messages/{sessionID}/send/video" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-api-key-for-authentication" \
  -d '{
    "to": "5511999999999@s.whatsapp.net",
    "base64": "data:video/mp4;base64,AAAAIGZ0eXBpc29tAAACAGlzb21pc28yYXZjMW1wNDE...",
    "caption": "Legenda do vídeo",
    "replyId": "optional_message_id"
  }'
```

**Padrão 2: Via URL**
```bash
curl -X POST "http://localhost:8080/messages/{sessionID}/send/video" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-api-key-for-authentication" \
  -d '{
    "to": "5511999999999@s.whatsapp.net",
    "url": "https://exemplo.com/video.mp4",
    "caption": "Legenda do vídeo",
    "replyId": "optional_message_id"
  }'
```

### Envio de Documento

**Padrão 1: Via Base64**
```bash
curl -X POST "http://localhost:8080/messages/{sessionID}/send/document" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-api-key-for-authentication" \
  -d '{
    "to": "5511999999999@s.whatsapp.net",
    "base64": "data:application/pdf;base64,JVBERi0xLjQKJcOkw7zDtsO8CjIgMCBvYmoKPDwKL0xlbmd0aCA...",
    "caption": "Descrição do documento",
    "replyId": "optional_message_id"
  }'
```

**Padrão 2: Via URL**
```bash
curl -X POST "http://localhost:8080/messages/{sessionID}/send/document" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-api-key-for-authentication" \
  -d '{
    "to": "5511999999999@s.whatsapp.net",
    "url": "https://exemplo.com/documento.pdf",
    "caption": "Descrição do documento",
    "replyId": "optional_message_id"
  }'
```

### Envio de Sticker
```bash
# Via arquivo local
curl -X POST "http://localhost:8080/messages/{sessionID}/send/sticker" \
  -H "X-API-Key: your-api-key-for-authentication" \
  -F "to_jid=5511999999999@s.whatsapp.net" \
  -F "sticker_file=@/caminho/para/sticker.webp" \
  -F "reply_to_id=optional_message_id"

# Via URL pública
curl -X POST "http://localhost:8080/messages/{sessionID}/send/sticker" \
  -H "X-API-Key: your-api-key-for-authentication" \
  -F "to_jid=5511999999999@s.whatsapp.net" \
  -F "sticker_url=https://exemplo.com/sticker.webp" \
  -F "reply_to_id=optional_message_id"

# Via Base64
curl -X POST "http://localhost:8080/messages/{sessionID}/send/sticker" \
  -H "X-API-Key: your-api-key-for-authentication" \
  -F "to_jid=5511999999999@s.whatsapp.net" \
  -F "sticker_base64=data:image/webp;base64,UklGRnoAAABXRUJQVlA4WAoAAAAQAAAAAAAAAAAAQUxQSAwAAAARBxAR..." \
  -F "reply_to_id=optional_message_id"
```

## 📊 Códigos de Status

| Código | Descrição |
|--------|-----------|
| 200 | Sucesso |
| 400 | Requisição inválida |
| 401 | Não autorizado |
| 404 | Recurso não encontrado |
| 500 | Erro interno do servidor |

## 🎯 Exemplos Práticos

### Exemplo 1: Fluxo Completo de Nova Sessão
```bash
# 1. Criar sessão
SESSION_RESPONSE=$(curl -s -X POST "http://localhost:8080/sessions/add" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-api-key-for-authentication" \
  -d '{"name": "Bot Vendas", "description": "Bot para atendimento"}')

# 2. Extrair ID da sessão
SESSION_ID=$(echo $SESSION_RESPONSE | jq -r '.id')

# 3. Conectar sessão
curl -X POST "http://localhost:8080/sessions/$SESSION_ID/connect" \
  -H "X-API-Key: your-api-key-for-authentication"

# 4. Obter QR Code
curl -X GET "http://localhost:8080/sessions/$SESSION_ID/qr" \
  -H "X-API-Key: your-api-key-for-authentication"

# 5. Após escanear QR, enviar mensagem
curl -X POST "http://localhost:8080/messages/$SESSION_ID/send/text" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-api-key-for-authentication" \
  -d '{
    "to_jid": "5511999999999@s.whatsapp.net",
    "content": "🎉 Sessão conectada com sucesso!"
  }'
```

### Exemplo 2: Envio de Mídia com Arquivos Locais
```bash
# Definir variáveis
SESSION_ID="1fd8bd19-d74e-41a0-bd7a-f984469fe6ea"
RECIPIENT="5511999999999@s.whatsapp.net"

# Enviar documento PDF
curl -X POST "http://localhost:8080/messages/$SESSION_ID/send/document" \
  -H "X-API-Key: your-api-key-for-authentication" \
  -F "to_jid=$RECIPIENT" \
  -F "document_file=@assets/document.pdf" \
  -F "caption=📄 Relatório mensal"

# Enviar imagem
curl -X POST "http://localhost:8080/messages/$SESSION_ID/send/image" \
  -H "X-API-Key: your-api-key-for-authentication" \
  -F "to_jid=$RECIPIENT" \
  -F "image_file=@assets/image.png" \
  -F "caption=🖼️ Captura de tela"

# Enviar vídeo
curl -X POST "http://localhost:8080/messages/$SESSION_ID/send/video" \
  -H "X-API-Key: your-api-key-for-authentication" \
  -F "to_jid=$RECIPIENT" \
  -F "video_file=@assets/video.mp4" \
  -F "caption=🎬 Demonstração do produto"
```

### Exemplo 3: Envio em Lote
```bash
# Script para envio em lote
SESSION_ID="1fd8bd19-d74e-41a0-bd7a-f984469fe6ea"

# Lista de destinatários
RECIPIENTS=(
  "5511999999999@s.whatsapp.net"
  "5511888888888@s.whatsapp.net"
  "5511777777777@s.whatsapp.net"
)

# Enviar para todos
for recipient in "${RECIPIENTS[@]}"; do
  curl -X POST "http://localhost:8080/messages/$SESSION_ID/send/text" \
    -H "Content-Type: application/json" \
    -H "X-API-Key: your-api-key-for-authentication" \
    -d "{
      \"to_jid\": \"$recipient\",
      \"content\": \"🚀 Mensagem promocional para todos!\"
    }"
  sleep 2  # Aguardar 2 segundos entre envios
done
```

## 📋 Tipos MIME Suportados

### Imagens
- `image/jpeg`, `image/jpg`
- `image/png`
- `image/gif`
- `image/webp`

### Vídeos
- `video/mp4`
- `video/avi`
- `video/mov`
- `video/mkv`
- `video/webm`

### Áudios
- `audio/mpeg`, `audio/mp3`
- `audio/wav`
- `audio/ogg`
- `audio/aac`
- `audio/m4a`

### Documentos
- `application/pdf`
- `application/msword`
- `application/vnd.openxmlformats-officedocument.wordprocessingml.document`
- `application/vnd.ms-excel`
- `application/vnd.openxmlformats-officedocument.spreadsheetml.sheet`
- `text/plain`
- `application/zip`
- `application/rar`

### Stickers
- `image/webp`
- `image/png`

## 🔧 Parâmetros Padronizados

### Para Mensagens de Texto
| Parâmetro | Tipo | Descrição |
|-----------|------|-----------|
| `to_jid` | string | **Obrigatório.** JID do destinatário |
| `content` | string | **Obrigatório.** Conteúdo da mensagem |
| `reply_to_id` | string | Opcional. ID da mensagem sendo respondida |

### Para Mensagens de Mídia

#### 📁 Arquivo Local
| Parâmetro | Formato | Exemplo |
|-----------|---------|---------|
| `image_file` | `@caminho/arquivo` | `@assets/image.png` |
| `audio_file` | `@caminho/arquivo` | `@assets/audio.mp3` |
| `video_file` | `@caminho/arquivo` | `@assets/video.mp4` |
| `document_file` | `@caminho/arquivo` | `@assets/document.pdf` |
| `sticker_file` | `@caminho/arquivo` | `@assets/sticker.webp` |

#### 🌐 URL Pública
| Parâmetro | Formato | Exemplo |
|-----------|---------|---------|
| `url` | `https://...` | `https://exemplo.com/arquivo.jpg` |

#### 📋 Base64
| Parâmetro | Formato | Exemplo |
|-----------|---------|---------|
| `image:` | `{tipo}:{base64_string}` | `image:iVBORw0KGgoAAAANSUhEUgAA...` |
| `audio:` | `{tipo}:{base64_string}` | `audio:SUQzAwAAAAAfdlBSSVYA...` |
| `video:` | `{tipo}:{base64_string}` | `video:AAAAIGZ0eXBpc29tAAA...` |
| `document:` | `{tipo}:{base64_string}` | `document:JVBERi0xLjQKJcOkw7...` |
| `sticker:` | `{tipo}:{base64_string}` | `sticker:UklGRnoAAABXRUJQVl...` |

#### 📝 Parâmetros Opcionais
| Parâmetro | Tipo | Descrição |
|-----------|------|-----------|
| `caption` | string | Opcional. Legenda da mídia |
| `reply_to_id` | string | Opcional. ID da mensagem sendo respondida |
| `file_name` | string | Opcional. Nome personalizado do arquivo |
| `mime_type` | string | Opcional. Tipo MIME personalizado |

## 📱 Formato do JID

O JID (Jabber ID) é o identificador único do destinatário no WhatsApp:

### Para Contatos Individuais
```
{número_com_código_país}@s.whatsapp.net
```
**Exemplos:**
- Brasil: `5511999999999@s.whatsapp.net`
- EUA: `1234567890@s.whatsapp.net`
- Argentina: `5491123456789@s.whatsapp.net`

### Para Grupos
```
{id_do_grupo}@g.us
```
**Exemplo:**
- `120363025246125486@g.us`

## 🔄 Respostas da API

### Resposta de Sucesso (Texto)
```json
{
  "whatsapp_id": "3EB078C20421B3046EA341",
  "status": "sent",
  "timestamp": "2025-07-20T20:08:31-03:00"
}
```

### Resposta de Sucesso (Mídia)
```json
{
  "whatsapp_id": "3EB0F68233FDD15A7D7467",
  "status": "sent",
  "timestamp": "2025-07-20T20:13:22-03:00"
}
```

### Resposta de Erro
```json
{
  "error": "ID da sessão inválido",
  "message": "O ID da sessão deve ser um UUID válido"
}
```

## 🚨 Tratamento de Erros

### Erros Comuns

#### 400 - Bad Request
```json
{
  "error": "JID do destinatário obrigatório",
  "message": "O campo to_jid é obrigatório"
}
```

#### 401 - Unauthorized
```json
{
  "error": "Não autorizado",
  "message": "API Key inválida ou ausente"
}
```

#### 404 - Not Found
```json
{
  "error": "Sessão não encontrada",
  "message": "A sessão especificada não existe"
}
```

#### 500 - Internal Server Error
```json
{
  "error": "Erro interno do servidor",
  "message": "Cliente não está conectado para sessão"
}
```

## 🔍 Monitoramento e Logs

### Health Check
```bash
# Verificar se a API está funcionando
curl -X GET "http://localhost:8080/health"

# Resposta esperada
{
  "status": "ok",
  "timestamp": "2025-07-20T20:00:00Z"
}
```

### Status da Aplicação
```bash
# Verificar status detalhado
curl -X GET "http://localhost:8080/ready"

# Informações da API
curl -X GET "http://localhost:8080/"
```

## 🛠️ Configuração e Deployment

### Variáveis de Ambiente
```bash
# Configurações básicas
export DB_HOST=localhost
export DB_PORT=5432
export DB_NAME=zapcore
export DB_USER=postgres
export DB_PASSWORD=password

# Configurações da API
export API_KEY=your-api-key-for-authentication
export SERVER_PORT=8080
export LOG_LEVEL=info

# Configurações de upload
export UPLOAD_MAX_SIZE=10MB
export UPLOAD_ALLOWED_TYPES=image/jpeg,image/png,video/mp4,audio/mpeg
```

### Docker
```bash
# Build da imagem
docker build -t zapcore .

# Executar container
docker run -d \
  --name zapcore-api \
  -p 8080:8080 \
  -e DB_HOST=postgres \
  -e API_KEY=your-api-key \
  zapcore
```

## 📞 Suporte e Contribuição

### Reportar Problemas
- Verifique os logs da aplicação
- Inclua o `whatsapp_id` da mensagem com problema
- Forneça o payload da requisição

### Contribuir
1. Fork do repositório
2. Criar branch para feature
3. Implementar testes
4. Submeter Pull Request

---

**ZapCore WhatsApp API v1.0.0**
Desenvolvido com ❤️ em Go
