package jsonx

import (
	"encoding/json"
	"testing"
	"time"
)

func TestParseISO8601(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    ISO8601
		wantErr bool
	}{
		{
			name:  "Timestamp with microseconds",
			input: "2024-01-02T15:04:05.999999Z",
			want:  ISO8601(time.Date(2024, 01, 02, 15, 04, 05, 999999*1000, time.UTC)),

			wantErr: false,
		},
		{
			name:    "Timestamp with timezone but no microseconds",
			input:   "2024-01-02T15:04:05+07:00",
			want:    ISO8601(time.Date(2024, 01, 02, 15, 04, 05, 0, time.FixedZone("", 7*3600))),
			wantErr: false,
		},
		{
			name:  "Timestamp with timezone of UTC with microseconds",
			input: "2024-04-08T08:05:04.830140+00:00",
			// time.Date function interprets the nanosecond parameter. The time.Date function expects the nanosecond parameter to be the entire nanosecond part of the time, not just the microsecond part.
			// When we pass 830140 as the nanosecond argument, Go interprets this as 830140 nanoseconds,
			// which is equivalent to 000830140 microseconds (padded with leading zeros to fill the nanosecond precision).
			// This is why we see 2024-04-08 08:05:04.00083014 +0000 UTC as the output.
			// To correctly represent 830140 microseconds, we need to convert it to nanoseconds by multiplying by 1000 (or set the value to 830140000).
			want:    ISO8601(time.Date(2024, 04, 8, 8, 05, 04, 830140*1000, time.UTC)),
			wantErr: false,
		},
		{
			name:    "Timestamp with timezone but no microseconds (2)",
			input:   "2024-04-08T04:47:10+03:00",
			want:    ISO8601(time.Date(2024, 04, 8, 4, 47, 10, 0, time.FixedZone("", 3*3600))),
			wantErr: false,
		},
		{
			name:    "Timestamp with Zulu time",
			input:   "2024-01-02T15:04:05Z",
			want:    ISO8601(time.Date(2024, 01, 02, 15, 04, 05, 0, time.UTC)),
			wantErr: false,
		},
		{
			name:    "Timestamp with Zulu time with microseconds",
			input:   "2024-04-08T08:05:04.830140",
			want:    ISO8601(time.Date(2024, 04, 8, 8, 05, 04, 830140*1000, time.UTC)),
			wantErr: false,
		},
		{
			name:    "Basic ISO8601 layout",
			input:   "2024-01-02T15:04:05",
			want:    ISO8601(time.Date(2024, 01, 02, 15, 04, 05, 0, time.UTC)),
			wantErr: false,
		},
		{
			name:    "Invalid format",
			input:   "2024-01-02",
			want:    ISO8601{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseISO8601(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseISO8601() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !time.Time(got).Equal(time.Time(tt.want)) {
				t.Errorf("ParseISO8601() = %v (%s), want %v (%s)", got, got.ToTime().String(), tt.want, tt.want.ToTime().String())
			}
		})
	}
}

func TestISO8601(t *testing.T) {
	data := `{"start": "2021-08-20T10:05:01", "end": "2021-12-01T17:05:06", "nothing": null, "empty": ""}`
	v := struct {
		Start   ISO8601 `json:"start"`
		End     ISO8601 `json:"end"`
		Nothing ISO8601 `json:"nothing"`
		Empty   ISO8601 `json:"empty"`
	}{}
	err := json.Unmarshal([]byte(data), &v)
	if err != nil {
		t.Fatal(err)
	}

	if !v.Nothing.IsZero() {
		t.Fatalf("expected 'nothing' to be zero but got: %v", v.Nothing)
	}

	if !v.Empty.IsZero() {
		t.Fatalf("expected 'empty' to be zero but got: %v", v.Empty)
	}

	loc := time.UTC

	if expected, got := time.Date(2021, time.August, 20, 10, 5, 1, 0, loc), v.Start.ToTime(); expected != got {
		t.Fatalf("expected 'start' to be: %v but got: %v", expected, got)
	}

	if expected, got := time.Date(2021, time.December, 1, 17, 5, 6, 0, loc), v.End.ToTime(); expected != got {
		t.Fatalf("expected 'start' to be: %v but got: %v", expected, got)
	}
}

func TestISO8601WithZoneUTCOffset(t *testing.T) {
	data := `{"start": "2022-08-10T03:21:00.000000+03:00", "end": "2022-08-10T09:49:00.000000+03:00", "nothing": null, "empty": ""}`
	v := struct {
		Start   ISO8601 `json:"start"`
		End     ISO8601 `json:"end"`
		Nothing ISO8601 `json:"nothing"`
		Empty   ISO8601 `json:"empty"`
	}{}
	err := json.Unmarshal([]byte(data), &v)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// t.Logf("Start: %s, location: %s\n", v.Start.String(), v.Start.ToTime().Location().String())

	if !v.Nothing.IsZero() {
		t.Fatalf("expected 'nothing' to be zero but got: %v", v.Nothing)
	}

	if !v.Empty.IsZero() {
		t.Fatalf("expected 'empty' to be zero but got: %v", v.Empty)
	}

	loc := time.FixedZone("EEST", 10800)

	if expected, got := time.Date(2022, time.August, 10, 3, 21, 0, 0, loc).String(), v.Start.ToTime().String(); expected != got {
		t.Fatalf("expected 'start' string to be: %v but got: %v", expected, got)
	}

	if expected, got := time.Date(2022, time.August, 10, 9, 49, 0, 0, loc).String(), v.End.ToTime().String(); expected != got {
		t.Fatalf("expected 'end' string to be: %v but got: %v", expected, got)
	}

	if expected, got := time.Date(2022, time.August, 10, 3, 21, 0, 0, loc), v.Start.ToTime().In(loc); expected != got {
		t.Fatalf("expected 'start' to be: %v but got: %v", expected, got)
	}

	if expected, got := time.Date(2022, time.August, 10, 9, 49, 0, 0, loc), v.End.ToTime().In(loc); expected != got {
		t.Fatalf("expected 'end' to be: %v but got: %v", expected, got)
	}
}

