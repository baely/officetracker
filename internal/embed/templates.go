package embed

import (
	"embed"
	"html/template"
)

// templates
var (
	//go:embed html
	templates embed.FS

	Form    = template.Must(template.ParseFS(templates, "html/bases/*.html", "html/form.html"))
	Hero    = template.Must(template.ParseFS(templates, "html/bases/*.html", "html/hero.html"))
	Login   = template.Must(template.ParseFS(templates, "html/bases/*.html", "html/login.html"))
	Tos     = template.Must(template.ParseFS(templates, "html/bases/*.html", "html/tos.html"))
	Privacy = template.Must(template.ParseFS(templates, "html/bases/*.html", "html/privacy.html"))
	Error   = template.Must(template.ParseFS(templates, "html/bases/*.html", "html/error.html"))
)

// static files
var (
	//go:embed static/github-mark-white.png
	GitHubMark []byte

	//go:embed html/setup_old.html
	Setup []byte
)
