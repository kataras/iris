package context

// TransactionErrResult could be named also something like 'MaybeError',
// it is useful to send it on transaction.Complete in order to execute a custom error mesasge to the user.
//
// in simple words it's just a 'traveler message' between the transaction and its scope.
// it is totally optional
type TransactionErrResult struct {
	StatusCode int
	// if reason is empty then the already relative registered (custom or not)
	// error will be executed if the scope allows that.
	Reason      string
	ContentType string
}

// Error returns the reason given by the user or an empty string
func (err TransactionErrResult) Error() string {
	return err.Reason
}

// IsFailure returns true if this is an actual error
func (err TransactionErrResult) IsFailure() bool {
	return StatusCodeNotSuccessful(err.StatusCode)
}

// NewTransactionErrResult returns a new transaction result with the given error message,
// it can be empty too, but if not then the transaction's scope is decided what to do with that
func NewTransactionErrResult() TransactionErrResult {
	return TransactionErrResult{}
}

// TransactionScope is the manager of the transaction's response, can be resseted and skipped
// from its parent context or execute an error or skip other transactions
type TransactionScope interface {
	// EndTransaction returns if can continue to the next transactions or not (false)
	// called after Complete, empty or not empty error
	EndTransaction(maybeErr TransactionErrResult, ctx Context) bool
}

// TransactionScopeFunc the transaction's scope signature
type TransactionScopeFunc func(maybeErr TransactionErrResult, ctx Context) bool

// EndTransaction ends the transaction with a callback to itself, implements the TransactionScope interface
func (tsf TransactionScopeFunc) EndTransaction(maybeErr TransactionErrResult, ctx Context) bool {
	return tsf(maybeErr, ctx)
}

//  +------------------------------------------------------------+
//  | Transaction Implementation                                 |
//  +------------------------------------------------------------+

// Transaction gives the users the opportunity to code their route handlers  cleaner and safier
// it receives a scope which is decided when to send an error to the user, recover from panics
// stop the execution of the next transactions and so on...
//
// it's default scope is the TransientTransactionScope which is silently
// skips the current transaction's response if transaction.Complete accepts a non-empty error.
//
// Create and set custom transactions scopes with transaction.SetScope.
//
// For more information please visit the tests.
type Transaction struct {
	context  Context
	parent   Context
	hasError bool
	scope    TransactionScope
}

func newTransaction(from *context) *Transaction {
	tempCtx := *from
	writer := tempCtx.ResponseWriter().Clone()
	tempCtx.ResetResponseWriter(writer)
	t := &Transaction{
		parent:  from,
		context: &tempCtx,
		scope:   TransientTransactionScope,
	}

	return t
}

// Context returns the current context of the transaction.
func (t *Transaction) Context() Context {
	return t.context
}

// SetScope sets the current transaction's scope
// iris.RequestTransactionScope || iris.TransientTransactionScope (default).
func (t *Transaction) SetScope(scope TransactionScope) {
	t.scope = scope
}

// Complete completes the transaction
// rollback and send an error when the error is not empty.
// The next steps depends on its Scope.
//
// The error can be a type of context.NewTransactionErrResult().
func (t *Transaction) Complete(err error) {
	maybeErr := TransactionErrResult{}

	if err != nil {
		t.hasError = true

		statusCode := 400 // bad request
		reason := err.Error()
		cType := "text/plain; charset=" + t.context.Application().ConfigurationReadOnly().GetCharset()

		if errWstatus, ok := err.(TransactionErrResult); ok {
			if errWstatus.StatusCode > 0 {
				statusCode = errWstatus.StatusCode
			}

			if errWstatus.Reason != "" {
				reason = errWstatus.Reason
			}
			// get the content type used on this transaction
			if cTypeH := t.context.ResponseWriter().Header().Get(contentTypeHeaderKey); cTypeH != "" {
				cType = cTypeH
			}

		}
		maybeErr.StatusCode = statusCode
		maybeErr.Reason = reason
		maybeErr.ContentType = cType
	}
	// the transaction ends with error or not error, it decides what to do next with its Response
	// the Response is appended to the parent context an all cases but it checks for empty body,headers and all that,
	// if they are empty (silent error or not error at all)
	// then all transaction's actions are skipped as expected
	canContinue := t.scope.EndTransaction(maybeErr, t.context)
	if !canContinue {
		t.parent.SkipTransactions()
	}
}

// TransientTransactionScope explanation:
//
// independent 'silent' scope, if transaction fails (if transaction.IsFailure() == true)
// then its response is not written to the real context no error is provided to the user.
// useful for the most cases.
var TransientTransactionScope = TransactionScopeFunc(func(maybeErr TransactionErrResult, ctx Context) bool {
	if maybeErr.IsFailure() {
		ctx.Recorder().Reset() // this response is skipped because it's empty.
	}
	return true
})

// RequestTransactionScope explanation:
//
// if scope fails (if transaction.IsFailure() == true)
// then the rest of the context's response  (transaction or normal flow)
// is not written to the client, and an error status code is written instead.
var RequestTransactionScope = TransactionScopeFunc(func(maybeErr TransactionErrResult, ctx Context) bool {
	if maybeErr.IsFailure() {

		// we need to register a beforeResponseFlush event here in order
		// to execute last the FireStatusCode
		// (which will reset the whole response's body, status code and headers setted from normal flow or other transactions too)
		ctx.ResponseWriter().SetBeforeFlush(func() {
			// we need to re-take the context's response writer
			// because inside here the response writer is changed to the original's
			// look ~context:1306
			w := ctx.ResponseWriter().(*ResponseRecorder)
			if maybeErr.Reason != "" {
				// send the error with the info user provided
				w.SetBodyString(maybeErr.Reason)
				w.WriteHeader(maybeErr.StatusCode)
				ctx.ContentType(maybeErr.ContentType)
			} else {
				// else execute the registered user error and skip the next transactions and all normal flow,
				ctx.StatusCode(maybeErr.StatusCode)
				ctx.StopExecution()
			}
		})

		return false
	}

	return true
})
