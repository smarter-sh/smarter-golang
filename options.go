package smarter

import (
	"log/slog"
	"net/http"
	"os"
	"time"
)

// Option configures a [Client]. Every option is optional: see [NewClient] for where each value is
// looked up when it isn't given.
type Option func(*options)

type options struct {
	apiKey, baseURL string
	environment     Environment
	timeout         *time.Duration
	maxRetries      *int
	headers         http.Header
	httpClient      *http.Client
	logger          *slog.Logger
	lookupEnv       func(string) (string, bool)
}

// WithAPIKey sets the Smarter api key.
func WithAPIKey(apiKey string) Option {
	return func(o *options) { o.apiKey = apiKey }
}

// WithBaseURL sets the platform's url, e.g. https://platform.example.com or http://localhost:9357.
func WithBaseURL(baseURL string) Option {
	return func(o *options) { o.baseURL = baseURL }
}

// WithEnvironment sets the environment of the smarter cli's configuration file to use: local, alpha,
// beta, next or prod.
func WithEnvironment(environment Environment) Option {
	return func(o *options) { o.environment = environment }
}

// WithTimeout sets the timeout of each attempt of a request. For a stream, it applies to each read.
func WithTimeout(timeout time.Duration) Option {
	return func(o *options) { o.timeout = &timeout }
}

// WithMaxRetries sets how many times a request that failed for a transient reason (a timeout, a 429,
// a 502, ...) is retried. 0 disables retries, e.g. so that a prompt is never sent twice.
func WithMaxRetries(maxRetries int) Option {
	return func(o *options) { o.maxRetries = &maxRetries }
}

// WithHeader adds a header to every request.
func WithHeader(key, value string) Option {
	return func(o *options) {
		if o.headers == nil {
			o.headers = http.Header{}
		}
		o.headers.Add(key, value)
	}
}

// WithHTTPClient sets the *http.Client that sends the requests, e.g. one with a proxy. Its Timeout
// should be 0: the client applies its own timeout to each attempt.
func WithHTTPClient(client *http.Client) Option {
	return func(o *options) { o.httpClient = client }
}

// WithLogger sets the logger. Without one, the client logs nothing, unless SMARTER_LOG is debug or
// info, which log to stderr. See ADR-007.
func WithLogger(logger *slog.Logger) Option {
	return func(o *options) { o.logger = logger }
}

// WithLookupEnv sets the function that reads environment variables, os.LookupEnv by default, e.g. to
// configure a client from a map in a test.
func WithLookupEnv(lookup func(string) (string, bool)) Option {
	return func(o *options) { o.lookupEnv = lookup }
}

func newOptions(opts []Option) *options {
	o := &options{lookupEnv: os.LookupEnv}
	for _, opt := range opts {
		opt(o)
	}
	return o
}
