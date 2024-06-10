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
	OldIndex = template.Must(template.ParseFS(templates, "html/index.html"))
	OldLogin = template.Must(template.ParseFS(templates, "html/login.html"))

	Form    = template.Must(template.ParseFS(templates, "html/form.html"))
	Hero    = template.Must(template.ParseFS(templates, "html/hero.html"))
	Login   = template.Must(template.ParseFS(templates, "html/login.html"))
	Tos     = template.Must(template.ParseFS(templates, "html/tos.html"))
	Privacy = template.Must(template.ParseFS(templates, "html/privacy.html"))
)

// static files
var (
	//go:embed static/github-mark-white.png
	GitHubMark []byte

	//go:embed html/setup_old.html
	Setup []byte
)
