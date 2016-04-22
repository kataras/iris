// Copyright 2014 Unknwon
//
// Licensed under the Apache License, Version 2.0 (the "License"): you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.

package ini

import (
	"bytes"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func Test_Version(t *testing.T) {
	Convey("Get version", t, func() {
		So(Version(), ShouldEqual, _VERSION)
	})
}

const _CONF_DATA = `
; Package name
NAME        = ini
; Package version
VERSION     = v1
; Package import path
IMPORT_PATH = gopkg.in/%(NAME)s.%(VERSION)s

# Information about package author
# Bio can be written in multiple lines.
[author]
NAME   = Unknwon  ; Succeeding comment
E-MAIL = fake@localhost
GITHUB = https://github.com/%(NAME)s
BIO    = """Gopher.
Coding addict.
Good man.
"""  # Succeeding comment

[package]
CLONE_URL = https://%(IMPORT_PATH)s

[package.sub]
UNUSED_KEY = should be deleted

[features]
-: Support read/write comments of keys and sections
-: Support auto-increment of key names
-: Support load multiple files to overwrite key values

[types]
STRING     = str
BOOL       = true
BOOL_FALSE = false
FLOAT64    = 1.25
INT        = 10
TIME       = 2015-01-01T20:17:05Z
DURATION   = 2h45m
UINT       = 3

[array]
STRINGS  = en, zh, de
FLOAT64S = 1.1, 2.2, 3.3
INTS     = 1, 2, 3
UINTS    = 1, 2, 3
TIMES    = 2015-01-01T20:17:05Z,2015-01-01T20:17:05Z,2015-01-01T20:17:05Z

[note]
empty_lines = next line is empty\

; Comment before the section
[comments] ; This is a comment for the section too
; Comment before key
key  = "value"
key2 = "value2" ; This is a comment for key2
key3 = "one", "two", "three"

[advance]
value with quotes = "some value"
value quote2 again = 'some value'
true = 2+3=5
"1+1=2" = true
"""6+1=7""" = true
"""` + "`" + `5+5` + "`" + `""" = 10
` + "`" + `"6+6"` + "`" + ` = 12
` + "`" + `7-2=4` + "`" + ` = false
ADDRESS = ` + "`" + `404 road,
NotFound, State, 50000` + "`" + `

two_lines = how about \
	continuation lines?
lots_of_lines = 1 \
	2 \
	3 \
	4 \
`

func Test_Load(t *testing.T) {
	Convey("Load from data sources", t, func() {

		Convey("Load with empty data", func() {
			So(Empty(), ShouldNotBeNil)
		})

		Convey("Load with multiple data sources", func() {
			cfg, err := Load([]byte(_CONF_DATA), "testdata/conf.ini")
			So(err, ShouldBeNil)
			So(cfg, ShouldNotBeNil)

			f, err := Load([]byte(_CONF_DATA), "testdata/404.ini")
			So(err, ShouldNotBeNil)
			So(f, ShouldBeNil)
		})
	})

	Convey("Bad load process", t, func() {

		Convey("Load from invalid data sources", func() {
			_, err := Load(_CONF_DATA)
			So(err, ShouldNotBeNil)

			f, err := Load("testdata/404.ini")
			So(err, ShouldNotBeNil)
			So(f, ShouldBeNil)

			_, err = Load(1)
			So(err, ShouldNotBeNil)

			_, err = Load([]byte(""), 1)
			So(err, ShouldNotBeNil)
		})

		Convey("Load with bad section name", func() {
			_, err := Load([]byte("[]"))
			So(err, ShouldNotBeNil)

			_, err = Load([]byte("["))
			So(err, ShouldNotBeNil)
		})

		Convey("Load with bad keys", func() {
			_, err := Load([]byte(`"""name`))
			So(err, ShouldNotBeNil)

			_, err = Load([]byte(`"""name"""`))
			So(err, ShouldNotBeNil)

			_, err = Load([]byte(`""=1`))
			So(err, ShouldNotBeNil)

			_, err = Load([]byte(`=`))
			So(err, ShouldNotBeNil)

			_, err = Load([]byte(`name`))
			So(err, ShouldNotBeNil)
		})

		Convey("Load with bad values", func() {
			_, err := Load([]byte(`name="""Unknwon`))
			So(err, ShouldNotBeNil)
		})
	})
}

func Test_LooseLoad(t *testing.T) {
	Convey("Loose load from data sources", t, func() {
		Convey("Loose load mixed with nonexistent file", func() {
			cfg, err := LooseLoad("testdata/404.ini")
			So(err, ShouldBeNil)
			So(cfg, ShouldNotBeNil)
			var fake struct {
				Name string `ini:"name"`
			}
			So(cfg.MapTo(&fake), ShouldBeNil)

			cfg, err = LooseLoad([]byte("name=Unknwon"), "testdata/404.ini")
			So(err, ShouldBeNil)
			So(cfg.Section("").Key("name").String(), ShouldEqual, "Unknwon")
			So(cfg.MapTo(&fake), ShouldBeNil)
			So(fake.Name, ShouldEqual, "Unknwon")
		})
	})

}

func Test_File_Append(t *testing.T) {
	Convey("Append data sources", t, func() {
		cfg, err := Load([]byte(""))
		So(err, ShouldBeNil)
		So(cfg, ShouldNotBeNil)

		So(cfg.Append([]byte(""), []byte("")), ShouldBeNil)

		Convey("Append bad data sources", func() {
			So(cfg.Append(1), ShouldNotBeNil)
			So(cfg.Append([]byte(""), 1), ShouldNotBeNil)
		})
	})
}

func Test_File_WriteTo(t *testing.T) {
	Convey("Write to somewhere", t, func() {
		var buf bytes.Buffer
		cfg := Empty()
		cfg.WriteTo(&buf)
	})
}

func Test_File_SaveTo(t *testing.T) {
	Convey("Save file", t, func() {
		cfg, err := Load([]byte(_CONF_DATA), "testdata/conf.ini")
		So(err, ShouldBeNil)
		So(cfg, ShouldNotBeNil)

		cfg.Section("").Key("NAME").Comment = "Package name"
		cfg.Section("author").Comment = `Information about package author
# Bio can be written in multiple lines.`
		cfg.Section("advanced").Key("val w/ pound").SetValue("my#password")
		So(cfg.SaveTo("testdata/conf_out.ini"), ShouldBeNil)

		cfg.Section("author").Key("NAME").Comment = "This is author name"
		So(cfg.SaveToIndent("testdata/conf_out.ini", "\t"), ShouldBeNil)
	})
}

// Helpers for slice tests.
func float64sEqual(values []float64, expected ...float64) {
	So(values, ShouldHaveLength, len(expected))
	for i, v := range expected {
		So(values[i], ShouldEqual, v)
	}
}

func intsEqual(values []int, expected ...int) {
	So(values, ShouldHaveLength, len(expected))
	for i, v := range expected {
		So(values[i], ShouldEqual, v)
	}
}

func int64sEqual(values []int64, expected ...int64) {
	So(values, ShouldHaveLength, len(expected))
	for i, v := range expected {
		So(values[i], ShouldEqual, v)
	}
}

func uintsEqual(values []uint, expected ...uint) {
	So(values, ShouldHaveLength, len(expected))
	for i, v := range expected {
		So(values[i], ShouldEqual, v)
	}
}

func uint64sEqual(values []uint64, expected ...uint64) {
	So(values, ShouldHaveLength, len(expected))
	for i, v := range expected {
		So(values[i], ShouldEqual, v)
	}
}

func timesEqual(values []time.Time, expected ...time.Time) {
	So(values, ShouldHaveLength, len(expected))
	for i, v := range expected {
		So(values[i].String(), ShouldEqual, v.String())
	}
}
