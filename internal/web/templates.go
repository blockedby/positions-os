package web

import (
	"html/template"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// TemplateEngine handles HTML template rendering
type TemplateEngine struct {
	templatesDir string
	templates    *template.Template
	reload       bool // dev mode: reload on each request
}

// NewTemplateEngine creates a new template engine
func NewTemplateEngine(templatesDir string, reload bool) *TemplateEngine {
	return &TemplateEngine{
		templatesDir: templatesDir,
		reload:       reload,
	}
}

// Load parses all templates from the templates directory
func (te *TemplateEngine) Load() error {
	// Parse all HTML files recursively, except pages directory
	tmpl := template.New("").Funcs(template.FuncMap{
		"dict": func(values ...interface{}) (map[string]interface{}, error) {
			if len(values)%2 != 0 {
				return nil, io.ErrShortBuffer // simple error
			}
			dict := make(map[string]interface{}, len(values)/2)
			for i := 0; i < len(values); i += 2 {
				key, ok := values[i].(string)
				if !ok {
					return nil, io.ErrShortBuffer // key must be string
				}
				dict[key] = values[i+1]
			}
			return dict, nil
		},
		"lower": strings.ToLower,
	})

	err := filepath.Walk(te.templatesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip pages directory - these are loaded on-demand
		if info.IsDir() && info.Name() == "pages" {
			return filepath.SkipDir
		}

		if !info.IsDir() && filepath.Ext(path) == ".html" {
			_, err = tmpl.ParseFiles(path)
			return err
		}
		return nil
	})

	if err != nil {
		return err
	}

	te.templates = tmpl
	return nil
}

// Render renders a template with the given data
func (te *TemplateEngine) Render(w io.Writer, name string, data interface{}) error {
	if te.reload {
		if err := te.Load(); err != nil {
			return err
		}
	}

	// Clone base templates and parse page-specific template
	tmpl, err := te.templates.Clone()
	if err != nil {
		return err
	}

	// Parse page template
	pageFile := filepath.Join(te.templatesDir, "pages", name+".html")
	tmpl, err = tmpl.ParseFiles(pageFile)
	if err != nil {
		return err
	}

	// Execute layout with content
	return tmpl.ExecuteTemplate(w, "layout", data)
}

// RenderContent renders only the content template without layout (for HTMX)
func (te *TemplateEngine) RenderContent(w io.Writer, name string, data interface{}) error {
	if te.reload {
		if err := te.Load(); err != nil {
			return err
		}
	}

	// Clone base templates and parse page-specific template
	tmpl, err := te.templates.Clone()
	if err != nil {
		return err
	}

	// Parse page template
	pageFile := filepath.Join(te.templatesDir, "pages", name+".html")
	tmpl, err = tmpl.ParseFiles(pageFile)
	if err != nil {
		return err
	}

	// Execute only the content template
	return tmpl.ExecuteTemplate(w, "content", data)
}

// RenderPartial renders a named template (partial)
func (te *TemplateEngine) RenderPartial(w io.Writer, name string, data interface{}) error {
	return te.templates.ExecuteTemplate(w, name, data)
}
