package smarter

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestErrorMessage(t *testing.T) {
	journaled, err := os.ReadFile(filepath.Join("testdata", "describe_missing.json"))
	if err != nil {
		t.Fatal(err)
	}
	journal := journalError(journaled)
	if journal == nil || journal.ErrorClass != "SAMBrokerErrorNotReady" {
		t.Fatalf("the journal error was not parsed: %v", journal)
	}
	cases := []struct {
		body, want string
	}{
		{string(journaled), journal.Description},
		{`{"error": "text is required."}`, "text is required."},
		{`{"detail": "Not found."}`, "Not found."},
		{`{"error": {"description": "nested"}}`, "nested"},
		{"Forbidden: no", "Forbidden: no"},
		{"<html><body>debug page</body></html>", "500 Internal Server Error"},
		{"", "500 Internal Server Error"},
	}
	for _, c := range cases {
		if got := errorMessage(500, []byte(c.body), journalError([]byte(c.body))); got != c.want {
			t.Errorf("%q: got %q, want %q", c.body, got, c.want)
		}
	}
	if got := errorMessage(500, nil, &SAMJournalAPIErrorBody{ErrorClass: "X"}); got != "X" {
		t.Errorf("got %q", got)
	}
	if journalError([]byte(`{"error": {"unexpected": true}}`)) != nil || journalError([]byte(`["a"]`)) != nil {
		t.Error("a non-journal body was parsed")
	}
}

func TestIsRetryable(t *testing.T) {
	journaled, _ := os.ReadFile(filepath.Join("testdata", "describe_missing.json"))
	cases := []struct {
		status int
		body   string
		want   bool
	}{
		{502, "", true}, {429, "", true}, {408, "", true}, {504, "", true},
		{503, "Service Unavailable", true}, {503, string(journaled), false},
		{500, "", false}, {404, "", false},
	}
	for _, c := range cases {
		if got := isRetryable(c.status, []byte(c.body)); got != c.want {
			t.Errorf("%d: got %v", c.status, got)
		}
	}
}

func TestRetryDelay(t *testing.T) {
	if got := retryDelay(0, http.Header{"Retry-After": {"3"}}); got != 3*time.Second {
		t.Errorf("got %v", got)
	}
	for _, value := range []string{"3600", "soon"} {
		if got := retryDelay(0, http.Header{"Retry-After": {value}}); got > 500*time.Millisecond {
			t.Errorf("%s: got %v", value, got)
		}
	}
	if got := retryDelay(3, nil); got < 2900*time.Millisecond {
		t.Errorf("got %v", got)
	}
	if got := retryDelay(40, nil); got > 8*time.Second {
		t.Errorf("got %v", got)
	}
}

func TestSSE(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "prompt_sse.txt"))
	if err != nil {
		t.Fatal(err)
	}
	var events []ServerSentEvent
	reader := newSSEReader(strings.NewReader(string(data)))
	for {
		event, err := reader.next()
		if err != nil {
			break
		}
		events = append(events, event)
	}
	if len(events) == 0 || events[len(events)-1].Event != "result" {
		t.Fatalf("got %v", events)
	}
	var first struct{ Type string }
	if err := events[0].JSON(&first); err != nil || first.Type != "llm_request" {
		t.Errorf("got %v %v", first, err)
	}

	text := ": comment\r\nretry: 3000\r\n\r\nevent: x\nid: 7\nretry: 10\ndata: a\ndata:b\n\nfoo: bar\rretry: soon\rdata: last"
	reader = newSSEReader(strings.NewReader(text))
	want := []ServerSentEvent{{Event: "x", Data: "a\nb", ID: "7", Retry: 10}, {Data: "last"}}
	for _, w := range want {
		got, err := reader.next()
		if err != nil || got != w {
			t.Errorf("got %+v %v, want %+v", got, err, w)
		}
	}
	if _, err := reader.next(); err == nil {
		t.Error("expected the end of the stream")
	}
}

func TestTypes(t *testing.T) {
	if IntID(5) != "5" {
		t.Error("IntID")
	}
	if segment, err := ID(" rMTAwMDAwNQx ").segment(); err != nil || segment != "rMTAwMDAwNQx" {
		t.Errorf("got %q %v", segment, err)
	}
	for _, id := range []ID{"", "a/b"} {
		if _, err := id.segment(); err == nil {
			t.Errorf("%q: no error", id)
		}
	}
	if *Ptr(true) != true || Deref[int](nil) != 0 || Deref(Ptr(3)) != 3 {
		t.Error("Ptr or Deref")
	}
	if MaskSecret("short") != "****" || MaskSecret("") != "None" {
		t.Error("MaskSecret")
	}
	if _, err := kindSegment("a/b"); err == nil {
		t.Error("kindSegment")
	}
}
