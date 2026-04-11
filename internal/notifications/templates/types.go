package templates

type TemplateName = string

const (
	Confirmation      TemplateName = "confirm"
	RepositoryUpdated TemplateName = "repository_update"
)

type ConfirmEmailPayload struct {
	Code string
}

type RepositoryUpdateEmailPayload struct {
	Name    string
	Owner   string
	Version string
}

type RenderedTemplate struct {
	Body    string
	Subject string
}
