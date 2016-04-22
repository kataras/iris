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
	"strings"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func Test_Section(t *testing.T) {
	Convey("Test CRD sections", t, func() {
		cfg, err := Load([]byte(_CONF_DATA), "testdata/conf.ini")
		So(err, ShouldBeNil)
		So(cfg, ShouldNotBeNil)

		Convey("Get section strings", func() {
			So(strings.Join(cfg.SectionStrings(), ","), ShouldEqual, "DEFAULT,author,package,package.sub,features,types,array,note,comments,advance")
		})

		Convey("Delete a section", func() {
			cfg.DeleteSection("")
			So(cfg.SectionStrings()[0], ShouldNotEqual, DEFAULT_SECTION)
		})

		Convey("Create new sections", func() {
			cfg.NewSections("test", "test2")
			_, err := cfg.GetSection("test")
			So(err, ShouldBeNil)
			_, err = cfg.GetSection("test2")
			So(err, ShouldBeNil)
		})
	})
}
