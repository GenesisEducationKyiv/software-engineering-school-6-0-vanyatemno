package templates

import (
	"testing"

	"ghnotify/contract"
)

func TestInitBodyTemplates(t *testing.T) {
	templatesMap, err := initBodyTemplatesMap()
	if err != nil {
		t.Fatalf("initTemplates() returned error: %v", err)
	}

	if templatesMap == nil {
		t.Fatal("initTemplates() returned nil map")
	}
	testCases := []contract.TemplateName{
		contract.Confirmation,
		contract.RepositoryUpdated,
	}

	for _, templateName := range testCases {
		tmpl, ok := templatesMap.Load(templateName)
		if !ok {
			t.Fatalf("template %q was not loaded", templateName)
		}

		if tmpl == nil {
			t.Fatalf("template %q is nil", templateName)
		}
	}
}
