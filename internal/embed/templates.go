package embed

import (
	"embed"
	"html/template"
)

// templates
var (
	//go:embed html
	templates embed.FS
)
var (
	Index = template.Must(template.ParseFS(templates, "html/index.html"))
	Login = template.Must(template.ParseFS(templates, "html/login.html"))
)

// static files
var (
	//go:embed static/github-mark-white.png
	GitHubMark []byte

	//go:embed html/setup.html
	Setup []byte
)
