package unis

// Validator is just another interface
// for string utilities.
// All validators should implement this interface.
// Contains only one function "Valid" which accepts
// a string and returns a boolean and an error.
// It should compare that string "str" with
// something and returns a true, nil or false, err.
//
// Validators can be used side by side with Processors.
//
// See .If for more.
type Validator interface {
	Valid(str string) (ok bool, err error)
}

// ValidatorFunc is just an "alias" for the Validator interface.
// It implements the Validator.
type ValidatorFunc func(str string) (bool, error)

// Valid accepts
// a string and returns a boolean and an error.
// It should compare that string "str" with
// something and returns a true, nil or false, err.
func (v ValidatorFunc) Valid(str string) (bool, error) {
	return v(str)
}

func newFailure(err error) ValidatorFunc {
	return func(string) (bool, error) {
		return false, err
	}
}

// If receives a "validator" Validator and two Processors,
// the first processor will be called when that validator passed,
// the second processor will be called when the validator failed.
// Both of the processors ("succeed" and "failure"), as always,
// can be results of .NewChain.
//
// Returns a new string processor which checks the "validator"
// against the "original" string, if passed then it runs the
// "succeed", otherwise it runs the "failure".
//
// Remember: it returns a ProcessorFunc, meaning that can be used in a new chain too.
func If(validator Validator, succeed Processor, failure Processor) ProcessorFunc {
	return func(original string) string {
		if ok, _ := validator.Valid(original); ok {
			return succeed.Process(original)
		}
		return failure.Process(original)
	}
}
