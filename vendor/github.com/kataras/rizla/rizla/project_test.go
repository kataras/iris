package rizla

import (
	"path/filepath"
	"testing"
)

func TestProjectMatcher(t *testing.T) {
	files := map[string]bool{
		"main.go":                               true,
		"./sdsds.go":                            true,
		"C:/mydsadsa.go":                        true,
		"C:\\something\\go\\anything.go\\ok.go": true,
		"22312main.go":                          true,
		"323232.go":                             true,
		"-myfile2.go":                           true,
		"_____.go":                              true,
		".gooutput":                             !isWindows, // on non-windows the event is .gooutputblablabla, so we check for '.go'
		".god":                                  !isWindows,
		".goo":                                  !isWindows,
		".go.dgo":                               !isWindows,
		"":                                      false,
	}
	for k, v := range files {
		if gotV := DefaultGoMatcher(k); gotV != v {
			t.Fatalf("Matcher, expected %#v but got %#v for filename %s", v, gotV, k)
		}
	}
}

func TestProjectPrepare(t *testing.T) {
	p := NewProject("project_test.go")

	if p.Matcher == nil {
		t.Fatal("Matcher is nil, not defaulted")
	}

	mainfile, _ := filepath.Abs("project_test.go")
	if p.MainFile != mainfile {
		t.Fatalf("Mainfile is not the correct, expected %s but got %s", mainfile, p.MainFile)
	}

	if p.dir != filepath.Dir(mainfile) {
		t.Fatalf("Dir is not the correct, expected %s but got %s", filepath.Dir(mainfile), p.dir)
	}

}
