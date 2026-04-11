package templates

import (
	"bytes"
	"fmt"
	"html/template"

	"github.com/puzpuzpuz/xsync/v4"
	"go.uber.org/zap"
)

type Service struct {
	bodyTemplatesMap *xsync.Map[TemplateName, *template.Template]
	subjectsMap      *xsync.Map[TemplateName, string]
}

func New() *Service {
	bodyTemplatesMap, err := initBodyTemplatesMap()
	if err != nil {
		zap.L().Fatal("Failed to initialize templates", zap.Error(err))
	}
	subjectsMap := initSubjectTemplatesMap()

	return &Service{
		bodyTemplatesMap: bodyTemplatesMap,
		subjectsMap:      subjectsMap,
	}
}

func (s *Service) RenderTemplate(name TemplateName, payload any) (*RenderedTemplate, error) {
	bodyBuffer, err := s.renderTemplateBody(name, payload)
	if err != nil {
		return nil, err
	}
	subject, ok := s.subjectsMap.Load(name)
	if !ok {
		zap.L().Error("could not find subject for template", zap.String("template_name", name))
		return nil, fmt.Errorf("could not find subject for template: %s", name)
	}

	return &RenderedTemplate{
		Body:    bodyBuffer.String(),
		Subject: subject,
	}, nil
}

func (s *Service) renderTemplateBody(name TemplateName, payload any) (*bytes.Buffer, error) {
	tmpl, ok := s.bodyTemplatesMap.Load(name)
	if !ok {
		return nil, fmt.Errorf("template %q was not loaded", name)
	}
	var buf bytes.Buffer
	err := tmpl.Execute(&buf, payload)
	if err != nil {
		zap.L().Error("failed to execute template", zap.Error(err))
		return nil, err
	}

	return &buf, nil
}
