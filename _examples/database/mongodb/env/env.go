package env

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
)

var (
	// Port is the PORT environment variable or 8080 if missing.
	// Used to open the tcp listener for our web server.
	Port string
	// DSN is the DSN environment variable or mongodb://localhost:27017 if missing.
	// Used to connect to the mongodb.
	DSN string
)

func parse() {
	Port = getDefault("PORT", "8080")
	DSN = getDefault("DSN", "mongodb://localhost:27017")

	log.Printf("• Port=%s\n", Port)
	log.Printf("• DSN=%s\n", DSN)
}

// Load loads environment variables that are being used across the whole app.
// Loading from file(s), i.e .env or dev.env
//
// Example of a 'dev.env':
// PORT=8080
// DSN=mongodb://localhost:27017
//
// After `Load` the callers can get an environment variable via `os.Getenv`.
func Load(envFileName string) {
	if args := os.Args; len(args) > 1 && args[1] == "help" {
		fmt.Fprintln(os.Stderr, "https://github.com/kataras/iris/blob/main/_examples/database/mongodb/README.md")
		os.Exit(-1)
	}

	// If more than one filename passed with comma separated then load from all
	// of these, a env file can be a partial too.
	envFiles := strings.Split(envFileName, ",")
	for _, envFile := range envFiles {
		if filepath.Ext(envFile) == "" {
			envFile += ".env"
		}

		if fileExists(envFile) {
			log.Printf("Loading environment variables from file: %s\n", envFile)

			if err := godotenv.Load(envFile); err != nil {
				panic(fmt.Sprintf("error loading environment variables from [%s]: %v", envFile, err))
			}
		}
	}

	// envMap, _ := godotenv.Read(envFiles...)
	// for k, v := range envMap {
	// 	log.Printf("◽ %s=%s\n", k, v)
	// }

	parse()
}

func getDefault(key string, def string) string {
	value := os.Getenv(key)
	if value == "" {
		os.Setenv(key, def)
		value = def
	}

	return value
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}
