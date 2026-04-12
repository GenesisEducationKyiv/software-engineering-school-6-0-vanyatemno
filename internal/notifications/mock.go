package notifications

import "se-school/internal/notifications/templates"

type NotificationsServiceMock struct {
}

func NewNotificationsServiceMock() *NotificationsServiceMock {
	return &NotificationsServiceMock{}
}

func SendEmail(receivers []string, template templates.TemplateName, data any) error {
	return nil
}
