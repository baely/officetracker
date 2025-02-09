package embed

import (
	"embed"
	"html/template"
)

// templates
var (
	//go:embed html
	templates embed.FS

	Form      = template.Must(template.ParseFS(templates, "html/bases/*", "html/form.html"))
	Hero      = template.Must(template.ParseFS(templates, "html/bases/*", "html/hero.html"))
	Login     = template.Must(template.ParseFS(templates, "html/bases/*", "html/login.html"))
	Settings  = template.Must(template.ParseFS(templates, "html/bases/*", "html/settings.html"))
	Developer = template.Must(template.ParseFS(templates, "html/bases/*", "html/developer.html"))
	Tos       = template.Must(template.ParseFS(templates, "html/bases/*", "html/tos.html"))
	Privacy   = template.Must(template.ParseFS(templates, "html/bases/*", "html/privacy.html"))
	Error     = template.Must(template.ParseFS(templates, "html/bases/*", "html/error.html"))
)

// static files
var (
	//go:embed static/github-mark-white.png
	GitHubMark []byte

	//go:embed static/office-building.png
	OfficeBuilding []byte

	//go:embed static/settings.js
	SettingsJS []byte

	//go:embed html/setup_old.html
	Setup []byte
)
