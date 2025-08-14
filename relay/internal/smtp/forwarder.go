package smtp

import (
	"fmt"
	"log"
	"net/smtp"
	"strings"

	"github.com/Renespeare/mailpulse/relay/internal/auth"
	"github.com/Renespeare/mailpulse/relay/internal/crypto"
	"github.com/Renespeare/mailpulse/relay/internal/storage"
)

// EmailForwarder handles forwarding emails to upstream SMTP servers
type EmailForwarder struct {
	authManager auth.AuthManager
	storage     storage.Storage
}

// NewEmailForwarder creates a new email forwarder
func NewEmailForwarder(authManager auth.AuthManager, storage storage.Storage) *EmailForwarder {
	return &EmailForwarder{
		authManager: authManager,
		storage:     storage,
	}
}

// ForwardEmail forwards an email using the project's SMTP settings
func (f *EmailForwarder) ForwardEmail(email *storage.Email, projectID string) error {
	// Get project details from database
	project, err := f.storage.GetProject(projectID)
	if err != nil {
		return fmt.Errorf("failed to get project configuration: %w", err)
	}
	
	// Check if project is active
	if project.Status != "active" {
		return fmt.Errorf("project %s is not active", projectID)
	}
	
	// Check if project has SMTP configuration for real forwarding
	if project.SMTPHost != nil && *project.SMTPHost != "" && 
	   project.SMTPUser != nil && *project.SMTPUser != "" && 
	   project.SMTPPasswordEnc != nil && *project.SMTPPasswordEnc != "" {
		
		// Decrypt SMTP password
		smtpPassword, err := crypto.DecryptSMTPPassword(*project.SMTPPasswordEnc)
		if err != nil {
			log.Printf("⚠️  Failed to decrypt SMTP password for project %s: %v", projectID, err)
			return fmt.Errorf("failed to decrypt SMTP password: %w", err)
		}
		
		smtpHost := *project.SMTPHost
		smtpPort := 587 // default
		if project.SMTPPort != nil && *project.SMTPPort > 0 {
			smtpPort = *project.SMTPPort
		}
		smtpUser := *project.SMTPUser
		
		log.Printf("📤 Real SMTP forwarding email %s for project %s (%s) via %s:%d", 
			email.ID, project.Name, projectID, smtpHost, smtpPort)
		
		// Use real SMTP forwarding
		return f.realSMTPForwarding(email, smtpHost, smtpPort, smtpUser, smtpPassword)
	}
	
	// Fallback to simulation mode if no SMTP configuration
	log.Printf("📤 [SIMULATION MODE] No SMTP config found for project %s - simulating forwarding", projectID)
	return f.simulateSMTPForwarding(email, "smtp.gmail.com", 587, "simulation@example.com", "simulation-password")
}

// simulateSMTPForwarding simulates actual SMTP forwarding
func (f *EmailForwarder) simulateSMTPForwarding(email *storage.Email, host string, port int, _, _ string) error {
	log.Printf("📤 [SIMULATION] Attempting to forward email %s via %s:%d", email.ID, host, port)
	log.Printf("   From: %s", email.From)
	log.Printf("   To: %v", email.To)
	log.Printf("   Subject: %s", email.Subject)
	log.Printf("   ⚠️  NOTE: This is simulated - not connecting to real SMTP server")
	
	// Simulate connection and sending
	// In real implementation, you would:
	// 1. Connect to upstream SMTP server with real credentials
	// 2. Authenticate with project SMTP settings
	// 3. Send the actual email content
	// 4. Handle responses and errors
	
	// For demo, simulate realistic success/failure scenarios:
	
	// Simulate different failure scenarios
	if strings.Contains(strings.ToLower(email.Subject), "fail") {
		return fmt.Errorf("[SIMULATED] recipient mailbox full")
	}
	
	if len(email.To) > 5 {
		return fmt.Errorf("[SIMULATED] too many recipients")
	}
	
	// Simulate network timeout for emails ending in 0
	if email.ID[len(email.ID)-1:] == "0" {
		return fmt.Errorf("[SIMULATED] SMTP connection timeout - would need real SMTP credentials")
	}
	
	// Simulate auth failure for emails ending in 1  
	if email.ID[len(email.ID)-1:] == "1" {
		return fmt.Errorf("[SIMULATED] SMTP authentication failed - invalid credentials")
	}
	
	// Otherwise simulate success
	log.Printf("✅ [SIMULATION] Email %s would be forwarded successfully to upstream SMTP", email.ID)
	return nil
}

