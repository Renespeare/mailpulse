# MailPulse SMTP Relay 📧

**Secure Authenticated SMTP Server**

## 🔐 Authentication Required

MailPulse requires authentication for every connection. It will **never** relay emails without valid credentials.

## What is MailPulse Relay?

MailPulse is a secure Go-based SMTP server that:
- ✅ Requires mandatory authentication (API key + password)
- ✅ Forwards emails to configured upstream SMTP servers
- ✅ Provides comprehensive monitoring and logging
- ✅ Enforces rate limits and quotas
- ✅ Uses TLS/STARTTLS encryption
- ❌ **NEVER** accepts anonymous connections
- ❌ **NEVER** acts as an open relay

## Quick Start

### Prerequisites
- Go 1.21+
- PostgreSQL 15+

### Environment Variables
```bash
# Required
DATABASE_URL=postgres://user:pass@localhost:5432/mailpulse
ENCRYPTION_KEY=your-32-character-encryption-key

# SMTP Configuration
SMTP_PORT=2525
HTTP_PORT=8080
SMTP_TLS_REQUIRED=true
```

### Run Locally
```bash
cd relay
go mod download
go run cmd/main.go
```

### Docker
```bash
docker-compose up relay
```

## Authentication Flow

Every SMTP connection must authenticate:

```bash
# 1. Connect to server
telnet your-server.com 2525
> 220 MailPulse SMTP Server Ready

# 2. Start TLS (required)
EHLO client.example.com
> 250-AUTH PLAIN LOGIN
> 250 STARTTLS
STARTTLS
> 220 Ready to start TLS

# 3. Authenticate with API key (base64 encoded)
AUTH PLAIN AG1wX2xpdmVfY2M3ZTcyNjY5NzVmZGU4Njk5Y2RhZDkyNjE3NDc4NGUAc2FkaS1iaW5vLXNhdm8tNDYxNg==
> 235 Authentication successful

# Note: The base64 string encodes: \0mp_live_your-api-key\0your-password

# 4. Now you can send emails
MAIL FROM:<sender@example.com>
> 250 OK
```

**Without authentication:**
```bash
MAIL FROM:<sender@example.com>
> 530 Authentication required
```

For complete examples of sending email, see [docs/SENDING_EMAIL.md](../docs/SENDING_EMAIL.md)

## API Endpoints

### Health Check
```bash
curl http://localhost:8080/health
```

### Email Management
```bash
# List emails with pagination (optional ?project=projectId filter)
curl http://localhost:8080/api/emails

# Get email statistics for a project
curl http://localhost:8080/api/emails/stats/{projectId}

# Resend failed email
curl -X POST http://localhost:8080/api/emails/{emailId}/resend
```

### Project Management
```bash
# List all projects
curl http://localhost:8080/api/projects

# Create new project
curl -X POST http://localhost:8080/api/projects \
  -H "Content-Type: application/json" \
  -d '{"name":"My App","password":"secret","quotaDaily":500,"quotaPerMinute":10}'

# Get specific project
curl http://localhost:8080/api/projects/{projectId}

# Update project settings
curl -X PATCH http://localhost:8080/api/projects/{projectId} \
  -H "Content-Type: application/json" \
  -d '{"name":"Updated Name"}'

# Delete project (soft delete)
curl -X DELETE http://localhost:8080/api/projects/{projectId}
```

### Quota Monitoring
```bash
# Get real-time quota usage and limits
curl http://localhost:8080/api/quota/{projectId}
```

## Security Features

### Mandatory Authentication
- Every connection requires valid API key
- Passwords hashed with bcrypt
- Failed auth attempts are rate limited
- All attempts logged for audit

### Rate Limiting
- Per-project quotas (daily/per-minute)
- In-memory quota tracking
- Automatic blocking on quota exceeded
- Configurable limits per project

### Encryption
- TLS/STARTTLS required for all connections
- AES-256 for sensitive data storage
- Secure API key generation
- Optional email content encryption

