package memsizeui

import "html/template"

var templateBase = template.Must(template.New("base").Parse(`<!DOCTYPE html>
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
			 margin: 0;
			 background-color: #eee;
			 border: 1px solid #999;
			 border-radius: 2pt;
		 }
		</style>
	</head>
	<body>
		{{template "content" .}}
	</body>
</html>
`))

func contentTemplate(source string) *template.Template {
	base := template.Must(templateBase.Clone())
	template.Must(base.New("content").Parse(source))
	return base
}

var rootTemplate = contentTemplate(`
<h1>Memsize</h1>
<form method="POST" action="scan">
	<button type="submit">Scan Now</button>
</form>
<hr/>
<h3>Reports</h3>
<ul>
	{{range .Reports}}
	   <li><a href="report/{{.ID}}">{{.Date}}</a></li>
	{{end}}
</ul>
`)

var reportTemplate = contentTemplate(`
<h1>Memsize Report {{.ID}}</h1>
<form method="POST" action="../../scan">
	<a class="button" href="../..">Overview</a>
	<button type="submit">Scan Again</button>
</form>
<pre>
Date: {{.Date}}
Duration: {{.Duration}}
</pre>
<hr/>
<pre>
{{.Sizes.Report}}
</pre>
`)