// realSMTPForwarding implements actual SMTP forwarding
func (f *EmailForwarder) realSMTPForwarding(email *storage.Email, host string, port int, user, pass string) error {
	// 1. Connect to SMTP server
	addr := fmt.Sprintf("%s:%d", host, port)
	auth := smtp.PlainAuth("", user, pass, host)
	
	// 2. Prepare email content
	to := email.To
	subject := email.Subject
	
	// Build proper RFC 822 email message
	var message strings.Builder
	message.WriteString(fmt.Sprintf("From: %s\r\n", email.From))
	message.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(to, ", ")))
	message.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	message.WriteString("MIME-Version: 1.0\r\n")
	message.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	message.WriteString("\r\n") // Empty line between headers and body
	
	// Add email body - parse and clean the original content
	if email.ContentEnc != nil && len(email.ContentEnc) > 0 {
		bodyContent := parseEmailBody(string(email.ContentEnc))
		message.WriteString(bodyContent)
	} else {
		message.WriteString("This email was forwarded through MailPulse SMTP relay.\r\n")
	}
	
	body := message.String()
	
	log.Printf("📤 Connecting to SMTP server %s:%d as %s", host, port, user)
	log.Printf("📧 Email details - From: %s, To: %v, Subject: %s", email.From, to, subject)
	
	// 3. Send email
	err := smtp.SendMail(addr, auth, email.From, to, []byte(body))
	if err != nil {
		log.Printf("❌ SMTP forwarding failed for email %s: %v", email.ID, err)
		log.Printf("🔍 Debug - Host: %s, Port: %d, User: %s", host, port, user)
		return fmt.Errorf("SMTP forwarding failed: %w", err)
	}
	
	log.Printf("✅ Successfully forwarded email %s via real SMTP to %v", email.ID, to)
	return nil
}

// parseEmailBody extracts just the body content from raw SMTP DATA
func parseEmailBody(rawContent string) string {
	// Split by double newline to separate headers from body
	parts := strings.Split(rawContent, "\r\n\r\n")
	if len(parts) < 2 {
		// Try single newline format
		parts = strings.Split(rawContent, "\n\n")
		if len(parts) < 2 {
			// No clear header/body separation, return cleaned content
			return cleanBodyContent(rawContent)
		}
	}
	
	// Join all parts after headers as body (in case body contains double newlines)
	bodyParts := parts[1:]
	body := strings.Join(bodyParts, "\r\n\r\n")
	
	return cleanBodyContent(body)
}

// cleanBodyContent removes SMTP artifacts like trailing dots
func cleanBodyContent(content string) string {
	// Remove trailing SMTP termination dot if present
	content = strings.TrimSpace(content)
	if strings.HasSuffix(content, "\r\n.") {
		content = strings.TrimSuffix(content, "\r\n.")
	} else if strings.HasSuffix(content, "\n.") {
		content = strings.TrimSuffix(content, "\n.")
	} else if strings.HasSuffix(content, ".") && strings.HasSuffix(strings.TrimSuffix(content, "."), "\n") {
		content = strings.TrimSuffix(content, ".")
	}
	
	// Ensure proper line endings
	content = strings.ReplaceAll(content, "\n", "\r\n")
	
	return content + "\r\n"
}