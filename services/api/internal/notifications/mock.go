package notifications

import (
	"context"
	"sync"

	"ghnotify/contract"
)

// NotificationsServiceMock records Notify calls and can inject failures — either
// for every call (NotifyErr) or for specific recipients (NotifyErrFor, used to
// exercise per-subscriber failure isolation).
type NotificationsServiceMock struct {
	mu           sync.Mutex
	NotifyCalls  []NotifyCall
	NotifyErr    error
	NotifyErrFor map[string]error
}

type NotifyCall struct {
	Recipient string
	Template  contract.TemplateName
	Data      any
}

func NewNotificationsServiceMock() *NotificationsServiceMock {
	return &NotificationsServiceMock{}
}

func (m *NotificationsServiceMock) Notify(_ context.Context, recipient string, template contract.TemplateName, data any) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.NotifyCalls = append(m.NotifyCalls, NotifyCall{
		Recipient: recipient,
		Template:  template,
		Data:      data,
	})
	if m.NotifyErr != nil {
		return m.NotifyErr
	}
	if m.NotifyErrFor != nil {
		if err, ok := m.NotifyErrFor[recipient]; ok {
			return err
		}
	}
	return nil
}

// Calls returns a copy of the recorded Notify invocations. Safe to call
// concurrently with Notify.
func (m *NotificationsServiceMock) Calls() []NotifyCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]NotifyCall, len(m.NotifyCalls))
	copy(out, m.NotifyCalls)
	return out
}

// SetNotifyErr atomically swaps the error returned by future Notify calls.
func (m *NotificationsServiceMock) SetNotifyErr(err error) {
	m.mu.Lock()
	m.NotifyErr = err
	m.mu.Unlock()
}
