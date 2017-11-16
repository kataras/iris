package maintenance

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"

	"github.com/kataras/iris/core/maintenance/client"
	"github.com/kataras/iris/core/maintenance/encoding"

	"github.com/kataras/survey"
)

// question describes the question which will be used
// for the survey in order to authenticate the local iris.
type question struct {
	Message string `json:"message"`
}

func hasInternetConnection() (bool, bool) {
	r, err := client.PostForm("", nil)
	if err != nil {
		// no internet connection
		return false, false
	}
	defer r.Body.Close()
	return true, r.StatusCode == 204
}

func ask() bool {
	qs := fetchQuestions()
	var lastResponseUnsed string
	for _, q := range qs {
		survey.AskOne(&survey.Input{Message: q.Message}, &lastResponseUnsed, validate(q))
	}

	return lastResponseUnsed != ""
}

// fetchQuestions returns a list of questions
// fetched by the authority server.
func fetchQuestions() (qs []question) {
	r, err := client.PostForm("/survey/ask", nil)
	if err != nil {
		return
	}
	defer r.Body.Close()
	if err := encoding.UnmarshalBody(r.Body, &qs, json.Unmarshal); err != nil {
		return
	}

	return
}

func validate(q question) survey.Validator {
	return func(answer interface{}) error {
		if err := survey.Required(answer); err != nil {
			return err
		}

		ans, ok := answer.(string)
		if !ok {
			return fmt.Errorf("bug: expected string but got %v", answer)
		}
		data := url.Values{
			"q":               []string{q.Message},
			"ans":             []string{ans},
			"current_version": []string{Version},
		}

		r, err := client.PostForm("/survey/submit", data)
		if err != nil {
			// error from server-side, allow.
			return nil
		}
		defer r.Body.Close()

		if r.StatusCode == 200 {
			// read the whole thing, it has nothing.
			io.Copy(ioutil.Discard, r.Body)
			return nil // pass, no any errors.
		}
		// now, if invalid;
		got, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return nil
		}
		errMsg := string(got)
		return fmt.Errorf(errMsg)
	}
}
