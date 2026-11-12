package sessions

import (
	"bytes"
	"encoding"
	"encoding/gob"
	"errors"
	"net/http"
	"net/url"
	"time"
	"uuid"

	"github.com/gorilla/securecookie"
)

const defaultMaxAge time.Duration = 24 * time.Hour

// Options represents the configuration settings for a session.
// These settings determine how the session cookies are managed and stored.
//
// The defaults described on the fields (a MaxAge of 24 hours, SameSite Lax,
// Secure true, Path "/") are set by NewOptions. A zero Options{} has none of
// them: MaxAge 0, no SameSite attribute, Secure false, no DataCodecs.
// Start from NewOptions and change the fields you need.
type Options struct {
	// Path specifies the cookie path. It is optional.
	// NewOptions sets it to "/". An empty path writes no Path attribute, so
	// the browser uses its own default: the directory of the request's path.
	Path string `json:"path" yaml:"Path"`

	// Domain specifies the cookie domain. It is optional.
	// If not set, the default domain is the host of the current document URL.
	Domain string `json:"domain" yaml:"Domain"`

	// MaxAge defines the maximum age of the cookie, as a time.Duration
	// (the cookie's Max-Age attribute is written in whole seconds).
	// MaxAge=0 means no 'Max-Age' attribute is specified and the
	// session is removed after the request life-cycle ends.
	// MaxAge<0 means delete the cookie immediately, equivalently 'Max-Age: 0'.
	//
	// NewOptions sets it to 24 hours.
	MaxAge time.Duration `json:"max_age" yaml:"MaxAge"`

	// SameSite controls the SameSite attribute of the cookie.
	// It can be set to Lax, Strict, or None to control the cookie's
	// cross-site request behavior.
	//
	// NewOptions sets SameSite to Lax (changed from None in v14) to
	// mitigate CSRF while still allowing top-level navigations.
	SameSite http.SameSite `json:"same_site" yaml:"SameSite"`

	// Partitioned indicates whether the cookie is partitioned.
	// Partitioned cookies are only sent in a third-party context if the
	// request is initiated by the same site that set the cookie.
	//
	// The default value is false.
	Partitioned bool `json:"partitioned" yaml:"Partitioned"`

	// Secure indicates whether the cookie should only be transmitted
	// over secure protocols such as HTTPS.
	//
	// NewOptions sets it to true.
	Secure bool `json:"secure" yaml:"Secure"`

	// DataCodecs are the securecookie codecs used to encode and decode
	// the cookie value and the stored session data. NewOptions builds them
	// from its key pairs. DataCodecs are shared between all sessions and
	// should not be modified per-session. Instead, use each database's
	// codecs.
	DataCodecs []securecookie.Codec `json:"-" yaml:"-"`
}

// NewOptions creates and returns a new Options value with default values.
// The keyPairs parameter is used to initialize the securecookie.Codec for encoding and decoding session data.
func NewOptions(keyPairs ...[]byte) Options {
	return Options{
		Path:       "/",                                     // Default path is the root.
		Domain:     "",                                      // Default domain is the host of the current document URL.
		MaxAge:     defaultMaxAge,                           // Default maximum age for the cookie.
		SameSite:   http.SameSiteLaxMode,                    // Secure-by-default: mitigate CSRF (was unset/None before v14).
		Secure:     true,                                    // Secure-by-default: send only over HTTPS. Set to false for plain-HTTP (e.g. local development).
		DataCodecs: keyRotation(defaultMaxAge, keyPairs...), // Initialize codecs with key rotation.
	}
}

// Apply applies any changes to the Options before using it in databases.
// It updates the MaxAge for each securecookie.Codec and returns the Options back.
//
// Note that Options is a value type: copying it copies every field except the
// DataCodecs slice, whose codecs are shared by design (they hold the keys).
func (opts Options) Apply() Options {
	for _, codec := range opts.DataCodecs {
		if cookie, ok := codec.(*securecookie.SecureCookie); ok {
			cookie.MaxAge(int(opts.MaxAge.Seconds())) // Set the max-age for the cookie.
		}
	}
	return opts
}

// keyRotation initializes securecookie.Codecs with the provided key pairs and sets their MaxAge and serializer.
func keyRotation(maxAge time.Duration, keyPairs ...[]byte) []securecookie.Codec {
	if len(keyPairs) == 0 {
		return nil
	}

	codecs := securecookie.CodecsFromPairs(keyPairs...)
	for _, codec := range codecs {
		if cookie, ok := codec.(*securecookie.SecureCookie); ok {
			cookie.MaxAge(int(maxAge.Seconds()))            // Set the max-age for the cookie.
			cookie.SetSerializer(securecookie.NopEncoder{}) // Use a no-op serializer.
		}
	}
	return codecs
}

