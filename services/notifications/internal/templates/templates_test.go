package templates

import (
	"testing"

	"ghnotify/contract"

	"go.uber.org/zap"
)

func TestTemplates(t *testing.T) {
	zap.ReplaceGlobals(zap.Must(zap.NewDevelopment()))
	service := New()

	_, err := service.RenderTemplate(
		contract.Confirmation,
		map[string]string{
			"Code": "code",
			"Link": "link",
		},
	)
	if err != nil {
		t.Fatal(err)
	}
}
