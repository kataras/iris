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
// DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER AND CONTRIBUTOR, GERASIMOS MAROPOULOS
// BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
// (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
// LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
// ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
// SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
package typescript

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"reflect"
)

type (
	// Tsconfig the struct for tsconfig.json
	Tsconfig struct {
		CompilerOptions CompilerOptions `json:"compilerOptions"`
		Exclude         []string        `json:"exclude"`
	}

	CompilerOptions struct {
		Declaration                      bool   `json:"declaration"`
		Module                           string `json:"module"`
		Target                           string `json:"target"`
		Watch                            bool   `json:"watch"`
		Charset                          string `json:"charset"`
		Diagnostics                      bool   `json:"diagnostics"`
		EmitBOM                          bool   `json:"emitBOM"`
		EmitDecoratorMetadata            bool   `json:"emitDecoratorMetadata"`
		ExperimentalDecorators           bool   `json:"experimentalDecorators"`
		InlineSourceMap                  bool   `json:"inlineSourceMap"`
		InlineSources                    bool   `json:"inlineSources"`
		IsolatedModules                  bool   `json:"isolatedModules"`
		Jsx                              string `json:"jsx"`
		ReactNamespace                   string `json:"reactNamespace"`
		ListFiles                        bool   `json:"listFiles"`
		Locale                           string `json:"locale"`
		MapRoot                          string `json:"mapRoot"`
		ModuleResolution                 string `json:"moduleResolution"`
		NewLine                          string `json:"newLine"`
		NoEmit                           bool   `json:"noEmit"`
		NoEmitOnError                    bool   `json:"noEmitOnError"`
		NoEmitHelpers                    bool   `json:"noEmitHelpers"`
		NoImplicitAny                    bool   `json:"noImplicitAny"`
		NoLib                            bool   `json:"noLib"`
		NoResolve                        bool   `json:"noResolve"`
		SkipDefaultLibCheck              bool   `json:"skipDefaultLibCheck"`
		OutDir                           string `json:"outDir"`
		OutFile                          string `json:"outFile"`
		PreserveConstEnums               bool   `json:"preserveConstEnums"`
		Pretty                           bool   `json:"pretty"`
		RemoveComments                   bool   `json:"removeComments"`
		RootDir                          string `json:"rootDir"`
		SourceMap                        bool   `json:"sourceMap"`
		SourceRoot                       string `json:"sourceRoot"`
		StripInternal                    bool   `json:"stripInternal"`
		SuppressExcessPropertyErrors     bool   `json:"suppressExcessPropertyErrors"`
		SuppressImplicitAnyIndexErrors   bool   `json:"suppressImplicitAnyIndexErrors"`
		AllowUnusedLabels                bool   `json:"allowUnusedLabels"`
		NoImplicitReturns                bool   `json:"noImplicitReturns"`
		NoFallthroughCasesInSwitch       bool   `json:"noFallthroughCasesInSwitch"`
		AllowUnreachableCode             bool   `json:"allowUnreachableCode"`
		ForceConsistentCasingInFileNames bool   `json:"forceConsistentCasingInFileNames"`
		AllowSyntheticDefaultImports     bool   `json:"allowSyntheticDefaultImports"`
		AllowJs                          bool   `json:"allowJs"`
		NoImplicitUseStrict              bool   `json:"noImplicitUseStrict"`
	}
)

// CompilerArgs returns the CompilerOptions' contents of the Tsconfig
// it reads the json tags, add '--' at the start of each one and returns an array of strings
func (tsconfig *Tsconfig) CompilerArgs() []string {
	val := reflect.ValueOf(tsconfig).Elem().FieldByName("CompilerOptions")
	compilerOpts := make([]string, val.NumField())
	for i := 0; i < val.NumField(); i++ {
		typeField := val.Type().Field(i)
		compilerOpts[i] = "--" + typeField.Tag.Get("json")
	}

	return compilerOpts
}

func FromFile(tsConfigAbsPath string) *Tsconfig {
	file, err := ioutil.ReadFile(tsConfigAbsPath)
	if err != nil {
		panic("[IRIS TypescriptPlugin.FromConfig]" + err.Error())
		os.Exit(1) //we have to exit at this point, we don't want to messup with the typescript files of the user if something goes wrong here...
	}
	config := &Tsconfig{}
	json.Unmarshal(file, config)
	return config
}

func DefaultTsconfig() *Tsconfig {
	return &Tsconfig{
		CompilerOptions: CompilerOptions{
			Module:        "commonjs",
			Target:        "es5",
			NoImplicitAny: false,
			SourceMap:     false,
		},
		Exclude: []string{"node_modules"},
	}

}
