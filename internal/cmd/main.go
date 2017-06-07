package main

import (
	"os"
	"text/template"

	"github.com/kataras/iris/internal/cmd/gen/website/recipe/example"
)

const tmpl = `
{{ range $key, $example := . -}}
{{ if $example.HasChildren }}
<h2 id="{{$example.Name}}"><a href="#{{$example.Name}}" class="headerlink" title="{{$example.Name}}"></a>{{$example.Name}}</h2>
{{ range $key, $child := $example.Children -}}
    <h3 id="{{ $child.Name }}">
        <a href="#{{ $child.Name }}" class="headerlink" title="{{ $child.Name }}"></a>
        {{ $child.Name }}
    </h3>
    <pre data-src="{{ $child.DataSource }}" 
         data-visible="true" class ="line-numbers codepre"></pre>
{{- end }}
{{- end }}
{{ if .HasNotChildren }}
<h2 id="{{ $example.Name }}">
	<a href="#{{ $example.Name }}" class="headerlink" title="{{ $example.Name }}"></a>
	{{ $example.Name }}
</h2>
<pre data-src="{{ $example.DataSource }}" 
		data-visible="true" class ="line-numbers codepre"></pre>
{{- end }}
{{- end }}
`

func main() {
	// just for testing, the cli will be coded when I finish at least with this one command.
	examples, err := example.Parse("master")
	if err != nil {
		println(err.Error())
		return
	}

	text, err := template.New("").Parse(tmpl)
	if err != nil {
		println(err.Error())
	}

	if err := text.Execute(os.Stdout, examples); err != nil {
		println("err in template : " + err.Error())
	}
}
