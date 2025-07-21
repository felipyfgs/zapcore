# 📱 ZapCore WhatsApp API - Documentação Completa

API REST completa para integração com WhatsApp usando a biblioteca whatsmeow com suporte a múltiplas sessões e envio de mídia.

## 🚀 Funcionalidades

- ✅ **Mensagens de Texto** - Envio simples e com reply
- ✅ **Documentos** - PDF, DOC, XLSX, etc. com caption
- ✅ **Imagens** - JPG, PNG, GIF, etc. com caption
- ✅ **Vídeos** - MP4, AVI, MOV, etc. com caption
- ✅ **Áudios** - MP3, WAV, OGG, etc.
- ✅ **Múltiplos Formatos** - Arquivo local, URL pública, base64
- ✅ **Auto-detecção** - MIME type automático por extensão
- ✅ **Validações** - Entrada robusta e tratamento de erros
- ✅ **Logs Detalhados** - Debug completo para desenvolvimento

## 📋 Índice
- [🚀 Visão Geral](#-visão-geral)
- [🔐 Autenticação](#-autenticação)
- [📱 Gerenciamento de Sessões](#-gerenciamento-de-sessões)
- [💬 Mensagens de Texto](#-mensagens-de-texto)
- [📎 Envio de Mídia](#-envio-de-mídia)
- [⚠️ Códigos de Status](#️-códigos-de-status)
- [💡 Exemplos Práticos](#-exemplos-práticos)

## 🚀 Visão Geral

A ZapCore WhatsApp API é uma solução completa para integração com WhatsApp usando o protocolo Multi-Device. Permite gerenciar múltiplas sessões simultaneamente e enviar diversos tipos de mensagens e mídias.

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

## 💬 Mensagens de Texto

### Envio de Texto Simples
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

**Resposta de Sucesso:**
```json
{
  "whatsapp_id": "3EB0123456789ABCDEF",
  "status": "sent",
  "timestamp": "2025-07-20T21:00:00Z",
  "message": "Mensagem enviada com sucesso"
}
```

## 📎 Envio de Mídia

A API suporte **3 formatos diferentes** para envio de mídia:
- 📁 **Arquivo Local**: `"file": "assets/documento.pdf"`
- 🌐 **URL Pública**: `"url": "https://exemplo.com/arquivo.jpg"`
- 📋 **Base64**: `"base64": "data:mime/type;base64,string..."`

> ⚠️ **Importante**: Use apenas **um** formato por requisição.

### 📄 Documentos

Suporte para PDF, DOC, XLSX, TXT, etc. com caption opcional.

**📁 Via Arquivo Local:**
```bash
curl -X POST "http://localhost:8080/messages/{sessionID}/send/document" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-api-key-for-authentication" \
  -d '{
    "to": "5511999999999@s.whatsapp.net",
    "file": "assets/document.pdf",
    "caption": "📄 Documento importante anexado!"
  }'
```

**🌐 Via URL Pública:**
```bash
curl -X POST "http://localhost:8080/messages/{sessionID}/send/document" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-api-key-for-authentication" \
  -d '{
    "to": "5511999999999@s.whatsapp.net",
    "url": "https://exemplo.com/documento.pdf",
    "caption": "📄 Documento via URL"
  }'
```

**📋 Via Base64:**
```bash
curl -X POST "http://localhost:8080/messages/{sessionID}/send/document" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-api-key-for-authentication" \
  -d '{
    "to": "5511999999999@s.whatsapp.net",
    "base64": "data:application/pdf;base64,JVBERi0xLjQKMSAwIG9iago8PAovVHlwZSAvQ2F0YWxvZwo...",
    "caption": "📄 Documento via base64"
  }'
```

### 🖼️ Imagens

Suporte para JPG, PNG, GIF, WEBP, etc. com caption opcional.

**📁 Via Arquivo Local:**
```bash
curl -X POST "http://localhost:8080/messages/{sessionID}/send/image" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-api-key-for-authentication" \
  -d '{
    "to": "5511999999999@s.whatsapp.net",
    "file": "assets/image.png",
    "caption": "🖼️ Imagem local com caption!"
  }'
```

**🌐 Via URL Pública:**
```bash
curl -X POST "http://localhost:8080/messages/{sessionID}/send/image" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-api-key-for-authentication" \
  -d '{
    "to": "5511999999999@s.whatsapp.net",
    "url": "https://exemplo.com/imagem.jpg",
    "caption": "🖼️ Imagem via URL"
  }'
```

**📋 Via Base64:**
```bash
curl -X POST "http://localhost:8080/messages/{sessionID}/send/image" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-api-key-for-authentication" \
  -d '{
    "to": "5511999999999@s.whatsapp.net",
    "base64": "data:image/jpeg;base64,/9j/4AAQSkZJRgABAQAAAQ...",
    "caption": "🖼️ Imagem via base64"
  }'
```

### 🎥 Vídeos

Suporte para MP4, AVI, MOV, MKV, etc. com caption opcional.

**📁 Via Arquivo Local:**
```bash
curl -X POST "http://localhost:8080/messages/{sessionID}/send/video" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-api-key-for-authentication" \
  -d '{
    "to": "5511999999999@s.whatsapp.net",
    "file": "assets/video.mp4",
    "caption": "🎥 Vídeo local com caption!"
  }'
```

**🌐 Via URL Pública:**
```bash
curl -X POST "http://localhost:8080/messages/{sessionID}/send/video" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-api-key-for-authentication" \
  -d '{
    "to": "5511999999999@s.whatsapp.net",
    "url": "https://exemplo.com/video.mp4",
    "caption": "🎥 Vídeo via URL"
  }'
```

**📋 Via Base64:**
```bash
curl -X POST "http://localhost:8080/messages/{sessionID}/send/video" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-api-key-for-authentication" \
  -d '{
    "to": "5511999999999@s.whatsapp.net",
    "base64": "data:video/mp4;base64,AAAAIGZ0eXBpc29tAAACAGlzb21pc28yYXZjMW1wNDE...",
    "caption": "🎥 Vídeo via base64"
  }'
```

### 🎵 Áudios

Suporte para MP3, WAV, OGG, AAC, etc.

**📁 Via Arquivo Local:**
```bash
curl -X POST "http://localhost:8080/messages/{sessionID}/send/audio" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-api-key-for-authentication" \
  -d '{
    "to": "5511999999999@s.whatsapp.net",
    "file": "assets/audio.mp3"
  }'
```

**🌐 Via URL Pública:**
```bash
curl -X POST "http://localhost:8080/messages/{sessionID}/send/audio" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-api-key-for-authentication" \
  -d '{
    "to": "5511999999999@s.whatsapp.net",
    "url": "https://exemplo.com/audio.mp3"
  }'
```

**📋 Via Base64:**
```bash
curl -X POST "http://localhost:8080/messages/{sessionID}/send/audio" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-api-key-for-authentication" \
  -d '{
    "to": "5511999999999@s.whatsapp.net",
    "base64": "data:audio/mpeg;base64,SUQzAwAAAAAfdlBSSVYAAAAOAAABWE1QMwAAAAAAAAA..."
  }'
```

## ⚠️ Códigos de Status

### Respostas de Sucesso

**200 OK** - Mídia enviada com sucesso
```json
{
  "whatsapp_id": "3EB06B4F968EC808B5C54E",
  "status": "sent",
  "timestamp": "2025-07-20T21:00:27-03:00",
  "message": "Mídia enviada com sucesso"
}
```

### Códigos de Erro

| Código | Descrição | Solução |
|--------|-----------|---------|
| **400** | Dados inválidos | Verificar formato JSON e campos obrigatórios |
| **401** | API Key inválida | Verificar header `X-API-Key` |
| **404** | Sessão não encontrada | Verificar se sessionID existe |
| **503** | Erro no WhatsApp | Verificar conexão da sessão |

### Exemplos de Erro

**400 - Validação falhou:**
```json
{
  "error": "validação de documento falhou: tipo MIME não especificado",
  "code": 400
}
```

**401 - Não autorizado:**
```json
{
  "error": "API Key inválida",
  "code": 401
}
```

**503 - Serviço indisponível:**
```json
{
  "error": "erro ao enviar mídia",
  "code": 503
}
```

## 💡 Exemplos Práticos

### Cenário 1: Envio de Relatório PDF
```bash
# Enviar relatório mensal via arquivo local
curl -X POST "http://localhost:8080/messages/1fd8bd19-d74e-41a0-bd7a-f984469fe6ea/send/document" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-api-key-for-authentication" \
  -d '{
    "to": "5511999999999@s.whatsapp.net",
    "file": "assets/relatorio-mensal.pdf",
    "caption": "📊 Relatório mensal de vendas - Janeiro 2025"
  }'
```

### Cenário 2: Compartilhar Imagem de Produto
```bash
# Enviar foto de produto via URL
curl -X POST "http://localhost:8080/messages/1fd8bd19-d74e-41a0-bd7a-f984469fe6ea/send/image" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-api-key-for-authentication" \
  -d '{
    "to": "5511999999999@s.whatsapp.net",
    "url": "https://loja.com/produtos/smartphone-x1.jpg",
    "caption": "📱 Novo Smartphone X1 - Disponível por R$ 1.299,00"
  }'
```

### Cenário 3: Envio de Vídeo Promocional
```bash
# Enviar vídeo promocional via base64
curl -X POST "http://localhost:8080/messages/1fd8bd19-d74e-41a0-bd7a-f984469fe6ea/send/video" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-api-key-for-authentication" \
  -d '{
    "to": "5511999999999@s.whatsapp.net",
    "base64": "data:video/mp4;base64,AAAAIGZ0eXBpc29tAAACAGlzb21pc28yYXZjMW1wNDE...",
    "caption": "🎥 Confira nosso novo produto em ação!"
  }'
```

## 🔧 Configuração

### Variáveis de Ambiente
```bash
# .env
DB_HOST=localhost
DB_PORT=5432
DB_NAME=zapcore
DB_USER=postgres
DB_PASSWORD=password
API_KEY=your-api-key-for-authentication
LOG_LEVEL=debug
```

### Estrutura de Arquivos
```
zapcore/
├── assets/           # Arquivos de mídia para teste
│   ├── document.pdf
│   ├── image.png
│   └── video.mp4
├── logs/            # Logs da aplicação
└── internal/        # Código fonte
```

### Formatos Suportados

| Tipo | Extensões | MIME Types |
|------|-----------|------------|
| **Documentos** | pdf, doc, docx, xls, xlsx, ppt, pptx, txt | application/pdf, application/msword, etc. |
| **Imagens** | jpg, jpeg, png, gif, webp, bmp | image/jpeg, image/png, image/gif, etc. |
| **Vídeos** | mp4, avi, mov, mkv, webm | video/mp4, video/avi, video/quicktime, etc. |
| **Áudios** | mp3, wav, ogg, aac, m4a | audio/mpeg, audio/wav, audio/ogg, etc. |

---

## 📞 Suporte

Para dúvidas ou problemas:
- 📧 Email: suporte@zapcore.com
- 📱 WhatsApp: +55 11 99999-9999
- 🐛 Issues: GitHub Issues

---

## 🎯 Resumo Rápido

### Texto
```bash
curl -X POST "http://localhost:8080/messages/{sessionID}/send/text" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-api-key-for-authentication" \
  -d '{"to_jid": "5511999999999@s.whatsapp.net", "content": "Olá!"}'
```

### Mídia (3 formatos)
```bash
# Arquivo local
{"to": "5511999999999@s.whatsapp.net", "file": "assets/document.pdf", "caption": "Legenda"}

# URL pública
{"to": "5511999999999@s.whatsapp.net", "url": "https://exemplo.com/arquivo.jpg", "caption": "Legenda"}

# Base64
{"to": "5511999999999@s.whatsapp.net", "base64": "data:image/jpeg;base64,/9j/4AAQ...", "caption": "Legenda"}
```

### Endpoints
- `/messages/{sessionID}/send/text` - Texto
- `/messages/{sessionID}/send/document` - PDF, DOC, etc.
- `/messages/{sessionID}/send/image` - JPG, PNG, etc.
- `/messages/{sessionID}/send/video` - MP4, AVI, etc.
- `/messages/{sessionID}/send/audio` - MP3, WAV, etc.

**Versão:** v1.0.0 | **Atualização:** 2025-07-20
