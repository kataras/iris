package typescript

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"reflect"
	"strconv"

	"github.com/kataras/iris/typescript/npm"
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

	// Config the configs for the Typescript plugin
	// Has five (5) fields
	//
	// 1. Bin: 	string, the typescript installation directory/typescript/lib/tsc.js, if empty it will search inside global npm modules
	// 2. Dir:     string, Dir set the root, where to search for typescript files/project. Default "./"
	// 3. Ignore:  string, comma separated ignore typescript files/project from these directories. Default "" (node_modules are always ignored)
	// 4. Tsconfig:  &typescript.Tsconfig{}, here you can set all compilerOptions if no tsconfig.json exists inside the 'Dir'
	// 5. Editor: 	typescript.Editor("username","password"), if setted then alm-tools browser-based typescript IDE will be available. Defailt is nil
	Config struct {
		// Bin the path of the tsc binary file
		// if empty then the plugin tries to find it
		Bin string
		// Dir the client side directory, which typescript (.ts) files are live
		Dir string
		// Ignore ignore folders, default is /node_modules/
		Ignore string
		// Tsconfig the typescript build configs, including the compiler's options
		Tsconfig *Tsconfig
	}
)

// CompilerArgs returns the CompilerOptions' contents of the Tsconfig
// it reads the json tags, add '--' at the start of each one and returns an array of strings
// this is from file
func (tsconfig *Tsconfig) CompilerArgs() []string {
	val := reflect.ValueOf(tsconfig).Elem().FieldByName("CompilerOptions") // -> for tsconfig *Tsconfig
	// val := reflect.ValueOf(tsconfig.CompilerOptions)
	compilerOpts := make([]string, 0) // 0 because we don't know the real valid options yet.
	for i := 0; i < val.NumField(); i++ {
		typeField := val.Type().Field(i)
		valueFieldG := val.Field(i)
		var valueField string
		// only if it's string or int we need to put that
		if valueFieldG.Kind() == reflect.String {
			//if valueFieldG.String() != "" {
			//valueField = strconv.QuoteToASCII(valueFieldG.String())
			//	}
			valueField = valueFieldG.String()
		} else if valueFieldG.Kind() == reflect.Int {
			if valueFieldG.Int() > 0 {
				valueField = strconv.Itoa(int(valueFieldG.Int()))
			}
		} else if valueFieldG.Kind() == reflect.Bool {
			valueField = strconv.FormatBool(valueFieldG.Bool())
		}

		if valueField != "" && valueField != "false" {
			// var opt string

			// key := typeField.Tag.Get("json")
			// // it's bool value of true then just --key, for example --watch
			// if valueField == "true" {
			// 	opt = "--" + key
			// } else {
			// 	// it's a string now, for example  -m commonjs
			// 	opt = "-" + string(key[0]) + " " + valueField
			// }
			key := "--" + typeField.Tag.Get("json")
			compilerOpts = append(compilerOpts, key)
			// the form is not '--module ES6' but os.Exec should recognise them as arguments
			// so we need to put the values on the next index
			if valueField != "true" {
				// it's a string now, for example  -m commonjs
				compilerOpts = append(compilerOpts, valueField)
			}

		}

	}

	return compilerOpts
}

// FromFile reads a file & returns the Tsconfig by its contents
func FromFile(tsConfigAbsPath string) (config Tsconfig, err error) {
	file, err := ioutil.ReadFile(tsConfigAbsPath)
	if err != nil {
		return
	}
	config = Tsconfig{}
	err = json.Unmarshal(file, &config)

	return
}

// DefaultTsconfig returns the default Tsconfig, with CompilerOptions module: commonjs, target: es5 and ignore the node_modules
func DefaultTsconfig() Tsconfig {
	return Tsconfig{
		CompilerOptions: CompilerOptions{
			Module:           "commonjs",
			Target:           "ES6",
			Jsx:              "react",
			ModuleResolution: "classic",
			Locale:           "en",
			Watch:            false,
			NoImplicitAny:    false,
			SourceMap:        false,
			Diagnostics:      true,
			NoEmit:           false,
			OutDir:           "", // taken from Config.Dir if it's not empty, otherwise ./ on Run()
		},
		Exclude: []string{"node_modules"},
	}

}

// DefaultConfig returns the default Options of the Typescript adaptor
// Bin and Editor are setting in runtime via the adaptor
func DefaultConfig() Config {
	root, err := os.Getwd()
	if err != nil {
		panic("typescript: cannot get the cwd")
	}
	compilerTsConfig := DefaultTsconfig()
	c := Config{
		Dir:      root + pathSeparator,
		Ignore:   nodeModules,
		Tsconfig: &compilerTsConfig,
	}
	c.Bin = npm.NodeModuleAbs("typescript/lib/tsc.js")
	return c
}
