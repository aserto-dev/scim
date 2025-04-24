package common

import (
	_ "embed"
)

//go:embed assets/template.tmpl
var template []byte

func LoadDefaultTemplate() []byte {
	return template
}
