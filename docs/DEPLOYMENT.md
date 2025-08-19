# MailPulse Deployment Guide

## Quick Start (5 Minutes)

### Prerequisites
- Docker & Docker Compose
- 2GB RAM minimum
- Open ports: 2525 (SMTP), 3000 (Dashboard), 8080 (API)

### 1. Clone and Configure

```bash
git clone <your-mailpulse-repo>
cd mailpulse
cp .env.example .env
```

### 2. Update Environment Variables

```bash
# .env file
POSTGRES_PASSWORD=your-secure-password-here
ENCRYPTION_KEY=your-32-character-encryption-key
```

### 3. Start Services

```bash
docker-compose up -d
```

### 4. Access Dashboard

Wait for all services to start (check with `docker-compose ps`).

### 5. Create First User

Visit http://localhost:3000 and register your admin account.

## Production Deployment

### Security Hardening

#### 1. Environment Variables
```bash
# Use strong, unique passwords
POSTGRES_PASSWORD=$(openssl rand -base64 32)
ENCRYPTION_KEY=$(openssl rand -base64 32)

# Production URLs
VITE_RELAY_API_URL=https://your-domain.com:8080
```

#### 2. TLS Configuration
```nginx
# nginx configuration
server {
    listen 443 ssl;
    server_name your-domain.com;
    
    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;
    
    # Dashboard
    location / {
        proxy_pass http://localhost:3000;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
    
    # API
    location /api/ {
        proxy_pass http://localhost:8080/;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}

# SMTP with TLS (port 587 or 465)
server {
    listen 587;
    proxy_pass localhost:2525;
    proxy_ssl on;
}
```

#### 3. Firewall Rules
```bash
# Allow only necessary ports
ufw allow 22    # SSH
ufw allow 80    # HTTP (redirect to HTTPS)
ufw allow 443   # HTTPS
ufw allow 587   # SMTP TLS
ufw enable
```

### Database Configuration

#### PostgreSQL Production Settings
```yaml
# docker-compose.prod.yml
services:
  postgres:
    command: >
      postgres
      -c max_connections=200
      -c shared_buffers=256MB
      -c effective_cache_size=1GB
      -c maintenance_work_mem=64MB
      -c checkpoint_completion_target=0.9
      -c wal_buffers=16MB
      -c default_statistics_target=100
      -c random_page_cost=1.1
    environment:
      POSTGRES_DB: mailpulse
      POSTGRES_USER: mailpulse
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD}
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./backups:/backups
```

#### Backup Strategy
```bash
# Daily backup script
#!/bin/bash
DATE=$(date +%Y%m%d_%H%M%S)
docker-compose exec -T postgres pg_dump -U mailpulse mailpulse > backups/backup_$DATE.sql
find backups/ -name "backup_*.sql" -mtime +7 -delete
```

### Monitoring & Alerts

#### Health Checks
```bash
# Check all services
curl -f http://localhost:3000/api/health || exit 1
curl -f http://localhost:8080/health || exit 1
```

#### Log Aggregation
```yaml
# Add to docker-compose.yml
logging:
  driver: "json-file"
  options:
    max-size: "100m"
    max-file: "3"
```

#### Prometheus Metrics (Optional)
```go
// Add to relay service
import "github.com/prometheus/client_golang/prometheus/promhttp"

http.Handle("/metrics", promhttp.Handler())
```

### Performance Tuning

#### Go Relay Optimization
```dockerfile
# Build with optimizations
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o mailpulse-relay ./cmd/main.go
```

### Scaling

#### Horizontal Scaling
```yaml
# Multiple relay instances
relay1:
  <<: *relay-service
  container_name: mailpulse-relay-1
  
relay2:
  <<: *relay-service  
  container_name: mailpulse-relay-2

# Load balancer
nginx:
  image: nginx
  ports:
    - "2525:2525"
  volumes:
    - ./nginx.conf:/etc/nginx/nginx.conf
```

#### Database Scaling
```yaml
# Read replica
postgres-replica:
  image: postgres:15-alpine
  environment:
    PGUSER: replica
    POSTGRES_PASSWORD: ${REPLICA_PASSWORD}
  command: |
    postgres
    -c wal_level=replica
    -c max_wal_senders=3
    -c max_replication_slots=3
```

## Troubleshooting

### Common Issues

#### "Authentication Required" Error
```bash
# Check API key configuration
docker-compose logs relay | grep -i auth

# Verify database connection
docker-compose exec postgres psql -U mailpulse -d mailpulse -c "SELECT * FROM projects LIMIT 1;"
```

#### High Memory Usage
```bash
# Check container resources
docker stats

# Tune PostgreSQL memory
# Adjust shared_buffers and effective_cache_size
```

#### SMTP Connection Refused
```bash
# Check if relay is running
docker-compose ps relay
curl -f http://localhost:8080/health

# Check firewall
sudo ufw status
netstat -tlnp | grep 2525
```

### Maintenance

#### Updates
```bash
# Backup before updates
./backup.sh

# Pull latest images
docker-compose pull

# Restart services
docker-compose up -d
```

#### Log Rotation
```bash
# Setup logrotate
cat > /etc/logrotate.d/mailpulse << EOF
/var/lib/docker/containers/*/*-json.log {
    daily
    rotate 7
    compress
    missingok
    notifempty
    create 0644 root root
    postrotate
        docker kill -s USR1 $(docker ps -q)
    endscript
}
EOF
```

---

**Remember: Always test deployments in staging first. MailPulse requires authentication - it's not an open relay.**