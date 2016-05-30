package config

import (
	"os"
	"reflect"

	"github.com/imdario/mergo"
)

var (
	pathSeparator = string(os.PathSeparator)
	nodeModules   = pathSeparator + "node_modules" + pathSeparator
)

type (
	// Tsconfig the struct for tsconfig.json
	Tsconfig struct {
		CompilerOptions CompilerOptions `json:"compilerOptions"`
		Exclude         []string        `json:"exclude"`
	}

	// CompilerOptions contains all the compiler options used by the tsc (typescript compiler)
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

	Typescript struct {
		Bin      string
		Dir      string
		Ignore   string
		Tsconfig Tsconfig
		Editor   Editor
	}
)

// CompilerArgs returns the CompilerOptions' contents of the Tsconfig
// it reads the json tags, add '--' at the start of each one and returns an array of strings
func (tsconfig Tsconfig) CompilerArgs() []string {
	//val := reflect.ValueOf(tsconfig).Elem().FieldByName("CompilerOptions") -> for tsconfig *Tsconfig
	val := reflect.ValueOf(tsconfig).FieldByName("CompilerOptions")
	compilerOpts := make([]string, val.NumField())
	for i := 0; i < val.NumField(); i++ {
		typeField := val.Type().Field(i)
		compilerOpts[i] = "--" + typeField.Tag.Get("json")
	}

	return compilerOpts
}

// DefaultTsconfig returns the default Tsconfig, with CompilerOptions module: commonjs, target: es5 and ignore the node_modules
func DefaultTsconfig() Tsconfig {
	return Tsconfig{
		CompilerOptions: CompilerOptions{
			Module:        "commonjs",
			Target:        "es5",
			NoImplicitAny: false,
			SourceMap:     false,
		},
		Exclude: []string{"node_modules"},
	}

}

// DefaultTypescript returns the default Options of the Typescript plugin
// Bin and Editor are setting in runtime via the plugin
func DefaultTypescript() Typescript {
	root, err := os.Getwd()
	if err != nil {
		panic("Typescript Plugin: Cannot get the Current Working Directory !!! [os.getwd()]")
	}
	c := Typescript{Dir: root + pathSeparator, Ignore: nodeModules, Tsconfig: DefaultTsconfig()}
	return c

}

// Merge merges the default with the given config and returns the result
func (c Typescript) Merge(cfg []Typescript) (config Typescript) {

	if len(cfg) > 0 {
		config = cfg[0]
		mergo.Merge(&config, c)
	} else {
		_default := c
		config = _default
	}

	return
}
