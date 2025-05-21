package embed

import (
	"embed"
	"html/template"
	"io/fs"
	"path"
)

//go:embed html
var templates embed.FS

//go:embed themes/infinite_city
var infiniteCityThemeFS embed.FS

//go:embed static
var staticFiles embed.FS

// Base templates (shared or default)
var (
	Form      = template.Must(template.ParseFS(templates, "html/bases/*", "html/form.html"))
	Hero      = template.Must(template.ParseFS(templates, "html/bases/*", "html/hero.html"))
	Login     = template.Must(template.ParseFS(templates, "html/bases/*", "html/login.html")) // This might become the "default" login
	Settings  = template.Must(template.ParseFS(templates, "html/bases/*", "html/settings.html"))
	Developer = template.Must(template.ParseFS(templates, "html/bases/*", "html/developer.html"))
	Tos       = template.Must(template.ParseFS(templates, "html/bases/*", "html/tos.html"))
	Privacy   = template.Must(template.ParseFS(templates, "html/bases/*", "html/privacy.html"))
	Error     = template.Must(template.ParseFS(templates, "html/bases/*", "html/error.html"))
)

// Theme definition structure
type Theme struct {
	Name      string
	Login     *template.Template
	Index     *template.Template
	StaticFS  fs.FS
	LoginPath string
	IndexPath string
	CSSPath   string
	JSPath    string
}

// AvailableThemes holds all loaded themes
var AvailableThemes = make(map[string]Theme)

func mustLoadTemplate(efs fs.FS, basePath string, patterns ...string) *template.Template {
	// Prepend the base path to all patterns
	fullPatterns := make([]string, len(patterns))
	for i, p := range patterns {
		fullPatterns[i] = path.Join(basePath, p)
	}
	return template.Must(template.ParseFS(efs, fullPatterns...))
}

func init() {
	// Load Infinite City Theme
	infiniteCityBase := "themes/infinite_city"
	infiniteCityStaticFS, err := fs.Sub(infiniteCityThemeFS, infiniteCityBase)
	if err != nil {
		panic(err)
	}

	AvailableThemes["infinite_city"] = Theme{
		Name:      "Infinite City",
		Login:     mustLoadTemplate(infiniteCityThemeFS, infiniteCityBase, "login.html"),
		Index:     mustLoadTemplate(infiniteCityThemeFS, infiniteCityBase, "index.html"),
		StaticFS:  infiniteCityStaticFS,
		LoginPath: "login.html", // Relative to theme's static FS
		IndexPath: "index.html", // Relative to theme's static FS
		CSSPath:   "style.css",  // Relative to theme's static FS
		JSPath:    "script.js",  // Relative to theme's static FS
	}

	// TODO: Load a "default" theme, possibly using the existing Login, Hero templates
	// For example, if you have an index.html for the default theme:
	// defaultIndexFS, _ := fs.Sub(templates, "html") // Assuming default index is in html/
	// AvailableThemes["default"] = Theme{
	// 	Name: "Default",
	// 	Login: Login, // Using the existing global Login template
	// 	Index: template.Must(template.ParseFS(templates, "html/bases/*", "html/hero.html")), // Or your main index page
	// 	StaticFS: staticFiles, // Or a sub-FS if default theme has its own static assets
	//      CSSPath: "path/to/default/style.css", // if applicable
	//      JSPath: "path/to/default/script.js",   // if applicable
	// }
}

// Static files (global or for default theme)
var (
	//go:embed static/github-mark-white.png
	GitHubMark []byte

	//go:embed static/office-building.png
	OfficeBuilding []byte

	//go:embed html/setup_old.html
	Setup []byte
)

// GetTheme returns a theme by name.
func GetTheme(name string) (Theme, bool) {
	theme, ok := AvailableThemes[name]
	return theme, ok
}