### Audit Logging
- Complete connection logs
- Authentication attempt tracking
- Email send/failure records
- Security event monitoring

## Configuration

### Project Setup
Projects are configured via the dashboard or API:
```json
{
  "name": "My App",
  "apiKey": "mp_live_...",
  "password": "password",
  "smtpHost": "smtp.gmail.com",
  "smtpPort": 587,
  "quotaDaily": 500,
  "quotaPerMinute": 10
}
```

### SMTP Client Configuration

For complete examples in multiple languages, see [docs/SENDING_EMAIL.md](../docs/SENDING_EMAIL.md).

## Architecture

```
┌──────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   Your App       │     │  MailPulse      │     │   Gmail/Outlook │
│                  │────▶│  SMTP Relay     │────▶│   etc.          │
│ (API key & pass) │     │ (AUTH REQUIRED) │     │                 │
└──────────────────┘     └─────────────────┘     └─────────────────┘
                               │
                               ▼
                        ┌─────────────────┐
                        │   PostgreSQL    │
                        │ (Email + Quotas)│
                        └─────────────────┘
```

## Not an Open Relay - Legal Notice

### What This Service IS:
- ✅ Authenticated SMTP forwarding service
- ✅ Email monitoring and analytics platform  
- ✅ Secure relay requiring valid credentials
- ✅ Tool for monitoring your own email sending

### What This Service is NOT:
- ❌ Open relay accepting anonymous connections
- ❌ Service for sending spam or unsolicited emails
- ❌ Way to bypass email authentication
- ❌ Public email sending service

### Authentication Requirements
Every SMTP connection must provide:
1. ✅ Valid API key (username)
2. ✅ Valid password (project-specific)
3. ✅ TLS/STARTTLS encryption

**Without these, connections are immediately rejected with 530 Authentication required.**

### Relay Restrictions
Even with valid authentication, MailPulse:
- Only forwards to **your** configured upstream SMTP server
- Does **not** relay to arbitrary destinations
- Enforces sender verification
- Applies strict rate limits and quotas
- Logs all activity for monitoring


## Development

### Project Structure
```
relay/
├── cmd/
│   └── main.go              # Application entry point
├── internal/
│   ├── api/                 # HTTP API server
│   ├── auth/                # Authentication & authorization
│   ├── smtp/                # SMTP server implementation
│   ├── storage/             # PostgreSQL integration
│   ├── security/            # Rate limiting & security
│   └── queue/               # Email retry queue
├── go.mod
└── README.md
```

### Building
```bash
# Development build
go build -o mailpulse-relay cmd/main.go

# Production build with optimizations
CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o mailpulse-relay cmd/main.go
```

### Testing
```bash
# Run all tests
go test ./...

# Test with coverage
go test -cover ./...

# Benchmark tests
go test -bench=. ./...
```

## Monitoring

### Metrics Available
- Email send success/failure rates
- Authentication attempt statistics  
- Rate limiting events
- Connection counts and duration
- Queue depth and processing times

### Health Checks
```bash
# Basic health
curl http://localhost:8080/health
```

## Troubleshooting

### Common Issues

**"Authentication Required" Error**
- Verify API key is correct
- Ensure password matches project configuration
- Check TLS/STARTTLS is enabled
- Review audit logs for failed attempts

**Connection Refused**
- Check if relay is running: `curl http://localhost:8080/health`
- Verify port 2525 is open
- Check firewall rules
- Review Docker/service logs

**Rate Limited**
- Check project quota settings
- Review current usage in dashboard
- Check for abuse/unusual traffic
- Review in-memory quota limits

### Debug Mode
```bash
# Enable debug logging
LOG_LEVEL=debug go run cmd/main.go

# Or with Docker
docker-compose -f docker-compose.yml -f docker-compose.debug.yml up relay
```

## License

MIT License - see [LICENSE](../LICENSE) file for details.

---

**Security Note:** MailPulse requires authentication for all connections.