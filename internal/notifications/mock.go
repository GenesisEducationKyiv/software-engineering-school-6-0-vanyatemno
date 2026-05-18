package notifications

import (
	"sync"

	"se-school/internal/notifications/templates"
)

type NotificationsServiceMock struct {
	mu             sync.Mutex
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
	m.mu.Lock()
	m.SendEmailCalls = append(m.SendEmailCalls, SendEmailCall{
		Receivers: receivers,
		Template:  template,
		Data:      data,
	})
	err := m.SendEmailErr
	m.mu.Unlock()
	return err
}

// Calls returns a copy of the recorded SendEmail invocations. Safe to call
// concurrently with SendEmail.
func (m *NotificationsServiceMock) Calls() []SendEmailCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]SendEmailCall, len(m.SendEmailCalls))
	copy(out, m.SendEmailCalls)
	return out
}

// SetSendEmailErr atomically swaps the error returned by future SendEmail
// calls.
func (m *NotificationsServiceMock) SetSendEmailErr(err error) {
	m.mu.Lock()
	m.SendEmailErr = err
	m.mu.Unlock()
}
