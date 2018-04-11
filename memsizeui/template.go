package memsizeui

import (
	"html/template"
	"strconv"
	"sync"

	"github.com/fjl/memsize"
)

var (
	base         *template.Template // the "base" template
	baseInitOnce sync.Once
)

func baseInit() {
	base = template.Must(template.New("base").Parse(`<!DOCTYPE html>
<html>
	<head>
		<meta charset="UTF-8">
		<title>memsize</title>
		<style>
		 body {
			 font-family: sans-serif;
		 }
		 button, .button {
			 display: inline-block;
			 font-weight: bold;
			 color: black;
			 text-decoration: none;
			 font-size: inherit;
			 padding: 3pt;
			 margin: 3pt;
			 background-color: #eee;
			 border: 1px solid #999;
			 border-radius: 2pt;
		 }
         form.inline {
             display: inline-block;
         }
		</style>
	</head>
	<body>
		{{template "content" .}}
	</body>
</html>`))

	base.Funcs(template.FuncMap{
		"quote":     strconv.Quote,
		"humansize": memsize.HumanSize,
	})

	template.Must(base.New("rootbuttons").Parse(`
<a class="button" href="/">Overview</a>
{{- range .Roots -}}
<form class="inline" method="POST" action="/scan?root={{.}}">
	<button type="submit">Scan {{quote .}}</button>
</form>
{{- end -}}`))
}

func contentTemplate(source string) *template.Template {
	baseInitOnce.Do(baseInit)
	t := template.Must(base.Clone())
	template.Must(t.New("content").Parse(source))
	return t
}

var rootTemplate = contentTemplate(`
<h1>Memsize</h1>
{{template "rootbuttons" .}}
<hr/>
<h3>Reports</h3>
<ul>
	{{range .Reports}}
	   <li><a href="report/{{.ID}}">{{quote .RootName}} @ {{.Date}}</a></li>
	{{end}}
</ul>
`)

var notFoundTemplate = contentTemplate(`
<h1>Report not found</h1>
{{template "rootbuttons" .}}
`)

var reportTemplate = contentTemplate(`
<h1>Memsize Report {{.ID}}</h1>
<form method="POST" action="../../scan?root={{.RootName}}">
	<a class="button" href="../..">Overview</a>
	<button type="submit">Scan Again</button>
</form>
<pre>
Root: {{quote .RootName}}
Date: {{.Date}}
Duration: {{.Duration}}
Bitmap Size: {{.Sizes.BitmapSize | humansize}}
Bitmap Utilization: {{.Sizes.BitmapUtilization}}
</pre>
<hr/>
<pre>
{{.Sizes.Report}}
</pre>
`)
