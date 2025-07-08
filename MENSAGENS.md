# 📨 WAMEX - Guia de Envio de Mensagens

## 🚀 Funcionalidade Implementada

✅ **Envio de mensagens de texto** está **100% funcional**!
✅ **Reconexão automática** das sessões ao iniciar o servidor
✅ **Recuperação de devices existentes** do banco de dados
✅ **Múltiplas sessões simultâneas**

---

## 📋 Endpoint Principal

```
POST /sessions/{sessionID}/send
```

### Headers Obrigatórios:
- `Content-Type: application/json`
- `apiKey: njhfyikg`

### Body (JSON):
```json
{
  "to": "559981769536@s.whatsapp.net",
  "message": "Sua mensagem aqui"
}
```

---

## 🔧 Exemplos Práticos

### 1. Envio Básico (usando curl)

**⚠️ IMPORTANTE: Use aspas simples para evitar problemas com `!` no bash**

```bash
curl -X POST -H "Content-Type: application/json" -H "apiKey: njhfyikg" \
  -d '{"to": "559981769536@s.whatsapp.net", "message": "Olá! Mensagem de teste!"}' \
  http://localhost:8080/sessions/minha-sessao/send
```

### 2. Mensagem com Emoji

```bash
curl -X POST -H "Content-Type: application/json" -H "apiKey: njhfyikg" \
  -d '{"to": "559981769536@s.whatsapp.net", "message": "🎉 WAMEX funcionando! 🚀"}' \
  http://localhost:8080/sessions/minha-sessao/send
```

### 3. Mensagem para Grupo

```bash
curl -X POST -H "Content-Type: application/json" -H "apiKey: njhfyikg" \
  -d '{"to": "120363123456789012@g.us", "message": "Mensagem para o grupo! 👥"}' \
  http://localhost:8080/sessions/minha-sessao/send
```

---

## 📱 Formatos de JID

| Tipo | Formato | Exemplo |
|------|---------|---------|
| **Número Individual** | `{número}@s.whatsapp.net` | `559981769536@s.whatsapp.net` |
| **Grupo** | `{groupID}@g.us` | `120363123456789012@g.us` |
| **Status/Stories** | `status@broadcast` | `status@broadcast` |

---

## ✅ Respostas da API

### Sucesso (200):
```json
{
  "success": true,
  "message": "Mensagem enviada com sucesso",
  "to": "559981769536@s.whatsapp.net",
  "content": "Sua mensagem aqui"
}
```

### Erros Comuns:

#### Sessão não encontrada (404):
```json
{
  "error": "session_not_found",
  "message": "Sessão não encontrada: nome-sessao",
  "code": 404
}
```

#### Sessão não conectada (409):
```json
{
  "error": "session_not_connected",
  "message": "Sessão não está conectada ao WhatsApp",
  "code": 409
}
```

#### Campos obrigatórios (400):
```json
{
  "error": "missing_to",
  "message": "Campo 'to' é obrigatório",
  "code": 400
}
```

---

## 🔄 Reconexão Automática

O sistema agora **reconecta automaticamente** as sessões que estavam conectadas:

```
18:21:02.765 [SessionManager INFO] Device existente recuperado para sessão 0e11ca32-1442-4110-9452-1d673a40a0a2 (JID: 559981769536:96@s.whatsapp.net)
18:21:02.767 [SessionManager INFO] Tentando reconectar sessão: 0e11ca32-1442-4110-9452-1d673a40a0a2 (nome: minha-sessao)
18:21:03.111 [SessionManager INFO] Sessão 0e11ca32-1442-4110-9452-1d673a40a0a2 reconectada com sucesso
```

---

## 🛠️ Solução para Problemas no Bash

### Problema: `bash: !: event not found`

**Causa:** O bash interpreta `!` como history expansion.

**Solução:** Use **aspas simples** em vez de aspas duplas:

❌ **Errado:**
```bash
curl -d "{\"message\": \"Olá!\"}"  # Erro: bash: !: event not found
```

✅ **Correto:**
```bash
curl -d '{"message": "Olá!"}'      # Funciona perfeitamente
```

---

## 📊 Status das Funcionalidades

| Funcionalidade | Status | Descrição |
|----------------|--------|-----------|
| ✅ Envio de texto | **Implementado** | Mensagens de texto funcionando |
| ✅ Reconexão automática | **Implementado** | Sessões reconectam ao iniciar |
| ✅ Múltiplas sessões | **Implementado** | Várias sessões simultâneas |
| ✅ Recuperação de device | **Implementado** | Devices salvos são recuperados |
| 🔄 Envio de mídia | **Planejado** | Imagens, áudios, documentos |
| 🔄 Webhooks | **Planejado** | Notificações de eventos |
| 🔄 Histórico de mensagens | **Planejado** | Armazenamento e consulta |

---

## 🎯 Próximos Passos

1. **Implementar envio de mídia** (imagens, áudios, documentos)
2. **Sistema de webhooks** para receber mensagens
3. **Histórico de mensagens** no banco de dados
4. **Verificação de números** válidos
5. **Gerenciamento de grupos** (criar, adicionar membros)

---

**🎉 O sistema WAMEX está funcionando perfeitamente para envio de mensagens de texto!**
