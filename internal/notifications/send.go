package notifications

import (
	"se-school/internal/notifications/mailer"
	"se-school/internal/notifications/templates"

	"go.uber.org/zap"
)

func (n *Service) SendEmail(receivers []string, template templates.TemplateName, data any) error {
	renderedTemplate, err := n.templatesService.RenderTemplate(template, data)
	if err != nil {
		zap.L().Error("failed to render template", zap.Error(err))
		return err
	}
	err = n.mailer.Send(&mailer.Message{
		To:      receivers,
		Subject: renderedTemplate.Subject,
		Body:    renderedTemplate.Body,
	})
	if err != nil {
		zap.L().Error("failed to send email(s)", zap.Error(err))
		return err
	}

	return nil
}
