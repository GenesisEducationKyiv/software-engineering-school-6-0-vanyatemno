package mailer

type Mailer interface {
	SendEmail() error
	SendEmailBulk() error
}
