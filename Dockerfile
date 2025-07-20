# Build stage
FROM golang:1.23-alpine AS builder

# Instalar dependências necessárias
RUN apk add --no-cache git ca-certificates tzdata gcc musl-dev

# Definir diretório de trabalho
WORKDIR /app

# Configurar proxy Go se necessário
ENV GOPROXY=https://proxy.golang.org,direct
ENV GOSUMDB=sum.golang.org

# Copiar arquivos de dependências
COPY go.mod go.sum ./

# Baixar dependências com retry
RUN go mod download || (sleep 5 && go mod download) || (sleep 10 && go mod download)

# Copiar código fonte
COPY . .

# Build da aplicação
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/server

# Production stage
FROM alpine:latest

# Instalar ca-certificates para HTTPS
RUN apk --no-cache add ca-certificates tzdata

# Criar usuário não-root
RUN addgroup -g 1001 -S zapcore && \
    adduser -u 1001 -S zapcore -G zapcore

# Definir diretório de trabalho
WORKDIR /app

# Copiar binário do stage de build
COPY --from=builder /app/main .

# Criar diretórios necessários
RUN mkdir -p logs uploads sessions && \
    chown -R zapcore:zapcore /app

# Mudar para usuário não-root
USER zapcore

# Expor porta
EXPOSE 8080

# Comando para executar a aplicação
CMD ["./main"]
