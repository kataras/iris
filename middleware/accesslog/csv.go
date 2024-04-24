package accesslog

import (
	"encoding/csv"
	"io"
	"strconv"
	"sync"
)

// CSV is a Formatter type for csv encoded logs.
type CSV struct {
	writerPool *sync.Pool
	ac         *AccessLog

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

	// TODO: Fields []string // field name, position?
}

// SetOutput initializes the csv writer.
// It uses the "dest" as AccessLog to
// write the first csv record which
// contains the names of the future log values.
func (f *CSV) SetOutput(dest io.Writer) {
	f.ac, _ = dest.(*AccessLog)
	f.writerPool = &sync.Pool{
		New: func() interface{} {
			return csv.NewWriter(dest)
		},
	}

	if !f.Header {
		return
	}

	{
		// If the destination is not a reader
		// we can't detect if the header already inserted
		// so we exit, we dont want to malform the contents.
		destReader, ok := f.ac.Writer.(io.Reader)
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
	w := csv.NewWriter(dest)

	keys := []string{"Timestamp", "Latency", "Code", "Method", "Path"}

	if f.ac.IP {
		keys = append(keys, "IP")
	}

	// keys = append(keys, []string{"Params", "Query"}...)
	keys = append(keys, "Req Values")

	if f.ac.BytesReceived || f.ac.BytesReceivedBody {
		keys = append(keys, "In")
	}

	if f.ac.BytesSent || f.ac.BytesSentBody {
		keys = append(keys, "Out")
	}

	if f.ac.RequestBody {
		keys = append(keys, "Request")
	}

	if f.ac.ResponseBody {
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

	values := []string{
		timestamp,
		log.Latency.String(),
		strconv.Itoa(log.Code),
		log.Method,
		log.Path,
	}

	if f.ac.IP {
		values = append(values, log.IP)
	}

	if s := log.RequestValuesLine(); s != "" || f.Header {
		// even if it's empty, if Header was set, then add it.
		values = append(values, s)
	}

	if f.ac.BytesReceived || f.ac.BytesReceivedBody {
		values = append(values, strconv.Itoa(log.BytesReceived))
	}

	if f.ac.BytesSent || f.ac.BytesSentBody {
		values = append(values, strconv.Itoa(log.BytesSent))
	}

	if f.ac.RequestBody && (log.Request != "" || f.Header) {
		values = append(values, log.Request)
	}

	if f.ac.ResponseBody && (log.Response != "" || f.Header) {
		values = append(values, log.Response)
	}

	w := f.writerPool.Get().(*csv.Writer)
	err := w.Write(values)
	w.Flush() // it works as "reset" too.
	f.writerPool.Put(w)
	return true, err
}
