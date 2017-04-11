package entry

import (
	"time"

	"github.com/geekypanda/httpcache/cfg"
) 

// Entry is the cache entry
// contains the expiration datetime and the response
type Entry struct {
	life time.Duration
	// ExpiresAt is the time which this cache will not be available
	expiresAt time.Time

	// Response the response should be served to the client
	response *Response
	// but we need the key to invalidate manually...xmm
	// let's see for that later, maybe we make a slice instead
	// of store map
}

// NewEntry returns a new cache entry
// it doesn't sets the expiresAt & the response
// because these are setting each time on Reset
func NewEntry(duration time.Duration) *Entry {
	// if given duration is not <=0 (which means finds from the headers)
	// then we should check for the MinimumCacheDuration here
	if duration >= 0 && duration < cfg.MinimumCacheDuration {
		duration = cfg.MinimumCacheDuration
	}

	return &Entry{
		life:     duration,
		response: &Response{},
	}
}

// Response gets the cache response contents
// if it's valid returns them with a true value
// otherwise returns nil, false
func (e *Entry) Response() (*Response, bool) {
	if !e.valid() {
		// it has been expired
		return nil, false
	}
	return e.response, true
}

// valid returns true if this entry's response is still valid
// or false if the expiration time passed
func (e *Entry) valid() bool {
	return !time.Now().After(e.expiresAt)
}

// LifeChanger is the function which returns
// a duration which will be compared with the current
// entry's (cache life)  duration
// and execute the LifeChanger func
// to set the new life time
type LifeChanger func() time.Duration

// ChangeLifetime modifies the life field
// which is the life duration of the cached response
// of this cache entry
//
// useful when we find a max-age header from the handler
func (e *Entry) ChangeLifetime(fdur LifeChanger) {
	if e.life < cfg.MinimumCacheDuration {
		newLifetime := fdur()
		if newLifetime > e.life {
			e.life = newLifetime
		} else {
			// if even the new lifetime is less than MinimumCacheDuration
			// then change set it explicitly here
			e.life = cfg.MinimumCacheDuration
		}
	}
}

// Reset called each time the entry is expired
// and the handler calls this after the original handler executed
// to re-set the response with the new handler's content result
func (e *Entry) Reset(statusCode int, contentType string,
	body []byte, lifeChanger LifeChanger) {

	if e.response == nil {
		e.response = &Response{}
	}
	if statusCode > 0 {
		e.response.statusCode = statusCode
	}

	if contentType != "" {
		e.response.contentType = contentType
	}

	e.response.body = body
	// check if a given life changer provided
	// and if it does then execute the change life time
	if lifeChanger != nil {
		e.ChangeLifetime(lifeChanger)
	}
	e.expiresAt = time.Now().Add(e.life)
}
