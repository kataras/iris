///This is my first 'benchmark' excuse me if it is fail I will fix it when I learn more for these things*
package gapi

import (
	"net/http"
	"strconv"
	"testing"
)


//For now: run the example first and after go test -run=XXX -bench=.
func BenchmarkRouter(b *testing.B) {

	for i := 0; i <= b.N; i++ {
		println("Test ", i, " Started")
		resp, err := http.Get("http://localhost/api/user/" + strconv.Itoa(i))
		defer resp.Body.Close()
		if err != nil {
			b.Fatal("error on req ", err)

		}
		if resp.StatusCode != 200 {
			b.Fatalf("Received non-200 response: %d\n", resp.StatusCode)
		}

		//

		resp, err = http.Get("http://localhost/profile/user/kataras/details/dsadsa" + strconv.Itoa(i))

		if err != nil {
			//b.Fatal("error on req ", err)
			println("error on req ", err)
			//	resp.Body.Close()

		}
		if resp.StatusCode != 200 {
			b.Fatalf("Received non-200 response: %d\n", resp.StatusCode)
		}

		//

		resp, err = http.Get("http://localhost/register")

		if err != nil {
			b.Fatal("error on req ", err)

		}
		if resp.StatusCode != 200 {
			b.Fatalf("Received non-200 response: %d\n", resp.StatusCode)
		}

		//

	}

}
