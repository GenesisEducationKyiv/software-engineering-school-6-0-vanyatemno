package notifications

import (
	"se-school/internal/notifications/mailer"
	"se-school/internal/notifications/templates"
)

type Service struct {
	mailer           mailer.Mailer
	templatesService templates.TemplateService
}

func New(mailer mailer.Mailer, templatesService templates.TemplateService) *Service {
	return &Service{
		mailer:           mailer,
		templatesService: templatesService,
	}
}
