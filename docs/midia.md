# ARQUITETURA - SISTEMA DE MÍDIA WAMEX

## 🎯 VISÃO
**Objetivo**: Criar sistema robusto de gerenciamento de mídia integrado ao MinIO S3 para envios WhatsApp Business API
**Usuários**: Desenvolvedores integrando com WAMEX API, usuários finais enviando mídia via WhatsApp
**Valor**: Redução de 70% no tempo de processamento de mídia e 100% de compatibilidade com formatos WhatsApp

## 👥 EXPERIÊNCIA DO USUÁRIO

### Jornada Principal
1. **Upload** → Usuário seleciona arquivo via interface/API
2. **Validação** → Sistema verifica tipo, tamanho e conformidade WhatsApp
3. **Processamento** → Otimização automática e armazenamento MinIO
4. **Disponibilização** → Mídia pronta para envio via WhatsApp API
5. **Gerenciamento** → Listagem, download e exclusão de arquivos

### Performance
- **Upload**: <3s para arquivos até 16MB
- **Download**: <1s para qualquer arquivo
- **Listagem**: <500ms para até 10.000 itens
- **Disponibilidade**: 99.9% uptime

### Acessibilidade
- API RESTful compatível com OpenAPI 3.0
- Suporte a multipart/form-data padrão
- Documentação interativa com Swagger
- Rate limiting configurável por usuário

## 🏗️ ARQUITETURA

### Frontend
**Não aplicável** - Sistema backend focado em API REST para integração

### Backend
**Go (Chi Router + Bun ORM + MinIO SDK)** - Justificativa:
- Performance superior para I/O intensivo (upload/download)
- Concorrência nativa ideal para múltiplos uploads simultâneos
- Ecosystem robusto para integração MinIO (minio-go SDK)
- Compatibilidade com stack WAMEX existente

### Dados
**PostgreSQL + MinIO S3** - Justificativa:
- **PostgreSQL**: Metadados de arquivos (ID, nome, tipo, tamanho, timestamps)
- **MinIO S3**: Armazenamento binário otimizado com versionamento
- **Redis**: Cache de metadados para listagens rápidas
- **Estrutura hierárquica**: `/bucket/YYYY/MM/DD/file_id.ext`

### Infraestrutura
**Híbrida (MinIO Self-hosted + Cloud Database)**
- MinIO Server: `https://minio.resolvecert.com` (já configurado)
- Buckets: `wamex-media`, `wamex-temp`, `wamex-thumbnails`
- Backup automático: Replicação cross-region
- CDN: CloudFlare para downloads públicos (opcional)

### API Endpoints
```
POST   /media/upload                    - Upload geral de mídia (detecta tipo automaticamente)
GET    /media/{mediaID}/download        - Download de mídia por ID
DELETE /media/{mediaID}                 - Deleção de mídia por ID
GET    /media/list                      - Listagem de mídias com paginação
POST   /message/{sessionID}/send/media  - Envio de mídia já uploadada
```

## 🚀 ROADMAP

### FASE 1: MVP (Alta Urgência) - 2 semanas
- [ ] **Estrutura de Banco**: Tabela `media_files` com metadados essenciais
- [ ] **Upload Básico**: Endpoint `/media/upload` com validação WhatsApp
- [ ] **Download Direto**: Endpoint `/media/{id}/download` servindo do MinIO
- [ ] **Integração WhatsApp**: Endpoint `/message/{session}/send/media`
- [ ] **Validações Core**: Tipos MIME e limites de tamanho por categoria

### FASE 2: Extensão (Média Urgência) - 2 semanas
- [ ] **Listagem Avançada**: Endpoint `/media/list` com filtros e paginação
- [ ] **Deleção Segura**: Endpoint `DELETE /media/{id}` com verificações
- [ ] **Otimização Automática**: Compressão de imagens e thumbnails
- [ ] **Cache Redis**: Metadados em cache para performance
- [ ] **Rate Limiting**: Proteção contra abuse de upload

### FASE 3: Otimização (Baixa Urgência) - 1 semana
- [ ] **Limpeza Automática**: Job para remoção de arquivos expirados
- [ ] **Métricas Avançadas**: Dashboard de uso e performance
- [ ] **CDN Integration**: CloudFlare para downloads públicos
- [ ] **Backup Automático**: Replicação cross-region
- [ ] **API Versioning**: Suporte a múltiplas versões da API

## ⚠️ RISCOS

