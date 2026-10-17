package mailer

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"embed"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net"
	"net/http"
	"net/mail"
	"net/smtp"
	"strings"
	"time"

	"github.com/resend/resend-go/v4"
)

type MailDriver string

const (
	SMTP   MailDriver = "smtp"
	Resend MailDriver = "resend"
	API    MailDriver = "api"
	// Log dumps the composed email (recipients, subject, rendered HTML) to
	// the log instead of sending it — no SMTP/API credentials needed, so
	// it's a natural default for local dev and CI.
	Log MailDriver = "log"
)

type Config struct {
	APIKey      string
	Driver      MailDriver
	Host        string
	Port        string
	FromAddress string
	FromName    string
	Username    string
	Password    string
	Encryption  string
}

type Individual struct {
	Name  string
	Email string
}

type Attachment struct {
	Filename string
	Content  []byte
	MIMEType string
}

type Mailable interface {
	Subject() string
	To() []Individual
	Content() string
	Data() map[string]any
	Attachments() []Attachment
}

type Mailer struct {
	to         Individual
	cfg        Config
	httpClient *http.Client
	templates  embed.FS
}

func New(cfg Config, templates embed.FS) Mailer {
	return Mailer{cfg: cfg, templates: templates}
}

func (m Mailer) To(email, name string) Mailer {
	m.to = Individual{Email: email, Name: name}
	return m
}

// Send preserves the original fire-and-log API. Use SendContext to handle failures.
func (m Mailer) Send(mailable Mailable) {
	if err := m.SendContext(context.Background(), mailable); err != nil {
		log.Printf("failed to send email: %v", err)
	}
}

// SendContext renders and sends an email, returning template and provider errors.
func (m Mailer) SendContext(ctx context.Context, mailable Mailable) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if err := ctx.Err(); err != nil {
		return err
	}
	switch m.cfg.Driver {
	case SMTP, API, Log, Resend:
	default:
		return fmt.Errorf("unknown mail driver %q", m.cfg.Driver)
	}
	html, err := m.composeHTML(mailable.Content(), mailable.Data())
	if err != nil {
		return err
	}
	switch m.cfg.Driver {
	case SMTP:
		return m.sendViaSMTP(ctx, mailable, html)
	case Log:
		m.sendViaLog(mailable, html)
		return nil
	case Resend:
		return m.sendViaResend(ctx, mailable, html)
	default:
		return m.sendViaAPI(ctx, mailable, html)
	}
}

func (m Mailer) sendViaResend(ctx context.Context, message Mailable, html string) error {
	if strings.TrimSpace(m.cfg.APIKey) == "" {
		return fmt.Errorf("resend API key is required")
	}
	if strings.TrimSpace(m.cfg.FromAddress) == "" {
		return fmt.Errorf("resend sender address is required")
	}
	address := func(p Individual) string { return (&mail.Address{Name: p.Name, Address: p.Email}).String() }
	params := &resend.SendEmailRequest{
		From: address(Individual{Name: m.cfg.FromName, Email: m.cfg.FromAddress}),
		To:   []string{address(m.to)}, Subject: message.Subject(), Html: html,
	}
	for _, to := range message.To() {
		params.To = append(params.To, address(to))
	}
	for _, att := range message.Attachments() {
		params.Attachments = append(params.Attachments, &resend.Attachment{Filename: att.Filename, Content: att.Content, ContentType: att.MIMEType})
	}
	httpClient := m.httpClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	client := resend.NewCustomClient(httpClient, m.cfg.APIKey)
	_, err := client.Emails.SendWithContext(ctx, params)
	if err != nil {
		return fmt.Errorf("resend send: %w", err)
	}
	return nil
}

// sendViaLog renders the email exactly as sendViaSMTP/sendViaAPI would
// (same composeHTML call, same recipient list) and writes it to the log
// instead of a wire — the full rendered HTML, not just a one-line notice,
// so a dev can actually read what an email would have said.
func (m Mailer) sendViaLog(mailable Mailable, html string) {
	to := []string{fmt.Sprintf("%s <%s>", m.to.Name, m.to.Email)}
	for _, t := range mailable.To() {
		to = append(to, fmt.Sprintf("%s <%s>", t.Name, t.Email))
	}

	var body strings.Builder
	fmt.Fprintf(&body, "\n----- mail (log driver) -----\n")
	fmt.Fprintf(&body, "To:      %s\n", strings.Join(to, ", "))
	fmt.Fprintf(&body, "From:    %s <%s>\n", m.cfg.FromName, m.cfg.FromAddress)
	fmt.Fprintf(&body, "Subject: %s\n", mailable.Subject())
	if attachments := mailable.Attachments(); len(attachments) > 0 {
		names := make([]string, len(attachments))
		for i, a := range attachments {
			names[i] = a.Filename
		}
		fmt.Fprintf(&body, "Attachments: %s\n", strings.Join(names, ", "))
	}
	fmt.Fprintf(&body, "\n%s\n", html)
	fmt.Fprintf(&body, "----- end mail -----\n")

	log.Print(body.String())
}

