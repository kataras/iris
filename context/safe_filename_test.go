package context

import (
	"path/filepath"
	"testing"
)

func TestSafeFilenameRejectsReservedDeviceNames(t *testing.T) {
	dir := t.TempDir()

	// These pass ValidFilenameRegexp and the canonical-prefix check, but on
	// Windows os.Create opens a device rather than a file.
	reserved := []string{
		"NUL", "nul", "Nul",
		"CON", "CON.txt", "con.log",
		"PRN", "AUX",
		"COM1", "com1.txt", "COM9",
		"LPT1", "lpt3.dat",
	}

	for _, name := range reserved {
		_, _, ok, err := SafeFilename(dir, name)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if ok {
			t.Errorf("expected the reserved device name %q to be rejected", name)
		}
	}
}

func TestSafeFilenameAcceptsOrdinaryNames(t *testing.T) {
	dir := t.TempDir()

	// Names that merely start with the reserved prefix are fine.
	ok_names := []string{
		"report.txt",
		"CONFIG.json",
		"console.log",
		"communications.csv",
		"nullable-values.txt",
		"lpt.txt",
		"COM.txt",
		"my_file-1.tar.gz",
	}

	for _, name := range ok_names {
		destPath, filename, ok, err := SafeFilename(dir, name)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if !ok {
			t.Errorf("expected %q to be accepted", name)
			continue
		}
		if filename != name {
			t.Errorf("%s: filename = %q", name, filename)
		}
		if filepath.Dir(destPath) == "" {
			t.Errorf("%s: empty destination directory", name)
		}
	}
}

// A traversal attempt is neutralized rather than rejected: filepath.Base strips
// every directory component, so the upload lands in the destination directory
// under its base name. The property that matters is that the returned path can
// never leave that directory.
func TestSafeFilenameNeutralizesTraversal(t *testing.T) {
	dir := t.TempDir()

	canonical, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}

	for _, name := range []string{
		"../escape.txt",
		"..\\escape.txt",
		"/etc/passwd",
		"C:\\Windows\\System32\\drivers\\etc\\hosts",
		"sub/dir/file.txt",
		"../../../../../../etc/shadow",
	} {
		destPath, filename, ok, err := SafeFilename(dir, name)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if !ok {
			// Rejected outright is also an acceptable outcome (it is what happens
			// when the stripped name still contains a disallowed character, e.g.
			// a backslash on non-Windows).
			continue
		}

		if filepath.Dir(destPath) != canonical {
			t.Errorf("%q: destPath = %q, escaped the destination directory %q", name, destPath, canonical)
		}
		if filename != filepath.Base(filepath.ToSlash(name)) {
			t.Errorf("%q: filename = %q, expected the sanitized base name", name, filename)
		}
	}
}

func TestSafeFilenameRejectsDisallowedCharacters(t *testing.T) {
	dir := t.TempDir()

	for _, name := range []string{
		"file\x00.txt",  // NUL byte
		"file name.txt", // the space is outside the allowlist
		"file:stream",   // NTFS alternate data stream
		"file*.txt",
		"file?.txt",
		"<script>.txt",
		"",
	} {
		_, _, ok, err := SafeFilename(dir, name)
		if err != nil {
			t.Fatalf("%q: %v", name, err)
		}
		if ok {
			t.Errorf("expected %q to be rejected", name)
		}
	}
}

// The destination is resolved through EvalSymlinks and the upload is joined onto
// the resolved directory. Building the path from the raw directory and then
// comparing it against the resolved one failed closed whenever the two differed,
// silently skipping every upload.
func TestSafeFilenameResolvesDestinationDirectory(t *testing.T) {
	dir := t.TempDir()

	destPath, filename, ok, err := SafeFilename(dir, "photo.png")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected photo.png to be accepted")
	}
	if filename != "photo.png" {
		t.Fatalf("filename = %q", filename)
	}

	canonical, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(canonical, "photo.png"); destPath != want {
		t.Fatalf("destPath = %q, expected %q", destPath, want)
	}
}
