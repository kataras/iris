package iris

/*
Contains tests for plugin, no end-to-end, just local-object tests, these are enoguh for now.

CONTRIBUTE & DISCUSSION ABOUT TESTS TO: https://github.com/iris-contrib/tests
*/

import (
	"fmt"
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

func (t *testPluginEx) Activate(p PluginContainer) error {
	fmt.Println("Activate Struct")
	t.activated = true
	return nil
}

func (t *testPluginEx) PreListen(*Framework) {
	fmt.Println("PreListen Struct")
	t.prelistenran = true
}

func (t *testPluginEx) PostListen(*Framework) {
	fmt.Println("PostListen Struct")
	t.postlistenran = true
}

func (t *testPluginEx) PreClose(*Framework) {
	fmt.Println("PreClose Struct")
	t.precloseran = true
}

func ExamplePlugins_Add() {
	initDefault()
	Plugins.Add(PreListenFunc(func(*Framework) {
		fmt.Println("PreListen Func")
	}))

	Plugins.Add(PostListenFunc(func(*Framework) {
		fmt.Println("PostListen Func")
	}))

	Plugins.Add(PreCloseFunc(func(*Framework) {
		fmt.Println("PreClose Func")
	}))

	myplugin := &testPluginEx{}
	Plugins.Add(myplugin)
	desc := Plugins.GetDescription(myplugin)
	fmt.Println(desc)

	ListenVirtual()
	Close()

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
	var plugins pluginContainer
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
	if len(plugins.activatedPlugins) != 1 {
		t.Fatalf("Expected: %d activated plugin but we got: %d", 1, len(plugins.activatedPlugins))
	}
}

type testPluginActivationType struct {
	shouldError bool
}

func (t testPluginActivationType) Activate(p PluginContainer) error {
	p.Add(&testPluginEx{})
	if t.shouldError {
		return fmt.Errorf("An error happens, this plugin and the added plugins by this plugin should not be registered")
	}
	return nil
}

func TestPluginActivate(t *testing.T) {
	var plugins pluginContainer
	myplugin := testPluginActivationType{shouldError: false}
	plugins.Add(myplugin)

	if len(plugins.activatedPlugins) != 2 { // 2 because it registeres a second plugin also
		t.Fatalf("Expected activated plugins to be: %d but we got: %d", 0, len(plugins.activatedPlugins))
	}
}

// if any error returned from the Activate plugin's method, then this plugin and the plugins it registers should not be registered at all
func TestPluginActivationError(t *testing.T) {
	var plugins pluginContainer
	myplugin := testPluginActivationType{shouldError: true}
	plugins.Add(myplugin)

	if len(plugins.activatedPlugins) > 0 {
		t.Fatalf("Expected activated plugins to be: %d but we got: %d", 0, len(plugins.activatedPlugins))
	}
}

func TestPluginEvents(t *testing.T) {
	var plugins pluginContainer
	var prelistenran, postlistenran, precloseran bool

	plugins.Add(PreListenFunc(func(*Framework) {
		prelistenran = true
	}))

	plugins.Add(PostListenFunc(func(*Framework) {
		postlistenran = true
	}))

	plugins.Add(PreCloseFunc(func(*Framework) {
		precloseran = true
	}))

	myplugin := &testPluginEx{}
	plugins.Add(myplugin)
	if len(plugins.activatedPlugins) != 4 {
		t.Fatalf("Expected: %d plugins to be registed but we got: %d", 4, len(plugins.activatedPlugins))
	}
	desc := plugins.GetDescription(myplugin)
	if desc != testPluginExDescription {
		t.Fatalf("Expected: %s as Description of the plugin but got: %s", testPluginExDescription, desc)
	}

	plugins.DoPreListen(nil)
	plugins.DoPostListen(nil)
	plugins.DoPreClose(nil)

	if !prelistenran {
		t.Fatalf("Expected to run PreListen Func but it doesnt!")
	}
	if !postlistenran {
		t.Fatalf("Expected to run PostListen Func but it doesnt!")
	}
	if !precloseran {
		t.Fatalf("Expected to run PostListen Func but it doesnt!")
	}

	if !myplugin.named {
		t.Fatalf("Plugin should be named with: %s!", testPluginExName)
	}
	if !myplugin.activated {
		t.Fatalf("Plugin should be activated but it's not!")
	}
	if !myplugin.prelistenran {
		t.Fatalf("Expected to run PreListen Struct but it doesnt!")
	}
	if !myplugin.postlistenran {
		t.Fatalf("Expected to run PostListen Struct but it doesnt!")
	}
	if !myplugin.precloseran {
		t.Fatalf("Expected to run PostListen Struct but it doesnt!")
	}

}