func TestISO8601WithZoneUTCOffsetWithoutMilliseconds(t *testing.T) {
	data := `{"start": "2023-02-04T09:48:14+00:00", "end": "2023-02-05T00:03:16+00:00", "nothing": null, "empty": ""}`
	v := struct {
		Start   ISO8601 `json:"start"`
		End     ISO8601 `json:"end"`
		Nothing ISO8601 `json:"nothing"`
		Empty   ISO8601 `json:"empty"`
	}{}
	err := json.Unmarshal([]byte(data), &v)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// t.Logf("Start: %s, location: %s\n", v.Start.String(), v.Start.ToTime().Location().String())

	if !v.Nothing.IsZero() {
		t.Fatalf("expected 'nothing' to be zero but got: %v", v.Nothing)
	}

	if !v.Empty.IsZero() {
		t.Fatalf("expected 'empty' to be zero but got: %v", v.Empty)
	}

	loc := time.FixedZone("UTC", 0)

	if expected, got := time.Date(2023, time.February, 04, 9, 48, 14, 0, loc).String(), v.Start.ToTime().String(); expected != got {
		t.Fatalf("expected 'start' string to be: %v but got: %v", expected, got)
	}

	if expected, got := time.Date(2023, time.February, 05, 0, 3, 16, 0, loc).String(), v.End.ToTime().String(); expected != got {
		t.Fatalf("expected 'end' string to be: %v but got: %v", expected, got)
	}

	if expected, got := time.Date(2023, time.February, 04, 9, 48, 14, 0, loc), v.Start.ToTime().In(loc); expected != got {
		t.Fatalf("expected 'start' to be: %v but got: %v", expected, got)
	}

	if expected, got := time.Date(2023, time.February, 05, 0, 3, 16, 0, loc), v.End.ToTime().In(loc); expected != got {
		t.Fatalf("expected 'end' to be: %v but got: %v", expected, got)
	}
}

func TestParseISO8601_StandardLayouts(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Time
		hasError bool
	}{
		{
			input:    "2024-05-21T18:06:07Z",
			expected: time.Date(2024, 5, 21, 18, 6, 7, 0, time.UTC),
			hasError: false,
		},
		{
			input:    "2024-05-21T18:06:07-04:00",
			expected: time.Date(2024, 5, 21, 22, 6, 7, 0, time.UTC),
			hasError: false,
		},
		{
			input:    "2024-05-21T18:06:07",
			expected: time.Date(2024, 5, 21, 18, 6, 7, 0, time.UTC), // no time local.
			hasError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			parsedTime, err := ParseISO8601(tt.input)
			if (err != nil) != tt.hasError {
				t.Errorf("ParseISO8601() error = %v, wantErr %v", err, tt.hasError)
				return
			}
			if !tt.hasError && !parsedTime.ToTime().Equal(tt.expected) {
				t.Errorf("ParseISO8601() = %v, want %v", parsedTime, tt.expected)
			}
		})
	}
}
func TestParseISO8601_UnconventionalOffset(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Time
		hasError bool
	}{
		{
			input:    "2024-05-21T18:06:07.000000-04:01:19",
			expected: time.Date(2024, 5, 21, 22, 7, 26, 0, time.UTC),
			hasError: false,
		},
		{
			input:    "2024-12-31T23:59:59.000000-00:00:59",
			expected: time.Date(2025, 1, 1, 0, 0, 58, 0, time.UTC),
			hasError: false,
		},
		{
			input:    "2024-05-21T18:06:07.000000+03:30:15",
			expected: time.Date(2024, 5, 21, 14, 35, 52, 0, time.UTC),
			hasError: false,
		},
		{
			input:    "2024-05-21T18:06:07.000000-24:00:00",
			expected: time.Date(2024, 5, 22, 18, 6, 7, 0, time.UTC),
			hasError: false,
		},
		{
			input:    "2024-05-21T18:06:07.000000-04:61:19", // Invalid minute part in offset
			expected: time.Time{},
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			parsedTime, err := ParseISO8601(tt.input)
			if (err != nil) != tt.hasError {
				t.Errorf("ParseISO8601() error = %v, wantErr %v", err, tt.hasError)
				return
			}
			if !tt.hasError && !parsedTime.ToTime().Equal(tt.expected) {
				t.Errorf("ParseISO8601() = %v, want %v", parsedTime, tt.expected)
			}
		})
	}
}
