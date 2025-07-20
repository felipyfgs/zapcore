-- Script de inicialização do banco de dados
-- Este arquivo será executado primeiro pelo PostgreSQL

-- Criar extensão para UUID se não existir
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Criar função para atualizar updated_at (será usada por todas as tabelas)
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Log de inicialização
DO $$
BEGIN
    RAISE NOTICE 'ZapCore Database initialized successfully!';
END $$;
