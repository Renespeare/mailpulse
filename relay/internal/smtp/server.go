package smtp

import (
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"log"
	"net"
	"net/mail"
	"strings"
	"time"

	"github.com/Renespeare/mailpulse/relay/internal/auth"
	"github.com/Renespeare/mailpulse/relay/internal/security"
	"github.com/Renespeare/mailpulse/relay/internal/storage"
)

// Server represents an SMTP server with authentication
type Server struct {
	addr         string
	authManager  auth.AuthManager
	storage      storage.Storage
	rateLimiter  security.RateLimiter
	forwarder    *EmailForwarder
	tlsConfig    *tls.Config
	requireAuth  bool
	requireTLS   bool
}

// Config holds server configuration
type Config struct {
	Address     string
	AuthManager auth.AuthManager
	Storage     storage.Storage
	RateLimiter security.RateLimiter
	Forwarder   *EmailForwarder
	TLSConfig   *tls.Config
	RequireAuth bool
	RequireTLS  bool
}

// NewServer creates a new SMTP server
func NewServer(config Config) *Server {
	return &Server{
		addr:        config.Address,
		authManager: config.AuthManager,
		storage:     config.Storage,
		rateLimiter: config.RateLimiter,
		forwarder:   config.Forwarder,
		tlsConfig:   config.TLSConfig,
		requireAuth: config.RequireAuth,
		requireTLS:  config.RequireTLS,
	}
}

// Start starts the SMTP server
func (s *Server) Start() error {
	listener, err := net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", s.addr, err)
	}
	defer listener.Close()

	log.Printf("🔐 SMTP Server listening on %s (AUTH REQUIRED)", s.addr)
	log.Printf("⚠️  SECURITY: This is NOT an open relay - authentication mandatory")

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			continue
		}

		go s.handleConnection(conn)
	}
}

// handleConnection handles a single SMTP connection
func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()
	
	remoteAddr := conn.RemoteAddr().String()
	log.Printf("New connection from %s", remoteAddr)

	// Send greeting
	if err := s.sendResponse(conn, "220 MailPulse SMTP Server Ready (AUTH REQUIRED)"); err != nil {
		log.Printf("Failed to send greeting to %s: %v", remoteAddr, err)
		return
	}

	session := &SMTPSession{
		conn:        conn,
		remoteAddr:  remoteAddr,
		server:      s,
		state:       StateGreeting,
		authManager: s.authManager,
		storage:     s.storage,
		rateLimiter: s.rateLimiter,
	}

	session.handle()
}

// sendResponse sends an SMTP response
func (s *Server) sendResponse(conn net.Conn, response string) error {
	_, err := conn.Write([]byte(response + "\r\n"))
	return err
}

// SMTPState represents the current state of an SMTP session
type SMTPState int

const (
	StateGreeting SMTPState = iota
	StateHelo
	StateAuth
	StateAuthenticated
	StateMail
	StateRcpt
	StateData
	StateQuit
)

// SMTPSession represents an active SMTP session
type SMTPSession struct {
	conn        net.Conn
	remoteAddr  string
	server      *Server
	state       SMTPState
	authManager auth.AuthManager
	storage     storage.Storage
	rateLimiter security.RateLimiter
	
	// Session data
	authenticated bool
	project       *auth.Project
	mailFrom      string
	rcptTo        []string
	data          []byte
}

// handle processes SMTP commands
func (s *SMTPSession) handle() {
	buffer := make([]byte, 1024)
	
	for {
		n, err := s.conn.Read(buffer)
		if err != nil {
			log.Printf("Connection closed by %s: %v", s.remoteAddr, err)
			return
		}
		
		command := strings.TrimSpace(string(buffer[:n]))
		log.Printf("<%s RECV: %s", s.remoteAddr, command)
		
		if err := s.processCommand(command); err != nil {
			log.Printf("Error processing command from %s: %v", s.remoteAddr, err)
			s.sendResponse("500 Command error")
			return
		}
	}
}

// processCommand processes individual SMTP commands
func (s *SMTPSession) processCommand(command string) error {
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return s.sendResponse("500 Command not recognized")
	}
	
	// Only uppercase the command verb, keep parameters case-sensitive
	cmd := strings.ToUpper(parts[0])
	
	switch cmd {
	case "HELO", "EHLO":
		return s.handleHelo(cmd, parts)
	case "AUTH":
		return s.handleAuth(parts, command)
	case "STARTTLS":
		return s.handleStartTLS()
	case "MAIL":
		return s.handleMail(command)
	case "RCPT":
		return s.handleRcpt(command)
	case "DATA":
		return s.handleData()
	case "QUIT":
		return s.handleQuit()
	case "RSET":
		return s.handleReset()
	case "NOOP":
		return s.sendResponse("250 OK")
	default:
		return s.sendResponse("500 Command not recognized")
	}
}

