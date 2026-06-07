package mailer

import (
	"se-school/internal/config"

	"go.uber.org/zap"
	"gopkg.in/gomail.v2"
)

type MailService struct {
	cfg *config.Mailer
}

func NewMailerService(cfg *config.Mailer) *MailService {
	return &MailService{
		cfg: cfg,
	}
}

func (m *MailService) Send(message *Message) error {
	dialer := gomail.NewDialer(m.cfg.SMTP, m.cfg.Port, m.cfg.From, m.cfg.Password)
	s, err := dialer.Dial()
	if err != nil {
		return err
	}

	msg := gomail.NewMessage()
	msg.SetHeader("From", m.cfg.From)
	msg.SetHeader("Subject", message.Subject)
	msg.SetBody("text/html", message.Body)

	var failedEmails []string
	for i := range message.To {
		msg.SetHeader("To", message.To[i])
		err = gomail.Send(s, msg)
		if err != nil {
			zap.L().Error("Error sending mail", zap.Error(err))
			failedEmails = append(failedEmails, message.To[i])
		}
	}
	if len(failedEmails) > 0 {
		return UnsentMailsError{
			EmailAddresses: failedEmails,
		}
	}

	err = s.Close()
	if err != nil {
		return err
	}

	return nil
}
