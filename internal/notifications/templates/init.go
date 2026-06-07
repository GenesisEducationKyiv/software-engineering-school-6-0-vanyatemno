package templates

import (
	"fmt"
	"html/template"
	"path/filepath"
	"se-school/internal/utils"
	"strings"

	"github.com/puzpuzpuz/xsync/v4"
)

const (
	htmlTemplatesRelativePath = "internal/notifications/templates/htmls/"
	repositoryUpdatedSubject  = "Github repository update"
	confirmationSubject       = "Confirm your subscription"
)

func initBodyTemplatesMap() (*xsync.Map[TemplateName, *template.Template], error) {
	templatesMap := xsync.NewMap[TemplateName, *template.Template]()
	projectDir, err := utils.GetRootProjectDir()
	if err != nil {
		return nil, err
	}

	matches, err := filepath.Glob(
		filepath.Join(projectDir, htmlTemplatesRelativePath, "*.html"),
	)
	if err != nil {
		return nil, fmt.Errorf("find html templates: %w", err)
	}

	for _, path := range matches {
		name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		tmpl, err := template.ParseFiles(path)
		if err != nil {
			return nil, fmt.Errorf("parse template %s: %w", path, err)
		}

		templatesMap.Store(name, tmpl)
	}

	return templatesMap, nil
}

func initSubjectTemplatesMap() *xsync.Map[TemplateName, string] {
	subjectsMap := xsync.NewMap[TemplateName, string]()
	subjectsMap.Store(RepositoryUpdated, repositoryUpdatedSubject)
	subjectsMap.Store(Confirmation, confirmationSubject)

	return subjectsMap
}
