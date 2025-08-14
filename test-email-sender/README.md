# MailPulse Email Sender (Go)

A Go script to send emails through your MailPulse SMTP relay server.

## Setup

1. **Get your project credentials** from MailPulse dashboard:
   - API Key (from project card)
   - Project Password (from project creation)

2. **Update configuration** either in code or via environment variables

## Usage

### Option 1: Environment Variables (Recommended)
```bash
export MAILPULSE_API_KEY="mp_live_your_api_key_here"
export MAILPULSE_PASSWORD="your_project_password"
export FROM_EMAIL="sender@yourdomain.com"
export TO_EMAIL="recipient@gmail.com"

# Optional
export MAILPULSE_HOST="localhost"  # Default: localhost
export MAILPULSE_PORT="2525"       # Default: 2525

go run main.go
```

### Option 2: Edit Code Directly
Update the `config` struct in `main.go`:
```go
config := EmailConfig{
    SMTPHost: "localhost",                    // MailPulse relay server
    SMTPPort: "2525",                        // MailPulse relay port
    Username: "your-api-key-here",           // Your project API key
    Password: "your-project-password-here",   // Your project password  
    From:     "sender@yourdomain.com",
    To:       []string{"recipient@gmail.com"},
    Subject:  "Test Email from MailPulse Go Client",
    Body:     "Your email content here...",
}
```

Then run:
```bash
go run main.go
```


## Expected Output

```
🔌 Connecting to MailPulse relay at localhost:2525
🔑 Authenticating with API key: mp_live_...
✅ Email sent successfully!
📧 From: sender@yourdomain.com
📧 To: recipient@gmail.com
📧 Subject: Test Email from MailPulse Go Client
🔧 Via: localhost:2525
```

## Features

- ✅ SMTP authentication using API key and password
- ✅ Proper RFC 822 email formatting
- ✅ Support for multiple recipients
- ✅ Environment variable configuration
- ✅ Detailed logging and error handling
- ✅ No external dependencies (uses Go standard library)

## Troubleshooting

- **Authentication failed**: Check your API key and project password
- **Connection refused**: Make sure MailPulse relay server is running on port 2525  
- **Email not delivered**: Check MailPulse logs and your SMTP provider configuration