# WhatsApp Notifier

A robust WhatsApp notification service built with Go that allows you to send WhatsApp messages programmatically via HTTP API. It uses the WhatsApp Web protocol through the `whatsmeow` library to provide a reliable messaging interface.

## Features

- 🚀 **HTTP API**: Send WhatsApp messages via REST API
- 📱 **QR Code Authentication**: Easy WhatsApp Web login via QR code
- 👥 **Contact & Group Management**: Retrieve contacts and joined groups
- 🔄 **Auto Reconnection**: Automatic reconnection with exponential backoff
- 🛡️ **Security**: API key authentication and input validation
- 📊 **Health Monitoring**: Health check endpoints with connection status
- 🗂️ **Structured Logging**: JSON and text logging with configurable levels
- 💾 **Persistent Sessions**: SQLite database for session storage
- ⚡ **Graceful Shutdown**: Clean shutdown handling with proper resource cleanup

## Quick Start

### Prerequisites

- Go 1.21+ 
- SQLite3 (for session storage)
- WhatsApp account

### Installation

1. **Clone the repository**:
   ```bash
   git clone https://github.com/nahidhasan98/whatsapp-notifier.git
   cd whatsapp-notifier
   ```

2. **Install dependencies**:
   ```bash
   go mod download
   ```

3. **Configure environment** (optional):
   ```bash
   cp .env.example .env
   # Edit .env with your configuration
   ```

4. **Build the application**:
   ```bash
   go build -o bin/whatsapp-notifier cmd/server/main.go
   ```

5. **Run the service**:
   ```bash
   ./bin/whatsapp-notifier
   ```

6. **Authenticate with WhatsApp**:
   - On first run, scan the QR code displayed in the terminal with your WhatsApp mobile app
   - Go to WhatsApp > Settings > Linked Devices > Link a Device

## Configuration

The application can be configured using environment variables or a `.env` file:

### Server Configuration
```bash
SERVER_HOST=0.0.0.0              # Server bind address (default: "")
SERVER_PORT=8080                 # Server port (default: 8080)
SERVER_READ_TIMEOUT=15s          # HTTP read timeout (default: 15s)
SERVER_WRITE_TIMEOUT=15s         # HTTP write timeout (default: 15s)
SERVER_SHUTDOWN_TIMEOUT=10s      # Graceful shutdown timeout (default: 10s)
```

### Database Configuration
```bash
DB_DRIVER=sqlite3                                    # Database driver (default: sqlite3)
DB_DSN=file:mywhatsapp.db?_foreign_keys=on          # Database connection string
```

### WhatsApp Configuration
```bash
WHATSAPP_LOG_LEVEL=INFO          # WhatsApp client log level (default: INFO)
WHATSAPP_DEVICE_NAME="macOS"     # Custom device name shown in WhatsApp (default: "macOS")
```

### Logging Configuration
```bash
LOG_LEVEL=info                   # Application log level (default: info)
LOG_FORMAT=text                  # Log format: "json" or "text" (default: text)
```

### Security Configuration
```bash
API_KEYS=api-key-123,api-key-456,api-key-789   # Comma-separated API keys
```

**⚠️ Important**: Set secure API keys before deploying to production. The default keys will cause validation errors.

### Gitea Webhook Configuration
```bash
GITEA_WEBHOOK_SECRET=webhook-secret         # Webhook secret for authentication
GITEA_RECIPIENT=1234567890@s.whatsapp.net   # WhatsApp JID to receive notifications
```

## API Endpoints

### Authentication
All API endpoints require authentication via the `X-API-Key` header:
```bash
curl -H "X-API-Key: your-secure-api-key" http://localhost:8080/health
```

### Health Check
```http
GET /health
```

**Response**:
```json
{
  "status": "ok",
  "connected": true,
  "timestamp": 1698765432
}
```

**Detailed health check**:
```http
GET /health?detailed=true
```

### Send Message
```http
POST /send
Content-Type: application/json
X-API-Key: your-secure-api-key

{
  "to": "1234567890@s.whatsapp.net",
  "message": "Hello from WhatsApp Notifier!"
}
```

