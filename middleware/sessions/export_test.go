package sessions

// IsValidSessionID exposes isValidSessionID to the external test package,
// see uuid_leniency_test.go.
var IsValidSessionID = isValidSessionID

// CopyThroughGob exposes copyThroughGob to the external test package,
// see gob_copier_test.go.
var CopyThroughGob = copyThroughGob
