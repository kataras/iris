package context

import "testing"

// The JSONP callback is written into a "text/javascript" response body verbatim,
// and the canonical way to fill it is straight from the request:
//
//	ctx.JSONP(v, iris.JSONP{Callback: ctx.URLParam("callback")})
//
// so anything that is not a plain identifier reference is attacker-authored
// script served from the application's own origin.
func TestIsValidJSONPCallback(t *testing.T) {
	valid := []string{
		"cb",
		"callback",
		"_private",
		"$jquery",
		"window.cb",
		"my.deep.callback",
		"callbacks[0]",
		"a.b[12].c",
		"cb123",
	}

	for _, callback := range valid {
		if !IsValidJSONPCallback(callback) {
			t.Errorf("expected %q to be a valid JSONP callback", callback)
		}
	}

	invalid := []string{
		"",
		"</script><script>alert(1)//",
		"alert(1)",
		"cb;alert(1)",
		"cb()",
		"cb,alert",
		"1cb",
		".cb",
		"cb.",
		"cb['x']",
		"cb[\"x\"]",
		"cb\n",
		"cb ",
		" cb",
		"cb+cb",
		"<img src=x onerror=alert(1)>",
		"cb\t",
		"eval\x00",
	}

	for _, callback := range invalid {
		if IsValidJSONPCallback(callback) {
			t.Errorf("expected %q to be rejected as a JSONP callback", callback)
		}
	}

	// A pathological length is rejected before it reaches the regexp.
	long := make([]byte, 200)
	for i := range long {
		long[i] = 'a'
	}
	if IsValidJSONPCallback(string(long)) {
		t.Error("expected an over-long callback to be rejected")
	}
}
