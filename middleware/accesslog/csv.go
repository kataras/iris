package accesslog

import (
	"encoding/csv"
	"io"
	"strconv"
	"time"
)

// CSV is a Formatter type for csv encoded logs.
type CSV struct {
	writer *csv.Writer
	ac     *AccessLog

	// Add header fields to the first line if it's not exist.
	// Note that the destination should be a compatible io.Reader
	// with access to write.
	Header bool
	// Google Spreadsheet's Script to wrap the Timestamp field
	// in order to convert it into a readable date.
	// Example: "FROM_UNIX" when
	// function FROM_UNIX(epoch_in_millis) {
	// 	return new Date(epoch_in_millis);
	// }
	DateScript string
	// Latency Round base, e.g. time.Second.
	LatencyRound time.Duration
	// Writes immediately every record.
	AutoFlush bool

	// TODO: Fields []string // field name, position?
}

// SetOutput initializes the csv writer.
// It uses the "dest" as AccessLog to
// write the first csv record which
// contains the names of the future log values.
func (f *CSV) SetOutput(dest io.Writer) {
	ac, ok := dest.(*AccessLog)
	if !ok {
		panic("SetOutput with invalid type. Report it as bug.")
	}

	w := csv.NewWriter(dest)
	f.writer = w
	f.ac = ac

	if !f.Header {
		return
	}

	{
		// If the destination is not a reader
		// we can't detect if the header already inserted
		// so we exit, we dont want to malform the contents.
		destReader, ok := ac.Writer.(io.Reader)
		if !ok {
			return
		}

		r := csv.NewReader(destReader)
		if header, err := r.Read(); err == nil && len(header) > 0 && header[0] == "Timestamp" {
			// we assume header already exists, exit.
			return
		}
	}

	// Write the header.

	keys := []string{"Timestamp", "Latency", "Code", "Method", "Path"}

	if ac.IP {
		keys = append(keys, "IP")
	}

	// keys = append(keys, []string{"Params", "Query"}...)
	keys = append(keys, "Req Values")

	/*
		if len(ac.FieldSetters) > 0 {
			keys = append(keys, "Fields")
		} // Make fields their own headers?
	*/

	if ac.BytesReceived {
		keys = append(keys, "In")
	}

	if ac.BytesSent {
		keys = append(keys, "Out")
	}

	if ac.RequestBody {
		keys = append(keys, "Request")
	}

	if ac.ResponseBody {
		keys = append(keys, "Response")
	}

	w.Write(keys)
	w.Flush()
}

// Format writes an incoming log using CSV encoding.
func (f *CSV) Format(log *Log) (bool, error) {
	// Timestamp, Latency, Code, Method, Path, IP, Path Params Query Fields
	//|Bytes Received|Bytes Sent|Request|Response|

	timestamp := strconv.FormatInt(log.Timestamp, 10)

	if f.DateScript != "" {
		timestamp = "=" + f.DateScript + "(" + timestamp + ")"
	}

	lat := ""
	if f.LatencyRound > 0 {
		lat = log.Latency.Round(f.LatencyRound).String()
	} else {
		lat = log.Latency.String()
	}

	values := []string{
		timestamp,
		lat,
		strconv.Itoa(log.Code),
		log.Method,
		log.Path,
	}

	if f.ac.IP {
		values = append(values, log.IP)
	}

	parseRequestValues(log.Code, log.PathParams, log.Query, log.Fields)
	values = append(values, log.RequestValuesLine())

	if f.ac.BytesReceived {
		values = append(values, strconv.Itoa(log.BytesReceived))
	}

	if f.ac.BytesSent {
		values = append(values, strconv.Itoa(log.BytesSent))
	}

	if f.ac.RequestBody {
		values = append(values, log.Request)
	}

	if f.ac.ResponseBody {
		values = append(values, log.Response)
	}

	f.writer.Write(values)

	if f.AutoFlush {
		return true, f.Flush()
	}
	return true, nil
}

// Flush implements the Fluster interface.
// Flushes any buffered csv records to the destination.
func (f *CSV) Flush() error {
	f.writer.Flush()
	return f.writer.Error()
}
