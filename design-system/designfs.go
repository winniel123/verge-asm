// Package designfs embeds the design-system templates, tokens and fixtures as a
// read-only file system. It sits at the design-system root because an embed
// directive forbids a parent path and the module root holds no Go package.
package designfs

import (
	"embed"
	"io/fs"
)

//go:embed templates/*.tmpl tokens/*.css fixtures/*.json
var files embed.FS

var FS fs.FS = files
