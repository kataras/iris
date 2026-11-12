// An optional JSON request body with ctx.BindJSONOptional[T].
//
// BindJSONOptional behaves like ctx.BindJSON except that an EMPTY body is a
// success: it returns the zero value of T with true and skips validation.
// That makes it the right reader for a filter object the client may omit.
// A body that is present but malformed still gets the usual 400.
//
// Learn: accept a request whose JSON body may be missing, and still answer
// 400 when a body is there but malformed.
// Before: ../read-json-validate
//
// Run:
//
//	go run main.go
//
// Try:
//
//	curl -i -X POST http://localhost:8080/reports/search                    # 200, no body: every report
//	curl -i -X POST http://localhost:8080/reports/search \
//	  -H 'Content-Type: application/json' -d '{"status":"open","min_amount":100}'   # 200, the two matching reports
//	curl -i -X POST http://localhost:8080/reports/search \
//	  -H 'Content-Type: application/json' -d '{"status":}'                 # 400 INVALID_ARGUMENT, "malformed JSON"
//
// Next: ../raw-body
package main

import (
	"strings"

	"github.com/kataras/iris/v14"
)

func main() {
	app := iris.New()
	app.Post("/reports/search", searchReports)
	app.Listen(":8080")
}

// Report is the resource being listed.
type Report struct {
	Title  string  `json:"title"`
	Status string  `json:"status"`
	Amount float64 `json:"amount"`
}

// ReportFilter is the optional body. Its zero value means "no filtering",
// which is exactly what ctx.BindJSONOptional returns for an empty body.
type ReportFilter struct {
	Status    string  `json:"status"`
	MinAmount float64 `json:"min_amount"`
}

func (f ReportFilter) matches(r Report) bool {
	if f.Status != "" && !strings.EqualFold(f.Status, r.Status) {
		return false
	}

	return r.Amount >= f.MinAmount
}

var reports = []Report{
	{Title: "Q1 hosting", Status: "open", Amount: 420.50},
	{Title: "Q1 travel", Status: "closed", Amount: 89.90},
	{Title: "Q2 hosting", Status: "open", Amount: 96.00},
	{Title: "Q2 licenses", Status: "open", Amount: 1250.00},
}

func searchReports(ctx iris.Context) {
	filter, ok := ctx.BindJSONOptional[ReportFilter]()
	if !ok {
		return // the body was present but could not be decoded: 400 written.
	}

	matched := make([]Report, 0, len(reports))
	for _, r := range reports {
		if filter.matches(r) {
			matched = append(matched, r)
		}
	}

	ctx.OK(iris.Map{"filter": filter, "items": matched}, nil)
}
