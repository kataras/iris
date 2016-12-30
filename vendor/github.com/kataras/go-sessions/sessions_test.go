package sessions

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.com/gavv/httpexpect"
	"github.com/kataras/go-errors"
	"github.com/kataras/go-serializer"
)

var errReadBody = errors.New("While trying to read %s from the request body. Trace %s")

// ReadJSON reads JSON from request's body
func ReadJSON(jsonObject interface{}, req *http.Request) error {
	b, err := ioutil.ReadAll(req.Body)
	if err != nil && err != io.EOF {
		return err
	}
	decoder := json.NewDecoder(strings.NewReader(string(b)))
	err = decoder.Decode(jsonObject)

	if err != nil && err != io.EOF {
		return errReadBody.Format("JSON", err.Error())
	}
	return nil
}

func getTester(mux *http.ServeMux, t *testing.T) *httpexpect.Expect {

	testConfiguration := httpexpect.Config{
		BaseURL: "http://localhost:8080",
		Client: &http.Client{
			Transport: httpexpect.NewBinder(mux),
			Jar:       httpexpect.NewJar(),
		},
		Reporter: httpexpect.NewAssertReporter(t),
	}

	return httpexpect.WithConfig(testConfiguration)
}

func writeValues(res http.ResponseWriter, values map[string]interface{}) error {

	result, err := serializer.Serialize("application/json", values)
	if err != nil {
		return err
	}

	res.Header().Set("Content-Type", "application/json")
	res.Write(result)
	return nil
}

func TestSessionsNetHTTP(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	values := map[string]interface{}{
		"Name":   "go-sessions",
		"Days":   "1",
		"Secret": "dsads£2132215£%%Ssdsa",
	}

	setHandler := http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		vals := make(map[string]interface{}, 0)
		if err := ReadJSON(&vals, req); err != nil {
			t.Fatalf("Cannot readjson. Trace %s", err.Error())
		}
		sess := Start(res, req)
		for k, v := range vals {
			sess.Set(k, v)
		}

		res.WriteHeader(http.StatusOK)
	})
	mux.Handle("/set/", setHandler)

	writeSessValues := func(res http.ResponseWriter, req *http.Request) {
		sess := Start(res, req)
		sessValues := sess.GetAll()
		if err := writeValues(res, sessValues); err != nil {
			t.Fatalf("While serialize the session values: %s", err.Error())
		}
	}

	getHandler := http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		writeSessValues(res, req)
	})
	mux.Handle("/get/", getHandler)

	clearHandler := http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		sess := Start(res, req)
		sess.Clear()
		writeSessValues(res, req)
	})
	mux.Handle("/clear/", clearHandler)

	destroyHandler := http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		Destroy(res, req)
		writeSessValues(res, req)
		res.WriteHeader(http.StatusOK)
		// the cookie and all values should be empty
	})
	mux.Handle("/destroy/", destroyHandler)

	afterDestroyHandler := http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		res.WriteHeader(http.StatusOK)
	})
	// request cookie should be empty
	mux.Handle("/after_destroy/", afterDestroyHandler)

	e := getTester(mux, t)

	e.POST("/set/").WithJSON(values).Expect().Status(http.StatusOK).Cookies().NotEmpty()
	e.GET("/get/").Expect().Status(http.StatusOK).JSON().Object().Equal(values)

	// test destory which also clears first
	d := e.GET("/destroy/").Expect().Status(http.StatusOK)
	d.JSON().Object().Empty()
	e.GET("/after_destroy/").Expect().Status(http.StatusOK).Cookies().Empty()
	// set and clear again
	e.POST("/set/").WithJSON(values).Expect().Status(http.StatusOK).Cookies().NotEmpty()
	e.GET("/clear/").Expect().Status(http.StatusOK).JSON().Object().Empty()
}

func TestFlashMessages(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()

	valueSingleKey := "Name"
	valueSingleValue := "go-sessions"

	values := map[string]interface{}{
		valueSingleKey: valueSingleValue,
		"Days":         "1",
		"Secret":       "dsads£2132215£%%Ssdsa",
	}

	setHandler := http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		if req.RequestURI == "/set/" {
			vals := make(map[string]interface{}, 0)
			if err := ReadJSON(&vals, req); err != nil {
				t.Fatalf("Cannot readjson. Trace %s", err.Error())
			}
			sess := Start(res, req)
			for k, v := range vals {
				sess.SetFlash(k, v)
			}

			res.WriteHeader(http.StatusOK)
		}
	})
	mux.Handle("/set/", setHandler)

	writeFlashValues := func(res http.ResponseWriter, req *http.Request) {
		sess := Start(res, req)
		flashes := sess.GetFlashes()
		if err := writeValues(res, flashes); err != nil {
			t.Fatalf("While serialize the flash values: %s", err.Error())
		}
	}

	getSingleHandler := http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		if req.RequestURI == "/get_single/" {
			sess := Start(res, req)
			flashMsgString := sess.GetFlashString(valueSingleKey)
			res.Write([]byte(flashMsgString))
		}
	})

	mux.Handle("/get_single/", getSingleHandler)

	getHandler := http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		if req.RequestURI == "/get/" {
			writeFlashValues(res, req)
		}
	})

	mux.Handle("/get/", getHandler)

	clearHandler := http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		if req.RequestURI == "/clear/" {
			sess := Start(res, req)
			sess.ClearFlashes()
			writeFlashValues(res, req)
		}
	})

	mux.Handle("/clear/", clearHandler)

	destroyHandler := http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		if req.RequestURI == "/destroy/" {
			Destroy(res, req)
			writeFlashValues(res, req)
			res.WriteHeader(http.StatusOK)
		}
		// the cookie and all values should be empty
	})

	mux.Handle("/destroy/", destroyHandler)

	afterDestroyHandler := http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		if req.RequestURI == "/after_destroy/" {
			res.WriteHeader(http.StatusOK)
		}
	})

	// request cookie should be empty
	mux.Handle("/after_destroy/", afterDestroyHandler)

	e := getTester(mux, t)

	e.POST("/set/").WithJSON(values).Expect().Status(http.StatusOK).Cookies().NotEmpty()
	// get all
	e.GET("/get/").Expect().Status(http.StatusOK).JSON().Object().Equal(values)
	// get the same flash on other request should return nothing because the flash message is removed after fetch once
	e.GET("/get/").Expect().Status(http.StatusOK).JSON().Object().Empty()
	// test destory which also clears first
	d := e.GET("/destroy/").Expect().Status(http.StatusOK)
	d.JSON().Object().Empty()
	e.GET("/after_destroy/").Expect().Status(http.StatusOK).Cookies().Empty()
	// set and clear again
	e.POST("/set/").WithJSON(values).Expect().Status(http.StatusOK).Cookies().NotEmpty()
	e.GET("/clear/").Expect().Status(http.StatusOK).JSON().Object().Empty()

	// set again in order to take the single one ( we don't test Cookies.NotEmpty because httpexpect default conf reads that from the request-only)
	e.POST("/set/").WithJSON(values).Expect().Status(http.StatusOK)
	//	e.GET("/get/").Expect().Status(http.StatusOK).JSON().Object().Equal(values)
	e.GET("/get_single/").Expect().Status(http.StatusOK).Body().Equal(valueSingleValue)
}
