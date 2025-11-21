package report

// ThemeConfig allows us to change the report's branding globally
// Fonts are hardcoded in templates to avoid Go template escaping issues
type ThemeConfig struct {
	PrimaryColor    string
	SecondaryColor  string
	AccentColor     string
	BackgroundColor string
}

// DefaultTheme returns the IMENSIAH standard brand
// Based on globals.css: Primary is Deep Blue (hsl 195), Accent is Gold (hsl 45)
// Fonts: Montserrat (headings) and Roboto (body) are loaded directly in templates
func DefaultTheme() ThemeConfig {
	return ThemeConfig{
		PrimaryColor:    "#002A45", // Deep Ocean Blue
		SecondaryColor:  "#003A5C", // Slightly lighter blue
		AccentColor:     "#F59E0B", // Gold/Amber
		BackgroundColor: "#F8FAFC", // Light Gray/White
	}
}

// GetCSSVariables returns CSS custom properties for the theme
func (t ThemeConfig) GetCSSVariables() map[string]string {
	return map[string]string{
		"--primary":   t.PrimaryColor,
		"--secondary": t.SecondaryColor,
		"--accent":    t.AccentColor,
		"--bg":        t.BackgroundColor,
	}
}
