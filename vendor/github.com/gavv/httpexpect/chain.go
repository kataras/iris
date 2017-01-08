package httpexpect

type chain struct {
	reporter Reporter
	failbit  bool
}

func makeChain(reporter Reporter) chain {
	return chain{reporter, false}
}

func (c *chain) failed() bool {
	return c.failbit
}

func (c *chain) fail(message string, args ...interface{}) {
	if c.failbit {
		return
	}
	c.failbit = true
	c.reporter.Errorf(message, args...)
}

func (c *chain) reset() {
	c.failbit = false
}

func (c *chain) assertFailed(r Reporter) {
	if !c.failbit {
		r.Errorf("expected chain is failed, but it's ok")
	}
}

func (c *chain) assertOK(r Reporter) {
	if c.failbit {
		r.Errorf("expected chain is ok, but it's failed")
	}
}
