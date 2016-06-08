//go:generate templates -s templates -o templates/templates.go
package main

import (
	"bytes"
	"go/format"
	"regexp"
	"sort"
	"strings"
	"text/template"

	"github.com/kaneshin/schematic"
	bundle "github.com/kaneshin/schematic/cmd/schematic/templates"
)

var (
	templates *template.Template
	newlines  = regexp.MustCompile(`(?m:\s*$)`)
)

func init() {
	templates = template.Must(bundle.Parse(schematic.Templates()))
}

// Generate generates code according to the schema.
func Generate(s *schematic.Schema) ([]byte, error) {
	var buf bytes.Buffer

	for i := 0; i < 2; i++ {
		s.Resolve(nil)
	}

	name := strings.ToLower(strings.Split(s.Title, " ")[0])
	templates.ExecuteTemplate(&buf, "package.tmpl", name)

	// TODO: Check if we need time.
	templates.ExecuteTemplate(&buf, "imports.tmpl", []string{
		"encoding/json", "fmt", "io", "reflect",
		"net/http", "runtime", "time", "bytes",
		// TODO: Change for google/go-querystring if pull request #5 gets merged
		// https://github.com/google/go-querystring/pull/5
		"github.com/ernesto-jimenez/go-querystring/query",
	})
	templates.ExecuteTemplate(&buf, "service.tmpl", struct {
		Name    string
		URL     string
		Version string
	}{
		Name:    name,
		URL:     s.URL(),
		Version: s.Version,
	})

	for _, name := range sortedKeys(s.Properties) {
		schema := s.Properties[name]
		// Skipping definitions because there is no links, nor properties.
		if schema.Links == nil && schema.Properties == nil {
			continue
		}

		context := struct {
			Name       string
			Definition *schematic.Schema
		}{
			Name:       name,
			Definition: schema,
		}

		templates.ExecuteTemplate(&buf, "struct.tmpl", context)
		templates.ExecuteTemplate(&buf, "funcs.tmpl", context)
	}

	// Remove blank lines added by text/template
	bytes := newlines.ReplaceAll(buf.Bytes(), []byte(""))

	// Format sources
	clean, err := format.Source(bytes)
	if err != nil {
		return buf.Bytes(), err
	}
	return clean, nil
}

func sortedKeys(m map[string]*schematic.Schema) (keys []string) {
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return
}