// Encode encodes the given idOrData into a string using the specified cookieName.
func (opts Options) Encode(cookieName string, idOrData any) (string, error) {
	var encodedBytes []byte

	switch v := idOrData.(type) {
	case string:
		// It's an ID, we don't encode it using gob.
		encodedBytes = []byte(v)
	case encoding.BinaryMarshaler:
		// Data type completes the BinaryMarshaler, let's take advantage of it.
		data, err := v.MarshalBinary()
		if err != nil {
			return "", err
		}
		encodedBytes = data
	default:
		// It's not a string or encoding.BinaryMarshaler.
		// Use gob to encode the data.
		buf := new(bytes.Buffer)
		enc := gob.NewEncoder(buf)
		if err := enc.Encode(idOrData); err != nil {
			return "", err
		}
		encodedBytes = buf.Bytes()
	}

	if len(opts.DataCodecs) > 0 {
		return securecookie.EncodeMulti(cookieName, encodedBytes, opts.DataCodecs...)
	}

	return url.QueryEscape(string(encodedBytes)), nil
}

// Decode decodes the given cookieValue into idOrDataPtr using the specified cookieName.
func (opts Options) Decode(cookieName, cookieValue string, idOrDataPtr any) error {
	if len(opts.DataCodecs) == 0 {
		val, err := url.QueryUnescape(cookieValue)
		if err != nil {
			return err
		}

		switch v := idOrDataPtr.(type) {
		case *string:
			// It's an ID.
			//
			// Note(@kataras): validate its SHAPE before it becomes a storage key.
			//
			// With no DataCodecs configured the id is unauthenticated, and it is used
			// verbatim as the Redis/Postgres key ("cookieName:id"). Without this
			// check a client could drive reads and writes against any key under
			// that prefix, with an id of any length and any contents. Ids are
			// generated with uid.NewV7String (UUIDs), so requiring that shape
			// costs nothing and closes the hole. The check is version-agnostic,
			// so v4 ids issued before the v7 switch keep validating.
			if !isValidSessionID(val) {
				return ErrInvalidSessionID
			}

			*v = val
			return nil
		case encoding.BinaryUnmarshaler:
			// Data type completes the BinaryMarshaler, let's take advantage of it.
			return v.UnmarshalBinary([]byte(val))
		default:
			// It's not a string or encoding.BinaryUnmarshaler.
			// Use gob to decode the data.
			dec := gob.NewDecoder(bytes.NewReader([]byte(val)))
			return dec.Decode(idOrDataPtr)
		}
	}

	var idOrDataDecoded []byte
	if err := securecookie.DecodeMulti(cookieName, cookieValue, &idOrDataDecoded, opts.DataCodecs...); err != nil {
		return err
	}

	switch v := idOrDataPtr.(type) {
	case *string:
		// It's an ID.
		*v = string(idOrDataDecoded)
		return nil
	case encoding.BinaryUnmarshaler:
		return v.UnmarshalBinary(idOrDataDecoded)
	default:
		// dec := gob.NewDecoder(bytes.NewBuffer(idOrDataDecoded))
		dec := gob.NewDecoder(bytes.NewReader(idOrDataDecoded))
		return dec.Decode(idOrDataPtr)
	}
}

// Cookie converts the Options instance to an http.Cookie with a value.
// It sets the cookie's attributes based on the Options fields.
func (opts Options) Cookie(cookieName, rawValue string) *http.Cookie {
	var expires time.Time

	if opts.MaxAge > 0 {
		expires = time.Now().Add(opts.MaxAge)
	} else if opts.MaxAge < 0 {
		expires = time.Unix(1, 0)
	}

	return &http.Cookie{
		Name:        cookieName,
		Value:       rawValue,
		Path:        opts.Path,
		Domain:      opts.Domain,
		MaxAge:      int(opts.MaxAge.Seconds()),
		Expires:     expires,
		Secure:      opts.Secure,
		HttpOnly:    true,
		SameSite:    opts.SameSite,
		Partitioned: opts.Partitioned,
	}
}

// ErrInvalidSessionID is returned when a client-supplied session id does not
// have the shape this package generates (a UUID).
var ErrInvalidSessionID = errors.New("sessions: invalid session id")

// isValidSessionID reports whether the given id has the shape of a generated
// session id. See the note in Decode.
func isValidSessionID(id string) bool {
	_, err := uuid.Parse(id)
	return err == nil
}
