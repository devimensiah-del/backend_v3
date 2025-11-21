package report

import (
	"bytes"
	"html/template"
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

	// Parse all files in the templates folder
	// Note: Ensure your HTML files use {{.Theme.PrimaryColor}} or CSS var(--primary)
	tmpl := template.New("report").Funcs(template.FuncMap{
		"safeHTML": func(s string) template.HTML { return template.HTML(s) },
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
