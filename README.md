# WAMEX - WhatsApp Message Exchange System

A robust WhatsApp messaging system built with Go, featuring Clean Architecture and multi-source media support.

## Features

### Core Functionality
- **WhatsApp Session Management** with QR code authentication
- **Multi-format Messaging** (text, image, audio, document)
- **Multi-source Media Support** (Base64, URL, MinIO, Upload)
- **Clean Architecture** with domain-driven design
- **RESTful API** with OpenAPI 3.0 documentation
- **PostgreSQL + MinIO** integration

### Architecture
- **Domain Layer**: Entities and business interfaces
- **Use Case Layer**: Business logic orchestration
- **Infrastructure Layer**: Database, storage, WhatsApp client
- **Transport Layer**: HTTP handlers and middleware

## Tech Stack

- **Go 1.24.4** with idiomatic project structure
- **Chi router** for HTTP routing
- **whatsmeow** for WhatsApp Web API
- **Bun ORM** for database operations
- **PostgreSQL** for data persistence
- **MinIO** for object storage
- **Docker Compose** for development environment

## Quick Start

### Prerequisites
- Go 1.24.4+
- Docker & Docker Compose
- PostgreSQL
- MinIO

### Installation

1. **Clone the repository**
```bash
git clone https://github.com/felipyfgs/wamex.git
cd wamex
```

2. **Start services**
```bash
docker-compose up -d
```

3. **Build and run**
```bash
go build -o wamex cmd/wamex/main.go
./wamex
```

4. **Access API documentation**
- OpenAPI: http://localhost:8080/docs
- Health check: http://localhost:8080/health

## API Usage

### Authentication
1. **Get QR Code**
```bash
GET /api/v1/whatsapp/qr
```

2. **Check Session Status**
```bash
GET /api/v1/whatsapp/status
```

### Messaging
1. **Send Text Message**
```bash
POST /api/v1/messages/text
{
  "phone": "5511999999999",
  "message": "Hello World!"
}
```

2. **Send Media Message**
```bash
POST /api/v1/messages/media
{
  "phone": "5511999999999",
  "media_type": "image",
  "media_source": "base64",
  "media_data": "data:image/jpeg;base64,..."
}
```

## Project Structure

```
wamex/
├── cmd/wamex/              # Application entry point
├── internal/
│   ├── domain/             # Business entities and interfaces
│   ├── usecase/            # Business logic orchestration
│   ├── infra/              # Infrastructure implementations
│   └── transport/          # HTTP handlers and middleware
├── pkg/                    # Reusable packages
├── api/                    # API documentation
├── docs/                   # Project documentation
├── test/                   # Test files
└── docker-compose.yml      # Development environment
```

## Configuration

Environment variables:
```bash
# Database
DB_HOST=localhost
DB_PORT=5432
DB_NAME=wamex
DB_USER=postgres
DB_PASSWORD=postgres

# MinIO
MINIO_ENDPOINT=localhost:9000
MINIO_ACCESS_KEY=minioadmin
MINIO_SECRET_KEY=minioadmin

# Server
SERVER_PORT=8080
```

## Development

### Running Tests
```bash
go test ./...
```

### Building for Production
```bash
go build -ldflags="-s -w" -o wamex cmd/wamex/main.go
```

### Docker Build
```bash
docker build -t wamex:latest .
```

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Support

For support, please open an issue on GitHub or contact the maintainers.

---

**WAMEX v0.0.1** - Foundation release with all core functionality implemented and tested.
