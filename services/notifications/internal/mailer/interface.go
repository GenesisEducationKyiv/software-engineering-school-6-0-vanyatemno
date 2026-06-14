package mailer

type Mailer interface {
	Send(message *Message) error
}
