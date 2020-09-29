package main_test

import (
	"strings"
	"testing"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/httptest"
)

const (
	female = iota + 1
	male
)

func TestI18nPlurals(t *testing.T) {
	handler := func(ctx iris.Context) {
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
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	defer r.Body.Close()
	httptest.Do(w, r, handler, func(app *iris.Application) {
		err := app.I18n.Load("./locales/*/*", "en-US", "el-GR")
		if err != nil {
			panic(err)
		}
	})

	expected := `Classic=classic
YouLate=You are 1 minute late.
YouLate=You are 2 minutes late.
FreeDay=You have a day off
FreeDay=You have 5 free days
FreeDay=You have three days and 15 minutes off.
HeIsHome=Peter is home
HouseCount=She (Maria) has 2 houses
HouseCount=He (Peter) has 1 house
nav.home=Home
nav.user=Account
nav.more.what=this
nav.more.even.more=yes
nav.more.even.aplural=You are 1 minute late.
nav.more.even.aplural=You are 15 minutes late.
VarTemplate=(He) Peter is home
VarTemplatePlural=She is awesome
VarTemplatePlural=other (She) has 1 house
VarTemplatePlural=other (She) has 5 houses
VarTemplatePlural=He is awesome
VarTemplatePlural=other (He) has 1 house
VarTemplatePlural=other (He) has 2 houses
VarTemplatePlural=Makis, Peter are awesome
TemplatePlural=Peter is unique
TemplatePlural=Makis, Peter are awesome
VarTemplatePlural=other (He) has 4 houses
TemplateVarTemplatePlural=These 3 are wonderful, feeding 5 dogsssss in total!
message.HostResult=Store Encrypted Message Online
LocalVarsHouseCount.Text=She has 4 houses
`
	if got := w.Body.String(); expected != got {
		t.Fatalf("expected:\n'%s'\n\nbut got:\n'%s'", expected, got)
	}
}

func tr(ctx iris.Context, key string, args ...interface{}) {
	translation := ctx.Tr(key, args...)
	ctx.Writef("%s=%s\n", key, translation)
}
