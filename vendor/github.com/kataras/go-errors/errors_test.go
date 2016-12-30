package errors

import (
	"fmt"
	"testing"
)

var errMessage = "User with mail: %s already exists"
var errUserAlreadyExists = New(errMessage)
var userMail = "user1@mail.go"
var expectedUserAlreadyExists = "User with mail: user1@mail.go already exists"

func getNewLine() string {
	if NewLine {
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
	// Error: User with mail: user1@mail.go already exists
	// Error: User with mail: user1@mail.go already exists
	// Please change your mail addr
}

func do(method string, testErr *Error, expectingMsg string, t *testing.T) {
	formattedErr := func() error {
		return testErr.Format(userMail)
	}()

	if formattedErr.Error() != expectingMsg {
		t.Fatalf("Error %s failed, expected:\n%s got:\n%s", method, expectingMsg, formattedErr.Error())
	}
}

func TestFormat(t *testing.T) {
	expected := Prefix + expectedUserAlreadyExists + getNewLine()
	do("Format Test", errUserAlreadyExists, expected, t)
}

func TestAppendErr(t *testing.T) {
	NewLine = true
	Prefix = "Error: "

	errChangeMailMsg := "Please change your mail addr"
	errChangeMail := fmt.Errorf(errChangeMailMsg)                                                           // test go standard error
	expectedErrorMessage := errUserAlreadyExists.Format(userMail).Error() + errChangeMailMsg + getNewLine() // first Prefix and last newline lives inside do
	errAppended := errUserAlreadyExists.AppendErr(errChangeMail)
	do("Append Test Standard error type", &errAppended, expectedErrorMessage, t)
}

func TestAppendError(t *testing.T) {
	NewLine = true
	Prefix = "Error: "

	errChangeMailMsg := "Please change your mail addr"
	errChangeMail := New(errChangeMailMsg)                                                                       // test Error struct
	expectedErrorMessage := errUserAlreadyExists.Format(userMail).Error() + errChangeMail.Error() + getNewLine() // first Prefix and last newline lives inside do
	errAppended := errUserAlreadyExists.AppendErr(errChangeMail)
	do("Append Test Error type", &errAppended, expectedErrorMessage, t)
}

func TestAppend(t *testing.T) {
	NewLine = true
	Prefix = "Error: "

	errChangeMailMsg := "Please change your mail addr"
	expectedErrorMessage := errUserAlreadyExists.Format(userMail).Error() + errChangeMailMsg + getNewLine() // first Prefix and last newline lives inside do
	errAppended := errUserAlreadyExists.Append(errChangeMailMsg)
	do("Append Test string Message", &errAppended, expectedErrorMessage, t)
}

func TestNewLine(t *testing.T) {
	NewLine = false

	errNoNewLine := New(errMessage)
	expected := Prefix + expectedUserAlreadyExists
	do("NewLine Test", errNoNewLine, expected, t)

	NewLine = true
}

func TestPrefix(t *testing.T) {
	Prefix = "MyPrefix: "

	errUpdatedPrefix := New(errMessage)
	expected := Prefix + expectedUserAlreadyExists + "\n"
	do("Prefix Test with "+Prefix, errUpdatedPrefix, expected, t)
}