// handleHelo handles HELO/EHLO commands
func (s *SMTPSession) handleHelo(cmd string, parts []string) error {
	if len(parts) < 2 {
		return s.sendResponse("501 Syntax error")
	}
	
	s.state = StateHelo
	
	if cmd == "EHLO" {
		response := fmt.Sprintf("250-%s Hello %s\r\n", "mailpulse", parts[1])
		response += "250-AUTH PLAIN LOGIN\r\n"
		response += "250-STARTTLS\r\n"
		response += "250 SIZE 52428800\r\n" // 50MB limit
		return s.sendResponseRaw(response)
	}
	
	return s.sendResponse(fmt.Sprintf("250 %s Hello %s", "mailpulse", parts[1]))
}

// handleAuth handles AUTH command
func (s *SMTPSession) handleAuth(parts []string, fullCommand string) error {
	if s.server.requireAuth && s.authenticated {
		return s.sendResponse("503 Already authenticated")
	}
	
	if len(parts) < 2 {
		return s.sendResponse("501 Syntax error")
	}
	
	mechanism := parts[1]
	
	switch mechanism {
	case "PLAIN":
		return s.handleAuthPlain(parts, fullCommand)
	case "LOGIN":
		return s.handleAuthLogin()
	default:
		return s.sendResponse("504 Authentication mechanism not supported")
	}
}

// handleAuthPlain handles PLAIN authentication
func (s *SMTPSession) handleAuthPlain(parts []string, fullCommand string) error {
	// Check rate limit for auth attempts
	clientIP := strings.Split(s.remoteAddr, ":")[0]
	if err := s.rateLimiter.CheckAuthAttempt(clientIP); err != nil {
		log.Printf("Rate limit exceeded for auth attempts from %s: %v", clientIP, err)
		return s.sendResponse("421 Too many authentication attempts")
	}
	
	// Record auth attempt
	s.authManager.RecordAuthAttempt(s.remoteAddr, false)
	
	// AUTH PLAIN should have base64 encoded credentials
	if len(parts) < 3 {
		return s.sendResponse("535 Authentication failed")
	}
	
	log.Printf("🔍 Debug: Full command: %q", fullCommand)
	log.Printf("🔍 Debug: Command parts: %q", parts)
	log.Printf("🔍 Debug: Base64 part: %q", parts[2])
	
	// Decode base64 credentials
	authData, err := base64.StdEncoding.DecodeString(parts[2])
	if err != nil {
		log.Printf("Failed to decode auth data from %s: %v", s.remoteAddr, err)
		return s.sendResponse("535 Authentication failed")
	}
	
	// Debug: Show raw bytes
	log.Printf("🔍 Debug: Raw auth bytes: %x", authData)
	log.Printf("🔍 Debug: Raw auth string: %q", string(authData))
	
	// AUTH PLAIN format: \0username\0password
	authParts := strings.Split(string(authData), "\x00")
	log.Printf("🔍 Debug: Auth parts: %q", authParts)
	
	if len(authParts) != 3 {
		log.Printf("Invalid auth format from %s, expected 3 parts, got %d: %q", s.remoteAddr, len(authParts), authParts)
		return s.sendResponse("535 Authentication failed")
	}
	
	username := authParts[1] // authParts[0] is empty (authorization identity)
	password := authParts[2]
	
	log.Printf("🔍 Debug: Extracted username='%s', password='%s'", username, password)
	
	// Validate credentials
	project, err := s.authManager.ValidateAPIKey(username, password)
	if err != nil {
		log.Printf("Authentication failed for %s from %s: %v", username, s.remoteAddr, err)
		return s.sendResponse("535 Authentication failed")
	}
	
	// Check IP allowlist if required
	if project.RequireIPAllow {
		clientIP := strings.Split(s.remoteAddr, ":")[0]
		if !s.authManager.IsIPAllowed(project.ID, clientIP) {
			log.Printf("IP %s not allowed for project %s", clientIP, project.ID)
			return s.sendResponse("535 IP not authorized")
		}
	}
	
	// Check rate limits
	if err := s.authManager.CheckRateLimit(project.ID); err != nil {
		log.Printf("Rate limit exceeded for project %s: %v", project.ID, err)
		return s.sendResponse("421 Rate limit exceeded")
	}
	
	// Authentication successful
	s.authenticated = true
	s.project = project
	s.state = StateAuthenticated
	
	// Record successful auth
	s.authManager.RecordAuthAttempt(s.remoteAddr, true)
	
	log.Printf("✅ Authentication successful for project %s from %s", project.ID, s.remoteAddr)
	return s.sendResponse("235 Authentication successful")
}

