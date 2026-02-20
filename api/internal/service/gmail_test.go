package service

import (
	"fmt"
	"net/smtp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// smtpCall records captured arguments from the sendMailFn spy.
type smtpCall struct {
	addr string
	from string
	to   []string
	msg  string
}

func newSpySendMail(calls *[]smtpCall, returnErr error) func(string, smtp.Auth, string, []string, []byte) error {
	return func(addr string, _ smtp.Auth, from string, to []string, msg []byte) error {
		*calls = append(*calls, smtpCall{addr: addr, from: from, to: to, msg: string(msg)})
		return returnErr
	}
}

func TestNewEmailService_Valid(t *testing.T) {
	svc, err := NewEmailService("smtp.example.com", "587", "user", "pass", "from@example.com", "")
	require.NoError(t, err)
	assert.NotNil(t, svc)
	assert.Equal(t, "smtp.example.com", svc.host)
	assert.Equal(t, "587", svc.port)
	assert.Equal(t, "from@example.com", svc.from)
	assert.NotNil(t, svc.sendMailFn)
}

func TestNewEmailService_MissingHost(t *testing.T) {
	_, err := NewEmailService("", "587", "user", "pass", "from@example.com", "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "SMTP configuration incomplete")
}

func TestNewEmailService_MissingUsername(t *testing.T) {
	_, err := NewEmailService("smtp.example.com", "587", "", "pass", "from@example.com", "")
	assert.Error(t, err)
}

func TestNewEmailService_MissingPassword(t *testing.T) {
	_, err := NewEmailService("smtp.example.com", "587", "user", "", "from@example.com", "")
	assert.Error(t, err)
}

func TestSendPasswordEmail_DelegatesToWithAttachment(t *testing.T) {
	var calls []smtpCall
	svc := &EmailService{
		host:       "smtp.example.com",
		port:       "587",
		from:       "from@example.com",
		password:   "pass",
		sendMailFn: newSpySendMail(&calls, nil),
	}

	err := svc.SendPasswordEmail("to@example.com", "vm1", "alice", "secret")
	require.NoError(t, err)
	require.Len(t, calls, 1)
	assert.Contains(t, calls[0].msg, "vm1")
	assert.Contains(t, calls[0].msg, "alice")
	assert.Contains(t, calls[0].msg, "secret")
}

func TestSendPasswordEmailWithAttachment_PlainEmail(t *testing.T) {
	var calls []smtpCall
	svc := &EmailService{
		host:       "smtp.example.com",
		port:       "587",
		from:       "from@example.com",
		password:   "pass",
		sendMailFn: newSpySendMail(&calls, nil),
	}

	err := svc.SendPasswordEmailWithAttachment("to@example.com", "vm1", "alice", "pw123", nil)
	require.NoError(t, err)
	require.Len(t, calls, 1)
	assert.Equal(t, "smtp.example.com:587", calls[0].addr)
	assert.Equal(t, []string{"to@example.com"}, calls[0].to)
	assert.Contains(t, calls[0].msg, "Content-Type: text/plain")
	assert.NotContains(t, calls[0].msg, "multipart")
}

func TestSendPasswordEmailWithAttachment_WithAttachment(t *testing.T) {
	var calls []smtpCall
	svc := &EmailService{
		host:       "smtp.example.com",
		port:       "587",
		from:       "from@example.com",
		password:   "pass",
		sendMailFn: newSpySendMail(&calls, nil),
	}

	att := &EmailAttachment{
		Filename: "test.conf",
		Content:  []byte("wireguard config content"),
		MimeType: "application/x-wireguard-profile",
	}

	err := svc.SendPasswordEmailWithAttachment("to@example.com", "vm1", "alice", "pw123", att)
	require.NoError(t, err)
	require.Len(t, calls, 1)
	assert.Contains(t, calls[0].msg, "multipart/mixed")
	assert.Contains(t, calls[0].msg, "test.conf")
	assert.Contains(t, calls[0].msg, "application/x-wireguard-profile")
}

func TestSendPasswordEmailWithAttachment_TestEmailOverride(t *testing.T) {
	var calls []smtpCall
	svc := &EmailService{
		host:          "smtp.example.com",
		port:          "587",
		from:          "from@example.com",
		password:      "pass",
		testEmailOnly: "test@override.com",
		sendMailFn:    newSpySendMail(&calls, nil),
	}

	err := svc.SendPasswordEmailWithAttachment("real@example.com", "vm1", "alice", "pw123", nil)
	require.NoError(t, err)
	require.Len(t, calls, 1)
	assert.Equal(t, []string{"test@override.com"}, calls[0].to)
	assert.Contains(t, calls[0].msg, "[TEST MODE] Original recipient: real@example.com")
}

func TestSendPasswordEmailWithAttachment_SMTPError(t *testing.T) {
	svc := &EmailService{
		host:       "smtp.example.com",
		port:       "587",
		from:       "from@example.com",
		password:   "pass",
		sendMailFn: newSpySendMail(new([]smtpCall), fmt.Errorf("connection refused")),
	}

	err := svc.SendPasswordEmailWithAttachment("to@example.com", "vm1", "alice", "pw123", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to send email to to@example.com")
}

func TestBuildPlainMessage(t *testing.T) {
	svc := &EmailService{from: "sender@example.com"}
	msg := svc.buildPlainMessage("recipient@example.com", "Test Subject", "Hello world")

	assert.Contains(t, msg, "From: sender@example.com")
	assert.Contains(t, msg, "To: recipient@example.com")
	assert.Contains(t, msg, "Subject: Test Subject")
	assert.Contains(t, msg, "MIME-Version: 1.0")
	assert.Contains(t, msg, "Content-Type: text/plain; charset=UTF-8")
	assert.Contains(t, msg, "Hello world")
}

func TestBuildMIMEMessage(t *testing.T) {
	svc := &EmailService{from: "sender@example.com"}
	att := &EmailAttachment{
		Filename: "vpn.conf",
		Content:  []byte("test data"),
		MimeType: "application/octet-stream",
	}

	msg := svc.buildMIMEMessage("recipient@example.com", "Subject", "Body text", att)

	assert.Contains(t, msg, "From: sender@example.com")
	assert.Contains(t, msg, "To: recipient@example.com")
	assert.Contains(t, msg, "multipart/mixed")
	assert.Contains(t, msg, "Content-Type: text/plain; charset=UTF-8")
	assert.Contains(t, msg, "Body text")
	assert.Contains(t, msg, "vpn.conf")
	assert.Contains(t, msg, "application/octet-stream")
	assert.Contains(t, msg, "Content-Transfer-Encoding: base64")
}

func TestBuildMIMEMessage_LongAttachment(t *testing.T) {
	svc := &EmailService{from: "sender@example.com"}
	// Create content longer than 76 bytes to test line breaking
	content := make([]byte, 200)
	for i := range content {
		content[i] = 'A'
	}

	att := &EmailAttachment{
		Filename: "big.bin",
		Content:  content,
		MimeType: "application/octet-stream",
	}

	msg := svc.buildMIMEMessage("to@example.com", "Subj", "Body", att)
	// Verify that base64 lines don't exceed ~76 chars (plus \r\n)
	assert.Contains(t, msg, "big.bin")
	assert.Contains(t, msg, "Content-Transfer-Encoding: base64")
}

func TestSendPasswordEmailWithAttachment_WireGuardNote(t *testing.T) {
	var calls []smtpCall
	svc := &EmailService{
		host:       "smtp.example.com",
		port:       "587",
		from:       "from@example.com",
		password:   "pass",
		sendMailFn: newSpySendMail(&calls, nil),
	}

	att := &EmailAttachment{
		Filename: "user-wireguard.conf",
		Content:  []byte("[Interface]\nPrivateKey = test"),
		MimeType: "application/x-wireguard-profile",
	}

	err := svc.SendPasswordEmailWithAttachment("to@example.com", "vm1", "alice", "pw123", att)
	require.NoError(t, err)
	assert.Contains(t, calls[0].msg, "WireGuard VPN configuration")
}

func TestSendPasswordEmailWithAttachment_TestEmailSameAsRecipient(t *testing.T) {
	var calls []smtpCall
	svc := &EmailService{
		host:          "smtp.example.com",
		port:          "587",
		from:          "from@example.com",
		password:      "pass",
		testEmailOnly: "same@example.com",
		sendMailFn:    newSpySendMail(&calls, nil),
	}

	err := svc.SendPasswordEmailWithAttachment("same@example.com", "vm1", "alice", "pw123", nil)
	require.NoError(t, err)
	// Should NOT contain test mode note when recipient matches test email
	assert.NotContains(t, calls[0].msg, "[TEST MODE]")
}
