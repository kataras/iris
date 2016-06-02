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

// Send sends a mail to recipients
// the body can be html also
func (m *mailer) Send(to []string, subject, body string) error {
	if m.config.UseCommand {
		return m.sendCmd(to, subject, body)
	}

	return m.sendSMTP(to, subject, body)
}

func (m *mailer) sendSMTP(to []string, subject, body string) error {
	buffer := buf.Get()
	defer buf.Put(buffer)

	if !m.authenticated {
		if m.config.Username == "" || m.config.Password == "" || m.config.Host == "" || m.config.Port <= 0 {
			return fmt.Errorf("Username, Password, Host & Port cannot be empty when using SMTP!")
		}
		m.auth = smtp.PlainAuth("", m.config.Username, m.config.Password, m.config.Host)
		m.authenticated = true
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

func (m *mailer) sendCmd(to []string, subject, body string) error {
	buffer := buf.Get()
	defer buf.Put(buffer)

	cmd := utils.CommandBuilder("mail", "-s", subject, strings.Join(to, ","))
	cmd.AppendArguments("-a", "Content-type: text/html") //always html on

	cmd.Stdin = buffer
	_, err := cmd.Output()
	return err
}
