# 🧪 WAMEX API Tests

Este diretório contém testes HTTP para validar todas as funcionalidades do sistema WAMEX.

## 📁 Arquivos de Teste

### `wamex-api-tests.http`
- **Testes básicos** de todos os tipos de mensagem
- **Testes de validação** e tratamento de erros
- **Testes de performance** com múltiplas mensagens
- **Testes de gerenciamento** de sessão

### `wamex-media-tests.http`
- **Testes avançados** de mídia
- **Exemplos com arquivos reais**
- **Validações específicas** de tipos MIME
- **Testes de tamanho** de arquivo

## 🚀 Como Usar

### 1. **Pré-requisitos**
- Visual Studio Code com extensão **REST Client**
- Servidor WAMEX rodando em `http://localhost:8080`
- Sessão WhatsApp conectada (nome: `teste`)

### 2. **Executar Testes**
1. Abra qualquer arquivo `.http` no VS Code
2. Clique em **"Send Request"** acima de cada teste
3. Veja a resposta no painel lateral

### 3. **Configurar Variáveis**
Edite as variáveis no topo dos arquivos:
```http
@baseUrl = http://localhost:8080
@sessionName = teste
@testPhone = 559981769536
```

## 📱 Tipos de Mensagem Testados

### ✅ **Implementados e Funcionando**

| Tipo | Endpoint | Status |
|------|----------|--------|
| 📝 **Texto** | `/send/text` | ✅ Funcionando |
| 🖼️ **Imagem** | `/send/image` | ✅ Funcionando |
| 🎵 **Áudio** | `/send/audio` | ✅ Funcionando |
| 📄 **Documento** | `/send/document` | ✅ Funcionando |
| 🎭 **Sticker** | `/send/sticker` | ✅ Funcionando |

## 🔧 Preparar Arquivos de Mídia

### **Converter arquivo para Base64**

**Windows (PowerShell):**
```powershell
[Convert]::ToBase64String([IO.File]::ReadAllBytes('acets\audio.ogg'))
```

**Linux/Mac:**
```bash
base64 -w 0 acets/audio.ogg
```

### **Formatos Suportados**

#### 🖼️ **Imagens**
- PNG: `data:image/png;base64,`
- JPEG: `data:image/jpeg;base64,`
- WebP: `data:image/webp;base64,`

#### 🎵 **Áudio**
- OGG: `data:audio/ogg;base64,`
- MP3: `data:audio/mp3;base64,`
- WAV: `data:audio/wav;base64,`
- AAC: `data:audio/aac;base64,`

#### 📄 **Documentos**
- PDF: `data:application/pdf;base64,`
- TXT: `data:text/plain;base64,`
- DOCX: `data:application/vnd.openxmlformats-officedocument.wordprocessingml.document;base64,`

#### 🎭 **Stickers**
- WebP: `data:image/webp;base64,`

## 📊 Exemplo de Resposta

### **Sucesso:**
```json
{
  "success": true,
  "message": "Text message sent successfully",
  "details": {
    "phone": "559981769536",
    "type": "text",
    "status": "sent",
    "sentAt": "2025-07-13T12:59:15.4855786-03:00",
    "sessionName": "teste"
  },
  "timestamp": "2025-07-13T12:59:15.4855786-03:00"
}
```

### **Erro:**
```json
{
  "success": false,
  "message": "Phone number is required",
  "timestamp": "2025-07-13T12:59:15.4855786-03:00"
}
```

## 🎯 Sequência de Testes Recomendada

### **1. Teste Básico**
1. Verificar status da sessão
2. Enviar mensagem de texto
3. Verificar logs do servidor

### **2. Teste Completo**
1. Executar todos os 5 tipos de mensagem
2. Verificar recebimento no WhatsApp
3. Validar respostas da API

### **3. Teste de Validação**
1. Testar com dados inválidos
2. Verificar tratamento de erros
3. Validar códigos de resposta

### **4. Teste de Performance**
1. Enviar múltiplas mensagens
2. Verificar tempo de resposta
3. Monitorar logs do servidor

## 🚨 Troubleshooting

### **Erro: Connection refused**
- Verificar se o servidor está rodando
- Confirmar porta 8080 disponível

### **Erro: Session not found**
- Verificar nome da sessão nas variáveis
- Confirmar sessão conectada no WhatsApp

### **Erro: Invalid base64**
- Verificar formato do data URL
- Confirmar codificação base64 correta

### **Erro: File too large**
- Verificar limites de tamanho:
  - Imagem: 16MB
  - Áudio: 16MB  
  - Documento: 100MB
  - Sticker: 500KB

## 📝 Logs Úteis

### **Verificar logs do servidor:**
```bash
tail -f logs/wamex.log
```

### **Verificar status do Docker:**
```bash
docker-compose ps
```

### **Verificar banco de dados:**
```bash
docker exec -it wamex-postgres psql -U postgres -d wamex -c "SELECT * FROM sessions;"
```

---

**🎉 Sistema WAMEX - Todos os testes implementados e funcionando!**
