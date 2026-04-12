package notifications

import "se-school/internal/notifications/templates"

type NotificationsServiceMock struct {
	SendEmailCalls []SendEmailCall
	SendEmailErr   error
}

type SendEmailCall struct {
	Receivers []string
	Template  templates.TemplateName
	Data      any
}

func NewNotificationsServiceMock() *NotificationsServiceMock {
	return &NotificationsServiceMock{}
}

func (m *NotificationsServiceMock) SendEmail(receivers []string, template templates.TemplateName, data any) error {
	m.SendEmailCalls = append(m.SendEmailCalls, SendEmailCall{
		Receivers: receivers,
		Template:  template,
		Data:      data,
	})
	return m.SendEmailErr
}
