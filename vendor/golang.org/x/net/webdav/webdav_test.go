// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package webdav

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// TODO: add tests to check XML responses with the expected prefix path
func TestPrefix(t *testing.T) {
	const dst, blah = "Destination", "blah blah blah"

	do := func(method, urlStr string, body io.Reader, wantStatusCode int, headers ...string) error {
		req, err := http.NewRequest(method, urlStr, body)
		if err != nil {
			return err
		}
		for len(headers) >= 2 {
			req.Header.Add(headers[0], headers[1])
			headers = headers[2:]
		}
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		defer res.Body.Close()
		if res.StatusCode != wantStatusCode {
			return fmt.Errorf("got status code %d, want %d", res.StatusCode, wantStatusCode)
		}
		return nil
	}

	prefixes := []string{
		"/",
		"/a/",
		"/a/b/",
		"/a/b/c/",
	}
	for _, prefix := range prefixes {
		fs := NewMemFS()
		h := &Handler{
			FileSystem: fs,
			LockSystem: NewMemLS(),
		}
		mux := http.NewServeMux()
		if prefix != "/" {
			h.Prefix = prefix
		}
		mux.Handle(prefix, h)
		srv := httptest.NewServer(mux)
		defer srv.Close()

		// The script is:
		//	MKCOL /a
		//	MKCOL /a/b
		//	PUT   /a/b/c
		//	COPY  /a/b/c /a/b/d
		//	MKCOL /a/b/e
		//	MOVE  /a/b/d /a/b/e/f
		// which should yield the (possibly stripped) filenames /a/b/c and
		// /a/b/e/f, plus their parent directories.

		wantA := map[string]int{
			"/":       http.StatusCreated,
			"/a/":     http.StatusMovedPermanently,
			"/a/b/":   http.StatusNotFound,
			"/a/b/c/": http.StatusNotFound,
		}[prefix]
		if err := do("MKCOL", srv.URL+"/a", nil, wantA); err != nil {
			t.Errorf("prefix=%-9q MKCOL /a: %v", prefix, err)
			continue
		}

		wantB := map[string]int{
			"/":       http.StatusCreated,
			"/a/":     http.StatusCreated,
			"/a/b/":   http.StatusMovedPermanently,
			"/a/b/c/": http.StatusNotFound,
		}[prefix]
		if err := do("MKCOL", srv.URL+"/a/b", nil, wantB); err != nil {
			t.Errorf("prefix=%-9q MKCOL /a/b: %v", prefix, err)
			continue
		}

		wantC := map[string]int{
			"/":       http.StatusCreated,
			"/a/":     http.StatusCreated,
			"/a/b/":   http.StatusCreated,
			"/a/b/c/": http.StatusMovedPermanently,
		}[prefix]
		if err := do("PUT", srv.URL+"/a/b/c", strings.NewReader(blah), wantC); err != nil {
			t.Errorf("prefix=%-9q PUT /a/b/c: %v", prefix, err)
			continue
		}

		wantD := map[string]int{
			"/":       http.StatusCreated,
			"/a/":     http.StatusCreated,
			"/a/b/":   http.StatusCreated,
			"/a/b/c/": http.StatusMovedPermanently,
		}[prefix]
		if err := do("COPY", srv.URL+"/a/b/c", nil, wantD, dst, srv.URL+"/a/b/d"); err != nil {
			t.Errorf("prefix=%-9q COPY /a/b/c /a/b/d: %v", prefix, err)
			continue
		}

		wantE := map[string]int{
			"/":       http.StatusCreated,
			"/a/":     http.StatusCreated,
			"/a/b/":   http.StatusCreated,
			"/a/b/c/": http.StatusNotFound,
		}[prefix]
		if err := do("MKCOL", srv.URL+"/a/b/e", nil, wantE); err != nil {
			t.Errorf("prefix=%-9q MKCOL /a/b/e: %v", prefix, err)
			continue
		}

		wantF := map[string]int{
			"/":       http.StatusCreated,
			"/a/":     http.StatusCreated,
			"/a/b/":   http.StatusCreated,
			"/a/b/c/": http.StatusNotFound,
		}[prefix]
		if err := do("MOVE", srv.URL+"/a/b/d", nil, wantF, dst, srv.URL+"/a/b/e/f"); err != nil {
			t.Errorf("prefix=%-9q MOVE /a/b/d /a/b/e/f: %v", prefix, err)
			continue
		}

		got, err := find(nil, fs, "/")
		if err != nil {
			t.Errorf("prefix=%-9q find: %v", prefix, err)
			continue
		}
		sort.Strings(got)
		want := map[string][]string{
			"/":       {"/", "/a", "/a/b", "/a/b/c", "/a/b/e", "/a/b/e/f"},
			"/a/":     {"/", "/b", "/b/c", "/b/e", "/b/e/f"},
			"/a/b/":   {"/", "/c", "/e", "/e/f"},
			"/a/b/c/": {"/"},
		}[prefix]
		if !reflect.DeepEqual(got, want) {
			t.Errorf("prefix=%-9q find:\ngot  %v\nwant %v", prefix, got, want)
			continue
		}
	}
}

func TestFilenameEscape(t *testing.T) {
	re := regexp.MustCompile(`<D:href>([^<]*)</D:href>`)
	do := func(method, urlStr string) (string, error) {
		req, err := http.NewRequest(method, urlStr, nil)
		if err != nil {
			return "", err
		}
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			return "", err
		}
		defer res.Body.Close()

		b, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return "", err
		}
		m := re.FindStringSubmatch(string(b))
		if len(m) != 2 {
			return "", errors.New("D:href not found")
		}

		return m[1], nil
	}

	testCases := []struct {
		name, want string
	}{{
		name: `/foo%bar`,
		want: `/foo%25bar`,
	}, {
		name: `/こんにちわ世界`,
		want: `/%E3%81%93%E3%82%93%E3%81%AB%E3%81%A1%E3%82%8F%E4%B8%96%E7%95%8C`,
	}, {
		name: `/Program Files/`,
		want: `/Program%20Files`,
	}, {
		name: `/go+lang`,
		want: `/go+lang`,
	}, {
		name: `/go&lang`,
		want: `/go&amp;lang`,
	}}
	fs := NewMemFS()
	for _, tc := range testCases {
		if strings.HasSuffix(tc.name, "/") {
			if err := fs.Mkdir(tc.name, 0755); err != nil {
				t.Fatalf("name=%q: Mkdir: %v", tc.name, err)
			}
		} else {
			f, err := fs.OpenFile(tc.name, os.O_CREATE, 0644)
			if err != nil {
				t.Fatalf("name=%q: OpenFile: %v", tc.name, err)
			}
			f.Close()
		}
	}

	srv := httptest.NewServer(&Handler{
		FileSystem: fs,
		LockSystem: NewMemLS(),
	})
	defer srv.Close()

	u, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range testCases {
		u.Path = tc.name
		got, err := do("PROPFIND", u.String())
		if err != nil {
			t.Errorf("name=%q: PROPFIND: %v", tc.name, err)
			continue
		}
		if got != tc.want {
			t.Errorf("name=%q: got %q, want %q", tc.name, got, tc.want)
		}
	}
}
