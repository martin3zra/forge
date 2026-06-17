package mailer_test

import (
	"bytes"
	"encoding/base64"
	"net/smtp"
	"strings"
	"testing"
)

func TestSendWithAttachment(t *testing.T) {
	// Fake PDF content
	pdf := []byte("%PDF-1.4\n%Fake PDF for testing\n")

	// Build MIME message
	boundary := "BOUNDARY123"
	msg := bytes.Buffer{}
	msg.WriteString("To: recipient@example.com\r\n")
	msg.WriteString("From: test@example.com\r\n")
	msg.WriteString("Subject: Test Attachment\r\n")
	msg.WriteString("MIME-Version: 1.0\r\n")
	msg.WriteString("Content-Type: multipart/mixed; boundary=" + boundary + "\r\n\r\n")

	// Body
	msg.WriteString("--" + boundary + "\r\n")
	msg.WriteString("Content-Type: text/plain; charset=\"utf-8\"\r\n\r\n")
	msg.WriteString("This is a test email with attachment.\r\n")

	// Attachment
	encoded := make([]byte, base64.StdEncoding.EncodedLen(len(pdf)))
	base64.StdEncoding.Encode(encoded, pdf)
	msg.WriteString("--" + boundary + "\r\n")
	msg.WriteString("Content-Type: application/pdf\r\n")
	msg.WriteString("Content-Transfer-Encoding: base64\r\n")
	msg.WriteString("Content-Disposition: attachment; filename=\"test.pdf\"\r\n\r\n")
	msg.WriteString(splitBase64(string(encoded)) + "\r\n")
	msg.WriteString("--" + boundary + "--")

	// Send to MailHog
	addr := "localhost:1025"
	err := smtp.SendMail(addr, nil, "test@example.com", []string{"recipient@example.com"}, msg.Bytes())
	if err != nil {
		t.Fatalf("failed to send: %v", err)
	}

	t.Log("Message sent to MailHog. Check http://localhost:8025")
}

func splitBase64(s string) string {
	var lines []string
	for len(s) > 76 {
		lines = append(lines, s[:76])
		s = s[76:]
	}
	if len(s) > 0 {
		lines = append(lines, s)
	}
	return strings.Join(lines, "\r\n")
}
