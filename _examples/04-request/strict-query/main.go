// Query parameters that refuse to be wrong, and partial updates that refuse
// to be silent.
//
// ctx.Query().GetInt("limit") answers zero for "limit=lots", and every
// hand-written date filter answers the zero date for "from=yesterday". The
// endpoint then returns 200 with the wrong rows and nobody finds out. The
// strict readers render the 400 themselves and report whether the handler may
// continue, exactly like ctx.BindJSON:
//
//	ctx.QueryDate(key, opts...)  a types.SimpleDate, "2006-01-02" or "2006-1-2"
//	ctx.QueryInt(key, opts...)   an int
//	ctx.QueryBool(key, opts...)  strconv.ParseBool plus yes/no/on/off
//
// The three options they share:
//
//	iris.QueryRequired()          an absent or blank value is a 400 naming the key
//	iris.QueryDefault("20")       the raw value to parse when the key is absent,
//	                              written in the same wire form the client uses
//	iris.QueryZeroValues("0")     the placeholders older clients send for "unset";
//	                              such a value reads exactly like an absent one
//
// QueryDefault wins over QueryRequired: a reader with something to fall back on
// is never missing a value. A default that does not parse is a mistake in the
// program, so it panics on the first request rather than answering 400.
//
// ctx.BindUpdate[T] is the same idea for a PATCH: it decodes the body as
// ctx.BindJSON does and reads the "columns" query key, the members the client
// wants written. A column that is not a member of T answers 400 naming it and
// the members it could have been, because without that check a typo updates
// nothing and the client is told it succeeded. No columns at all means a full
// update and a nil slice.
//
// Learn: reject a bad query value with a 400 instead of a silent zero, and
// refuse a PATCH column that does not exist instead of reporting success.
// Before: ../raw-body
//
// Run:
//
//	go run main.go
//
// Try:
//
//	curl -i 'localhost:8080/events'                                        # 400 INVALID_ARGUMENT, from is required
//	curl -i 'localhost:8080/events?from=yesterday'                         # 400 INVALID_ARGUMENT, not a date
//	curl -i 'localhost:8080/events?from=2026-01-01&limit=lots'             # 400 INVALID_ARGUMENT, not an integer
//	curl -i 'localhost:8080/events?from=2026-01-01'                        # 200, limit defaults to 20
//	curl -i 'localhost:8080/events?from=2026-01-01&to=null&archived=yes'   # 200, "null" reads as absent
//	curl -i 'localhost:8080/events?from=0'                                 # 400, "0" reads as absent and from is required
//
//	curl -i -X PATCH 'localhost:8080/events/1?columns=note' -d '{"note":"checked"}'   # 204 No Content
//	curl -i -X PATCH 'localhost:8080/events/1?columns=note' -d '{"note":"checked"}'   # 304 Not Modified, same value
//	curl -i -X PATCH 'localhost:8080/events/1?columns=nte'  -d '{"note":"typo"}'      # 400 INVALID_ARGUMENT
//
// Both 400s carry the same rest/errors envelope, the one every other rejection
// in the framework renders ("id" is the tracking id written to the log too):
//
//	{"id":"56d67e5c-7bf1-428d-af86-e3b16d72abf1",
//	 "http_error_code":{"canonical_name":"INVALID_ARGUMENT","status":400},
//	 "message":"missing from query parameter: expected a date of the 2006-01-02 form"}
//
//	{"id":"213dddc3-5877-4ff5-90f4-e171bd4953c8",
//	 "http_error_code":{"canonical_name":"INVALID_ARGUMENT","status":400},
//	 "message":"unknown column: nte: allowed columns: id, note, severity"}
//
// Next: ../pagination
package main

import (
	"fmt"
	"sync"

	"github.com/kataras/iris/v14"
	"github.com/kataras/iris/v14/rest/errors"
)

func main() {
	app := iris.New()

	app.Get("/events", listEvents)
	app.Patch("/events/{id:int}", patchEvent)

	app.Listen(":8080")
}

// Event is the stored row. Its JSON member names are the column names the
// PATCH route accepts: "id", "note" and "severity", and nothing else.
type Event struct {
	ID       int    `json:"id"`
	Note     string `json:"note"`
	Severity string `json:"severity"`
}

var (
	mu     sync.Mutex
	events = map[int]Event{1: {ID: 1, Note: "", Severity: "warning"}}
)

// listEvents reads the window a client asked for, or answers 400 saying which
// parameter it could not read. Every reader's second result means "the
// response was already written", so the handler returns and writes nothing.
func listEvents(ctx iris.Context) {
	// Required: a search over every event ever recorded is not a default.
	// "0" and "null" are the placeholders old clients send for "no date", so
	// they read as absent and hit the required check rather than the parser.
	from, ok := ctx.QueryDate("from", iris.QueryRequired(), iris.QueryZeroValues("0", "null"))
	if !ok {
		return
	}

	// Optional, with the same placeholders. An omitted "to" is the zero date,
	// which this endpoint reads as "up to now".
	to, ok := ctx.QueryDate("to", iris.QueryZeroValues("0", "null"))
	if !ok {
		return
	}

	// The default is written the way a client would write it, and it is parsed
	// through the same path, so there is one spelling of "20" in the program.
	limit, ok := ctx.QueryInt("limit", iris.QueryDefault("20"))
	if !ok {
		return
	}

	// No option at all: absent means false, but "archived=maybe" is still a
	// 400 rather than a silent false.
	archived, ok := ctx.QueryBool("archived")
	if !ok {
		return
	}

	mu.Lock()
	total := len(events)
	mu.Unlock()

	ctx.JSON(iris.Map{
		"from":     from,
		"to":       to,
		"open_end": to.IsZero(),
		"limit":    limit,
		"archived": archived,
		"total":    total,
	})
}

// patchEvent applies only the members the client named. The 204 and the 304
// come from ctx.NoContent, which renders the "did anything change" branch every
// update endpoint needs.
func patchEvent(ctx iris.Context) {
	id := ctx.Params().GetInt("id")

	input, columns, ok := ctx.BindUpdate[Event]()
	if !ok {
		return // 400 already written: a bad body, or a column Event has not.
	}

	updated, err := updateEvent(id, input, columns...)
	ctx.NoContent(updated, err)
}

func updateEvent(id int, input Event, columns ...string) (bool, error) {
	mu.Lock()
	defer mu.Unlock()

	stored, found := events[id]
	if !found {
		return false, errors.NotFound.Wrap(fmt.Errorf("event %d not found", id))
	}

	// The client's names, mapped to this store's own fields. An empty list is
	// a full update; "id" is the server's and changes nothing.
	updated := stored
	for _, column := range columnsOrAll(columns) {
		switch column {
		case "note":
			updated.Note = input.Note
		case "severity":
			updated.Severity = input.Severity
		}
	}

	if updated == stored {
		return false, nil // 304 Not Modified.
	}

	events[id] = updated
	return true, nil // 204 No Content.
}

func columnsOrAll(columns []string) []string {
	if len(columns) == 0 {
		return []string{"note", "severity"}
	}

	return columns
}
