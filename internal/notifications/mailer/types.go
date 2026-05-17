package mailer

import "fmt"

// Message represents an email message.
// It contains all necessary fields for constructing and sending an email.
type Message struct {
	To      []string
	Subject string
	Body    string
}

type UnsentMailsError struct {
	EmailAddresses []string
}

func (e UnsentMailsError) Error() string {
	return fmt.Sprintf("could not send emails to %d user(s)", len(e.EmailAddresses))
}