// handleAuthLogin handles LOGIN authentication (placeholder)
func (s *SMTPSession) handleAuthLogin() error {
	return s.sendResponse("504 LOGIN authentication not implemented yet")
}

// handleStartTLS handles STARTTLS command
func (s *SMTPSession) handleStartTLS() error {
	if s.server.tlsConfig == nil {
		return s.sendResponse("502 TLS not available")
	}
	
	if err := s.sendResponse("220 Ready to start TLS"); err != nil {
		return err
	}
	
	// Upgrade connection to TLS
	tlsConn := tls.Server(s.conn, s.server.tlsConfig)
	if err := tlsConn.Handshake(); err != nil {
		return fmt.Errorf("TLS handshake failed: %w", err)
	}
	
	s.conn = tlsConn
	log.Printf("TLS enabled for connection from %s", s.remoteAddr)
	
	return nil
}

// handleMail handles MAIL FROM command
func (s *SMTPSession) handleMail(command string) error {
	if s.server.requireAuth && !s.authenticated {
		return s.sendResponse("530 Authentication required")
	}
	
	// Parse MAIL FROM:<address>
	parts := strings.SplitN(command, ":", 2)
	if len(parts) != 2 {
		return s.sendResponse("501 Syntax error")
	}
	
	from := strings.Trim(parts[1], "<> ")
	s.mailFrom = from
	s.state = StateMail
	
	return s.sendResponse("250 OK")
}

// handleRcpt handles RCPT TO command
func (s *SMTPSession) handleRcpt(command string) error {
	if s.state != StateMail && s.state != StateRcpt {
		return s.sendResponse("503 Bad sequence of commands")
	}
	
	// Parse RCPT TO:<address>
	parts := strings.SplitN(command, ":", 2)
	if len(parts) != 2 {
		return s.sendResponse("501 Syntax error")
	}
	
	to := strings.Trim(parts[1], "<> ")
	s.rcptTo = append(s.rcptTo, to)
	s.state = StateRcpt
	
	return s.sendResponse("250 OK")
}

// handleData handles DATA command
func (s *SMTPSession) handleData() error {
	if s.state != StateRcpt {
		return s.sendResponse("503 Bad sequence of commands")
	}
	
	// Re-check project status before accepting email data
	currentProject, err := s.storage.GetProject(s.project.ID)
	if err != nil {
		log.Printf("Failed to get current project status for %s: %v", s.project.ID, err)
		return s.sendResponse("451 Temporary server error")
	}
	
	if currentProject.Status != "active" {
		log.Printf("❌ Project %s is no longer active (status: %s), rejecting DATA command", currentProject.Name, currentProject.Status)
		return s.sendResponse("554 Transaction failed: Project not active")
	}
	
	if err := s.sendResponse("354 End data with <CR><LF>.<CR><LF>"); err != nil {
		return err
	}
	
	// Read email data until "."
	var data []byte
	buffer := make([]byte, 1024)
	
	for {
		n, err := s.conn.Read(buffer)
		if err != nil {
			return err
		}
		
		data = append(data, buffer[:n]...)
		
		// Check for end of data marker
		if strings.Contains(string(data), "\r\n.\r\n") {
			break
		}
	}
	
	s.data = data
	
	// Process the email
	if err := s.processEmail(); err != nil {
		log.Printf("Failed to process email: %v", err)
		return s.sendResponse("550 Transaction failed")
	}
	
	return s.sendResponse("250 OK: Message accepted")
}

