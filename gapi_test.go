package gapi

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

///Run with go test -v -gcflags "-N -l" ./...
///TODO:
//IMPORTANT** I FOUND AN ERROR ON NAMED PARAMETERS AFTER LAST CHANGE, DOESNT WORK ON MORE THAN ONE ROUTE MUST FIX.***
var (
	//left side key is the registed route by the test server/router
	//right side FIRST TWO values array are the requests will be maden by the test client seperated by whitespace
	// first string of the array seperated by whitespace are the requests that should return status 200 ok
	// the second string of the array, are the requests seperated by whitespace are the requests that we are expecte to be notfound 404 error
	//right side THIRD value of the array are the parameter's values and keys we are expecting from handler SERVER side
	testingRoutes = map[string][3]string{
		"/home":                                     {"/home", "/home2", ""},
		"/home/test1":                               {"/home/test1", "/home/test2", ""},
		"/profile/:username/friends/:friendId(int)": {"/profile/kataras/friends/1", "/profile/kataras/friends/astring", "username='kataras',friendId=1 username='nothingbecausenotfound',friendId='nothingbecausenotfound'"},
		"/profile/:username/friends":                {"/profile/kataras/friends /profile/gerasimos/friends /profile/23/friends", "/profile/kataras/nofriends", "userame='kataras' username='gerasimos' username=23 userame='nothingbecausenotfound'"},
		"/profile/:username":                        {"/profile/kataras /profile/gerasimos /profile/23 /profile/test$@test", "/profile/kataras/xmmm", "username='kataras' username='gerasimos' username=23 username='test$@test' username='nothingbecausenotfound'"},
		"/api/users/:userId(int)":                   {"/api/users/1", "/api/users/shouldNotBeFound", "userId=1,userId='nothingbecausenotfound'"},
		"/api/pages/:title(a-zA-Z+)":                {"/api/pages/thisIsOK", "/api/pages/1231 /api/pages/shouldN0tB3Found", "title='thisIsOK' title='nothingbecausenotfound' title='nothingbecausenotfound'"},
	}
)
var (
	api    *Gapi
	server *httptest.Server
)

func TestMain(m *testing.M) {
	setup()
	result := m.Run()
	teardown()
	os.Exit(result)
}

func setup() {
	api = New()
}

func SliceIndex(limit int, predicate func(i int) bool) int {
	for i := 0; i < limit; i++ {
		if predicate(i) {
			return i
		}
	}
	return -1
}

//Define the routes (GET)
//Initialization of the Server
func TestRouterServerSide(t *testing.T) {
	for route, arr := range testingRoutes {
		api.Get(route, func(c *Context) {
			var reqUrl = c.Request.URL.Path
			var testRouteReq = strings.Split(arr[0], " ")                                                         //split the expected request paths
			var indexOfReq = SliceIndex(len(testRouteReq), func(i int) bool { return testRouteReq[i] == reqUrl }) //find the index of the splited expected req paths of the real req url we will need this to find the group of keys and values of the parameters we are excepting
			if arr[2] != "" {                                                                                     //we are expected named parameters
				var parametersExc = strings.Split(arr[2], " ") //split the excepting named parameters

				if indexOfReq > len(parametersExc)-1 {

				} else {

					var thisRouteExceptingParams = parametersExc[indexOfReq]
					for _, param := range strings.Split(thisRouteExceptingParams, ",") { //these are the expected parameters(key=val) seperated by commas forthis route
						var keysValuesEq = strings.Split(param, "=")
						var key = keysValuesEq[0]
						var value = keysValuesEq[1]

						//compare this param with the request's params
						if value != c.Param(key) {
							t.Fatal("Request Parameters are not equal than the expected parameters on the Path: " + reqUrl + "->" + value + "!=" + c.Param(key))
						}
					}
				}
			}
		})
		t.Log("Route registed: ", route)
	}

	server = httptest.NewServer(api)
	t.Log("Server started at: ", server.URL)

}

func TestRouterClientSide(t *testing.T) {

	for _, arr := range testingRoutes {
		var reqUrlsShouldWork = strings.Split(arr[0], " ")

		for _, url := range reqUrlsShouldWork {
			var res, err = http.Get(server.URL + url)
			if err != nil {
				t.Fatal("This url(" + url + ") has error : " + err.Error())
			} else if res.StatusCode != 200 {
				t.Fatal("This url(" + url + ") should found and OK, but status: " + res.Status)
			}

			res.Body.Close()
		}

		var reqUrlsShouldNotWork = strings.Split(arr[1], " ")
		for _, url := range reqUrlsShouldNotWork {
			var res, err = http.Get(server.URL + url)
			if err != nil {
				t.Fatal("That url(" + url + ") has error : " + err.Error())
			} else if res.StatusCode != 404 {
				t.Fatal("That url(" + url + ") should not be founded but status: " + res.Status)
			}
			res.Body.Close()

		}
	}
}

//Any cleanup
func teardown() {
	server.Close()
}
