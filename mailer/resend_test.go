package mailer

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"
)

type mailTransport func(*http.Request) (*http.Response, error)

func (f mailTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type attachmentMail struct{ fakeMailable }

func (attachmentMail) To() []Individual {
	return []Individual{{Name: "Other", Email: "other@example.com"}}
}
func (attachmentMail) Attachments() []Attachment {
	return []Attachment{{Filename: "report.txt", Content: []byte("report"), MIMEType: "text/plain"}}
}

func TestResendPayloadAndErrors(t *testing.T) {
	for _, status := range []int{200, 401, 429, 500} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			m := New(Config{Driver: Resend, APIKey: "test-key", FromAddress: "sender@example.com", FromName: "Example"}, testTemplates).To("grace@example.com", "Grace")
			m.httpClient = &http.Client{Transport: mailTransport(func(r *http.Request) (*http.Response, error) {
				if r.URL.String() != "https://api.resend.com/emails" || r.Method != "POST" || r.Header.Get("Authorization") != "Bearer test-key" {
					t.Fatalf("unexpected request: %s %s", r.Method, r.URL)
				}
				var body struct {
					From        string
					To          []string
					Subject     string
					HTML        string
					Attachments []struct {
						Filename    string
						Content     []int
						ContentType string `json:"content_type"`
					}
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Fatal(err)
				}
				if body.From != "\"Example\" <sender@example.com>" || len(body.To) != 2 || body.Subject != "Welcome aboard" || !strings.Contains(body.HTML, "Hello, Grace!") {
					t.Fatalf("incorrect email: %+v", body)
				}
				if len(body.Attachments) != 1 || len(body.Attachments[0].Content) != 6 || body.Attachments[0].Content[0] != 114 || body.Attachments[0].ContentType != "text/plain" {
					t.Fatalf("incorrect attachment: %+v", body.Attachments)
				}
				response := `{"id":"email-123"}`
				if status != 200 {
					response = `{"name":"validation_error","message":"rejected","statusCode":` + strconv.Itoa(status) + `}`
				}
				return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(response))}, nil
			})}
			err := m.SendContext(context.Background(), attachmentMail{})
			if (err != nil) != (status != 200) {
				t.Fatalf("status %d: %v", status, err)
			}
		})
	}
}

func TestSendContextValidation(t *testing.T) {
	m := New(Config{Driver: Resend}, testTemplates).To("grace@example.com", "Grace")
	if err := m.SendContext(context.Background(), fakeMailable{}); err == nil {
		t.Fatal("missing key accepted")
	}
	m.cfg.APIKey = "key"
	if err := m.SendContext(context.Background(), fakeMailable{}); err == nil {
		t.Fatal("missing sender accepted")
	}
	m.cfg.Driver = "typo"
	if err := m.SendContext(context.Background(), fakeMailable{}); err == nil {
		t.Fatal("unknown driver accepted")
	}
	m.cfg.Driver = Log
	m.templates = testTemplates
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := m.SendContext(ctx, fakeMailable{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled: %v", err)
	}
	m.templates = emptyTemplates
	if err := m.SendContext(context.Background(), fakeMailable{}); err == nil {
		t.Fatal("missing template accepted")
	}
}

var emptyTemplates embed.FS

type addressMail struct {
	fakeMailable
	recipients []Individual
}

func (m addressMail) To() []Individual { return m.recipients }

func TestResendAddressFormatting(t *testing.T) {
	for _, tc := range []struct {
		name, displayName, wantFrom, wantTo, wantExtra string
	}{
		{"unnamed", "", "sender@example.com", "grace@example.com", "other@example.com"},
		{"blank", " \t ", "sender@example.com", "grace@example.com", "other@example.com"},
		{"named", "Doe, Grace", `"Doe, Grace" <sender@example.com>`, `"Doe, Grace" <grace@example.com>`, `"Doe, Grace" <other@example.com>`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := New(Config{Driver: Resend, APIKey: "test-key", FromAddress: "sender@example.com", FromName: tc.displayName}, testTemplates).To("grace@example.com", tc.displayName)
			m.httpClient = &http.Client{Transport: mailTransport(func(r *http.Request) (*http.Response, error) {
				var body struct {
					From string
					To   []string
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Fatal(err)
				}
				if body.From != tc.wantFrom || len(body.To) != 2 || body.To[0] != tc.wantTo || body.To[1] != tc.wantExtra {
					t.Errorf("unexpected addresses: %+v", body)
				}
				return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"id":"email-test"}`))}, nil
			})}
			if err := m.SendContext(context.Background(), addressMail{recipients: []Individual{{Name: tc.displayName, Email: "other@example.com"}}}); err != nil {
				t.Fatal(err)
			}
		})
	}
}
