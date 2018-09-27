package version

import (
	"time"

	"github.com/hashicorp/go-version"
)

// Version is a version wrapper which
// contains some additional customized properties.
type Version struct {
	version.Version
	WrittenAt    time.Time
	ChangelogURL string
}

// Result is the compare result type.
// Available types are Invalid, Smaller, Equal or Larger.
type Result int32

const (
	// Smaller when the compared version is smaller than the latest one.
	Smaller Result = -1
	// Equal when the compared version is equal with the latest one.
	Equal Result = 0
	// Larger when the compared version is larger than the latest one.
	Larger Result = 1
	// Invalid means that an error occurred when comparing the versions.
	Invalid Result = -2
)

// Compare compares the "versionStr" with the latest Iris version,
// opossite to the version package
// it returns the result of the "versionStr" not the "v" itself.
func (v *Version) Compare(versionStr string) Result {
	if len(v.Version.String()) == 0 {
		// if version not refreshed, by an internet connection lose,
		// then return Invalid.
		return Invalid
	}

	other, err := version.NewVersion(versionStr)
	if err != nil {
		return Invalid
	}
	return Result(other.Compare(&v.Version))
}

// Acquire returns the latest version info wrapper.
// It calls the fetch.
func Acquire() (v Version) {
	newVersion, changelogURL := fetch()
	if newVersion == nil { // if github was down then don't panic, just set it as the smallest version.
		newVersion, _ = version.NewVersion("0.0.1")
	}

	v = Version{
		Version:      *newVersion,
		WrittenAt:    time.Now(),
		ChangelogURL: changelogURL,
	}

	return
}
