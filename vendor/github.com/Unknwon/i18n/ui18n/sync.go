// Copyright 2013 Unknwon
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

package main

import (
	"log"
	"os"

	"github.com/Unknwon/com"
	"gopkg.in/ini.v1"
)

var cmdSync = &Command{
	UsageLine: "sync [source file] [target files]",
	Short:     "sync keys for locale files",
	Long: `to quickly sync keys for one or more locale files
based on the one you already have
`,
}

func init() {
	cmdSync.Run = syncLocales
}

func syncLocales(cmd *Command, args []string) {
	switch len(args) {
	case 0:
		log.Fatalln("No source locale file is specified")
	case 1:
		log.Fatalln("No target locale file is specified")
	}

	srcLocale, err := ini.Load(args[0])
	if err != nil {
		log.Fatalln(err)
	}

	// Load or create target locales.
	targets := args[1:]
	targetLocales := make([]*ini.File, len(targets))
	for i, target := range targets {
		if !com.IsExist(target) {
			os.Create(target)
		}

		targetLocales[i], err = ini.Load(target)
		if err != nil {
			log.Fatalln(err)
		}
	}

	for _, secName := range srcLocale.SectionStrings() {
		keyList := srcLocale.Section(secName).KeyStrings()
		for _, target := range targetLocales {
			sec, err := target.GetSection(secName)
			if err != nil {
				if sec, err = target.NewSection(secName); err != nil {
					log.Fatalln(err)
				}
			}
			for _, k := range keyList {
				if _, err = sec.GetKey(k); err != nil {
					sec.NewKey(k, "")
				}
			}
		}
	}

	for i, target := range targetLocales {
		if err = target.SaveTo(targets[i]); err != nil {
			log.Fatalln(err)
		}
	}
}
