package entity

import "github.com/kataras/iris/v14/rest/types"

// The initial idea was to have a list of allowed origins for CORS requests per oauth2 client,
// but let's don't coplicate things even more, just have a single one entry of point for all client access,
// which will also work for the JWKS endpoint (thing that we can't control if we have oauth2 client-based allowed origins).
type Origin struct {
	// CreatedAt is the timestamp when the user was created.
	CreatedAt types.ISO8601 `pg:"type=timestamp,default=clock_timestamp()" json:"created_at,omitempty"`
	// The allowed origin URL pattern.
	// Examples:
	// - "https://example.com"
	// - "https://*.example.com"
	// - "https://*.example.com:8080"
	// - "*" (the default one, allows all origins).
	Pattern string `pg:"type=varchar(255),unique" json:"pattern"`
}
