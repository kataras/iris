package context_test

import (
	"testing"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/context"
)

// GetApplications must return every registered application. The count is
// delta-based so applications registered by other tests are tolerated.
func TestGetApplicationsReturnsRegistered(t *testing.T) {
	before := len(context.GetApplications())

	app1 := iris.New()
	app2 := iris.New()

	apps := context.GetApplications()
	if got := len(apps) - before; got != 2 {
		t.Fatalf("expected 2 new registered applications, got %d (total %d)", got, len(apps))
	}

	var found1, found2 bool
	for _, app := range apps {
		if app == context.Application(app1) {
			found1 = true
		}
		if app == context.Application(app2) {
			found2 = true
		}
	}

	if !found1 || !found2 {
		t.Fatalf("expected both new applications in GetApplications result (found1=%v, found2=%v)", found1, found2)
	}
}
