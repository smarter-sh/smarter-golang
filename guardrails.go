package smarter

import (
	"context"
	"net/http"
	"net/url"
)

// Stage is the stage of a prompt that a Guardrail runs on.
type Stage string

// The stages of a prompt.
const (
	// StageInput is the prompt.
	StageInput Stage = "input"
	// StageOutput is the LLM's reply.
	StageOutput Stage = "output"
)

// GuardrailsService holds the operations of Guardrails: /api/v1/guardrails/, and the manifest api's
// Guardrail kind. A Guardrail is a check that the platform runs on prompts and on the LLM's replies.
type GuardrailsService struct {
	*KindService[Guardrail]
}

// GuardrailEvaluateParams are the optional arguments of [GuardrailsService.Evaluate].
type GuardrailEvaluateParams struct {
	// Stage is the stage to evaluate the text as, StageInput by default. The Guardrail must run on it.
	Stage Stage
}

// Evaluate dry-runs a Guardrail on a text, without saving an event, and returns whether it
// triggered, its disposition, and the text after its action. The error matches [ErrBadRequest] if
// the text is empty or too long, or the Guardrail does not run on the stage.
func (s *GuardrailsService) Evaluate(ctx context.Context, id ID, text string, params *GuardrailEvaluateParams) (*GuardrailEvaluation, error) {
	path, err := s.path(id, "evaluate/")
	if err != nil {
		return nil, err
	}
	stage := StageInput
	if params != nil && params.Stage != "" {
		stage = params.Stage
	}
	body := map[string]string{"text": text, "stage": string(stage)}
	var result GuardrailEvaluation
	if err := s.c.http.do(ctx, &request{method: http.MethodPost, path: path, body: body}, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GuardrailEventsParams are the optional arguments of [GuardrailsService.Events].
type GuardrailEventsParams struct {
	// Reviewed selects only reviewed (true) or unreviewed (false) events, when it isn't nil.
	Reviewed *bool
	// Disposition selects only events of this disposition, e.g. blocked.
	Disposition string
}

// Events returns the latest events of a Guardrail that the user owns, newest first.
func (s *GuardrailsService) Events(ctx context.Context, id ID, params *GuardrailEventsParams) (*GuardrailEvents, error) {
	path, err := s.path(id, "events/")
	if err != nil {
		return nil, err
	}
	q := query{}
	if params != nil {
		q.optionalBool("reviewed", params.Reviewed).str("disposition", params.Disposition)
	}
	var events GuardrailEvents
	if err := s.c.http.do(ctx, &request{method: http.MethodGet, path: path, query: url.Values(q)}, &events); err != nil {
		return nil, err
	}
	return &events, nil
}
