package smarter

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/rand/v2"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

// The SDK's http transport: authentication, retries, error mapping, and response decoding. Its
// retry rules are the Python SDK's (ADR-008).

// userAgent is the SDK's User-Agent.
var userAgent = "smarter-golang/" + Version

// ClientName is the SDK's X-Smarter-Client header, which identifies it to the platform's logs and journal.
const ClientName = "smarter-golang"

// maxRetryAfter is the longest Retry-After that is waited for.
const maxRetryAfter = 60 * time.Second

// retryableStatuses mean the request may succeed if it is sent again. 503 is retried only when the
// platform did not send it: see isRetryable.
var retryableStatuses = map[int]bool{
	http.StatusRequestTimeout:     true,
	http.StatusTooManyRequests:    true,
	http.StatusBadGateway:         true,
	http.StatusGatewayTimeout:     true,
	http.StatusServiceUnavailable: false,
}

// request is what one api call sends.
type request struct {
	method string
	path   string
	query  url.Values
	// body is a JSON body, encoded once, so that a retry can send it again.
	body any
	// form is a multipart body: its files, by field name.
	form []formFile
	// header overrides the client's headers for this request.
	header http.Header
}

// formFile is one file of a multipart body.
type formFile struct {
	field, name string
	content     []byte
}

// httpClient sends the requests of a Smarter client.
type httpClient struct {
	cfg     config
	client  *http.Client
	headers http.Header
	logger  *slog.Logger
}

// url returns the absolute url of an api path, relative to the api's base url.
func (h *httpClient) url(path string, query url.Values) string {
	u := path
	if !strings.HasPrefix(path, "http://") && !strings.HasPrefix(path, "https://") {
		u = h.cfg.baseURL + strings.TrimLeft(path, "/")
	}
	if len(query) > 0 {
		separator := "?"
		if strings.Contains(u, "?") {
			separator = "&"
		}
		u += separator + query.Encode()
	}
	return u
}

// encodeBody returns the request's body and its content type.
func (r *request) encodeBody() ([]byte, string, error) {
	if len(r.form) > 0 {
		var buffer bytes.Buffer
		writer := multipart.NewWriter(&buffer)
		for _, file := range r.form {
			part, err := writer.CreateFormFile(file.field, file.name)
			if err != nil {
				return nil, "", err
			}
			if _, err := part.Write(file.content); err != nil {
				return nil, "", err
			}
		}
		if err := writer.Close(); err != nil {
			return nil, "", err
		}
		return buffer.Bytes(), writer.FormDataContentType(), nil
	}
	if r.body == nil {
		return nil, "", nil
	}
	if raw, ok := r.body.(json.RawMessage); ok {
		return raw, "application/json", nil
	}
	data, err := json.Marshal(r.body)
	return data, "application/json", err
}

