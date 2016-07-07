package iris

/*
The most iris.go file implementation tested at other files like context_test, http_test, the untested are the Static methods, the favicon and some interfaces, which I already
tested them on production and I don't expect unexpected behavior but if you think we need more:

CONTRIBUTE & DISCUSSION ABOUT TESTS TO: https://github.com/iris-contrib/tests
*/

// Notes:
//
// We use Default := New() via initDefault() and not api := New() neither just Default. because we want to cover as much code as possible
// The tests are mostly end-to-end, except some features like plugins.
//
