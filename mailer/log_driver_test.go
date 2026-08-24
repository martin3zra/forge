package mailer

import (
	"embed"
	"log"
	"os"
	"strings"
	"testing"
)

//go:embed testdata
var testTemplates embed.FS

type fakeMailable struct{}

func (fakeMailable) Subject() string  { return "Welcome aboard" }
func (fakeMailable) To() []Individual { return nil }
func (fakeMailable) Content() string  { return "testdata/fake.html" }
func (fakeMailable) Data() map[string]any {
	return map[string]any{"Name": "Grace"}
}
func (fakeMailable) Attachments() []Attachment { return nil }

func TestLogDriverWritesRenderedEmailToLog(t *testing.T) {
	var out strings.Builder
	log.SetOutput(&out)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })

	cfg := Config{Driver: Log, FromAddress: "noreply@example.com", FromName: "Example"}
	m := New(cfg, testTemplates).To("grace@example.com", "Grace")
	m.Send(fakeMailable{})

	got := out.String()
	for _, want := range []string{
		"To:      Grace <grace@example.com>",
		"Subject: Welcome aboard",
		"Hello, Grace!",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected log output to contain %q, got:\n%s", want, got)
		}
	}
}
