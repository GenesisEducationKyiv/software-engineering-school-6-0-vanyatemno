package templates

// RenderedTemplate is the output of rendering an email template: a subject line
// and an HTML body, ready to be handed to the mailer.
type RenderedTemplate struct {
	Body    string
	Subject string
}
