package rule

import (
	"github.com/kataras/iris/context"
)

// Validators are introduced to implement the RFC about cache (https://tools.ietf.org/html/rfc7234#section-1.1).

// PreValidator like middleware, executes before the cache action begins, if a callback returns false
// then this specific cache action, with specific request, is ignored and the real (original)
// handler is executed instead.
//
// I'll not add all specifications here I'll give the opportunity (public API in the httpcache package-level)
// to the end-user to specify her/his ignore rules too (ignore-only for now).
//
// Each package, nethttp and fhttp should implement their own encapsulations because of different request object.
//
// One function, accepts the request and returns false if should be denied/ignore, otherwise true.
// if at least one return false then the original handler will execute as it's
// and the whole cache action(set & get) should be ignored, it will be never go to the step of post-cache validations.
type PreValidator func(context.Context) bool

// PostValidator type is is introduced to implement the second part of the RFC about cache.
//
// Q: What's the difference between this and a PreValidator?
// A: PreValidator runs BEFORE trying to get the cache, it cares only for the request
//    and if at least one PreValidator returns false then it just runs the original handler and stop there, at the other hand
//    a PostValidator runs if all PreValidators returns true and original handler is executed but with a response recorder,
//    also the PostValidator should return true to store the cached response.
//    Last, a PostValidator accepts a context
//    in order to be able to catch the original handler's response,
//    the PreValidator checks only for request.
//
// If a function of type of PostValidator returns true then the (shared-always) cache is allowed to be stored.
type PostValidator func(context.Context) bool

// validatorRule is a rule witch receives PreValidators and PostValidators
// it's a 'complete set of rules', you can call it as a Responsible Validator,
// it's used when you the user wants to check for special things inside a request and a response.
type validatorRule struct {
	// preValidators a list of PreValidator functions, execute before real cache begins
	// if at least one of them returns false then the original handler will execute as it's
	// and the whole cache action(set & get) will be skipped for this specific client's request.
	//
	// Read-only 'runtime'
	preValidators []PreValidator

	// postValidators a list of PostValidator functions, execute after the original handler is executed with the response recorder
	// and exactly before this cached response is saved,
	// if at least one of them returns false then the response will be not saved for this specific client's request.
	//
	// Read-only 'runtime'
	postValidators []PostValidator
}

var _ Rule = &validatorRule{}

// DefaultValidator returns a new validator which contains the default pre and post cache validators
func DefaultValidator() Rule { return Validator(nil, nil) }

// Validator receives the preValidators and postValidators and returns a new Validator rule
func Validator(preValidators []PreValidator, postValidators []PostValidator) Rule {
	return &validatorRule{
		preValidators:  preValidators,
		postValidators: postValidators,
	}
}

// Claim returns true if incoming request can claim for a cached handler
// the original handler should run as it is and exit
func (v *validatorRule) Claim(ctx context.Context) bool {
	// check for pre-cache validators, if at least one of them return false
	// for this specific request, then skip the whole cache
	for _, shouldProcess := range v.preValidators {
		if !shouldProcess(ctx) {
			return false
		}
	}
	return true
}

// Valid returns true if incoming request and post-response from the original handler
// is valid to be store to the cache, if not(false) then the consumer should just exit
// otherwise(true) the consumer should store the cached response
func (v *validatorRule) Valid(ctx context.Context) bool {
	// check if it's a valid response, if it's not then just return.
	for _, valid := range v.postValidators {
		if !valid(ctx) {
			return false
		}
	}
	return true
}
