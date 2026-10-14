package smarter

import (
	"bufio"
	"encoding/json"
	"io"
	"strconv"
	"strings"
)

// eventStreamContentType is the content type of a Server-Sent Events stream.
const eventStreamContentType = "text/event-stream"

// ServerSentEvent is one Server-Sent Event. The Smarter prompt api streams a prompt's progress as
// progress events while it runs, then one result event with the same JSON as a non-streaming
// response.
type ServerSentEvent struct {
	// Event is the event's type, e.g. "progress", or "" for an unnamed event.
	Event string
	// Data is the event's data: its data: lines, joined by newlines.
	Data string
	// ID is the event's id, if any.
	ID string
	// Retry is the reconnection time that the server asked for, in milliseconds, or 0.
	Retry int
}

// JSON decodes the event's data into v.
func (e ServerSentEvent) JSON(v any) error {
	return json.Unmarshal([]byte(e.Data), v)
}

// sseReader parses a stream into events, as the HTML Living Standard specifies.
type sseReader struct {
	scanner *bufio.Scanner
}

func newSSEReader(r io.Reader) *sseReader {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), 16*1024*1024)
	scanner.Split(scanLines)
	return &sseReader{scanner: scanner}
}

// scanLines splits at \n, \r\n or \r.
func scanLines(data []byte, atEOF bool) (int, []byte, error) {
	for i, b := range data {
		if b == '\n' {
			return i + 1, data[:i], nil
		}
		if b == '\r' {
			if i+1 < len(data) {
				if data[i+1] == '\n' {
					return i + 2, data[:i], nil
				}
				return i + 1, data[:i], nil
			}
			if atEOF {
				return i + 1, data[:i], nil
			}
			return 0, nil, nil // a \r at the end of the buffer may be the first half of a \r\n
		}
	}
	if atEOF && len(data) > 0 {
		return len(data), data, nil
	}
	return 0, nil, nil
}

// next returns the next event. A blank line ends an event. Comments (lines that start with :), and
// blocks that set only retry, yield nothing. It returns io.EOF at the end of the stream.
func (r *sseReader) next() (ServerSentEvent, error) {
	var event ServerSentEvent
	var data []string
	named := false
	for r.scanner.Scan() {
		line := r.scanner.Text()
		if line == "" {
			if len(data) > 0 || named {
				event.Data = strings.Join(data, "\n")
				return event, nil
			}
			event, data = ServerSentEvent{}, nil
			continue
		}
		if strings.HasPrefix(line, ":") {
			continue
		}
		field, value, _ := strings.Cut(line, ":")
		value = strings.TrimPrefix(value, " ")
		switch field {
		case "event":
			event.Event, named = value, true
		case "data":
			data = append(data, value)
		case "id":
			event.ID = value
		case "retry":
			if retry, err := strconv.Atoi(value); err == nil && retry >= 0 {
				event.Retry = retry
			}
		}
	}
	if err := r.scanner.Err(); err != nil {
		return ServerSentEvent{}, err
	}
	if len(data) > 0 || named {
		event.Data = strings.Join(data, "\n")
		return event, nil
	}
	return ServerSentEvent{}, io.EOF
}
