package unis

import (
	"regexp"
) // all validators based on regexp expressions should be live at this source file.

// NewMatcher returns a new validator which
// returns true and a nil error if the "expression"
// matches against a receiver string.
func NewMatcher(expression string) ValidatorFunc {
	r, err := regexp.Compile(expression)
	if err != nil {
		Logger(err.Error())
		return newFailure(err)
	}

	return func(str string) (bool, error) {
		return r.MatchString(str), nil
	}
}

const mailExpression = `^(([^<>()\[\]\\.,;:\s@"]+(\.[^<>()\[\]\\.,;:\s@"]+)*)|(".+"))@((\[[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}])|(([a-zA-Z\-0-9]+\.)+[a-zA-Z]{2,}))$`

// IsMail returns a validator which
// returns true and a nil error if the
// receiver string is an e-mail.
var IsMail = NewMatcher(mailExpression)