**Response**:
```json
{
  "status": "sent",
  "to": "1234567890@s.whatsapp.net",
  "timestamp": 1698765432
}
```

### Get Contacts
```http
GET /contacts
X-API-Key: your-secure-api-key
```

### Get Groups
```http
GET /groups
X-API-Key: your-secure-api-key
```

### Gitea Webhook
Receive push notifications from Gitea repositories and forward them to WhatsApp.

```http
POST /webhook/gitea
Content-Type: application/json

{
  "secret": "your-webhook-secret-here",
  "ref": "refs/heads/main",
  "repository": {
    "name": "my-repo",
    "full_name": "owner/my-repo",
    "html_url": "https://git.example.com/owner/my-repo"
  },
  "pusher": {
    "username": "john_doe",
    "full_name": "John Doe"
  },
  "commits": [
    {
      "id": "abc123def456",
      "message": "Fix bug in authentication",
      "url": "https://git.example.com/owner/my-repo/commit/abc123def456",
      "author": {
        "name": "John Doe",
        "email": "john@example.com"
      },
      "timestamp": "2024-01-15T10:30:00Z"
    }
  ],
  "compare_url": "https://git.example.com/owner/my-repo/compare/old...new"
}
```

**Setup in Gitea**:
1. Go to repository Settings > Webhooks
2. Add webhook with URL: `http://your-server:8080/webhook/gitea`
3. Set Content Type: `application/json`
4. Add the webhook payload with your secret
5. Choose "Push events" as trigger

**Response**:
```json
{
  "status": "notification sent"
}
```

## JID Format

WhatsApp uses JID (Jabber ID) format for addressing:

- **Individual contacts**: `[country_code][phone_number]@s.whatsapp.net`
  - Example: `1234567890@s.whatsapp.net` (US number: +1-234-567-890)
- **Group chats**: `[group_id]@g.us`
  - Example: `120363025343298765@g.us`

## Usage Examples

### Send a simple message
```bash
curl -X POST http://localhost:8080/send \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-secure-api-key" \
  -d '{
    "to": "1234567890@s.whatsapp.net",
    "message": "Hello, World!"
  }'
```

### Check service health
```bash
curl -H "X-API-Key: your-secure-api-key" http://localhost:8080/health
```

### Get all contacts
```bash
curl -H "X-API-Key: your-secure-api-key" http://localhost:8080/contacts
```

### Test Gitea webhook
```bash
curl -X POST http://localhost:8080/webhook/gitea \
  -H "Content-Type: application/json" \
  -d '{
    "secret": "your-webhook-secret-here",
    "ref": "refs/heads/main",
    "repository": {
      "name": "test-repo",
      "full_name": "user/test-repo"
    },
    "pusher": {
      "username": "johndoe"
    },
    "commits": [
      {
        "id": "abc123",
        "message": "Test commit"
      }
    ]
  }'
```

## Troubleshooting

### Common Issues

**Service fails to start**:
- Check logs: `sudo journalctl -u whatsapp-notifier.service -f`
- Verify user permissions on data directory
- Ensure database file is writable

**WhatsApp connection fails**:
- Re-scan QR code: Stop service, run interactively, scan QR, restart service
- Check WhatsApp Web session limits (max 4 linked devices)
- Verify internet connectivity

**API requests fail**:
- Verify API key in request headers
- Check service is running: `curl localhost:8080/health`
- Review application logs for errors

**Database errors**:
- Ensure SQLite is installed
- Check database file permissions
- Verify database directory exists and is writable

### Logging

View service logs:
```bash
# Real-time logs
sudo journalctl -u whatsapp-notifier.service -f

# Recent logs
sudo journalctl -u whatsapp-notifier.service -n 100

# Logs since boot
sudo journalctl -u whatsapp-notifier.service -b
```

## Acknowledgments

- [whatsmeow](https://github.com/tulir/whatsmeow) - WhatsApp Web Multi-Device API library
- [zerolog](https://github.com/rs/zerolog) - Fast and structured logging
- [godotenv](https://github.com/joho/godotenv) - Environment variable loading

## Disclaimer

This project is not affiliated with WhatsApp Inc. Use at your own risk and ensure compliance with WhatsApp's Terms of Service.