// processEmail processes the received email data
func (s *SMTPSession) processEmail() error {
	// Re-check project status before processing email (in case it was deactivated during session)
	currentProject, err := s.storage.GetProject(s.project.ID)
	if err != nil {
		log.Printf("Failed to get current project status for %s: %v", s.project.ID, err)
		return fmt.Errorf("project verification failed")
	}
	
	if currentProject.Status != "active" {
		log.Printf("❌ Project %s is no longer active (status: %s), rejecting email", currentProject.Name, currentProject.Status)
		return fmt.Errorf("project is not active")
	}
	
	// Check email quotas before processing
	if err := s.rateLimiter.CheckEmailQuota(s.project.ID, s.project.QuotaPerMinute, s.project.QuotaDaily); err != nil {
		log.Printf("Email quota exceeded for project %s: %v", s.project.ID, err)
		return fmt.Errorf("quota exceeded: %w", err)
	}
	
	// Generate unique message ID
	messageID := fmt.Sprintf("%d@mailpulse", time.Now().UnixNano())
	
	// Parse email content
	subject := "No Subject"
	emailContent := string(s.data)
	
	// Try to parse with Go's mail package first
	if msg, err := mail.ReadMessage(strings.NewReader(emailContent)); err == nil {
		if subjectHeader := msg.Header.Get("Subject"); subjectHeader != "" {
			subject = subjectHeader
		}
	} else {
		// Fallback: manually parse Subject line from raw content
		lines := strings.Split(emailContent, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(strings.ToLower(line), "subject:") {
				subject = strings.TrimSpace(line[8:]) // Remove "Subject:" prefix
				break
			}
		}
	}
	
	log.Printf("📧 Parsed subject: %q", subject)
	
	// Create email record
	email := &storage.Email{
		ID:        fmt.Sprintf("email_%d", time.Now().UnixNano()),
		MessageID: messageID,
		ProjectID: s.project.ID,
		From:      s.mailFrom,
		To:        s.rcptTo,
		Subject:   subject,
		ContentEnc: []byte(emailContent), // Store the full email content
		Size:      len(s.data),
		Status:    "processed",
		Attempts:  1,
		SentAt:    time.Now(),
	}
	
	// Store in database FIRST
	if err := s.storage.StoreEmail(email); err != nil {
		log.Printf("❌ Failed to store email in database: %v", err)
		// Don't increment quota if database storage fails
		return fmt.Errorf("failed to store email: %w", err)
	}
	
	log.Printf("✅ Email stored in database: %s", messageID)
	
	// Only record quota usage AFTER successful database storage
	if err := s.rateLimiter.RecordEmailSent(s.project.ID); err != nil {
		log.Printf("⚠️  Warning: Email stored but failed to update quota tracking: %v", err)
		// Don't fail the email send for quota tracking issues
	} else {
		log.Printf("✅ Quota counter updated for project %s", s.project.ID)
	}
	
	log.Printf("📧 Email processed successfully: %s from %s to %v (Project: %s)", 
		messageID, s.mailFrom, s.rcptTo, s.project.ID)
	
	// Forward to upstream SMTP server asynchronously
	go func() {
		if s.server.forwarder != nil {
			err := s.server.forwarder.ForwardEmail(email, s.project.ID)
			if err == nil {
				// Success - mark as delivered
				s.storage.UpdateEmailStatus(email.ID, "delivered", nil)
				log.Printf("✅ Email %s forwarded successfully via SMTP", email.ID)
			} else {
				// Failed - mark as failed with error
				errorMsg := fmt.Sprintf("SMTP forwarding failed: %s", err.Error())
				s.storage.UpdateEmailStatus(email.ID, "failed", &errorMsg)
				log.Printf("❌ Email %s forwarding failed: %s", email.ID, err.Error())
			}
		} else {
			log.Printf("⚠️  No email forwarder configured - email %s stored but not forwarded", email.ID)
		}
	}()
	
	return nil
}

// handleQuit handles QUIT command
func (s *SMTPSession) handleQuit() error {
	s.sendResponse("221 Goodbye")
	s.state = StateQuit
	return fmt.Errorf("client quit") // This will close the connection
}

// handleReset handles RSET command
func (s *SMTPSession) handleReset() error {
	s.mailFrom = ""
	s.rcptTo = nil
	s.data = nil
	s.state = StateHelo
	return s.sendResponse("250 OK")
}

// sendResponse sends an SMTP response
func (s *SMTPSession) sendResponse(response string) error {
	log.Printf(">%s SEND: %s", s.remoteAddr, response)
	_, err := s.conn.Write([]byte(response + "\r\n"))
	return err
}

// sendResponseRaw sends a raw SMTP response
func (s *SMTPSession) sendResponseRaw(response string) error {
	log.Printf(">%s SEND: %s", s.remoteAddr, strings.ReplaceAll(response, "\r\n", "\\r\\n"))
	_, err := s.conn.Write([]byte(response))
	return err
}