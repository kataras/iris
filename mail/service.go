package mail

import (
	"encoding/base64"
	"fmt"
	"net/mail"
	"net/smtp"
	"strings"

	"github.com/kataras/iris/config"
	"github.com/kataras/iris/utils"
)

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
		fromAddr      mail.Address
		auth          smtp.Auth
		authenticated bool
	}
)

// New creates and returns a new Service
func New(cfg config.Mail) Service {
	m := &mailer{config: cfg}

	// not necessary
	if !cfg.UseCommand && cfg.Username != "" && strings.Contains(cfg.Username, "@") {
		m.fromAddr = mail.Address{cfg.Username[0:strings.IndexByte(cfg.Username, '@')], cfg.Username}
	}

	return m
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

	fullhost := fmt.Sprintf("%s:%d", m.config.Host, m.config.Port)

	/* START: This one helped me https://gist.github.com/andelf/5004821 */
	header := make(map[string]string)
	header["From"] = m.fromAddr.String()
	header["To"] = strings.Join(to, ",")
	header["Subject"] = subject
	header["MIME-Version"] = "1.0"
	header["Content-Type"] = "text/html; charset=\"utf-8\""
	header["Content-Transfer-Encoding"] = "base64"

	message := ""
	for k, v := range header {
		message += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	message += "\r\n" + base64.StdEncoding.EncodeToString([]byte(body))

	/* END */

	return smtp.SendMail(
		fmt.Sprintf(fullhost),
		m.auth,
		m.config.Username,
		to,
		[]byte(message),
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
