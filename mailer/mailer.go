package mailer

import (
	"bytes"
	"crypto/rand"
	"embed"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/smtp"
	"strings"
)

type MailDriver string

const (
	SMTP MailDriver = "smtp"
	API  MailDriver = "api"
)

type Config struct {
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
	to        Individual
	cfg       Config
	templates embed.FS
}

func New(cfg Config, templates embed.FS) Mailer {
	return Mailer{cfg: cfg, templates: templates}
}

func (m Mailer) To(email, name string) Mailer {
	m.to = Individual{Email: email, Name: name}
	return m
}

func (m Mailer) Send(mailable Mailable) {
	if m.cfg.Driver == SMTP {
		m.sendViaSMTP(mailable)
		return
	}

	m.sendViaAPI(mailable)
}

func (m Mailer) sendViaAPI(mailable Mailable) {
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
		"html":        m.composeHTML(mailable.Content(), mailable.Data()),
		"attachments": attachments,
	}

	jsonData, _ := json.Marshal(emailData)
	client := &http.Client{}
	req, _ := http.NewRequest("POST", m.cfg.Host, bytes.NewBuffer(jsonData))
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", m.cfg.Password))
	req.Header.Add("Content-Type", "application/json")

	res, err := client.Do(req)
	if err != nil {
		log.Println(err)
		return
	}
	defer res.Body.Close()

	body, _ := io.ReadAll(res.Body)
	log.Println(string(body))
}

func (m Mailer) sendViaSMTP(mailable Mailable) {
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
		msg.WriteString(m.composeHTML(mailable.Content(), mailable.Data()) + "\r\n")

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
		msg.WriteString(m.composeHTML(mailable.Content(), mailable.Data()) + "\r\n")
	}

	addr := fmt.Sprintf("%s:%s", m.cfg.Host, m.cfg.Port)
	err := smtp.SendMail(addr, smtp.PlainAuth("", m.cfg.Username, m.cfg.Password, m.cfg.Host), m.cfg.FromAddress, []string{m.to.Email}, msg.Bytes())
	if err != nil {
		log.Fatalf("failed to send email: %v", err)
	}
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

func (m Mailer) composeHTML(view string, data map[string]any) string {

	tmpl, err := template.ParseFS(m.templates, view)
	if err != nil {
		log.Fatal(err.Error())
		return ""
	}

	var body bytes.Buffer
	if err := tmpl.Execute(&body, data); err != nil {
		log.Fatal(err.Error())
		return ""
	}

	return body.String()
}

func generateBoundary() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return "----=_Boundary_" + hex.EncodeToString(b)
}
