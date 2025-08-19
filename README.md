# MailPulse 📮

**Secure Self-hosted SMTP Monitoring Dashboard**

[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8.svg)](https://golang.org/)
[![React](https://img.shields.io/badge/React-19-61DAFB.svg)](https://reactjs.org/)

## ⚠️ Important Security Notice

**MailPulse is NOT an open relay.** It requires authentication for every connection and is designed as a secure email forwarding service with comprehensive monitoring capabilities.

## What is MailPulse?

MailPulse is a purpose-built secure SMTP monitoring solution with:

- 🔐 **Mandatory Authentication** - Every SMTP connection requires valid API keys
- 📊 **Email Analytics** - Comprehensive monitoring dashboard
- 🗄️ **Email Storage** - PostgreSQL-backed email logging and search
- 🔄 **Resend Functionality** - Retry failed emails from the dashboard
- 📈 **Quota Management** - Per-project rate limiting and quotas
- 🛡️ **Security First** - Audit logging, encryption, and abuse prevention

## Quick Start

### 1. Clone and Configure
```bash
git clone git@github.com:Renespeare/mailpulse.git
cd mailpulse
cp .env.example .env
# Edit .env with your secure passwords
```

### 2. Start Services
```bash
docker-compose up -d
```

### 3. Access Dashboard
Visit http://localhost:3000 and start creating projects.

## Architecture

```
┌──────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   Client App     │     │  MailPulse      │     │   Upstream      │
│                  │────▶│  SMTP Relay     │────▶│   SMTP Server   │
│ (API key & pass) │     │                 │     │  (Gmail, etc.)  │
└──────────────────┘     │   + HTTP API    │     └─────────────────┘
                         └─────────────────┘
                                  │
                                  ▼
                         ┌─────────────────┐
                         │   PostgreSQL    │
                         │ (Email + Quotas)│
                         └─────────────────┘
                                  ▲
                                  │ (HTTP API)
                                  │
                         ┌─────────────────┐
                         │    Dashboard    │
                         │  (React+Vite)   │
                         └─────────────────┘
```

## Core Components

### SMTP Relay (`relay/`)
- **Go-based** secure SMTP server
- **Authenticated only** - No open relay functionality
- **PostgreSQL integration** for email storage
- **In-memory** rate limiting and quota tracking
- **TLS/STARTTLS** required for all connections

### Dashboard (`dashboard/`)
- **React 19** with TypeScript and Vite
- **Direct API integration** for database operations
- **Built-in authentication** for secure access
- **Tailwind CSS v4** with modern components
- **Real-time** email monitoring and analytics

### Security Features
- **API Key Authentication** - bcrypt-hashed API keys
- **Rate Limiting** - In-memory per-project quotas
- **Audit Logging** - Comprehensive security event tracking
- **Encryption** - AES-256 for sensitive data
- **IP Allowlists** - Optional IP-based restrictions

## Configuration Example

### SMTP Client Configuration

For complete examples, see [docs/SENDING_EMAIL.md](docs/SENDING_EMAIL.md).

For additional test clients, see [test-email-sender/](test-email-sender/) directory.

### Environment Variables
```bash
# Database
DATABASE_URL=postgres://user:pass@localhost:5432/mailpulse

# Security
ENCRYPTION_KEY=your-32-character-encryption-key

# SMTP
SMTP_PORT=2525
HTTP_PORT=8080
SMTP_TLS_REQUIRED=true
```

## Project Structure

```
mailpulse/
├── relay/                 # Go SMTP relay
│   ├── cmd/               # Main application
│   └── internal/          # Core packages
│       ├── auth/          # Authentication
│       ├── storage/       # Database operations
│       ├── security/      # Rate limiting
│       └── smtp/          # SMTP server
├── dashboard/             # React + Vite dashboard
│   ├── src/               # Source files and components
├── docker/                # Docker configurations
├── docs/                  # Documentation
└── docker-compose.yml     # Development setup
```

## Development

### Prerequisites
- Go 1.21+
- Node.js 18+
- PostgreSQL 15+
- Docker & Docker Compose

### Local Development
```bash
# Start database
docker-compose up postgres -d

# Start relay
cd relay
go mod download
go run cmd/main.go

# Start dashboard
cd dashboard
npm install
npm run dev
```

## Production Deployment

See [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md) for comprehensive production setup including:
- TLS/SSL configuration
- Reverse proxy setup
- Database optimization
- Monitoring and alerts
- Backup strategies

## Security

**Critical:** MailPulse requires authentication for all connections. See [relay/README.md](relay/README.md) for security details.

## API Documentation

### Authentication
All API requests require authentication via API key:
```bash
curl -H "Authorization: Bearer mp_live_your-api-key" \
     https://your-server.com/api/emails
```

### API Endpoints

Complete API documentation available in [relay/README.md](relay/README.md#api-endpoints)

**Quick Reference:**
- Health: `GET /health`
- Projects: `GET /api/projects` 
- Emails: `GET /api/emails`
- Quotas: `GET /api/quota/{projectId}`
- Audit Logs: `GET /api/audit`


## License

MIT License - see [LICENSE](LICENSE) file for details.

---

**Remember: MailPulse requires authentication. It is not an open relay.**