package report

import (
	"bytes"
	"html/template"

	"github.com/microcosm-cc/bluemonday"
)

// ReportPageData is the "Master Object" passed to every HTML template
type ReportPageData struct {
	// The Branding
	Theme ThemeConfig

	// The Client Context
	CompanyName string
	Date        string

	// The Specific Page Data (e.g., just the SWOT struct)
	Content interface{}
}

type TemplateService struct {
	templates *template.Template
}

func NewTemplateService() *TemplateService {
	// We create a "Base" template with shared CSS variables
	// This allows us to change colors in Go, and CSS updates automatically
	const baseStyle = `
    <style>
        :root {
            --primary: {{.Theme.PrimaryColor}};
            --secondary: {{.Theme.SecondaryColor}};
            --accent: {{.Theme.AccentColor}};
            --bg: {{.Theme.BackgroundColor}};
            --heading-font: {{.Theme.HeadingFont}};
            --body-font: {{.Theme.BodyFont}};
        }
        body { font-family: var(--body-font); background: var(--bg); }
        h1, h2, h3 { font-family: var(--heading-font); color: var(--primary); }
        .accent-bar { background-color: var(--primary); }
    </style>
    `

	// SECURITY FIX: Create HTML sanitizer policy
	// Allows only safe HTML tags and attributes for report content
	// Prevents XSS attacks via malicious user input in PDF generation
	sanitizer := bluemonday.UGCPolicy() // User Generated Content policy
	sanitizer.AllowAttrs("class", "id").Globally()
	sanitizer.AllowAttrs("style").OnElements("div", "span", "p", "h1", "h2", "h3", "h4", "h5", "h6")

	// Parse all files in the templates folder
	// REMOVED: Unsafe "safeHTML" function that bypassed Go's auto-escaping
	// Now using bluemonday sanitization for any HTML that needs to be rendered
	tmpl := template.New("report").Funcs(template.FuncMap{
		"sanitizeHTML": func(s string) template.HTML {
			// Sanitize user-provided HTML to prevent XSS
			clean := sanitizer.Sanitize(s)
			return template.HTML(clean)
		},
	})

	tmpl = template.Must(tmpl.ParseGlob("templates/*.html"))

	// Inject the dynamic style block into every template
	tmpl = template.Must(tmpl.Parse(baseStyle))

	return &TemplateService{templates: tmpl}
}

func (t *TemplateService) RenderPage(pageName string, theme ThemeConfig, companyName string, content interface{}) (string, error) {
	data := ReportPageData{
		Theme:       theme,
		CompanyName: companyName,
		Date:        "October 2025", // Dynamic in real code
		Content:     content,
	}

	var buf bytes.Buffer
	if err := t.templates.ExecuteTemplate(&buf, pageName, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
