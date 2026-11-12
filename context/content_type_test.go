package context

import (
	"fmt"
	"net/http/httptest"
	"testing"
)

// TestParseContentType tests the ParseContentType function with various Content-Type strings.
func TestParseContentType(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    ContentType
		wantErr bool
	}{
		{
			name:  "Valid content type with charset and boundary",
			input: "text/html; charset=UTF-8; boundary=--something",
			want: ContentType{
				MediaType: "text/html",
				Charset:   "utf-8",
				Boundary:  "--something",
				Params:    nil,
			},
			wantErr: false,
		},
		{
			name:  "Valid content type with additional params",
			input: "application/json; charset=utf-8; version=2",
			want: ContentType{
				MediaType: "application/json",
				Charset:   "utf-8",
				Version:   2,
				Params:    nil,
			},
			wantErr: false,
		},
		{
			name:    "Invalid content type with no media type",
			input:   "; charset=utf-8; boundary=--something",
			want:    ContentType{},
			wantErr: true,
		},
		{
			name:  "Valid content type with uppercase media type and charset",
			input: "IMAGE/PNG; CHARSET=ISO-8859-1",
			want: ContentType{
				MediaType: "image/png",
				Charset:   "iso-8859-1",
				Params:    nil,
			},
			wantErr: false,
		},
		{
			name:    "Empty content type",
			input:   "",
			want:    ContentType{},
			wantErr: true,
		},
		{
			name:  "Valid content type with just media-type",
			input: "application/json",
			want: ContentType{
				MediaType: "application/json",
				Charset:   "",
				Params:    nil,
			},
			wantErr: false,
		},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprintf("[%d] %s", i, tt.name), func(t *testing.T) {
			gotContentType, err := ParseContentType(tt.input)
			if err != nil {
				if !tt.wantErr {
					t.Errorf("ParseContentType() error = %v, wantErr %v", err, tt.wantErr)
				}

				return
			}

			if expected, got := tt.want.String(), gotContentType.String(); expected != got {
				t.Errorf("ParseContentType() got = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestSetContentTypeOptions tests the setContentTypeOptions function to ensure it sets the correct headers.
func TestSetContentTypeOptions(t *testing.T) {
	ct := ContentType{
		MediaType: "text/html",
		Charset:   "UTF-8",
		Boundary:  "--something",
		Params:    map[string]string{},
		//
		NoSniff: true,
	}

	// Create a ResponseRecorder to record the response.
	w := httptest.NewRecorder()

	// Call the function to set the headers.
	setContentType(w, ct, "")

	// Check the Content-Type header.
	if got, want := w.Header().Get("Content-Type"), "text/html; charset=utf-8; boundary=--something"; got != want {
		t.Errorf("setContentTypeOptions() Content-Type = %v, want %v", got, want)
	}

	// Check the X-Content-Type-Options header.
	if got, want := w.Header().Get("X-Content-Type-Options"), "nosniff"; got != want {
		t.Errorf("setContentTypeOptions() X-Content-Type-Options = %v, want %v", got, want)
	}
}
