# WhatsApp Notifier

A robust WhatsApp notification service built with Go that allows you to send WhatsApp messages programmatically via HTTP API. It uses the WhatsApp Web protocol through the `whatsmeow` library to provide a reliable messaging interface.

## Features

- 🚀 **HTTP API**: Send WhatsApp messages via REST API
- 📱 **QR Code Authentication**: Easy WhatsApp Web login via QR code
- 👥 **Contact & Group Management**: Retrieve contacts and joined groups
- 🔄 **Auto Reconnection**: Automatic reconnection with exponential backoff
- 🛡️ **Security**: API key authentication, HMAC signature verification, and input validation
- 📊 **Health Monitoring**: Health check endpoints with connection status
- 🗂️ **Structured Logging**: JSON and text logging with configurable levels, file output support
- 💾 **Persistent Sessions**: SQLite database for session storage
- ⚡ **Graceful Shutdown**: Clean shutdown handling with proper resource cleanup
- 🔔 **Webhook Integration**: Receive notifications from Gitea and GitHub repositories

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
LOG_LEVEL=info                          # Application log level (default: info)
LOG_FORMAT=text                         # Log format: "json" or "text" (default: text)
LOG_FILE=./logs/whatsapp-notifier.log   # Log file path (default: ./logs/whatsapp-notifier.log)
```

### Security Configuration
```bash
API_KEYS=api-key-123,api-key-456,api-key-789   # Comma-separated API keys
```

**⚠️ Important**: Set secure API keys before deploying to production. The default keys will cause validation errors.

### Webhook Configuration

#### Gitea Webhook
```bash
GITEA_WEBHOOK_SECRET=gitea-webhook-secret    # Secret for HMAC SHA256 signature verification
GITEA_RECIPIENT=1234567890@s.whatsapp.net    # WhatsApp JID to receive notifications
```

#### GitHub Webhook
```bash
GITHUB_WEBHOOK_SECRET=github-webhook-secret  # Secret for HMAC SHA256 signature verification
GITHUB_RECIPIENT=1234567890@s.whatsapp.net   # WhatsApp JID to receive notifications
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
X-Gitea-Signature: <hmac-sha256-signature>
```

**Request body**:
```json
{
  "ref": "refs/heads/main",
  "repository": {
    "name": "my-repo",
    "full_name": "owner/my-repo"
  },
  "pusher": {
    "name": "John Doe"
  },
  "commits": [
    {
      "id": "abc123def456",
      "message": "Fix bug in authentication",
      "url": "https://git.example.com/owner/my-repo/commit/abc123def456"
    }
  ],
  "compare_url": "https://git.example.com/owner/my-repo/compare/old...new"
}
```

**Setup in Gitea**:
1. Go to repository Settings > Webhooks > Add Webhook
2. Set Payload URL: `http://your-server:8080/webhook/gitea`
3. Set Content Type: `application/json`
4. Set Secret: Use the same value as `GITEA_WEBHOOK_SECRET`
5. Choose "Push events" as trigger
6. Click "Add Webhook"

**Response**:
```json
{
  "status": "notification sent"
}
```

**WhatsApp notification format**:
```
🔔 *New Push to owner/my-repo*

👤 Pusher: John Doe
🌿 Branch: main
📊 Commits: 1

*Commits:*
• abc123d - Fix bug in authentication

🔗 View changes: https://git.example.com/owner/my-repo/compare/old...new
```

### GitHub Webhook
Receive push notifications from GitHub repositories and forward them to WhatsApp.

```http
POST /webhook/github
Content-Type: application/json
X-Hub-Signature-256: sha256=<hmac-sha256-signature>
```

**Request body**:
```json
{
  "ref": "refs/heads/main",
  "repository": {
    "name": "my-repo",
    "full_name": "owner/my-repo"
  },
  "pusher": {
    "name": "John Doe"
  },
  "commits": [
    {
      "id": "abc123def456",
      "message": "Fix bug in authentication",
      "url": "https://github.com/owner/my-repo/commit/abc123def456",
      "added": ["newfile.txt"],
      "modified": ["README.md"],
      "removed": ["oldfile.txt"]
    }
  ],
  "compare": "https://github.com/owner/my-repo/compare/old...new"
}
```

**Setup in GitHub**:
1. Go to repository Settings > Webhooks > Add webhook
2. Set Payload URL: `http://your-server:8080/webhook/github`
3. Set Content type: `application/json`
4. Set Secret: Use the same value as `GITHUB_WEBHOOK_SECRET`
5. Select "Just the push event"
6. Ensure "Active" is checked
7. Click "Add webhook"

**Response**:
```json
{
  "status": "notification sent"
}
```

**WhatsApp notification format**:
```
🔔 *New Push to owner/my-repo*

👤 Pusher: John Doe
🌿 Branch: main
📊 Commits: 1

*Commits:*
• abc123d - Fix bug in authentication

📁 *Files Changed:*
✅ Added: newfile.txt
📝 Modified: README.md
❌ Removed: oldfile.txt

🔗 View changes: https://github.com/owner/my-repo/compare/old...new
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
# Calculate HMAC signature
SECRET="your-webhook-secret"
PAYLOAD='{"ref":"refs/heads/main","repository":{"full_name":"user/test-repo"},"pusher":{"name":"John Doe"},"commits":[{"id":"abc123","message":"Test commit","url":"https://example.com"}],"compare_url":"https://example.com/compare"}'
SIGNATURE=$(echo -n "$PAYLOAD" | openssl dgst -sha256 -hmac "$SECRET" | sed 's/^.* //')

# Send webhook request
curl -X POST http://localhost:8080/webhook/gitea \
  -H "Content-Type: application/json" \
  -H "X-Gitea-Signature: $SIGNATURE" \
  -d "$PAYLOAD"
```

### Test GitHub webhook
```bash
# Calculate HMAC signature
SECRET="your-webhook-secret"
PAYLOAD='{"ref":"refs/heads/main","repository":{"full_name":"user/test-repo"},"pusher":{"name":"John Doe"},"commits":[{"id":"abc123","message":"Test commit","url":"https://example.com","added":[],"modified":["README.md"],"removed":[]}],"compare":"https://example.com/compare"}'
SIGNATURE="sha256=$(echo -n "$PAYLOAD" | openssl dgst -sha256 -hmac "$SECRET" | sed 's/^.* //')"

# Send webhook request
curl -X POST http://localhost:8080/webhook/github \
  -H "Content-Type: application/json" \
  -H "X-Hub-Signature-256: $SIGNATURE" \
  -d "$PAYLOAD"
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

**Webhook signature verification fails**:
- Ensure the webhook secret matches in both service configuration and webhook settings
- Verify the signature header format (Gitea: `X-Gitea-Signature`, GitHub: `X-Hub-Signature-256: sha256=...`)
- Check that payload is sent as raw JSON (not form-encoded)

### Logging

Application logs are written to the configured log file (default: `./logs/whatsapp-notifier.log`).

View logs:
```bash
# Follow log file
tail -f ./logs/whatsapp-notifier.log

# View recent logs
tail -n 100 ./logs/whatsapp-notifier.log

# Search for errors
grep -i error ./logs/whatsapp-notifier.log
```

If running as a systemd service:
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