func (m Mailer) sendViaAPI(ctx context.Context, mailable Mailable, html string) error {
	to := []map[string]string{{"email": m.to.Email, "name": m.to.Name}}
	for _, t := range mailable.To() {
		to = append(to, map[string]string{"email": t.Email, "name": t.Name})
	}

	attachments := []map[string]string{}
	for _, att := range mailable.Attachments() {
		encoded := base64.StdEncoding.EncodeToString(att.Content)
		attachments = append(attachments, map[string]string{
			"content":     encoded,
			"filename":    att.Filename,
			"type":        att.MIMEType,
			"disposition": "attachment",
		})
	}

	emailData := map[string]any{
		"from":        map[string]string{"email": m.cfg.FromAddress, "name": m.cfg.FromName},
		"to":          to,
		"subject":     mailable.Subject(),
		"html":        html,
		"attachments": attachments,
	}

	jsonData, _ := json.Marshal(emailData)
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "POST", m.cfg.Host, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("mail API request: %w", err)
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", m.cfg.Password))
	req.Header.Add("Content-Type", "application/json")

	res, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("mail API send: %w", err)
	}
	defer res.Body.Close()

	_, err = io.Copy(io.Discard, res.Body)
	if err != nil {
		return fmt.Errorf("mail API response: %w", err)
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("mail API status %d", res.StatusCode)
	}
	return nil
}

func (m Mailer) sendViaSMTP(ctx context.Context, mailable Mailable, html string) error {
	from := fmt.Sprintf("%s <%s>", m.cfg.FromName, m.cfg.FromAddress)
	to := []string{fmt.Sprintf("%s <%s>", m.to.Name, m.to.Email)}
	for _, t := range mailable.To() {
		to = append(to, fmt.Sprintf("%s <%s>", t.Name, t.Email))
	}

	subject := mailable.Subject()
	var msg bytes.Buffer

	if len(mailable.Attachments()) > 0 {
		boundary := generateBoundary()
		// Headers
		headers := []string{
			"To: " + strings.Join(to, ", "),
			"From: " + from,
			"Subject: " + subject,
			"MIME-Version: 1.0",
			"Content-Type: multipart/mixed; boundary=" + boundary,
		}
		msg.WriteString(strings.Join(headers, "\r\n") + "\r\n\r\n")

		// Body part
		msg.WriteString("--" + boundary + "\r\n")
		msg.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n\r\n")
		msg.WriteString(html + "\r\n")

		// Attachments
		for _, att := range mailable.Attachments() {
			encoded := make([]byte, base64.StdEncoding.EncodedLen(len(att.Content)))
			base64.StdEncoding.Encode(encoded, att.Content)

			msg.WriteString("--" + boundary + "\r\n")
			msg.WriteString(fmt.Sprintf("Content-Type: %s\r\n", att.MIMEType))
			msg.WriteString("Content-Transfer-Encoding: base64\r\n")
			msg.WriteString(fmt.Sprintf("Content-Disposition: attachment; filename=\"%s\"\r\n\r\n", att.Filename))
			msg.WriteString(splitBase64(string(encoded)) + "\r\n")
		}

		msg.WriteString("--" + boundary + "--")

	} else {

		// Headers
		headers := []string{
			"To: " + strings.Join(to, ", "),
			"From: " + from,
			"Subject: " + subject,
			"MIME-Version: 1.0",
			"Content-Type: text/html; charset=\"UTF-8\"",
		}

		// Body part
		msg.WriteString(strings.Join(headers, "\r\n") + "\r\n\r\n")
		msg.WriteString(html + "\r\n")
	}

	addr := fmt.Sprintf("%s:%s", m.cfg.Host, m.cfg.Port)

	// PlainAuth refuses to run over a non-TLS connection to anything but
	// localhost, even if the server advertises AUTH (e.g. a MailHog instance
	// reached over Tailscale). Omit it when no credentials are configured
	// instead of tripping that refusal for servers that don't need auth.
	var auth smtp.Auth
	if m.cfg.Username != "" {
		auth = smtp.PlainAuth("", m.cfg.Username, m.cfg.Password, m.cfg.Host)
	}

	// Keep SMTP operations bounded by the caller's context, including TLS and DATA.
	conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("SMTP connect: %w", err)
	}
	defer conn.Close()
	if deadline, ok := ctx.Deadline(); ok {
		if err := conn.SetDeadline(deadline); err != nil {
			return err
		}
	}
	rawConn := conn
	stop := context.AfterFunc(ctx, func() { rawConn.Close() })
	defer stop()
	encryption := strings.ToLower(m.cfg.Encryption)
	if encryption == "ssl" || encryption == "smtps" || m.cfg.Port == "465" {
		conn = tls.Client(conn, &tls.Config{ServerName: m.cfg.Host, MinVersion: tls.VersionTLS12})
	}
	client, err := smtp.NewClient(conn, m.cfg.Host)
	if err != nil {
		return fmt.Errorf("SMTP client: %w", err)
	}
	defer client.Close()
	if _, secure := conn.(*tls.Conn); !secure {
		if ok, _ := client.Extension("STARTTLS"); ok {
			if err := client.StartTLS(&tls.Config{ServerName: m.cfg.Host, MinVersion: tls.VersionTLS12}); err != nil {
				return err
			}
		} else if encryption == "tls" || encryption == "starttls" {
			return fmt.Errorf("SMTP server does not support STARTTLS")
		}
	}
	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return err
		}
	}
	if err := client.Mail(m.cfg.FromAddress); err != nil {
		return err
	}
	recipients := []string{m.to.Email}
	for _, recipient := range mailable.To() {
		recipients = append(recipients, recipient.Email)
	}
	for _, recipient := range recipients {
		if err := client.Rcpt(recipient); err != nil {
			return err
		}
	}
	writer, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := writer.Write(msg.Bytes()); err != nil {
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}
	return client.Quit()
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

func (m Mailer) composeHTML(view string, data map[string]any) (string, error) {

	tmpl, err := template.ParseFS(m.templates, view)
	if err != nil {
		return "", fmt.Errorf("mail template: %w", err)
	}

	var body bytes.Buffer
	if err := tmpl.Execute(&body, data); err != nil {
		return "", fmt.Errorf("mail template: %w", err)
	}

	return body.String(), nil
}

func generateBoundary() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return "----=_Boundary_" + hex.EncodeToString(b)
}
