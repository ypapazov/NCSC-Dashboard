package markdown

import (
	"bytes"
	"html/template"

	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
)

var (
	md       goldmark.Markdown
	sanitize *bluemonday.Policy
)

func init() {
	md = goldmark.New()
	sanitize = bluemonday.UGCPolicy()
	sanitize.AllowAttrs("class").OnElements("code", "pre", "span")
}

// Render converts Markdown to sanitized HTML safe for embedding.
func Render(input string) template.HTML {
	var buf bytes.Buffer
	if err := md.Convert([]byte(input), &buf); err != nil {
		return template.HTML(template.HTMLEscapeString(input))
	}
	safe := sanitize.SanitizeBytes(buf.Bytes())
	return template.HTML(safe)
}
