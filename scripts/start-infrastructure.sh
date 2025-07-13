#!/bin/bash

# Script para inicializar infraestrutura WAMEX (PostgreSQL + MinIO)
# Uso: ./scripts/start-infrastructure.sh

echo "🗄️  Iniciando infraestrutura WAMEX (PostgreSQL + MinIO)..."

# Verifica se Docker está rodando
if ! docker info > /dev/null 2>&1; then
    echo "❌ Docker não está rodando. Inicie o Docker primeiro."
    exit 1
fi

# Para containers existentes se estiverem rodando
echo "🛑 Parando containers existentes..."
docker-compose down

# Inicia toda a infraestrutura
echo "🚀 Iniciando PostgreSQL + MinIO..."
docker-compose up -d

# Aguarda serviços estarem prontos
echo "⏳ Aguardando serviços estarem prontos..."
sleep 15

# Verifica PostgreSQL
if docker-compose exec -T postgres pg_isready -U postgres > /dev/null 2>&1; then
    echo "✅ PostgreSQL iniciado com sucesso!"
else
    echo "⚠️  PostgreSQL pode não estar totalmente pronto ainda"
fi

# Verifica MinIO
if curl -f http://localhost:9000/minio/health/live > /dev/null 2>&1; then
    echo "✅ MinIO iniciado com sucesso!"
    echo ""
    echo "📊 Informações de Acesso:"
    echo ""
    echo "🐘 PostgreSQL:"
    echo "   Host: localhost:5432"
    echo "   Database: wamex"
    echo "   User: postgres"
    echo "   Password: postgres"
    echo ""
    echo "🗄️  MinIO:"
    echo "   API URL: http://localhost:9000"
    echo "   Console: http://localhost:9001"
    echo "   Usuário: wamex"
    echo "   Senha: wamex123456"
    echo ""
    echo "📁 Buckets MinIO (criar manualmente ou via código):"
    echo "   - wamex-media: Armazenamento de mídias enviadas"
    echo "   - wamex-temp: Arquivos temporários"
    echo "   - wamex-thumbnails: Miniaturas de imagens"
    echo ""
    echo "🔧 Para parar: docker-compose down"
    echo "📊 Para ver logs: docker-compose logs -f"
else
    echo "❌ Erro ao iniciar MinIO. Verifique os logs:"
    echo "   docker-compose logs minio"
fi
