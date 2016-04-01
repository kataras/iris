// Copyright (c) 2016, Gerasimos Maropoulos
// All rights reserved.
//
// Redistribution and use in source and binary forms, with or without modification,
// are permitted provided that the following conditions are met:
//
// 1. Redistributions of source code must retain the above copyright notice,
//    this list of conditions and the following disclaimer.
//
// 2. Redistributions in binary form must reproduce the above copyright notice,
//	  this list of conditions and the following disclaimer
//    in the documentation and/or other materials provided with the distribution.
//
// 3. Neither the name of the copyright holder nor the names of its contributors may be used to endorse
//    or promote products derived from this software without specific prior written permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
// ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
// WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
// DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER AND CONTRIBUTOR, GERASIMOS MAROPOULOS BE LIABLE FOR ANY
// DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
// (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
// LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
// ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
// SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
package iris

import (
	"net/http"
	"encoding/xml"
	"strings"
	"testing"
)

// from @wsantos
func TestContext_GetCookie(t *testing.T) {

	request, _ := http.NewRequest("GET", "/", nil)

	cookie := &http.Cookie{Name: "cookie-name", Value: "cookie-value"}
	request.AddCookie(cookie)

	context := &Context{Request: request}

	value := context.GetCookie("cookie-name")

	if value != "cookie-value" {
		t.Fatal("GetCookie should return \"cookie-value\", but returned: \"", value, "\"")
	}

}

// from @wsantos
func TestContext_GetCookie_Err(t *testing.T) {

	request, _ := http.NewRequest("GET", "/", nil)
	context := &Context{Request: request}

	value := context.GetCookie("cookie-name")

	if value != "" {
		t.Fatal("GetCookie should be empty, but returned: \"", value, "\"")
	}

}

// from @keuller
func TestContext_ReadJSON(t *testing.T) {

	content := strings.NewReader(`{"first_name":"John", "last_name": "Doe"}`)
	request, _ := http.NewRequest("POST", "/", content)
	context := &Context{Request: request}

	var obj map[string]string
	context.ReadJSON(&obj)
	if obj["first_name"] != "John" || obj["last_name"] != "Doe" {
		t.Fatalf("ReadJSON should return \"John\" and \"Doe\", but returned: %s and %s", obj["first_name"], obj["last_name"])
	}
}

func TestContext_ReadXML(t *testing.T) {

	type Contact struct {
		XMLName xml.Name `xml:"contact"`
		FirstName string `xml:"first_name"`
		LastName  string `xml:"last_name"`
	}

	content := strings.NewReader(`
		<?xml version="1.0"?>
		<contact>
			<first_name>John</first_name>
			<last_name>Doe</last_name>
		</contact>
	`)

	request, _ := http.NewRequest("POST", "/", content)
	context := &Context{Request: request}

	var obj Contact
	context.ReadXML(&obj)
	if obj.FirstName != "John" || obj.LastName != "Doe" {
		t.Fatalf("ReadXML should return \"John\" and \"Doe\", but returned: %s and %s", obj.FirstName, obj.LastName)
	}
}
