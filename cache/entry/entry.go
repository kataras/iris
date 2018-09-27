package entry

import (
	"time"

	"github.com/kataras/iris/cache/cfg"
)

// Entry is the cache entry
// contains the expiration datetime and the response
type Entry struct {
	life time.Duration
	// ExpiresAt is the time which this cache will not be available
	expiresAt time.Time

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

// CopyHeaders clones headers "src" to "dst" .
func CopyHeaders(dst map[string][]string, src map[string][]string) {
	if dst == nil || src == nil {
		return
	}

	for k, vv := range src {
		v := make([]string, len(vv))
		copy(v, vv)
		dst[k] = v
	}
}

// Reset called each time the entry is expired
// and the handler calls this after the original handler executed
// to re-set the response with the new handler's content result
func (e *Entry) Reset(statusCode int, headers map[string][]string,
	body []byte, lifeChanger LifeChanger) {

	if e.response == nil {
		e.response = &Response{}
	}
	if statusCode > 0 {
		e.response.statusCode = statusCode
	}

	if len(headers) > 0 {
		newHeaders := make(map[string][]string, len(headers))
		CopyHeaders(newHeaders, headers)
		e.response.headers = newHeaders
	}

	e.response.body = body
	// check if a given life changer provided
	// and if it does then execute the change life time
	if lifeChanger != nil {
		e.ChangeLifetime(lifeChanger)
	}

	now := time.Now()
	e.expiresAt = now.Add(e.life)
	e.LastModified = now
}
