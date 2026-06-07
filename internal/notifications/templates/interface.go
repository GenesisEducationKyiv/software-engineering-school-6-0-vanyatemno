package templates

type TemplateService interface {
	RenderTemplate(name TemplateName, payload any) (*RenderedTemplate, error)
}
