package assets_test

import (
	"bytes"
	_ "embed"
)

//go:embed config/topaz.yaml
var topazConfig []byte

//go:embed data/rick.json
var rickJson []byte

//go:embed data/morty.json
var mortyJson []byte

//go:embed data/patch.json
var patch []byte

//go:embed data/manifest.yaml
var manifest []byte

//go:embed data/group.json
var group []byte

func TopazConfigReader() *bytes.Reader {
	return bytes.NewReader(topazConfig)
}

func Rick() []byte {
	return rickJson
}

func Morty() []byte {
	return mortyJson
}

func PatchOp() []byte {
	return patch
}

func Group() []byte {
	return group
}

func Manifest() []byte {
	return manifest
}
