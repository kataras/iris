// black-box testing
package errors_test

import (
	"fmt"
	"testing"

	"github.com/kataras/iris/core/errors"
)

var errMessage = "User with mail: %s already exists"
var errUserAlreadyExists = errors.New(errMessage)
var userMail = "user1@mail.go"
var expectedUserAlreadyExists = "User with mail: user1@mail.go already exists"

func getNewLine() string {
	if errors.NewLine {
		return "\n"
	}
	return ""
}

func ExampleError() {
	fmt.Print(errUserAlreadyExists.Format(userMail))
	// first output first Output line
	fmt.Print(errUserAlreadyExists.Format(userMail).Append("Please change your mail addr"))
	// second output second and third Output lines

	// Output:
	// User with mail: user1@mail.go already exists
	// User with mail: user1@mail.go already exists
	// Please change your mail addr
}

func do(method string, testErr *errors.Error, expectingMsg string, t *testing.T) {
	formattedErr := func() error {
		return testErr.Format(userMail)
	}()

	if formattedErr.Error() != expectingMsg {
		t.Fatalf("error %s failed, expected:\n%s got:\n%s", method, expectingMsg, formattedErr.Error())
	}
}

func TestFormat(t *testing.T) {
	expected := errors.Prefix + expectedUserAlreadyExists + getNewLine()
	do("Format Test", errUserAlreadyExists, expected, t)
}

func TestAppendErr(t *testing.T) {
	errors.NewLine = true
	errors.Prefix = "error: "

	errChangeMailMsg := "Please change your mail addr"
	errChangeMail := fmt.Errorf(errChangeMailMsg)                                                           // test go standard error
	expectedErrorMessage := errUserAlreadyExists.Format(userMail).Error() + errChangeMailMsg + getNewLine() // first Prefix and last newline lives inside do
	errAppended := errUserAlreadyExists.AppendErr(errChangeMail)
	do("Append Test Standard error type", &errAppended, expectedErrorMessage, t)
}

func TestAppendError(t *testing.T) {
	errors.NewLine = true
	errors.Prefix = "error: "

	errChangeMailMsg := "Please change your mail addr"
	errChangeMail := errors.New(errChangeMailMsg)                                                                // test Error struct
	expectedErrorMessage := errUserAlreadyExists.Format(userMail).Error() + errChangeMail.Error() + getNewLine() // first Prefix and last newline lives inside do
	errAppended := errUserAlreadyExists.AppendErr(errChangeMail)
	do("Append Test Error type", &errAppended, expectedErrorMessage, t)
}

func TestAppend(t *testing.T) {
	errors.NewLine = true
	errors.Prefix = "error: "

	errChangeMailMsg := "Please change your mail addr"
	expectedErrorMessage := errUserAlreadyExists.Format(userMail).Error() + errChangeMailMsg + getNewLine() // first Prefix and last newline lives inside do
	errAppended := errUserAlreadyExists.Append(errChangeMailMsg)
	do("Append Test string Message", &errAppended, expectedErrorMessage, t)
}

func TestNewLine(t *testing.T) {
	errors.NewLine = false

	errNoNewLine := errors.New(errMessage)
	expected := errors.Prefix + expectedUserAlreadyExists
	do("NewLine Test", errNoNewLine, expected, t)

	errors.NewLine = true
}

func TestPrefix(t *testing.T) {
	errors.Prefix = "MyPrefix: "

	errUpdatedPrefix := errors.New(errMessage)
	expected := errors.Prefix + expectedUserAlreadyExists + "\n"
	do("Prefix Test with "+errors.Prefix, errUpdatedPrefix, expected, t)
}
