package common

import (
	"embed"
	"fmt"
)

//go:embed assets/*
var staticAssets embed.FS

func LoadTemplate(templateName string) ([]byte, error) {
	templateFile := fmt.Sprintf("assets/%s.tmpl", templateName)
	return staticAssets.ReadFile(templateFile)
}
