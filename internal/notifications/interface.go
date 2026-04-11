package notifications

import "se-school/internal/notifications/templates"

type NotificationsService interface {
	SendEmail(receivers []string, template templates.TemplateName, data any) error
}