// send sends a request, with retries, and returns the successful response, which the caller must
// close. For a stream, the timeout applies to each read instead of to the whole response.
func (h *httpClient) send(ctx context.Context, r *request, stream bool) (*http.Response, context.CancelFunc, error) {
	target := h.url(r.path, r.query)
	body, contentType, err := r.encodeBody()
	if err != nil {
		return nil, nil, &ValueError{Message: "the request's body can't be encoded: " + err.Error(), Err: err}
	}
	if h.logger.Enabled(ctx, slog.LevelDebug) {
		shown := string(body)
		if len(r.form) > 0 {
			shown = "<multipart>"
		} else if len(shown) > 2000 {
			shown = shown[:2000]
		}
		h.logger.DebugContext(ctx, "request", "source", "http", "method", r.method, "url", target, "api_key", MaskSecret(h.cfg.apiKey), "body", shown)
	}
	for attempt := 0; ; attempt++ {
		attemptCtx, cancel := context.WithCancel(ctx)
		timer := time.AfterFunc(h.cfg.timeout, cancel)
		timedOut := func() bool { return !timer.Stop() && ctx.Err() == nil }

		req, err := http.NewRequestWithContext(attemptCtx, r.method, target, bytes.NewReader(body))
		if err != nil {
			cancel()
			return nil, nil, &ValueError{Message: "invalid request: " + err.Error(), Err: err}
		}
		h.setHeaders(req, r.header, contentType, stream)
		started := time.Now()
		response, err := h.client.Do(req)
		if err != nil {
			expired := timedOut()
			cancel()
			if ctx.Err() != nil {
				return nil, nil, &APIConnectionError{Method: r.method, URL: target, Err: ctx.Err()}
			}
			if attempt < h.cfg.maxRetries {
				reason := "connection error"
				if expired {
					reason = "timeout"
				}
				if err := h.wait(ctx, attempt, r.method, target, reason, nil); err != nil {
					return nil, nil, err
				}
				continue
			}
			return nil, nil, &APIConnectionError{Method: r.method, URL: target, Err: err, timeout: expired}
		}
		h.logger.InfoContext(ctx, "response", "source", "http", "method", r.method, "url", target, "status", response.StatusCode, "duration", time.Since(started).Round(time.Millisecond).String())
		if response.StatusCode < 300 {
			if stream {
				// the timeout restarts on each read: see idleBody.
				timer.Stop()
				body := &idleBody{body: response.Body, timeout: h.cfg.timeout}
				body.timer = time.AfterFunc(h.cfg.timeout, func() {
					body.expired.Store(true)
					cancel()
				})
				response.Body = body
				return response, cancel, nil
			}
			return response, func() { timer.Stop(); cancel() }, nil
		}
		data, readErr := io.ReadAll(response.Body)
		_ = response.Body.Close()
		expired := timedOut()
		cancel()
		if readErr != nil {
			return nil, nil, &APIConnectionError{Method: r.method, URL: target, Err: readErr, timeout: expired}
		}
		if attempt < h.cfg.maxRetries && isRetryable(response.StatusCode, data) {
			if err := h.wait(ctx, attempt, r.method, target, strconv.Itoa(response.StatusCode), response.Header); err != nil {
				return nil, nil, err
			}
			continue
		}
		return nil, nil, makeStatusError(r.method, target, response, data)
	}
}

func (h *httpClient) setHeaders(req *http.Request, extra http.Header, contentType string, stream bool) {
	req.Header.Set("Authorization", "Token "+h.cfg.apiKey)
	req.Header.Set("Accept", "application/json")
	if stream {
		req.Header.Set("Accept", eventStreamContentType)
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("X-Smarter-Client", ClientName)
	req.Header.Set("X-Smarter-Capabilities", ChatContractV2Capability)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	for key, values := range h.headers {
		req.Header[key] = values
	}
	for key, values := range extra {
		req.Header[key] = values
	}
}

// wait sleeps before a retry, or returns early when the context is done.
func (h *httpClient) wait(ctx context.Context, attempt int, method, target, reason string, header http.Header) error {
	delay := retryDelay(attempt, header)
	h.logger.InfoContext(ctx, "retrying", "source", "http", "method", method, "url", target, "delay", delay.String(), "after", reason)
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return &APIConnectionError{Method: method, URL: target, Err: ctx.Err()}
	case <-timer.C:
		return nil
	}
}

