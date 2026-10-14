package smarter_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	smarter "github.com/smarter-sh/smarter-golang"
)

func TestStatusErrors(t *testing.T) {
	cases := map[int]error{
		400: smarter.ErrBadRequest,
		401: smarter.ErrAuthentication,
		403: smarter.ErrPermissionDenied,
		404: smarter.ErrNotFound,
		409: smarter.ErrConflict,
		422: smarter.ErrUnprocessableEntity,
		429: smarter.ErrRateLimit,
		500: smarter.ErrInternalServer,
		503: smarter.ErrInternalServer,
		418: smarter.ErrSmarter,
	}
	for status, sentinel := range cases {
		p := newMockPlatform(t)
		p.route("POST", "cli/whoami/", reply{status: status, json: map[string]any{}})
		_, err := p.client.WhoAmI(context.Background())
		var statusError *smarter.APIStatusError
		if !errors.As(err, &statusError) || !errors.Is(err, sentinel) || !errors.Is(err, smarter.ErrSmarter) {
			t.Errorf("%d: got %v", status, err)
			continue
		}
		equal(t, statusError.StatusCode, status)
		// wrapped by the caller, it still matches.
		if !errors.Is(fmt.Errorf("context: %w", err), sentinel) {
			t.Errorf("%d: a wrapped error doesn't match", status)
		}
	}
}

func TestNotFoundIsResourceNotFound(t *testing.T) {
	p := newMockPlatform(t)
	p.route("POST", "cli/describe/LLMClient/", reply{status: 404, json: map[string]any{"detail": "gone"}})
	_, err := p.client.Manifests.Describe(context.Background(), "LLMClient", "x")
	if !errors.Is(err, smarter.ErrResourceNotFound) || errors.Is(err, smarter.ErrBadRequest) {
		t.Fatalf("got %v", err)
	}
	p.route("GET", "guardrails/", reply{json: []any{}})
	_, err = p.client.Guardrails.Find(context.Background(), "nope")
	if !errors.Is(err, smarter.ErrResourceNotFound) || errors.Is(err, smarter.ErrNotFound) {
		t.Fatalf("got %v", err)
	}
	equal(t, err.Error(), `smarter: Guardrail "nope" not found`)
}

func TestStatusErrorDetails(t *testing.T) {
	p := newMockPlatform(t)
	p.route("POST", "cli/describe/LLMClient/", reply{status: 404, fixture: "describe_missing"})
	_, err := p.client.Manifests.Describe(context.Background(), "LLMClient", "x")
	var statusError *smarter.APIStatusError
	if !errors.As(err, &statusError) {
		t.Fatalf("got %v", err)
	}
	equal(t, statusError.ErrorClass(), "SAMBrokerErrorNotReady")
	equal(t, statusError.Error(), "smarter: error code: 404 - "+statusError.Message)
	equal(t, (&smarter.APIStatusError{StatusCode: 418}).ErrorClass(), "")
}

func TestOtherErrors(t *testing.T) {
	value := &smarter.ValueError{Message: "bad", Err: errors.New("cause")}
	if !errors.Is(value, smarter.ErrSmarter) || !errors.Is(value, value.Err) {
		t.Error("a ValueError is not a smarter error, or loses its cause")
	}
	equal(t, value.Error(), "smarter: bad")
	prompt := &smarter.PromptError{Message: "boom"}
	if !errors.Is(prompt, smarter.ErrSmarter) {
		t.Error("a PromptError is not a smarter error")
	}
	equal(t, prompt.Error(), "smarter: prompt failed: boom")
	if !errors.Is(&smarter.ConfigurationError{Message: "x"}, smarter.ErrSmarter) {
		t.Error("a ConfigurationError is not a smarter error")
	}
	validation := &smarter.APIResponseValidationError{Method: "GET", URL: "u", Err: errors.New("mismatch")}
	if !errors.Is(validation, smarter.ErrSmarter) || validation.Error() == "" {
		t.Error("an APIResponseValidationError is not a smarter error")
	}
}
