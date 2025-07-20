# Sistema de Logging Dual - ZapCore WhatsApp API

## 📋 Visão Geral

O ZapCore WhatsApp API agora suporta **logging dual**, permitindo exibir logs coloridos no terminal E simultaneamente salvar logs estruturados em JSON em arquivo, quando configurado via variáveis de ambiente.

## 🚀 Funcionalidades

- ✅ **Terminal**: Logs coloridos e legíveis para desenvolvimento
- ✅ **Arquivo**: Logs JSON estruturados para análise e monitoramento
- ✅ **Rotação**: Rotação diária automática de arquivos (zapcore-2025-01-20.log)
- ✅ **Diretório**: Criação automática da pasta `logs/` se não existir
- ✅ **Compatibilidade**: Mantém compatibilidade com sistema atual

## ⚙️ Configuração via .env

### Variáveis de Ambiente

```bash
# Ativar saída dupla (terminal + arquivo)
LOG_DUAL_OUTPUT=true

# Formato para terminal (colorido)
LOG_CONSOLE_FORMAT=console

# Formato para arquivo (estruturado)
LOG_FILE_FORMAT=json

# Caminho do arquivo de log
LOG_FILE_PATH=./logs/zapcore.log

# Nível de log aplicado a ambas as saídas
LOG_LEVEL=info
```

### Configuração Padrão (Compatibilidade)

```bash
# Sistema atual (sem dual output)
LOG_DUAL_OUTPUT=false
LOG_LEVEL=info
LOG_FORMAT=json
```

## 📁 Estrutura de Arquivos

```
logs/
├── zapcore-2025-01-20.log (JSON estruturado)
├── zapcore-2025-01-21.log (Rotação diária)
└── zapcore-2025-01-22.log
```

## 🎯 Exemplos de Uso

### 1. Desenvolvimento (Dual Output)

```bash
# Terminal colorido + arquivo JSON
export LOG_DUAL_OUTPUT=true
export LOG_CONSOLE_FORMAT=console
export LOG_FILE_FORMAT=json
export LOG_FILE_PATH=./logs/zapcore.log
export LOG_LEVEL=debug

./zapcore
```

**Saída no Terminal:**
```
2025-07-20T18:37:48-03:00 INF cmd/server/main.go:33 > Logger centralizado inicializado com saída dupla
2025-07-20T18:37:48-03:00 INF internal/infra/database/database.go:66 > Conexão com banco de dados estabelecida
```

**Saída no Arquivo (logs/zapcore-2025-07-20.log):**
```json
{"level":"info","console_format":"console","file_format":"json","file_path":"./logs/zapcore.log","dual_output":true,"time":"2025-07-20T18:37:48-03:00","caller":"/home/felipe/zapcore/cmd/server/main.go:33","message":"Logger centralizado inicializado com saída dupla"}
{"level":"info","time":"2025-07-20T18:37:48-03:00","caller":"/home/felipe/zapcore/internal/infra/database/database.go:66","message":"Conexão com banco de dados estabelecida com sucesso"}
```

### 2. Produção (Arquivo JSON apenas)

```bash
# Apenas arquivo JSON
export LOG_DUAL_OUTPUT=false
export LOG_LEVEL=info
export LOG_FORMAT=json

./zapcore
```

### 3. Desenvolvimento (Terminal apenas)

```bash
# Apenas terminal colorido
export LOG_DUAL_OUTPUT=false
export LOG_LEVEL=debug
export LOG_FORMAT=console

./zapcore
```

## 🔍 Logs Preservados

Todas as funcionalidades críticas são preservadas em ambas as saídas:

### Reconexão Automática Multi-Sessão
```json
{
  "level": "info",
  "session_id": "1fd8bd19-d74e-41a0-bd7a-f984469fe6ea",
  "session_name": "sessao-teste-reconexao",
  "jid": "559984059035:19@s.whatsapp.net",
  "time": "2025-07-20T18:29:57-03:00",
  "caller": "/home/felipe/zapcore/internal/infra/whatsapp/client.go:133",
  "message": "Sessão reconectada com sucesso"
}
```

### QR Code e Pareamento
```json
{
  "level": "info",
  "session_id": "uuid-da-sessao",
  "qr_code": "codigo-qr-base64",
  "time": "2025-07-20T18:29:57-03:00",
  "message": "QR CODE GERADO"
}
```

### Status de Conexão
```json
{
  "level": "info",
  "session_id": "uuid-da-sessao",
  "status": "connected",
  "time": "2025-07-20T18:29:57-03:00",
  "message": "Status da sessão atualizado com sucesso"
}
```

## 🛠️ Implementação Técnica

### Arquitetura

- **MultiLevelWriter**: Combina console e arquivo usando `zerolog.MultiLevelWriter`
- **Rotação Automática**: Arquivos nomeados com data atual (YYYY-MM-DD)
- **Fallback**: Se falhar ao criar arquivo, continua apenas com console
- **Compatibilidade**: Sistema atual mantido intacto

### Código Principal

```go
// pkg/logger/logger.go
func createDualOutputLogger(config Config) zerolog.Logger {
    // Writer para console (colorido)
    consoleWriter := zerolog.ConsoleWriter{
        Out:        os.Stdout,
        TimeFormat: time.RFC3339,
        NoColor:    false,
    }

    // Writer para arquivo (JSON)
    fileWriter, err := createFileWriter(config.FilePath)
    if err != nil {
        // Fallback para console apenas
        return zerolog.New(consoleWriter).With().Timestamp().Caller().Logger()
    }

    // Combinar os dois writers
    multiWriter := zerolog.MultiLevelWriter(consoleWriter, fileWriter)
    
    return zerolog.New(multiWriter).With().Timestamp().Caller().Logger()
}
```

## 📊 Monitoramento e Análise

### Análise de Logs JSON

```bash
# Contar logs por nível
jq -r '.level' logs/zapcore-2025-07-20.log | sort | uniq -c

# Filtrar logs de reconexão
jq 'select(.message | contains("reconexão"))' logs/zapcore-2025-07-20.log

# Extrair session_ids únicos
jq -r '.session_id // empty' logs/zapcore-2025-07-20.log | sort | uniq
```

### Integração com ELK Stack

Os logs JSON são compatíveis com Elasticsearch, Logstash e Kibana para análise avançada.

## 🎉 Benefícios

1. **Desenvolvimento**: Logs coloridos e legíveis no terminal
2. **Produção**: Logs estruturados para análise automatizada
3. **Monitoramento**: Integração fácil com sistemas de observabilidade
4. **Debugging**: Histórico completo em arquivos com rotação
5. **Compatibilidade**: Zero breaking changes no sistema atual

## 🔧 Troubleshooting

### Problema: Arquivo de log não é criado

**Solução**: Verificar permissões do diretório
```bash
mkdir -p logs
chmod 755 logs
```

### Problema: Logs duplicados

**Causa**: Configuração incorreta de dual output
**Solução**: Verificar variáveis de ambiente

### Problema: Performance

**Observação**: O dual output tem overhead mínimo (~5-10%)
**Recomendação**: Usar apenas em desenvolvimento ou quando necessário
