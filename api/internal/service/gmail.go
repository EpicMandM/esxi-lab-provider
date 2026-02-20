package service

import (
	"encoding/base64"
	"fmt"
	"net/smtp"
	"strings"
)

type EmailService struct {
	from          string
	password      string
	host          string
	port          string
	testEmailOnly string // If set, all emails go to this address (for testing)
	sendMailFn    func(addr string, a smtp.Auth, from string, to []string, msg []byte) error
}

// EmailAttachment represents a file attachment for an email
type EmailAttachment struct {
	Filename string
	Content  []byte
	MimeType string
}

// NewEmailService creates a new email service using SMTP
func NewEmailService(smtpHost, smtpPort, username, password, fromEmail, testEmailOnly string) (*EmailService, error) {
	if smtpHost == "" || username == "" || password == "" {
		return nil, fmt.Errorf("SMTP configuration incomplete")
	}

	return &EmailService{
		from:          fromEmail,
		password:      password,
		host:          smtpHost,
		port:          smtpPort,
		testEmailOnly: testEmailOnly,
		sendMailFn:    smtp.SendMail,
	}, nil
}

// SendPasswordEmail sends an email with the new password to the recipient
func (s *EmailService) SendPasswordEmail(to, vmName, username, password string) error {
	return s.SendPasswordEmailWithAttachment(to, vmName, username, password, nil)
}

// SendPasswordEmailWithAttachment sends an email with the new password and optional attachment
func (s *EmailService) SendPasswordEmailWithAttachment(to, vmName, username, password string, attachment *EmailAttachment) error {
	// Override recipient for testing if TEST_EMAIL_ONLY is set
	actualRecipient := to
	if s.testEmailOnly != "" {
		actualRecipient = s.testEmailOnly
	}

	subject := fmt.Sprintf("ESXi Lab Access - VM: %s", vmName)
	body := fmt.Sprintf(`Hello,

Your ESXi lab environment is now ready!

VM Name: %s
Username: %s
Password: %s

`, vmName, username, password)

	// Add note if email is being sent to test address
	if s.testEmailOnly != "" && to != actualRecipient {
		body += fmt.Sprintf("[TEST MODE] Original recipient: %s\n\n", to)
	}

	if attachment != nil {
		body += fmt.Sprintf(`A WireGuard VPN configuration file (%s) has been attached to this email.
To connect to the lab network:
1. Install WireGuard from https://www.wireguard.com/install/
2. Import the attached configuration file
3. Activate the tunnel

`, attachment.Filename)
	}

	body += `This password has been automatically generated for your lab session.

Best regards,
ESXi Lab Provider
`

	var message string
	if attachment != nil {
		message = s.buildMIMEMessage(actualRecipient, subject, body, attachment)
	} else {
		message = s.buildPlainMessage(actualRecipient, subject, body)
	}

	auth := smtp.PlainAuth("", s.from, s.password, s.host)
	addr := s.host + ":" + s.port

	err := s.sendMailFn(addr, auth, s.from, []string{actualRecipient}, []byte(message))
	if err != nil {
		return fmt.Errorf("failed to send email to %s: %w", actualRecipient, err)
	}

	return nil
}

// buildPlainMessage builds a plain text email message
func (s *EmailService) buildPlainMessage(to, subject, body string) string {
	return fmt.Sprintf("From: %s\r\n"+
		"To: %s\r\n"+
		"Subject: %s\r\n"+
		"MIME-Version: 1.0\r\n"+
		"Content-Type: text/plain; charset=UTF-8\r\n"+
		"\r\n"+
		"%s", s.from, to, subject, body)
}

// buildMIMEMessage builds a MIME email message with attachment
func (s *EmailService) buildMIMEMessage(to, subject, body string, attachment *EmailAttachment) string {
	boundary := "boundary-esxi-lab-provider"

	var sb strings.Builder

	// Headers
	sb.WriteString(fmt.Sprintf("From: %s\r\n", s.from))
	sb.WriteString(fmt.Sprintf("To: %s\r\n", to))
	sb.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	sb.WriteString("MIME-Version: 1.0\r\n")
	sb.WriteString(fmt.Sprintf("Content-Type: multipart/mixed; boundary=\"%s\"\r\n", boundary))
	sb.WriteString("\r\n")

	// Body part
	sb.WriteString(fmt.Sprintf("--%s\r\n", boundary))
	sb.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	sb.WriteString("\r\n")
	sb.WriteString(body)
	sb.WriteString("\r\n")

	// Attachment part
	if attachment != nil {
		sb.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		sb.WriteString(fmt.Sprintf("Content-Type: %s; name=\"%s\"\r\n", attachment.MimeType, attachment.Filename))
		sb.WriteString("Content-Transfer-Encoding: base64\r\n")
		sb.WriteString(fmt.Sprintf("Content-Disposition: attachment; filename=\"%s\"\r\n", attachment.Filename))
		sb.WriteString("\r\n")

		// Encode content to base64 and add line breaks every 76 characters (RFC 2045)
		encoded := base64.StdEncoding.EncodeToString(attachment.Content)
		for i := 0; i < len(encoded); i += 76 {
			end := i + 76
			if end > len(encoded) {
				end = len(encoded)
			}
			sb.WriteString(encoded[i:end])
			sb.WriteString("\r\n")
		}
	}

	// End boundary
	sb.WriteString(fmt.Sprintf("--%s--\r\n", boundary))

	return sb.String()
}
