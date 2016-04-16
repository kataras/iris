package cli

import (
	"os"
	"os/exec"
	"strings"
)

// PathSeparator is the string of os.PathSeparator
var PathSeparator = string(os.PathSeparator)

// ToDir converts & returns 'something' to PathSeparator+'something'+PathSeparator
func ToDir(folderWithoutSeparators string) string {
	return PathSeparator + folderWithoutSeparators + PathSeparator
}

// Command executes a command in shell and returns it's output, it's block version
func Command(command string, a ...string) (output string, err error) {
	var out []byte
	args := strings.Split(command, " ")
	if len(args) > 1 {
		command = args[0]
		args = append(args[1:], a...)
	} else {
		args = a
	}

	out, err = exec.Command(command, args...).Output()

	if err == nil {
		output = string(out)
	}

	return
}

// Exists, returns true if directory||file exists
func Exists(dir string) bool {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return false
	}
	return true
}
