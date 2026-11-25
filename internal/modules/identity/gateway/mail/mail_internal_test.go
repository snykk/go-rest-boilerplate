package mail

import (
	"bytes"
	"encoding/json"
	htmltemplate "html/template"
	"io/fs"
	"log/slog"
	"strings"
	"testing"
	"testing/fstest"
	texttemplate "text/template"
	"time"

	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/domain"
	"github.com/snykk/go-rest-boilerplate/internal/platform/mailer"
)

// NewMail parses the templates once and tries them, and refuses a set in which a template is missing, does not
// parse, names a value a message does not have, or leaves out the code or token its message carries.
func TestNewMail(t *testing.T) {
	tCases := []struct {
		name    string
		change  func(files fstest.MapFS)
		wantErr bool
	}{
		{name: "the embedded templates are accepted", change: func(fstest.MapFS) {}},
		{
			name:    "a missing template is refused",
			change:  func(files fstest.MapFS) { delete(files, "templates/password_reset.txt") },
			wantErr: true,
		},
		{
			name:    "a template that does not parse is refused",
			change:  replace("templates/verification.html", `{{define "content"}}{{.Code}{{end}}`),
			wantErr: true,
		},
		{
			name:    "a template that names a missing value is refused",
			change:  replace("templates/password_reset.txt", "{{.Missing}}"),
			wantErr: true,
		},
		{
			name:    "a layout that leaves out the content is refused",
			change:  replace("templates/layout.html", "<p>{{.Service}}</p>"),
			wantErr: true,
		},
		{
			name:    "a text that leaves out the code is refused",
			change:  replace("templates/verification.txt", "Your code expires in {{.Lifetime}}."),
			wantErr: true,
		},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			files := embeddedTemplates(t)
			tc.change(files)

			got, err := newMail(&mailer.Recorder{}, slog.New(slog.DiscardHandler), settingsFixture(), files)

			if gotErr := err != nil; gotErr != tc.wantErr {
				t.Fatalf("newMail = %v, want error: %t", err, tc.wantErr)
			}
			if tc.wantErr && got != nil {
				t.Errorf("newMail = %v, want nil with the error", got)
			}
		})
	}
}

// send queues a message it can render, and logs one it cannot at ERROR with its kind and the error, never its
// recipient or the secret it carries, and queues nothing.
func TestMailSend(t *testing.T) {
	tCases := []struct {
		name       string
		text       string // the source of the message's text template
		wantQueued bool
		wantLogged bool
	}{
		{name: "a message that renders is queued", text: "{{.Code}}", wantQueued: true},
		{name: "a message that cannot render is logged and not queued", text: "{{.Missing}}", wantLogged: true},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			var queue mailer.Recorder
			var logs bytes.Buffer
			sender := &Mail{
				queue:    &queue,
				log:      slog.New(slog.NewJSONHandler(&logs, nil)),
				settings: settingsFixture(),
			}
			verification := message{
				kind:    "identity.verification",
				subject: "Your verification code",
				html:    htmltemplate.Must(htmltemplate.New("html").Parse("<p>{{.Code}}</p>")),
				text:    texttemplate.Must(texttemplate.New("text").Parse(tc.text)),
			}

			sender.send(t.Context(), address(t), verification, content{Code: "123456"})

			if queued := len(queue.Messages()) == 1; queued != tc.wantQueued {
				t.Errorf("queued = %t, want %t", queued, tc.wantQueued)
			}
			if logged := logs.Len() > 0; logged != tc.wantLogged {
				t.Fatalf("logged = %t (%s), want %t", logged, logs.String(), tc.wantLogged)
			}
			if tc.wantLogged {
				assertRenderFailure(t, logs.String())
			}
		})
	}
}

// assertRenderFailure checks that logs hold one ERROR record that names the kind identity.verification and an
// error, and neither the code 123456 nor the address ada@example.com.
func assertRenderFailure(t *testing.T, logs string) {
	t.Helper()
	var record map[string]any
	if err := json.Unmarshal([]byte(logs), &record); err != nil {
		t.Fatalf("log record %q = %v, want one JSON record", logs, err)
	}
	logged, ok := record["error"].(string)
	if record["level"] != "ERROR" || record["kind"] != "identity.verification" || !ok || logged == "" {
		t.Errorf("log record = %v, want level ERROR, kind identity.verification and an error", record)
	}
	if strings.Contains(logs, "123456") || strings.Contains(logs, "ada@example.com") {
		t.Errorf("log record = %s, want neither the code nor the address", logs)
	}
}

func settingsFixture() Settings {
	return Settings{
		Service:  "Blog",
		CodeTTL:  time.Minute,
		ResetTTL: time.Minute,
	}
}

// address is ada@example.com as a domain.Email.
func address(t *testing.T) domain.Email {
	t.Helper()
	email, err := domain.NewEmail("ada@example.com")
	if err != nil {
		t.Fatalf("NewEmail = %v", err)
	}
	return email
}

// embeddedTemplates is a copy of the embedded templates that a test may change.
func embeddedTemplates(t *testing.T) fstest.MapFS {
	t.Helper()
	files := fstest.MapFS{}
	err := fs.WalkDir(templateFiles, ".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		data, err := fs.ReadFile(templateFiles, path)
		if err != nil {
			return err
		}
		files[path] = &fstest.MapFile{Data: data}
		return nil
	})
	if err != nil {
		t.Fatalf("copy the embedded templates = %v", err)
	}
	return files
}

// replace is a change that gives the file at path the content text.
func replace(path, text string) func(files fstest.MapFS) {
	return func(files fstest.MapFS) { files[path] = &fstest.MapFile{Data: []byte(text)} }
}