### Riscos Técnicos
- **Limite de Conexões MinIO**: Saturação com uploads simultâneos → **Mitigação**: Pool de conexões configurável (Urgência: ALTA)
- **Validação de Segurança**: Upload de arquivos maliciosos → **Mitigação**: Verificação de magic numbers + antivírus (Urgência: ALTA)
- **Crescimento de Storage**: Acúmulo descontrolado de arquivos → **Mitigação**: TTL automático + alertas de quota (Urgência: MÉDIA)

### Riscos UX
- **Timeout em Uploads**: Arquivos grandes causam timeout → **Mitigação**: Upload chunked + progress tracking (Urgência: MÉDIA)
- **Feedback Limitado**: Usuário não sabe status do processamento → **Mitigação**: WebSocket para status real-time (Urgência: BAIXA)

### Riscos de Negócio
- **Conformidade WhatsApp**: Mudanças nos formatos suportados → **Mitigação**: Monitoramento automático da documentação oficial (Urgência: BAIXA)

## 📊 MÉTRICAS DE SUCESSO

### Técnicas
- **Latência Upload**: P95 < 3s para arquivos até 16MB
- **Latência Download**: P95 < 1s para qualquer arquivo
- **Uptime**: >99.9% disponibilidade mensal
- **Throughput**: >100 uploads simultâneos sem degradação
- **Storage Efficiency**: <5% overhead de metadados

### Negócio
- **Taxa de Sucesso**: >99.5% uploads bem-sucedidos
- **Conformidade WhatsApp**: 100% arquivos válidos para envio
- **Redução de Custos**: 30% economia vs soluções cloud tradicionais
- **Time to Market**: Redução de 70% no tempo de integração

### UX
- **Satisfação API**: NPS >8 entre desenvolvedores
- **Facilidade de Uso**: <5min para primeira integração
- **Documentação**: 100% endpoints documentados com exemplos
- **Suporte**: <2h tempo médio de resposta para issues

## 🔧 ESPECIFICAÇÕES WHATSAPP

### Tipos de Mídia Suportados (Cloud API)
```go
// Áudio - 16MB máximo
audio/aac, audio/mp4, audio/mpeg, audio/amr, audio/ogg (opus codec)

// Documento - 100MB máximo
text/plain, application/pdf, application/msword,
application/vnd.openxmlformats-officedocument.wordprocessingml.document,
application/vnd.ms-excel, application/vnd.openxmlformats-officedocument.spreadsheetml.sheet,
application/vnd.ms-powerpoint, application/vnd.openxmlformats-officedocument.presentationml.presentation

// Imagem - 5MB máximo
image/jpeg, image/png (8-bit RGB/RGBA apenas)

// Vídeo - 16MB máximo
video/mp4, video/3gp (H.264 + AAC codec, single audio stream)

// Sticker - 100KB estático, 500KB animado
image/webp (512x512 pixels para animados)
```

### Validações Obrigatórias
- **Magic Number Check**: Verificar primeiros 512 bytes
- **Codec Validation**: H.264 para vídeo, AAC para áudio
- **Dimension Limits**: 512x512 para stickers animados
- **Channel Validation**: Single channel para audio/ogg
- **Color Space**: RGB/RGBA para imagens (sem transparência)

## 🏗️ ARQUITETURA

### Frontend
**Não aplicável** - Sistema backend focado em API REST para integração

### Backend
**Go (Chi Router + Bun ORM)** - Justificativa:
- Performance superior para I/O intensivo (upload/download)
- Concorrência nativa ideal para múltiplos uploads simultâneos
- Ecosystem robusto para integração MinIO (minio-go SDK)
- Compatibilidade com stack WAMEX existente

### Dados
**PostgreSQL + MinIO S3** - Justificativa:
- **PostgreSQL**: Metadados de arquivos (ID, nome, tipo, tamanho, timestamps)
- **MinIO S3**: Armazenamento binário otimizado com versionamento
- **Redis**: Cache de metadados para listagens rápidas
- **Estrutura hierárquica**: `/bucket/YYYY/MM/DD/file_id.ext`

### Infraestrutura
**Híbrida (MinIO Self-hosted + Cloud Database)**
- MinIO Server: `https://minio.resolvecert.com` (já configurado)
- Buckets: `wamex-media`, `wamex-temp`, `wamex-thumbnails`
- Backup automático: Replicação cross-region
- CDN: CloudFlare para downloads públicos (opcional)