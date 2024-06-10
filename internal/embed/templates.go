package embed

import (
	"embed"
	"html/template"
)

// templates
var (
	//go:embed html
	templates embed.FS
	
	Form    = template.Must(template.ParseFS(templates, "html/base.html", "html/form.html"))
	Hero    = template.Must(template.ParseFS(templates, "html/base.html", "html/hero.html"))
	Login   = template.Must(template.ParseFS(templates, "html/base.html", "html/login.html"))
	Tos     = template.Must(template.ParseFS(templates, "html/base.html", "html/tos.html"))
	Privacy = template.Must(template.ParseFS(templates, "html/base.html", "html/privacy.html"))
)

// static files
var (
	//go:embed static/github-mark-white.png
	GitHubMark []byte

	//go:embed html/setup_old.html
	Setup []byte
)
