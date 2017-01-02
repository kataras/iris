// Black-box Testing
package iris_test

import (
	"fmt"
	"github.com/kataras/iris"
	"testing"
)

const (
	testPluginExDescription = "Description for My test plugin"
	testPluginExName        = "My test plugin"
)

type testPluginEx struct {
	named, activated, descriptioned          bool
	prelistenran, postlistenran, precloseran bool
}

func (t *testPluginEx) GetName() string {
	fmt.Println("GetName Struct")
	t.named = true
	return testPluginExName
}

func (t *testPluginEx) GetDescription() string {
	fmt.Println("GetDescription Struct")
	t.descriptioned = true
	return testPluginExDescription
}

func (t *testPluginEx) Activate(p iris.PluginContainer) error {
	fmt.Println("Activate Struct")
	t.activated = true
	return nil
}

func (t *testPluginEx) PreListen(*iris.Framework) {
	fmt.Println("PreListen Struct")
	t.prelistenran = true
}

func (t *testPluginEx) PostListen(*iris.Framework) {
	fmt.Println("PostListen Struct")
	t.postlistenran = true
}

func (t *testPluginEx) PreClose(*iris.Framework) {
	fmt.Println("PreClose Struct")
	t.precloseran = true
}

func ExamplePlugins_Add() {
	iris.ResetDefault()
	iris.Default.Set(iris.OptionDisableBanner(true))
	iris.Plugins.Add(iris.PreListenFunc(func(*iris.Framework) {
		fmt.Println("PreListen Func")
	}))

	iris.Plugins.Add(iris.PostListenFunc(func(*iris.Framework) {
		fmt.Println("PostListen Func")
	}))

	iris.Plugins.Add(iris.PreCloseFunc(func(*iris.Framework) {
		fmt.Println("PreClose Func")
	}))

	myplugin := &testPluginEx{}
	iris.Plugins.Add(myplugin)
	desc := iris.Plugins.GetDescription(myplugin)
	fmt.Println(desc)

	// travis have problems if I do that using
	// Listen(":8080") and Close()
	iris.Plugins.DoPreListen(iris.Default)
	iris.Plugins.DoPostListen(iris.Default)
	iris.Plugins.DoPreClose(iris.Default)

	// Output:
	// GetName Struct
	// Activate Struct
	// GetDescription Struct
	// Description for My test plugin
	// PreListen Func
	// PreListen Struct
	// PostListen Func
	// PostListen Struct
	// PreClose Func
	// PreClose Struct
}

// if a plugin has GetName, then it should be registered only one time, the name exists for that reason, it's like unique  ID
func TestPluginDublicateName(t *testing.T) {
	iris.ResetDefault()
	var plugins = iris.Default.Plugins
	firstNamedPlugin := &testPluginEx{}
	sameNamedPlugin := &testPluginEx{}
	// err := plugins.Add(firstNamedPlugin, sameNamedPlugin) or
	err := plugins.Add(firstNamedPlugin)
	if err != nil {
		t.Fatalf("Unexpected error when adding a plugin with name: %s", testPluginExName)
	}
	err = plugins.Add(sameNamedPlugin)
	if err == nil {
		t.Fatalf("Expected an error because of dublicate named plugin!")
	}
	if plugins.Len() != 1 {
		t.Fatalf("Expected: %d activated plugin but we got: %d", 1, plugins.Len())
	}
}

type testPluginActivationType struct {
	shouldError bool
}

func (t testPluginActivationType) Activate(p iris.PluginContainer) error {
	p.Add(&testPluginEx{})
	if t.shouldError {
		return fmt.Errorf("An error happens, this plugin and the added plugins by this plugin should not be registered")
	}
	return nil
}

// a plugin may contain children plugins too, but,
// if an error happened then all of them are not activated/added to the plugin container
func AddPluginTo(t *testing.T, plugins iris.PluginContainer, plugin iris.Plugin, expectingCount int) {
	plugins.Add(plugin)
	if plugins.Len() != expectingCount { // 2 because it registeres a second plugin also
		t.Fatalf("Expected activated plugins to be: %d but we got: %d", expectingCount, plugins.Len())
	}
}

// if any error returned from the Activate plugin's method,
// then this plugin and the plugins it registers should not be registered at all
func TestPluginActivate(t *testing.T) {
	iris.ResetDefault()
	plugins := iris.Plugins

	myValidPluginWithChild := testPluginActivationType{shouldError: false}
	AddPluginTo(t, plugins, myValidPluginWithChild, 2) // 2 because its children registered also (its parent is not throwing an error)

	myInvalidPlugin := testPluginActivationType{shouldError: true}
	// should stay 2, even if we tried to add a new one,
	// because it cancels the registration of its children too (shouldError = true)
	AddPluginTo(t, plugins, myInvalidPlugin, 2)
}

func TestPluginEvents(t *testing.T) {
	iris.ResetDefault()
	var plugins = iris.Default.Plugins
	var prelistenran, postlistenran, precloseran bool

	plugins.Add(iris.PreListenFunc(func(*iris.Framework) {
		prelistenran = true
	}))

	plugins.Add(iris.PostListenFunc(func(*iris.Framework) {
		postlistenran = true
	}))

	plugins.Add(iris.PreCloseFunc(func(*iris.Framework) {
		precloseran = true
	}))

	myplugin := &testPluginEx{}
	plugins.Add(myplugin)
	if plugins.Len() != 4 {
		t.Fatalf("Expected: %d plugins to be registed but we got: %d", 4, plugins.Len())
	}
	desc := plugins.GetDescription(myplugin)
	if desc != testPluginExDescription {
		t.Fatalf("Expected: %s as Description of the plugin but got: %s", testPluginExDescription, desc)
	}

	plugins.DoPreListen(nil)
	plugins.DoPostListen(nil)
	plugins.DoPreClose(nil)

	if !prelistenran {
		t.Fatalf("Expected to run PreListen Func but it doesn't!")
	}
	if !postlistenran {
		t.Fatalf("Expected to run PostListen Func but it doesn't!")
	}
	if !precloseran {
		t.Fatalf("Expected to run PostListen Func but it doesn't!")
	}

	if !myplugin.named {
		t.Fatalf("Plugin should be named with: %s!", testPluginExName)
	}
	if !myplugin.activated {
		t.Fatalf("Plugin should be activated but it's not!")
	}
	if !myplugin.prelistenran {
		t.Fatalf("Expected to run PreListen Struct but it doesn't!")
	}
	if !myplugin.postlistenran {
		t.Fatalf("Expected to run PostListen Struct but it doesn't!")
	}
	if !myplugin.precloseran {
		t.Fatalf("Expected to run PostListen Struct but it doesn't!")
	}

}