// do sends a request, and decodes its JSON response into out, unless out is nil.
func (h *httpClient) do(ctx context.Context, r *request, out any) error {
	response, done, err := h.send(ctx, r, false)
	if err != nil {
		return err
	}
	defer done()
	defer func() { _ = response.Body.Close() }()
	target := h.url(r.path, r.query)
	data, err := io.ReadAll(response.Body)
	if err != nil {
		return &APIConnectionError{Method: r.method, URL: target, Err: err, timeout: errors.Is(err, context.Canceled) && ctx.Err() == nil}
	}
	if h.logger.Enabled(ctx, slog.LevelDebug) {
		shown := string(data)
		if len(shown) > 2000 {
			shown = shown[:2000]
		}
		h.logger.DebugContext(ctx, "response body", "source", "http", "status", response.StatusCode, "body", shown)
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(data, out); err != nil {
		return &APIResponseValidationError{Method: r.method, URL: target, StatusCode: response.StatusCode, Body: data, Err: err}
	}
	return nil
}

// idleBody is a response body whose timeout restarts on each read.
type idleBody struct {
	body    io.ReadCloser
	timer   *time.Timer
	timeout time.Duration
	expired atomic.Bool
}

func (b *idleBody) Read(p []byte) (int, error) {
	n, err := b.body.Read(p)
	b.timer.Reset(b.timeout)
	return n, err
}

func (b *idleBody) Close() error {
	b.timer.Stop()
	return b.body.Close()
}

// journalError returns the Smarter Journal error of a response body, if it is a journaled error response.
func journalError(body []byte) *SAMJournalAPIErrorBody {
	var document struct {
		Error json.RawMessage `json:"error"`
	}
	if json.Unmarshal(body, &document) != nil || len(document.Error) == 0 || document.Error[0] != '{' {
		return nil
	}
	var journal SAMJournalAPIErrorBody
	if json.Unmarshal(document.Error, &journal) != nil || journal.ErrorClass == "" {
		return nil
	}
	return &journal
}

// errorMessage returns a human-readable message for an error response. The platform's views return
// errors in several shapes: the journal's error object, an error string, Django REST framework's
// detail, plain text, or a Django error page.
func errorMessage(status int, body []byte, journal *SAMJournalAPIErrorBody) string {
	if journal != nil {
		if journal.Description != "" {
			return journal.Description
		}
		return journal.ErrorClass
	}
	var document map[string]any
	if json.Unmarshal(body, &document) == nil {
		for _, key := range []string{"error", "detail", "message"} {
			if text, ok := document[key].(string); ok && text != "" {
				return text
			}
		}
		if nested, ok := document["error"].(map[string]any); ok {
			if text, ok := nested["description"].(string); ok {
				return text
			}
		}
	} else if text := strings.TrimSpace(string(body)); text != "" {
		head := text
		if len(head) > 500 {
			head = head[:500]
		}
		if !strings.Contains(strings.ToLower(head), "<html") {
			return head
		}
	}
	return strings.TrimSpace(fmt.Sprintf("%d %s", status, http.StatusText(status)))
}

// makeStatusError returns the error of a 4xx or 5xx response.
func makeStatusError(method, target string, response *http.Response, body []byte) *APIStatusError {
	journal := journalError(body)
	return &APIStatusError{
		Method:       method,
		URL:          target,
		StatusCode:   response.StatusCode,
		Message:      errorMessage(response.StatusCode, body, journal),
		Body:         body,
		Header:       response.Header,
		JournalError: journal,
	}
}

// isRetryable reports whether a failed response may succeed when the request is sent again. A 503 is
// retried only when it is not a journaled response: the platform itself answers 503 for a resource
// that is not ready (e.g. SAMBrokerErrorNotReady), which a retry won't change, whereas a 503 from a
// load balancer or proxy means the platform is briefly unavailable.
func isRetryable(status int, body []byte) bool {
	if status == http.StatusServiceUnavailable {
		return journalError(body) == nil
	}
	return retryableStatuses[status]
}

// retryDelay returns how long to wait before a retry: the response's Retry-After, or exponential
// backoff with jitter.
func retryDelay(attempt int, header http.Header) time.Duration {
	if header != nil {
		if value := strings.TrimSpace(header.Get("Retry-After")); value != "" {
			if seconds, err := strconv.ParseFloat(value, 64); err == nil && seconds >= 0 {
				if delay := time.Duration(seconds * float64(time.Second)); delay <= maxRetryAfter {
					return delay
				}
			}
		}
	}
	backoff := min(0.5*float64(int(1)<<min(attempt, 10)), 8.0)
	jitter := 1 - 0.25*rand.Float64() //nolint:gosec // G404: jitter, not security
	return time.Duration(backoff * jitter * float64(time.Second))
}

// query builds a request's query parameters, leaving out the empty ones.
type query url.Values

// str adds a string parameter, unless it is empty.
func (q query) str(key, value string) query {
	if value != "" {
		url.Values(q).Set(key, value)
	}
	return q
}

// num adds an integer parameter.
func (q query) num(key string, value int) query {
	url.Values(q).Set(key, strconv.Itoa(value))
	return q
}

// optionalBool adds true or false, unless the value is nil.
func (q query) optionalBool(key string, value *bool) query {
	if value != nil {
		url.Values(q).Set(key, strconv.FormatBool(*value))
	}
	return q
}

// flag adds true, only when the value is true.
func (q query) flag(key string, value bool) query {
	if value {
		url.Values(q).Set(key, "true")
	}
	return q
}

func newJSONDecoder(r io.Reader) *json.Decoder { return json.NewDecoder(r) }

func isEOF(err error) bool { return errors.Is(err, io.EOF) }
