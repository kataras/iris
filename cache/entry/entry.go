package entry

import (
	"time"

	"github.com/kataras/iris/v12/core/memstore"
)

// Entry is the cache entry
// contains the expiration datetime and the response
type Entry struct {
	// ExpiresAt is the time which this cache will not be available
	lifeTime *memstore.LifeTime

	// when `Reset` this value is reseting to time.Now(),
	// it's used to send the "Last-Modified" header,
	// some clients may need it.
	LastModified time.Time

	// Response the response should be served to the client
	response *Response
	// but we need the key to invalidate manually...xmm
	// let's see for that later, maybe we make a slice instead
	// of store map
}

// reset called each time a new entry is acquired from the pool.
func (e *Entry) reset(lt *memstore.LifeTime, r *Response) {
	e.response = r
	e.LastModified = lt.Begun
}

// Response returns the cached response as it's.
func (e *Entry) Response() *Response {
	return e.response
}

// // Response gets the cache response contents
// // if it's valid returns them with a true value
// // otherwise returns nil, false
// func (e *Entry) Response() (*Response, bool) {
// 	if !e.isValid() {
// 		// it has been expired
// 		return nil, false
// 	}
// 	return e.response, true
// }

// // isValid reports whether this entry's response is still valid or expired.
// // If the entry exists in the store then it should be valid anyways.
// func (e *Entry) isValid() bool {
// 	return !e.lifeTime.HasExpired()
// }
