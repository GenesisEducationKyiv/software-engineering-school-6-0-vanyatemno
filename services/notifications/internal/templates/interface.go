package templates

import "ghnotify/contract"

type TemplateService interface {
	RenderTemplate(name contract.TemplateName, payload any) (*RenderedTemplate, error)
}
