package mail

import (
	"encoding/base64"
	"fmt"
	"net/mail"
	"net/smtp"
	"strings"
	"sync"

	"github.com/kataras/iris/config"
	"github.com/kataras/iris/utils"
)

var buf = utils.NewBufferPool(64)
var once sync.Once

type (
	// Service is the interface which mail sender should implement
	Service interface {
		// Send sends a mail to recipients
		// the body can be html also
		Send(string, string, ...string) error
		UpdateConfig(config.Mail)
	}

	mailer struct {
		config        *config.Mail
		fromAddr      mail.Address
		auth          smtp.Auth
		authenticated bool
	}
)

// New creates and returns a new Service
func New(cfg config.Mail) Service {
	m := &mailer{config: &cfg}
	if cfg.FromAlias == "" {
		if !cfg.UseCommand && cfg.Username != "" && strings.Contains(cfg.Username, "@") {
			m.fromAddr = mail.Address{Name: cfg.Username[0:strings.IndexByte(cfg.Username, '@')], Address: cfg.Username}
		}
	} else {
		m.fromAddr = mail.Address{Name: cfg.FromAlias, Address: cfg.Username}
	}
	return m
}

func (m *mailer) UpdateConfig(cfg config.Mail) {
	m.config = &cfg
}

// Send sends a mail to recipients
// the body can be html also
func (m *mailer) Send(subject string, body string, to ...string) error {
	if m.config.UseCommand {
		return m.sendCmd(subject, body, to)
	}

	return m.sendSMTP(subject, body, to)
}

func (m *mailer) sendSMTP(subject string, body string, to []string) error {
	buffer := buf.Get()
	defer buf.Put(buffer)

	if !m.authenticated {
		cfg := m.config
		if cfg.Username == "" || cfg.Password == "" || cfg.Host == "" || cfg.Port <= 0 {
			return fmt.Errorf("Username, Password, Host & Port cannot be empty when using SMTP!")
		}
		m.auth = smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)
		m.authenticated = true
	}

	fullhost := fmt.Sprintf("%s:%d", m.config.Host, m.config.Port)

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

	return smtp.SendMail(
		fmt.Sprintf(fullhost),
		m.auth,
		m.config.Username,
		to,
		[]byte(message),
	)
}

func (m *mailer) sendCmd(subject string, body string, to []string) error {
	buffer := buf.Get()
	defer buf.Put(buffer)

	header := make(map[string]string)
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
	buffer.WriteString(message)
	// fix by @qskousen
	cmd := utils.CommandBuilder("sendmail", "-F", m.fromAddr.Name, "-f", m.fromAddr.Address, "-t")

	cmd.Stdin = buffer
	_, err := cmd.CombinedOutput()
	return err
}
