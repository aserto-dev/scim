package common

import (
	"embed"
	"fmt"
)

//go:embed assets/*
var staticAssets embed.FS

func Assets() embed.FS {
	return staticAssets
}

func GetTemplateContent(templateName string) ([]byte, error) {
	var templateContent []byte
	var err error
	templateFile := fmt.Sprintf("assets/%s.tmpl", templateName)
	templateContent, err = Assets().ReadFile(templateFile)
	if err != nil {
		return nil, err
	}

	return templateContent, nil
}
