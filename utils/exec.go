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

package utils

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

var (
	// PathSeparator is the string of os.PathSeparator
	PathSeparator = string(os.PathSeparator)
)

type (
	// Cmd is a custom struch which 'implements' the *exec.Cmd
	Cmd struct {
		*exec.Cmd
	}
)

// Arguments sets the command line arguments, including the command as Args[0].
// If the args parameter is empty or nil, Run uses {Path}.
//
// In typical use, both Path and args are set by calling Command.
func (cmd *Cmd) Arguments(args ...string) *Cmd {
	cmd.Cmd.Args = append(cmd.Cmd.Args[0:1], args...) //we need the first argument which is the command
	return cmd
}

// AppendArguments appends the arguments to the exists
func (cmd *Cmd) AppendArguments(args ...string) *Cmd {
	cmd.Cmd.Args = append(cmd.Cmd.Args, args...)
	return cmd
}

// ResetArguments resets the arguments
func (cmd *Cmd) ResetArguments() *Cmd {
	cmd.Args = cmd.Args[0:1] //keep only the first because is the command
	return cmd
}

// Directory sets  the working directory of the command.
// If workingDirectory is the empty string, Run runs the command in the
// calling process's current directory.
func (cmd *Cmd) Directory(workingDirectory string) *Cmd {
	cmd.Cmd.Dir = workingDirectory
	return cmd
}

// CommandBuilder creates a Cmd object and returns it
// accepts 2 parameters, one is optionally
// first parameter is the command (string)
// second variatic parameter is the argument(s) (slice of string)
//
// the difference from the normal Command function is that you can re-use this Cmd, it doesn't execute until you  call its Command function
func CommandBuilder(command string, args ...string) *Cmd {
	return &Cmd{Cmd: exec.Command(command, args...)}
}

//the below is just for exec.Command:

// Command executes a command in shell and returns it's output, it's block version
func Command(command string, a ...string) (output string, err error) {
	var out []byte
	//if no args given, try to get them from the command
	if len(a) == 0 {
		commandArgs := strings.Split(command, " ")
		for _, commandArg := range commandArgs {
			if commandArg[0] == '-' { // if starts with - means that this is an argument, append it to the arguments
				a = append(a, commandArg)
			}
		}
	}
	out, err = exec.Command(command, a...).Output()

	if err == nil {
		output = string(out)
	}

	return
}

// MustCommand executes a command in shell and returns it's output, it's block version. It panics on an error
func MustCommand(command string, a ...string) (output string) {
	var out []byte
	var err error
	if len(a) == 0 {
		commandArgs := strings.Split(command, " ")
		for _, commandArg := range commandArgs {
			if commandArg[0] == '-' { // if starts with - means that this is an argument, append it to the arguments
				a = append(a, commandArg)
			}
		}
	}

	out, err = exec.Command(command, a...).Output()
	if err != nil {
		argsToString := strings.Join(a, " ")
		panic(fmt.Sprintf("\nError running the command %s", command+" "+argsToString))
	}

	output = string(out)

	return
}

// Exists returns true if directory||file exists
func Exists(dir string) bool {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return false
	}
	return true
}
