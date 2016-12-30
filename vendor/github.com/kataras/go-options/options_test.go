package options

import (
	"reflect"
	"testing"
)

type testOptions struct {
	Name    string
	Phone   int
	City    string
	IsAdult bool
}

func TestOptions(t *testing.T) {

	name := "kataras"
	phone := 6942
	city := "Xanthi"
	isAdult := true

	// 1. test dynamic options and static options with the .New
	defaultCity := "Athens"
	defaultOptions := Options{"City": defaultCity}

	expectedOptions := Options{"Name": name, "Phone": phone, "City": city, "IsAdult": isAdult}
	finalOptions, toStruct := New(defaultOptions, Options{"Name": name}, Options{"Phone": phone, "City": city}, Option("IsAdult", isAdult)) // each one does the same thing
	if !reflect.DeepEqual(finalOptions, expectedOptions) {
		t.Fatalf("Default (dynamic) options are not equal with the result, got:\n%#v\nwhile expected:\n%#v", finalOptions, expectedOptions)
	}
	expectedStaticOptions := testOptions{Name: name, Phone: phone, City: city, IsAdult: isAdult}
	staticOptions := testOptions{}
	err := toStruct(&staticOptions)
	if err != nil {
		t.Fatalf("Panic when dynamic options to static options from .New: %s", err.Error())
	}

	if !reflect.DeepEqual(staticOptions, expectedStaticOptions) {
		t.Fatalf("Static options are not equal with the expected static options, got:\n%#v\nwhile expected:\n%#v", staticOptions, expectedStaticOptions)
	}

	// 2. Test Default with dynamic options

	finalOptions = Default(Options{"Name": name, "Phone": phone, "City": city, "IsAdult": isAdult})
	if !reflect.DeepEqual(finalOptions, expectedOptions) {
		t.Fatalf("Default (dynamic) options are not equal with the result from .Default, got:\n%#v\nwhile expected:\n%#v", finalOptions, expectedOptions)
	}

	// 3. Test static (struct) options with default values
	staticOptions = testOptions{}
	err = Struct(&staticOptions, Options{"Name": name, "Phone": phone, "City": city, "IsAdult": isAdult})
	if err != nil {
		t.Fatalf("Panic when dynamic options to static options from .Struct: %s", err.Error())
	}

	if !reflect.DeepEqual(staticOptions, expectedStaticOptions) {
		t.Fatalf("Static options are not equal with the expected static options from .Struct, got:\n%#v\nwhile expected:\n%#v", staticOptions, expectedStaticOptions)
	}

	//	defaultStaticOptions := testStruct{City: defaultCity}
}
