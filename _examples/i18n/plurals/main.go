package main

import (
	"strings"

	"github.com/kataras/iris/v12"
)

const (
	female = iota + 1
	male
)

const tableStyle = `
<style>
a {
    padding: 8px 8px;
    text-decoration:none;
    cursor:pointer;
    color: #10a2ff;
}
table {
    position: absolute;
    top: 0;
    bottom: 0;
    left: 0;
    right: 0;
    height: 100%;
    width: 100%;
    border-collapse: collapse;
    border-spacing: 0;
    empty-cells: show;
    border: 1px solid #cbcbcb;
}

table caption {
    color: #000;
    font: italic 85%/1 arial, sans-serif;
    padding: 1em 0;
    text-align: center;
}

table td,
table th {
    border-left: 1px solid #cbcbcb;
    border-width: 0 0 0 1px;
    font-size: inherit;
    margin: 0;
    overflow: visible;
    padding: 0.5em 1em;
}

table thead {
    background-color: #10a2ff;
    color: #fff;
    text-align: left;
    vertical-align: bottom;
}

table td {
    background-color: transparent;
}

.table-odd td {
    background-color: #f2f2f2;
}

.table-bordered td {
    border-bottom: 1px solid #cbcbcb;
}
.table-bordered tbody > tr:last-child > td {
    border-bottom-width: 0;
}
</style>
`

/*
$ go run .
Visit http://localhost:8080
*/
func main() {
	app := iris.New()
	err := app.I18n.Load("./locales/*/*", "en-US")
	// ^ here we only use a single locale for the sake of the example,
	// on a real app you can register as many languages as you want to support.
	if err != nil {
		panic(err)
	}

	app.Get("/", func(ctx iris.Context) {
		ctx.HTML("<html><body>\n")
		ctx.WriteString(tableStyle)
		ctx.WriteString(`<table class="table-bordered table-odd">
<thead>
  <tr>
    <th>Key</th>
    <th>Translation</th>
    <th>Arguments</th>
  </tr>
</thead><tbody>
`)
		defer ctx.WriteString("</tbody></table></body></html>")

		tr(ctx, "Classic")

		tr(ctx, "YouLate", 1)
		tr(ctx, "YouLate", 2)

		tr(ctx, "FreeDay", 1)
		tr(ctx, "FreeDay", 5)

		tr(ctx, "FreeDay", 3, 15)

		tr(ctx, "HeIsHome", "Peter")

		tr(ctx, "HouseCount", female, 2, "Maria")
		tr(ctx, "HouseCount", male, 1, "Peter")

		tr(ctx, "nav.home")
		tr(ctx, "nav.user")
		tr(ctx, "nav.more.what")
		tr(ctx, "nav.more.even.more")
		tr(ctx, "nav.more.even.aplural", 1)
		tr(ctx, "nav.more.even.aplural", 15)

		tr(ctx, "VarTemplate", iris.Map{
			"Name":        "Peter",
			"GenderCount": male,
		})

		tr(ctx, "VarTemplatePlural", 1, female)
		tr(ctx, "VarTemplatePlural", 2, female, 1)
		tr(ctx, "VarTemplatePlural", 2, female, 5)
		tr(ctx, "VarTemplatePlural", 1, male)
		tr(ctx, "VarTemplatePlural", 2, male, 1)
		tr(ctx, "VarTemplatePlural", 2, male, 2)

		tr(ctx, "VarTemplatePlural", iris.Map{
			"PluralCount": 5,
			"Names":       []string{"Makis", "Peter"},
			"InlineJoin": func(arr []string) string {
				return strings.Join(arr, ", ")
			},
		})

		tr(ctx, "TemplatePlural", iris.Map{
			"PluralCount": 1,
			"Name":        "Peter",
		})
		tr(ctx, "TemplatePlural", iris.Map{
			"PluralCount": 5,
			"Names":       []string{"Makis", "Peter"},
			"InlineJoin": func(arr []string) string {
				return strings.Join(arr, ", ")
			},
		})
		tr(ctx, "VarTemplatePlural", 2, male, 4)

		tr(ctx, "TemplateVarTemplatePlural", iris.Map{
			"PluralCount": 3,
			"DogsCount":   5,
		})

		tr(ctx, "message.HostResult")

		tr(ctx, "LocalVarsHouseCount.Text", 3, 4)
	})

	app.Listen(":8080")
}

func tr(ctx iris.Context, key string, args ...interface{}) {
	translation := ctx.Tr(key, args...)
	ctx.Writef("<tr><td>%s</td><td>%s</td><td>%v</td></tr>\n", key, translation, args)
}
