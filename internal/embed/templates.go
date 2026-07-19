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
	Report    = template.Must(template.ParseFS(templates, "html/bases/*", "html/report.html"))
	Hero      = template.Must(template.ParseFS(templates, "html/bases/*", "html/hero.html"))
	Settings  = template.Must(template.ParseFS(templates, "html/bases/*", "html/settings.html"))
	Tos       = template.Must(template.ParseFS(templates, "html/bases/*", "html/tos.html"))
	Privacy   = template.Must(template.ParseFS(templates, "html/bases/*", "html/privacy.html"))
	Suspended = template.Must(template.ParseFS(templates, "html/bases/*", "html/suspended.html"))
	Error     = template.Must(template.ParseFS(templates, "html/bases/*", "html/error.html"))
	Stats     = template.Must(template.ParseFS(templates, "html/bases/*", "html/stats.html"))
)

// static files
var (
	//go:embed static/github-mark-white.png
	GitHubMark []byte

	//go:embed static/office-building.png
	OfficeBuilding []byte

	//go:embed static/themes.css
	ThemesCSS []byte

	//go:embed static/skyline.svg
	SkylineSVG []byte

	//go:embed html/setup_old.html
	Setup []byte
)
