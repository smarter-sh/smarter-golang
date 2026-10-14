package smarter_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	smarter "github.com/smarter-sh/smarter-golang"
)

func TestHeaders(t *testing.T) {
	p := newMockPlatform(t)
	p.route("POST", "cli/whoami/", reply{fixture: "whoami"})
	_, err := p.makeClient(t, smarter.WithHeader("X-Custom", "1")).WhoAmI(context.Background())
	check(t, err)
	header := p.last().Header
	equal(t, header.Get("Authorization"), "Token "+apiKey)
	equal(t, header.Get("X-Smarter-Client"), smarter.ClientName)
	equal(t, header.Get("X-Smarter-Capabilities"), "chat-contract-v2")
	equal(t, header.Get("X-Custom"), "1")
	if !strings.HasPrefix(header.Get("User-Agent"), "smarter-golang/") {
		t.Errorf("got %q", header.Get("User-Agent"))
	}
}

func TestRetriesThenSucceeds(t *testing.T) {
	p := newMockPlatform(t)
	var calls atomic.Int32
	p.route("POST", "cli/whoami/", reply{handler: func(req *http.Request) (*http.Response, error) {
		if calls.Add(1) == 1 {
			return response(req, 502, "{}", map[string]string{"Retry-After": "0"}), nil
		}
		return response(req, 200, string(loadBytes(t, "whoami.json")), nil), nil
	}})
	_, err := p.makeClient(t, smarter.WithMaxRetries(2)).WhoAmI(context.Background())
	check(t, err)
	equal(t, p.count(), 2)
}

func TestRetriesExhausted(t *testing.T) {
	p := newMockPlatform(t)
	p.route("POST", "cli/whoami/", reply{status: 429, json: map[string]any{"detail": "slow down"}, header: map[string]string{"Retry-After": "0"}})
	_, err := p.makeClient(t, smarter.WithMaxRetries(2)).WhoAmI(context.Background())
	if !errors.Is(err, smarter.ErrRateLimit) {
		t.Fatalf("got %v", err)
	}
	equal(t, p.count(), 3)
}

func TestNoRetryForClientErrors(t *testing.T) {
	p := newMockPlatform(t)
	p.route("POST", "cli/whoami/", reply{status: 401, json: map[string]any{"detail": "Invalid token."}})
	_, err := p.makeClient(t, smarter.WithMaxRetries(3)).WhoAmI(context.Background())
	var statusError *smarter.APIStatusError
	if !errors.As(err, &statusError) || !errors.Is(err, smarter.ErrAuthentication) {
		t.Fatalf("got %v", err)
	}
	equal(t, statusError.Message, "Invalid token.")
	equal(t, p.count(), 1)
}

// blockUntilCancelled answers nothing until the request is cancelled.
func blockUntilCancelled(req *http.Request) (*http.Response, error) {
	<-req.Context().Done()
	return nil, req.Context().Err()
}

func TestTimeout(t *testing.T) {
	p := newMockPlatform(t)
	p.route("POST", "cli/whoami/", reply{handler: blockUntilCancelled})
	_, err := p.makeClient(t, smarter.WithMaxRetries(1), smarter.WithTimeout(20*time.Millisecond)).WhoAmI(context.Background())
	var connection *smarter.APIConnectionError
	if !errors.As(err, &connection) || !connection.Timeout() {
		t.Fatalf("got %v", err)
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Errorf("got %v", err)
	}
	equal(t, p.count(), 2)
}

func TestConnectionError(t *testing.T) {
	p := newMockPlatform(t)
	p.route("POST", "cli/whoami/", reply{handler: func(*http.Request) (*http.Response, error) {
		return nil, errors.New("refused")
	}})
	_, err := p.makeClient(t, smarter.WithMaxRetries(1)).WhoAmI(context.Background())
	var connection *smarter.APIConnectionError
	if !errors.As(err, &connection) || connection.Timeout() || !strings.Contains(err.Error(), "refused") || !errors.Is(err, smarter.ErrSmarter) {
		t.Fatalf("got %v", err)
	}
	equal(t, p.count(), 2)
}

func TestCallerCancels(t *testing.T) {
	p := newMockPlatform(t)
	p.route("POST", "cli/whoami/", reply{handler: blockUntilCancelled})
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()
	_, err := p.makeClient(t, smarter.WithMaxRetries(3)).WhoAmI(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v", err)
	}
	equal(t, p.count(), 1)
}

func TestCallerCancelsDuringRetryWait(t *testing.T) {
	p := newMockPlatform(t)
	p.route("POST", "cli/whoami/", reply{status: 502, json: map[string]any{}, header: map[string]string{"Retry-After": "30"}})
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	_, err := p.makeClient(t, smarter.WithMaxRetries(3)).WhoAmI(ctx)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("got %v", err)
	}
}

func TestResponseValidationError(t *testing.T) {
	p := newMockPlatform(t)
	p.route("POST", "cli/whoami/", reply{json: map[string]any{"data": map[string]any{"user": "not an object"}}})
	_, err := p.client.WhoAmI(context.Background())
	var validation *smarter.APIResponseValidationError
	if !errors.As(err, &validation) || validation.StatusCode != 200 {
		t.Fatalf("got %v", err)
	}
}

