i18n [![GoDoc](https://godoc.org/github.com/iris-contrib/i18n?status.svg)](https://godoc.org/github.com/iris-contrib/i18n)
[![build status](https://img.shields.io/travis/iris-contrib/i18n/master.svg?style=flat-square)](https://travis-ci.org/iris-contrib/i18n)
====

Package i18n is for app Internationalization and Localization.

This package is a fork of the https://github.com/Unknwon/i18n.

It's heavly used inside the https://github.com/kataras/iris/tree/master/middleware/i18n middleware.

# Changes

This package provides some additional functionality compared to the original one;

PATCH by @j-lenoch at L129:

```go
// IsExistSimilar returns true if the language, or something similar
// exists (e.g. en-US maps to en).
// it returns the found name and whether it was able to match something.
func IsExistSimilar(lang string) (string, bool) {
_, ok := locales.store[lang]
if ok {
    return lang, true
}

// remove the internationalization element from the IETF code
code := strings.Split(lang, "-")[0]

for _, lc := range locales.store {
    if strings.Contains(lc.lang, code) {
        return lc.lang, true
    }
}

return "", false
}
```
