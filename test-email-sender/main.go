package main

import (
	"fmt"
	"log"
	"net"
	"net/smtp"
	"os"
	"strings"
	"time"
)

// EmailConfig holds the SMTP configuration
type EmailConfig struct {
	SMTPHost     string
	SMTPPort     string
	Username     string // API Key as username
	Password     string // Project password
	From         string
	To           []string
	Subject      string
	Body         string
}

func main() {
	// Configuration - update these with your project credentials
	config := EmailConfig{
		SMTPHost: "localhost",  // MailPulse relay server
		SMTPPort: "2525",       // MailPulse relay port
		Username: "your-api-key-here",        // Replace with your project API key
		Password: "your-project-password-here", // Replace with your project password
		From:     "sender@yourdomain.com",      // Sender email
		To:       []string{"recipient@gmail.com"}, // Recipient(s)
		Subject:  "Test Email from MailPulse Go Client",
		Body:     `Hello from MailPulse!

This email was sent using a Go script through the MailPulse SMTP relay.

Features tested:
- SMTP authentication with API key and password
- Real email forwarding through configured SMTP provider
- Proper email formatting and headers

Best regards,
MailPulse Go Client`,
	}

	// Override with environment variables if provided
	if host := os.Getenv("MAILPULSE_HOST"); host != "" {
		config.SMTPHost = host
	}
	if port := os.Getenv("MAILPULSE_PORT"); port != "" {
		config.SMTPPort = port
	}
	if apiKey := os.Getenv("MAILPULSE_API_KEY"); apiKey != "" {
		config.Username = apiKey
	}
	if password := os.Getenv("MAILPULSE_PASSWORD"); password != "" {
		config.Password = password
	}
	if from := os.Getenv("FROM_EMAIL"); from != "" {
		config.From = from
	}
	if to := os.Getenv("TO_EMAIL"); to != "" {
		config.To = []string{to}
	}

	// Validate configuration
	if err := validateConfig(config); err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	// Send email
	if err := sendEmail(config); err != nil {
		log.Fatalf("Failed to send email: %v", err)
	}

	fmt.Println("✅ Email sent successfully!")
	fmt.Printf("📧 From: %s\n", config.From)
	fmt.Printf("📧 To: %s\n", strings.Join(config.To, ", "))
	fmt.Printf("📧 Subject: %s\n", config.Subject)
	fmt.Printf("🔧 Via: %s:%s\n", config.SMTPHost, config.SMTPPort)
}

// validateConfig checks if all required fields are provided
func validateConfig(config EmailConfig) error {
	if config.Username == "your-api-key-here" || config.Username == "" {
		return fmt.Errorf("please set MAILPULSE_API_KEY environment variable or update Username in code")
	}
	if config.Password == "your-project-password-here" || config.Password == "" {
		return fmt.Errorf("please set MAILPULSE_PASSWORD environment variable or update Password in code")
	}
	if config.From == "" {
		return fmt.Errorf("sender email (From) is required")
	}
	if len(config.To) == 0 {
		return fmt.Errorf("at least one recipient email (To) is required")
	}
	return nil
}

// sendEmail sends the email through MailPulse SMTP relay
func sendEmail(config EmailConfig) error {
	// SMTP server address
	addr := fmt.Sprintf("%s:%s", config.SMTPHost, config.SMTPPort)
	
	// Build email message
	msg := buildEmailMessage(config)
	
	// Log connection attempt
	log.Printf("🔌 Connecting to MailPulse relay at %s", addr)
	log.Printf("🔑 Authenticating with API key: %s...", config.Username[:8])
	
	// Connect with timeout
	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server: %w", err)
	}
	
	client, err := smtp.NewClient(conn, config.SMTPHost)
	if err != nil {
		return fmt.Errorf("failed to create SMTP client: %w", err)
	}
	defer client.Close()
	
	// Say hello
	if err := client.Hello("localhost"); err != nil {
		return fmt.Errorf("EHLO failed: %w", err)
	}
	
	// Authenticate
	auth := smtp.PlainAuth("", config.Username, config.Password, config.SMTPHost)
	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}
	
	log.Printf("✅ Authentication successful")
	
	// Set sender
	if err := client.Mail(config.From); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}
	
	// Set recipients
	for _, to := range config.To {
		if err := client.Rcpt(to); err != nil {
			return fmt.Errorf("failed to set recipient %s: %w", to, err)
		}
	}
	
	// Send data
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to start data: %w", err)
	}
	
	_, err = w.Write([]byte(msg))
	if err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}
	
	err = w.Close()
	if err != nil {
		return fmt.Errorf("failed to close data: %w", err)
	}
	
	// Quit
	client.Quit()
	
	return nil
}

// buildEmailMessage creates a properly formatted email message
func buildEmailMessage(config EmailConfig) string {
	var msg strings.Builder
	
	// Headers
	msg.WriteString(fmt.Sprintf("From: %s\r\n", config.From))
	msg.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(config.To, ", ")))
	msg.WriteString(fmt.Sprintf("Subject: %s\r\n", config.Subject))
	msg.WriteString("MIME-Version: 1.0\r\n")
	msg.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	msg.WriteString("\r\n") // Empty line between headers and body
	
	// Body
	msg.WriteString(config.Body)
	msg.WriteString("\r\n")
	
	return msg.String()
}