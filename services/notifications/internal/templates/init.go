package templates

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"path/filepath"
	"strings"

	"ghnotify/contract"

	"github.com/puzpuzpuz/xsync/v4"
)

const (
	repositoryUpdatedSubject = "Github repository update"
	confirmationSubject      = "Confirm your subscription"
)

// htmlTemplates embeds the email bodies into the binary so the notifier has no
// runtime filesystem dependency (it can run from any working directory / a
// scratch container).
//
//go:embed htmls/*.html
var htmlTemplates embed.FS

func initBodyTemplatesMap() (*xsync.Map[contract.TemplateName, *template.Template], error) {
	templatesMap := xsync.NewMap[contract.TemplateName, *template.Template]()

	matches, err := fs.Glob(htmlTemplates, "htmls/*.html")
	if err != nil {
		return nil, fmt.Errorf("find html templates: %w", err)
	}

	for _, path := range matches {
		name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		tmpl, err := template.ParseFS(htmlTemplates, path)
		if err != nil {
			return nil, fmt.Errorf("parse template %s: %w", path, err)
		}

		templatesMap.Store(name, tmpl)
	}

	return templatesMap, nil
}

func initSubjectTemplatesMap() *xsync.Map[contract.TemplateName, string] {
	subjectsMap := xsync.NewMap[contract.TemplateName, string]()
	subjectsMap.Store(contract.RepositoryUpdated, repositoryUpdatedSubject)
	subjectsMap.Store(contract.Confirmation, confirmationSubject)

	return subjectsMap
}
