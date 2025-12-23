# Authly

![Authly Banner](.github/assets/banner.png)

Authly is a robust, self-hosted authentication and authorization service designed for modern applications. It provides a secure, compliant, and developer-friendly foundation for managing user identities and access controls.

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go 1.25+](https://img.shields.io/badge/go-1.25+-blue.svg)](https://go.dev/dl/)
[![Node.js 20+](https://img.shields.io/badge/node.js-20+-green.svg)](https://nodejs.org/)
[![Next.js](https://img.shields.io/badge/Next.js-16+-black.svg)](https://nextjs.org/)
[![CI](https://github.com/Anvoria/authly/actions/workflows/ci-backend.yml/badge.svg)](https://github.com/Anvoria/authly/actions/workflows/ci-backend.yml)

## Features

### Authentication & Sessions
- **Secure Authentication**: Complete login and registration flows.
- **Session Management**: HttpOnly cookies for secure session handling, preventing XSS attacks.
- **Hybrid Security Model**: Combines standard OAuth2 token flows for clients with secure cookie-based sessions for the frontend SPA.

### Authorization (OAuth2 / OIDC)
- **Standard Compliance**: Implements RFC 6749 (OAuth 2.0) and OpenID Connect.
- **Grant Types**: Supports Authorization Code, Refresh Token, Client Credentials, and Password grants.
- **PKCE Support**: Enforced Proof Key for Code Exchange (RFC 7636) for enhanced security on public clients.
- **Token Rotation**: Secure refresh token rotation to mitigate token theft.

### Access Control
- **RBAC**: Role-Based Access Control for managing user permissions.
- **Fine-Grained Permissions**: Detailed permission scoping for services and resources.

## Getting Started

### Prerequisites
- Docker and Docker Compose
- Go 1.25+ (for local development)
- Node.js 20+ (for local development)

### Running with Docker

1. Clone the repository:
   ```bash
   git clone https://github.com/Anvoria/authly.git
   cd authly
   ```

2. Start the services:
   ```bash
   docker compose up -d
   ```

The services will be available at:
- **Frontend**: http://localhost:3000
- **Backend API**: http://localhost:8000

## CLI

Authly includes a CLI tool for administrative tasks and key management.

To build and use the CLI:

```bash
cd backend
go build -o bin/authly-cli cmd/authly-cli/main.go
./bin/authly-cli --help
```

Available commands:
- `keys`: Manage JWK signing keys (generate, rotate).
- `admin`: Administrative tasks for user and system management.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
