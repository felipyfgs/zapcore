#!/bin/bash

# Script para executar o ZapCore com Docker

set -e

echo "🚀 Iniciando ZapCore..."

# Verificar se o Docker está rodando
if ! docker info > /dev/null 2>&1; then
    echo "❌ Docker não está rodando. Por favor, inicie o Docker primeiro."
    exit 1
fi

# Verificar se o docker-compose está disponível
if ! command -v docker-compose &> /dev/null; then
    echo "❌ docker-compose não encontrado. Por favor, instale o docker-compose."
    exit 1
fi

# Criar diretórios necessários
echo "📁 Criando diretórios necessários..."
mkdir -p logs uploads migrations

# Parar containers existentes se estiverem rodando
echo "🛑 Parando containers existentes..."
docker-compose down --remove-orphans

# Build e start dos containers
echo "🔨 Fazendo build e iniciando containers..."
docker-compose up --build -d

# Aguardar os serviços ficarem prontos
echo "⏳ Aguardando serviços ficarem prontos..."
sleep 10

# Verificar status dos containers
echo "📊 Status dos containers:"
docker-compose ps

# Mostrar logs da aplicação
echo "📝 Logs da aplicação (Ctrl+C para sair):"
echo "----------------------------------------"
docker-compose logs -f zapcore
