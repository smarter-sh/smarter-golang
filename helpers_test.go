package smarter_test

// The helpers of the SDK's unit tests: the counterpart of the Python SDK's SmarterTestBase.
// newMockPlatform gives each test a client whose requests never leave the process: a RoundTripper
// answers them from the routes that the test registers, usually with the sanitized JSON of a real
// platform response from testdata/.

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	smarter "github.com/smarter-sh/smarter-golang"
)

const (
	apiKey  = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaabeef"
	baseURL = "http://testserver"
	apiURL  = baseURL + "/api/v1/"
)

// reply is a route's answer.
type reply struct {
	fixture string            // the name of a JSON file in testdata/, without .json
	json    any               // the response's JSON
	status  int               // the response's status, 200 by default
	text    string            // the response's text, instead of JSON
	header  map[string]string // the response's headers
	handler func(*http.Request) (*http.Response, error)
}

// recorded is a request that a client sent, with its body.
type recorded struct {
	*http.Request
	body []byte
}

// mockPlatform answers a client's requests from its routes. An unrouted request gets a 404.
type mockPlatform struct {
	t        *testing.T
	mu       sync.Mutex
	routes   map[string]reply
	requests []recorded
	client   *smarter.Client
}

// testEnv is the environment that a test's client sees: none of the developer's SMARTER_* variables,
// and no smarter cli configuration file.
func testEnv(values map[string]string) func(string) (string, bool) {
	return func(name string) (string, bool) {
		if value, ok := values[name]; ok {
			return value, true
		}
		if name == "SMARTER_CONFIG_FILE" {
			return filepath.Join("testdata", "no-such-config.yaml"), true
		}
		return "", false
	}
}

func newMockPlatform(t *testing.T) *mockPlatform {
	t.Helper()
	p := &mockPlatform{t: t, routes: map[string]reply{}}
	p.client = p.makeClient(t)
	return p
}

// makeClient returns another client on the same routes.
func (p *mockPlatform) makeClient(t *testing.T, opts ...smarter.Option) *smarter.Client {
	t.Helper()
	base := []smarter.Option{
		smarter.WithAPIKey(apiKey),
		smarter.WithBaseURL(baseURL),
		smarter.WithMaxRetries(0),
		smarter.WithLookupEnv(testEnv(nil)),
		smarter.WithHTTPClient(&http.Client{Transport: p}),
	}
	client, err := smarter.NewClient(append(base, opts...)...)
	if err != nil {
		t.Fatal(err)
	}
	return client
}

// route answers requests to a path, relative to /api/v1/.
func (p *mockPlatform) route(method, path string, r reply) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.routes[method+" /api/v1/"+strings.TrimLeft(path, "/")] = r
}

// RoundTrip records the request, and answers it from its route.
func (p *mockPlatform) RoundTrip(req *http.Request) (*http.Response, error) {
	var body []byte
	if req.Body != nil {
		body, _ = io.ReadAll(req.Body)
		_ = req.Body.Close()
		req.Body = io.NopCloser(bytes.NewReader(body))
	}
	p.mu.Lock()
	p.requests = append(p.requests, recorded{Request: req, body: body})
	r, ok := p.routes[req.Method+" "+req.URL.Path]
	p.mu.Unlock()
	if !ok {
		return response(req, http.StatusNotFound, `{"detail": "no route"}`, nil), nil
	}
	if r.handler != nil {
		return r.handler(req)
	}
	status := r.status
	if status == 0 {
		status = http.StatusOK
	}
	if r.text != "" {
		return response(req, status, r.text, r.header), nil
	}
	var data []byte
	if r.fixture != "" {
		data = loadBytes(p.t, r.fixture+".json")
	} else if r.json != nil {
		data, _ = json.Marshal(r.json)
	}
	return response(req, status, string(data), r.header), nil
}

func response(req *http.Request, status int, body string, header map[string]string) *http.Response {
	h := http.Header{"Content-Type": {"application/json"}}
	for key, value := range header {
		h.Set(key, value)
	}
	return &http.Response{StatusCode: status, Status: http.StatusText(status), Header: h, Body: io.NopCloser(strings.NewReader(body)), Request: req}
}

// last returns the last request that a client sent.
func (p *mockPlatform) last() recorded {
	p.t.Helper()
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.requests) == 0 {
		p.t.Fatal("no request was sent")
	}
	return p.requests[len(p.requests)-1]
}

// lastJSON returns the JSON body of the last request.
func (p *mockPlatform) lastJSON() any {
	p.t.Helper()
	var body any
	if err := json.Unmarshal(p.last().body, &body); err != nil {
		p.t.Fatalf("the request's body is not JSON: %v", err)
	}
	return body
}

// count returns how many requests the clients sent.
func (p *mockPlatform) count() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.requests)
}

// loadBytes returns a file of testdata/.
func loadBytes(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// loadJSON returns a JSON file of testdata/, by its name without .json.
func loadJSON(t *testing.T, name string) any {
	t.Helper()
	var value any
	if err := json.Unmarshal(loadBytes(t, name+".json"), &value); err != nil {
		t.Fatal(err)
	}
	return value
}

// object returns a JSON value as an object, or fails.
func object(t *testing.T, value any) map[string]any {
	t.Helper()
	result, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("not a JSON object: %v", value)
	}
	return result
}

func check(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func equal[T comparable](t *testing.T, got, want T) {
	t.Helper()
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func jsonEqual(t *testing.T, got, want any) {
	t.Helper()
	g, _ := json.Marshal(got)
	w, _ := json.Marshal(want)
	if !bytes.Equal(g, w) {
		t.Errorf("got %s, want %s", g, w)
	}
}
