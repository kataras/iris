package mail

import (
	"fmt"
	"net/smtp"
	"strings"
	"text/template"

	"github.com/kataras/iris/config"
	"github.com/kataras/iris/utils"
)

const tmpl = `From: {{.From}}<br /> To: {{.To}}<br /> Subject: {{.Subject}}<br /> MIME-version: 1.0<br /> Content-Type: text/html; charset=&quot;UTF-8&quot;<br /> <br /> {{.Body}}`

var buf = utils.NewBufferPool(64)

type (
	// Service is the interface which mail sender should implement
	Service interface {
		// Send sends a mail to recipients
		// the body can be html also
		Send(to []string, subject, body string) error
	}

	mailer struct {
		config        config.Mail
		auth          smtp.Auth
		authenticated bool
	}
)

// New creates and returns a new Service
func New(cfg config.Mail) Service {
	return &mailer{config: cfg}
}

func (m *mailer) authenticate() error {
	if m.config.Username == "" || m.config.Password == "" || m.config.Host == "" {
		return fmt.Errorf("Username, Password & Host cannot be empty!")
	}
	m.auth = smtp.PlainAuth("", m.config.Username, m.config.Password, m.config.Host)
	m.authenticated = true
	return nil
}

// Send sends a mail to recipients
// the body can be html also
func (m *mailer) Send(to []string, subject, body string) error {
	buffer := buf.Get()
	defer buf.Put(buffer)

	if !m.authenticated {
		if err := m.authenticate(); err != nil {
			return err
		}
	}

	mailArgs := map[string]string{"To": strings.Join(to, ","), "Subject": subject, "Body": body}
	template := template.Must(template.New("mailTmpl").Parse(tmpl))

	if err := template.Execute(buffer, mailArgs); err != nil {
		return err
	}

	return smtp.SendMail(
		fmt.Sprintf("%s:%d", m.config.Host, m.config.Port),
		m.auth,
		m.config.Username,
		to,
		buffer.Bytes(),
	)
}
