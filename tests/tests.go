// Package tests empty
/*Why empty?
The only reason I don't make unit tests is because I think the whole story here is wrong. All unit tests may succed but in practise the app fail.
believe in the real micro example-usage-tests,
but if you have different opinion make PRs here.
*/
/*Alternative:
If you want to test your API use this, new, library ,which after my suggestion has fasthttp & Iris support: https://github.com/gavv/httpexpect
I have added some test examples to this directory also in order to help you
*/
package tests

// run all verbose mode:
// go test -v
// run all:
// go test .
// run specific:
// go test -run "Router"
