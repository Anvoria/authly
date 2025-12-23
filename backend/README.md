# Authly Backend

The core API service for Authly, providing the authentication logic, OAuth2/OIDC provider implementation, and data management.

## Tech Stack

- **Language**: Go
- **Web Framework**: Fiber
- **Database**: PostgreSQL with GORM
- **Caching**: Redis

## Getting Started

### Prerequisites

- Go 1.25 or higher
- PostgreSQL 16+
- Redis 7+

### Development

1. **Install dependencies**:
   ```bash
   go mod download
   ```

2. **Configuration**:
   Copy `.env.example` to `.env` and ensure `config.yaml` is correctly set up for your local environment:
   ```yaml
   # config.yaml
   database:
     host: "localhost"
     port: 5432
     # ...
   redis:
     host: "localhost"
     # ...
   ```

3. **Run Migrations**:
   The application runs migrations automatically on startup, or you can use the CLI tool.

4. **Start the Server**:
   For development with hot-reload (requires [Air](https://github.com/air-verse/air)):
   ```bash
   air
   ```
   
   Or standard Go run:
   ```bash
   go run cmd/main.go
   ```

   The API will be available at `http://localhost:8000`.

## CLI Tool

The backend includes a CLI for managing the system without direct DB access.

Build the CLI:
```bash
go build -o bin/authly-cli cmd/authly-cli/main.go
```

Usage:
```bash
# Generate new signing keys
./bin/authly-cli keys generate

# Rotate keys
./bin/authly-cli keys rotate

# View help
./bin/authly-cli --help
```

## Testing

Run the test suite:
```bash
go test ./...
```