func TestBrokenBody(t *testing.T) {
	p := newMockPlatform(t)
	p.route("POST", "cli/whoami/", reply{handler: func(req *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(&failingReader{}), Request: req}, nil
	}})
	_, err := p.client.WhoAmI(context.Background())
	var connection *smarter.APIConnectionError
	if !errors.As(err, &connection) {
		t.Fatalf("got %v", err)
	}
	p.route("POST", "cli/version/", reply{handler: func(req *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 500, Header: http.Header{}, Body: io.NopCloser(&failingReader{}), Request: req}, nil
	}})
	if _, err := p.client.Version(context.Background()); !errors.As(err, &connection) {
		t.Fatalf("got %v", err)
	}
}

type failingReader struct{}

func (*failingReader) Read([]byte) (int, error) { return 0, errors.New("reset") }

func TestStreamTimeoutAndErrors(t *testing.T) {
	p := newMockPlatform(t)
	stalled := func(req *http.Request) (*http.Response, error) {
		reader, writer := io.Pipe()
		go func() {
			<-req.Context().Done()
			_ = writer.CloseWithError(req.Context().Err())
		}()
		return &http.Response{StatusCode: 200, Header: http.Header{}, Body: reader, Request: req}, nil
	}
	p.route("POST", "llm-clients/5/prompt/", reply{handler: stalled})
	client := p.makeClient(t, smarter.WithTimeout(20*time.Millisecond))
	stream := client.LLMClients.Stream(context.Background(), smarter.IntID(5), smarter.Text("hi"), nil)
	defer func() { _ = stream.Close() }()
	if stream.Next() {
		t.Fatal("a stalled stream gave an event")
	}
	var connection *smarter.APIConnectionError
	if !errors.As(stream.Err(), &connection) || !connection.Timeout() {
		t.Fatalf("got %v", stream.Err())
	}
	if _, err := stream.Response(); !errors.Is(err, smarter.ErrSmarter) {
		t.Fatalf("got %v", err)
	}

	p.route("POST", "llm-clients/6/prompt/", reply{handler: func(req *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(&failingReader{}), Request: req}, nil
	}})
	broken := p.client.LLMClients.Stream(context.Background(), smarter.IntID(6), smarter.Text("hi"), nil)
	if broken.Next() || !errors.As(broken.Err(), &connection) || connection.Timeout() {
		t.Fatalf("got %v", broken.Err())
	}

	p.route("POST", "llm-clients/7/prompt/", reply{text: "event: progress\ndata: not json\n\n"})
	invalid := p.client.LLMClients.Stream(context.Background(), smarter.IntID(7), smarter.Text("hi"), nil)
	var validation *smarter.APIResponseValidationError
	if invalid.Next() || !errors.As(invalid.Err(), &validation) {
		t.Fatalf("got %v", invalid.Err())
	}
	p.route("POST", "llm-clients/8/prompt/", reply{text: "event: result\ndata: not json\n\n"})
	invalid = p.client.LLMClients.Stream(context.Background(), smarter.IntID(8), smarter.Text("hi"), nil)
	if invalid.Next() || !errors.As(invalid.Err(), &validation) {
		t.Fatalf("got %v", invalid.Err())
	}
	p.route("POST", "llm-clients/9/prompt/", reply{status: 404, json: map[string]any{"detail": "gone"}})
	missing := p.client.LLMClients.Stream(context.Background(), smarter.IntID(9), smarter.Text("hi"), nil)
	if missing.Next() || !errors.Is(missing.Err(), smarter.ErrNotFound) {
		t.Fatalf("got %v", missing.Err())
	}
	bad := p.client.LLMClients.Stream(context.Background(), smarter.ID("a/b"), smarter.Text("hi"), nil)
	if bad.Next() || !errors.Is(bad.Err(), smarter.ErrSmarter) {
		t.Fatalf("got %v", bad.Err())
	}
}

func TestStreamCallerCancels(t *testing.T) {
	p := newMockPlatform(t)
	p.route("POST", "llm-clients/5/prompt/", reply{handler: func(req *http.Request) (*http.Response, error) {
		reader, writer := io.Pipe()
		go func() {
			_, _ = writer.Write([]byte("event: progress\ndata: {\"type\": \"llm_request\", \"message\": \"m\"}\n\n"))
			<-req.Context().Done()
			_ = writer.CloseWithError(req.Context().Err())
		}()
		return &http.Response{StatusCode: 200, Header: http.Header{}, Body: reader, Request: req}, nil
	}})
	ctx, cancel := context.WithCancel(context.Background())
	stream := p.client.LLMClients.Stream(ctx, smarter.IntID(5), smarter.Text("hi"), nil)
	if !stream.Next() {
		t.Fatalf("no first event: %v", stream.Err())
	}
	cancel()
	if stream.Next() || !errors.Is(stream.Err(), context.Canceled) {
		t.Fatalf("got %v", stream.Err())
	}
	check(t, stream.Close())
}
