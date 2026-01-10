package service

import (
	"fmt"
	"net/smtp"
)

type EmailService struct {
	from          string
	password      string
	host          string
	port          string
	testEmailOnly string // If set, all emails go to this address (for testing)
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
	}, nil
}

// SendPasswordEmail sends an email with the new password to the recipient
func (s *EmailService) SendPasswordEmail(to, vmName, username, password string) error {
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

	body += `This password has been automatically generated for your lab session.

Best regards,
ESXi Lab Provider
`

	message := fmt.Sprintf("From: %s\r\n"+
		"To: %s\r\n"+
		"Subject: %s\r\n"+
		"MIME-Version: 1.0\r\n"+
		"Content-Type: text/plain; charset=UTF-8\r\n"+
		"\r\n"+
		"%s", s.from, actualRecipient, subject, body)

	auth := smtp.PlainAuth("", s.from, s.password, s.host)
	addr := s.host + ":" + s.port

	err := smtp.SendMail(addr, auth, s.from, []string{actualRecipient}, []byte(message))
	if err != nil {
		return fmt.Errorf("failed to send email to %s: %w", actualRecipient, err)
	}

	return nil
}
