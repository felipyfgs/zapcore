@echo off
setlocal enabledelayedexpansion

echo 🚀 Iniciando ZapCore...

REM Verificar se o Docker está rodando
docker info >nul 2>&1
if errorlevel 1 (
    echo ❌ Docker não está rodando. Por favor, inicie o Docker primeiro.
    pause
    exit /b 1
)

REM Verificar se o docker-compose está disponível
docker-compose --version >nul 2>&1
if errorlevel 1 (
    echo ❌ docker-compose não encontrado. Por favor, instale o docker-compose.
    pause
    exit /b 1
)

REM Criar diretórios necessários
echo 📁 Criando diretórios necessários...
if not exist "logs" mkdir logs
if not exist "uploads" mkdir uploads
if not exist "migrations" mkdir migrations

REM Parar containers existentes se estiverem rodando
echo 🛑 Parando containers existentes...
docker-compose down --remove-orphans

REM Build e start dos containers
echo 🔨 Fazendo build e iniciando containers...
docker-compose up --build -d

REM Aguardar os serviços ficarem prontos
echo ⏳ Aguardando serviços ficarem prontos...
timeout /t 10 /nobreak >nul

REM Verificar status dos containers
echo 📊 Status dos containers:
docker-compose ps

REM Mostrar logs da aplicação
echo 📝 Logs da aplicação (Ctrl+C para sair):
echo ----------------------------------------
docker-compose logs -f zapcore
