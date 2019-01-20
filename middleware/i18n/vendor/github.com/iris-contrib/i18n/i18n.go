// Copyright 2013 Unknwon
// Copyright 2017 Gerasimos (Makis) Maropoulos <https://github.com/@kataras>
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

// Package i18n is for app Internationalization and Localization.
package i18n

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"gopkg.in/ini.v1"
)

var (
	// ErrLangAlreadyExist throwed when language is already exists.
	ErrLangAlreadyExist = errors.New("lang already exists")

	locales = &localeStore{store: make(map[string]*locale)}
)

// add support for multi language file per language.
// ini has already implement a  *ini.File#Append
// BUT IT DOESN'T F WORKING, SO:
type localeFiles struct {
	files []*ini.File
}

// Get returns a the value from the "keyName" on the "sectionName"
// by searching all loc.files.
func (loc *localeFiles) GetKey(sectionName, keyName string) (*ini.Key, error) {
	for _, f := range loc.files {
		// returns the first available.
		// section is the same for both files if key exists.
		if sec, serr := f.GetSection(sectionName); serr == nil && sec != nil {
			if k, err := sec.GetKey(keyName); err == nil && k != nil {
				return k, err
			}
		}
	}

	return nil, fmt.Errorf("not found")
}

// Reload reloads and parses all data sources.
func (loc *localeFiles) Reload() error {
	for _, f := range loc.files {
		if err := f.Reload(); err != nil {
			return err
		}
	}
	return nil
}

func (loc *localeFiles) addFile(file *ini.File) error {
	loc.files = append(loc.files, file)
	return loc.Reload()
}

type locale struct {
	id       int
	lang     string
	langDesc string
	message  *localeFiles
}

type localeStore struct {
	langs       []string
	langDescs   []string
	store       map[string]*locale
	defaultLang string
}

// Get target language string
func (d *localeStore) Get(lang, section, format string) (string, bool) {
	if locale, ok := d.store[lang]; ok {
		// println(lang + " language found, let's see keys")
		if key, err := locale.message.GetKey(section, format); err == nil && key != nil {
			// println("value for section= " + section + "and key=" + format + " found")
			return key.Value(), true
		}
	}

	if len(d.defaultLang) > 0 && lang != d.defaultLang {
		// println("use the default lang: " + d.defaultLang)
		return d.Get(d.defaultLang, section, format)
	}

	return "", false
}

func (d *localeStore) Add(lang, langDesc string, source interface{}, others ...interface{}) error {

	file, err := ini.Load(source, others...)
	if err != nil {
		return err
	}
	file.BlockMode = false

	// if already exists add the file on this language.
	lc, ok := d.store[lang]
	if !ok {
		// println("add lang and init message: " + lang)
		// create a new one if doesn't exist.

		lc = new(locale)
		lc.message = new(localeFiles)
		lc.lang = lang
		lc.langDesc = langDesc
		lc.id = len(d.langs)
		// add the first language if not exists.
		d.langs = append(d.langs, lang)
		d.langDescs = append(d.langDescs, langDesc)
		d.store[lang] = lc
	}

	// println("append a file for language: " + lang)

	return lc.message.addFile(file)
}

func (d *localeStore) Reload(langs ...string) (err error) {
	if len(langs) == 0 {
		for _, lc := range d.store {
			if err = lc.message.Reload(); err != nil {
				return err
			}
		}
	} else {
		for _, lang := range langs {
			if lc, ok := d.store[lang]; ok {
				if err = lc.message.Reload(); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// SetDefaultLang sets default language which is a indicator that
// when target language is not found, try find in default language again.
func SetDefaultLang(lang string) {
	locales.defaultLang = lang
}

// ReloadLangs reloads locale files.
func ReloadLangs(langs ...string) error {
	return locales.Reload(langs...)
}

// Count returns number of languages that are registered.
func Count() int {
	return len(locales.langs)
}

// ListLangs returns list of all locale languages.
func ListLangs() []string {
	langs := make([]string, len(locales.langs))
	copy(langs, locales.langs)
	return langs
}

func ListLangDescs() []string {
	langDescs := make([]string, len(locales.langDescs))
	copy(langDescs, locales.langDescs)
	return langDescs
}

// IsExist returns true if given language locale exists.
func IsExist(lang string) bool {
	_, ok := locales.store[lang]
	return ok
}

// IsExistSimilar returns true if the language, or something similar
// exists (e.g. en-US maps to en).
// it returns the found name and whether it was able to match something.
// - PATCH by @j-lenoch.
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

// IndexLang returns index of language locale,
// it returns -1 if locale not exists.
func IndexLang(lang string) int {
	if lc, ok := locales.store[lang]; ok {
		return lc.id
	}
	return -1
}

// GetLangByIndex return language by given index.
func GetLangByIndex(index int) string {
	if index < 0 || index >= len(locales.langs) {
		return ""
	}
	return locales.langs[index]
}

func GetDescriptionByIndex(index int) string {
	if index < 0 || index >= len(locales.langDescs) {
		return ""
	}

	return locales.langDescs[index]
}

func GetDescriptionByLang(lang string) string {
	return GetDescriptionByIndex(IndexLang(lang))
}

func SetMessageWithDesc(lang, langDesc string, localeFile interface{}, otherLocaleFiles ...interface{}) error {
	return locales.Add(lang, langDesc, localeFile, otherLocaleFiles...)
}

// SetMessage sets the message file for localization.
func SetMessage(lang string, localeFile interface{}, otherLocaleFiles ...interface{}) error {
	return SetMessageWithDesc(lang, lang, localeFile, otherLocaleFiles...)
}

// Locale represents the information of localization.
type Locale struct {
	Lang string
}

// Tr translates content to target language.
func (l Locale) Tr(format string, args ...interface{}) string {
	return Tr(l.Lang, format, args...)
}

// Index returns lang index of LangStore.
func (l Locale) Index() int {
	return IndexLang(l.Lang)
}

// Tr translates content to target language.
func Tr(lang, format string, args ...interface{}) string {
	var section string

	idx := strings.IndexByte(format, '.')
	if idx > 0 {
		section = format[:idx]
		format = format[idx+1:]
	}

	value, ok := locales.Get(lang, section, format)
	if ok {
		format = value
	}

	if len(args) > 0 {
		params := make([]interface{}, 0, len(args))
		for _, arg := range args {
			if arg == nil {
				continue
			}

			val := reflect.ValueOf(arg)
			if val.Kind() == reflect.Slice {
				for i := 0; i < val.Len(); i++ {
					params = append(params, val.Index(i).Interface())
				}
			} else {
				params = append(params, arg)
			}
		}
		return fmt.Sprintf(format, params...)
	}
	return format
}
