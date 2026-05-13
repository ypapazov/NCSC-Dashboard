package main

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"
)

const wrapper = `<!DOCTYPE html>
<html lang="{{.Lang}}">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{{.Title}} — Fresnel Help</title>
<link rel="stylesheet" href="/static/css/fresnel.css">
<style>
.help-page { max-width: 52rem; margin: 2rem auto; padding: 0 1.5rem; }
.help-page h1 { font-size: 1.75rem; margin-bottom: 1rem; color: var(--text); }
.help-page h2 { font-size: 1.35rem; margin: 1.5rem 0 .75rem; color: var(--text); border-bottom: 1px solid var(--border); padding-bottom: .4rem; }
.help-page h3 { font-size: 1.1rem; margin: 1.25rem 0 .5rem; color: var(--text); }
.help-page p { margin: .6rem 0; line-height: 1.7; color: var(--text-muted); }
.help-page ul, .help-page ol { margin: .6rem 0 .6rem 1.5rem; color: var(--text-muted); }
.help-page li { margin: .25rem 0; line-height: 1.6; }
.help-page code { background: var(--surface-alt); padding: .15rem .35rem; border-radius: var(--radius-sm); font-family: var(--mono); font-size: .9em; }
.help-page pre { background: var(--surface); padding: 1rem; border-radius: var(--radius); overflow-x: auto; margin: .75rem 0; }
.help-page pre code { background: none; padding: 0; }
.help-page table { width: 100%%; border-collapse: collapse; margin: .75rem 0; }
.help-page th, .help-page td { padding: .5rem .75rem; border: 1px solid var(--border); text-align: left; }
.help-page th { background: var(--surface); font-weight: 600; color: var(--text); }
.help-page td { color: var(--text-muted); }
.help-page a { color: var(--primary); text-decoration: none; }
.help-page a:hover { text-decoration: underline; }
.help-page strong { color: var(--text); }
.help-nav { background: var(--surface); border-bottom: 1px solid var(--border); padding: .75rem 1.5rem; display: flex; align-items: center; gap: 1rem; }
.help-nav a { color: var(--text-muted); text-decoration: none; font-size: .9rem; }
.help-nav a:hover { color: var(--text); }
.help-nav .brand { color: var(--primary); font-weight: 600; font-size: 1rem; }
</style>
</head>
<body style="background:var(--bg);color:var(--text);font-family:var(--font);">
<nav class="help-nav">
	<a href="/" class="brand">Fresnel</a>
	<a href="/help/">Help</a>
</nav>
<div class="help-page">
{{.Body}}
</div>
</body>
</html>`

type page struct {
	Lang  string
	Title string
	Body  template.HTML
}

func main() {
	if len(os.Args) < 3 {
		log.Fatal("usage: helpgen <source-dir> <output-dir>")
	}
	srcDir := os.Args[1]
	outDir := os.Args[2]

	md := goldmark.New(
		goldmark.WithExtensions(extension.Table),
		goldmark.WithRendererOptions(html.WithUnsafe()),
	)

	tmpl, err := template.New("page").Parse(wrapper)
	if err != nil {
		log.Fatalf("template parse: %v", err)
	}

	entries, err := os.ReadDir(srcDir)
	if err != nil {
		log.Fatalf("read source dir %s: %v", srcDir, err)
	}

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		log.Fatalf("mkdir %s: %v", outDir, err)
	}

	lang := "en"
	if parts := strings.Split(filepath.ToSlash(srcDir), "/"); len(parts) > 0 {
		lang = parts[len(parts)-1]
	}

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}

		src, err := os.ReadFile(filepath.Join(srcDir, e.Name()))
		if err != nil {
			log.Fatalf("read %s: %v", e.Name(), err)
		}

		var buf bytes.Buffer
		if err := md.Convert(src, &buf); err != nil {
			log.Fatalf("convert %s: %v", e.Name(), err)
		}

		title := extractTitle(src, e.Name())

		var out bytes.Buffer
		if err := tmpl.Execute(&out, page{
			Lang:  lang,
			Title: title,
			Body:  template.HTML(buf.String()),
		}); err != nil {
			log.Fatalf("render %s: %v", e.Name(), err)
		}

		outName := strings.TrimSuffix(e.Name(), ".md") + ".html"
		outPath := filepath.Join(outDir, outName)
		if err := os.WriteFile(outPath, out.Bytes(), 0o644); err != nil {
			log.Fatalf("write %s: %v", outPath, err)
		}
		fmt.Printf("  %s -> %s\n", e.Name(), outPath)
	}
}

func extractTitle(src []byte, fallback string) string {
	for _, line := range strings.Split(string(src), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "# ") {
			return strings.TrimPrefix(line, "# ")
		}
	}
	return strings.TrimSuffix(fallback, ".md")
}
