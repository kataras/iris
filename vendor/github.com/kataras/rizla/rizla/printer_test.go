package rizla

import (
	"io"
	"io/ioutil"
	"os"
	"testing"
)

func TestPrinter(t *testing.T) {
	// I tried to use memfs and some other libraries for virtual file system but the colorable  doesn't accept these, if anyone can help send me a message on the chat...
	logggerpath := "mytestlogger.txt"
	p := NewProject("mytestproject.go")
	logger, ferr := os.Create(logggerpath)
	defer func() {
		logger.Close()
		p.Out.Close()
		os.RemoveAll(logggerpath)
	}()

	if ferr != nil {
		t.Fatal(ferr)
	}
	//set the global output for the printer
	p.Out = NewPrinter(logger)

	s := "Hello"
	p.Out.Print(s)

	contents, err := ioutil.ReadFile(logggerpath)
	if err != nil || len(contents) == 0 && err == io.EOF {
		t.Fatalf("While trying to read from the logger %s", err.Error())
	} else {
		if len(contents) != len(s) || string(contents) != s {
			t.Fatalf("Logger reads but the its contents are not valid, expected len bytes %d but got %d, expected %s but got %s", len(s), len(contents), s, string(contents))
		}
	}
}
