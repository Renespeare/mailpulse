# Email Sending Example

# SMTP Settings:
SMTP Host: localhost (or your MailPulse server)
SMTP Port: 2525
Username: mp_live_gs7e7526123feg8621chad158294284r (your API key)
Password: dmes-mnsk-laca-4801 (your project password)

Using Python (with smtplib)

import smtplib
from email.mime.text import MIMEText
from email.mime.multipart import MIMEMultipart

# MailPulse SMTP Configuration
SMTP_HOST = "localhost"  # or your MailPulse server IP
SMTP_PORT = 2525
USERNAME = "mp_live_gs7e7526123feg8621chad158294284r"  # Your API key
PASSWORD = "dmes-mnsk-laca-4801"  # Your project password

# Create message
msg = MIMEMultipart()
msg['From'] = "noreply@yourdomain.com"
msg['To'] = "recipient@example.com"
msg['Subject'] = "Test Email via MailPulse"

# Email body
body = """
Hello!

This is a test email sent through MailPulse SMTP relay.

Best regards,
Your Application
"""
msg.attach(MIMEText(body, 'plain'))

try:
    # Connect to MailPulse SMTP server
    server = smtplib.SMTP(SMTP_HOST, SMTP_PORT)
    server.starttls()  # Enable encryption
    server.login(USERNAME, PASSWORD)

    # Send email
    server.send_message(msg)
    server.quit()

    print("Email sent successfully!")

except Exception as e:
    print(f"Failed to send email: {e}")

Using Node.js (with nodemailer)

const nodemailer = require('nodemailer');

// MailPulse SMTP Configuration
const transporter = nodemailer.createTransporter({
    host: 'localhost',  // or your MailPulse server IP
    port: 2525,
    secure: false,  // true for 465, false for other ports
    auth: {
        user: 'mp_live_cc7e7266975fde8699cdad926174784e',  // Your API key
        pass: 'sadi-bino-savo-4616'  // Your project password
    }
});

// Email options
const mailOptions = {
    from: 'noreply@yourdomain.com',
    to: 'recipient@example.com',
    subject: 'Test Email via MailPulse',
    text: 'Hello!\n\nThis is a test email sent through MailPulse SMTP relay.\n\nBest regards,\nYour Application',
    html: '<p>Hello!</p><p>This is a test email sent through MailPulse SMTP relay.</p><p>Best regards,<br>Your Application</p>'
};

// Send email
transporter.sendMail(mailOptions, (error, info) => {
    if (error) {
        console.log('Error:', error);
    } else {
        console.log('Email sent successfully:', info.response);
    }
});

# Using Telnet (for testing)

telnet localhost 2525
> 220 MailPulse SMTP Server Ready
EHLO yourdomain.com
> 250-mailpulse Hello YOURDOMAIN.COM
> 250-AUTH PLAIN LOGIN
> 250-STARTTLS
> 250 SIZE 52428800
AUTH PLAIN AG1wX2xpdmVfY2M3ZTcyNjY5NzVmZGU4Njk5Y2RhZDkyNjE3NDc4NGUAc2FkaS1iaW5vLXNhdm8tNDYxNg==
> 235 Authentication successful
MAIL FROM:<noreply@yourdomain.com>
> 250 OK
RCPT TO:<recipient@example.com>
> 250 OK
DATA
> 354 Start mail input
Subject: Test Email via MailPulse
From: noreply@yourdomain.com
To: recipient@example.com

Hello! This is a test email sent through MailPulse.
.
> 250 OK
QUIT
> 221 Bye

Environment Variables (Recommended)

## .env file
MAILPULSE_HOST=localhost
MAILPULSE_PORT=2525
MAILPULSE_API_KEY=mp_live_cc7e7266975fde8699cdad926174784e
MAILPULSE_PASSWORD=sadi-bino-savo-4616

## AUTH PLAIN Format (Technical Details)

When using raw SMTP commands, the AUTH PLAIN mechanism requires base64 encoding:

```bash
# Format: base64("\0" + api_key + "\0" + password)
echo -n -e '\0mp_live_your_api_key\0your_password' | base64
```

**Example:**
- API Key: `mp_live_cc7e7266975fde8699cdad926174784e`
- Password: `sadi-bino-savo-4616`
- Encoded: `AG1wX2xpdmVfY2M3ZTcyNjY5NzVmZGU4Njk5Y2RhZDkyNjE3NDc4NGUAc2FkaS1iaW5vLXNhdm8tNDYxNg==`

Most SMTP libraries (Python smtplib, Node.js nodemailer, etc.) handle this encoding automatically.

## Important Notes

- ✅ Authentication is required for every email
- ✅ All emails will appear in your MailPulse dashboard  
- ✅ Quota limits apply (10 emails/minute, 500 emails/day by default)
- ✅ Each project has its own API key and password
- ✅ MailPulse forwards emails to your configured email provider (Gmail, Outlook, etc.)