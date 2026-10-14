package smarter

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

// cliPath is the path of the manifest api, relative to the api's base url.
const cliPath = "cli/"

var kindPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// kindSegment returns a kind as a url path segment, or a [*ValueError] if it isn't one identifier.
func kindSegment(kind string) (string, error) {
	text := strings.TrimSpace(kind)
	if !kindPattern.MatchString(text) {
		return "", &ValueError{Message: fmt.Sprintf("invalid manifest kind: %q", kind)}
	}
	return text, nil
}

// ManifestsService is the manifest api: the brokered, journaled /api/v1/cli/ endpoints that the
// smarter cli uses. Every resource kind is managed through the same verbs: apply, validate, get,
// describe, delete, deploy, undeploy and logs. The kind's broker on the platform validates the
// manifest with its Pydantic models and applies its business rules, so this is the only way that
// the SDK writes resources (ADR-003).
type ManifestsService struct {
	c *Client
}

// Apply creates or updates a resource from its manifest. The platform compares the manifest with the
// resource's current state and makes the changes that the difference requires. The error matches
// [ErrBadRequest] if the platform rejects the manifest.
func (s *ManifestsService) Apply(ctx context.Context, manifest *Manifest) (*JournalMessage, error) {
	body, err := manifest.ToApply()
	if err != nil {
		return nil, &ValueError{Message: "the manifest can't be encoded: " + err.Error(), Err: err}
	}
	var response JournalMessage
	if err := s.c.http.do(ctx, &request{method: http.MethodPost, path: cliPath + "apply/", body: json.RawMessage(body)}, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

// Validate validates a manifest without applying it, and returns whether it is valid, and its errors.
func (s *ManifestsService) Validate(ctx context.Context, manifest *Manifest) (*ManifestValidation, error) {
	body, err := manifest.ToApply()
	if err != nil {
		return nil, &ValueError{Message: "the manifest can't be encoded: " + err.Error(), Err: err}
	}
	var response JournaledResponse[ManifestValidation]
	if err := s.c.http.do(ctx, &request{method: http.MethodPost, path: cliPath + "validate/", body: json.RawMessage(body)}, &response); err != nil {
		return nil, err
	}
	if response.Data == nil {
		return &ManifestValidation{Valid: false}, nil
	}
	return response.Data, nil
}

// command sends a verb of the manifest api for a kind, with the resource's name, if any.
func (s *ManifestsService) command(ctx context.Context, verb, kind, name string, out any) error {
	segment, err := kindSegment(kind)
	if err != nil {
		return err
	}
	q := url.Values(query{}.str("name", name))
	return s.c.http.do(ctx, &request{method: http.MethodPost, path: cliPath + verb + "/" + segment + "/", query: q}, out)
}

// Get returns the resources of one kind, as a table: Data.Items are its rows, and Names() the
// resources' names. With a name, only the resource of that name.
func (s *ManifestsService) Get(ctx context.Context, kind, name string) (*ManifestList, error) {
	var response JournaledResponse[ManifestList]
	if err := s.command(ctx, "get", kind, name, &response); err != nil {
		return nil, err
	}
	result := response.Data
	if result == nil {
		result = &ManifestList{APIVersion: APIVersionV1, Kind: kind}
	}
	if name != "" {
		// the platform's get ignores the name filter for some kinds, so it is applied here too.
		var items []JSONObject
		for _, item := range result.Data.Items {
			if item["name"] == name {
				items = append(items, item)
			}
		}
		result.Data.Items = items
		result.Metadata.Count = len(items)
	}
	return result, nil
}

// Describe returns a resource's manifest, with its read-only status, which [ManifestsService.Apply]
// accepts as is. The error matches [ErrNotFound] if the resource doesn't exist.
func (s *ManifestsService) Describe(ctx context.Context, kind, name string) (*Manifest, error) {
	var response JournaledResponse[Manifest]
	if err := s.command(ctx, "describe", kind, name, &response); err != nil {
		return nil, err
	}
	if response.Data == nil {
		return nil, &ValueError{Message: fmt.Sprintf("the platform returned no manifest for %s %q", kind, name)}
	}
	return response.Data, nil
}

// Delete permanently deletes a resource. The error matches [ErrNotFound] if it doesn't exist.
func (s *ManifestsService) Delete(ctx context.Context, kind, name string) (*JournalMessage, error) {
	var response JournalMessage
	if err := s.command(ctx, "delete", kind, name, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

// Deploy deploys a resource, e.g. an LLMClient on its own host name.
//
// This creates real infrastructure on the platform's cloud, e.g. DNS records, TLS certificates and
// Kubernetes ingresses, which may be billable. See [Client.Kinds] for the kinds that can be deployed.
func (s *ManifestsService) Deploy(ctx context.Context, kind, name string) (*JournalMessage, error) {
	var response JournalMessage
	if err := s.command(ctx, "deploy", kind, name, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

// Undeploy undoes a resource's deployment.
func (s *ManifestsService) Undeploy(ctx context.Context, kind, name string) (*JournalMessage, error) {
	var response JournalMessage
	if err := s.command(ctx, "undeploy", kind, name, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

// Logs returns a resource's logs: the platform's message, and its data for the kinds that have logs.
func (s *ManifestsService) Logs(ctx context.Context, kind, name string) (*JournaledResponse[JSONObject], error) {
	var response JournaledResponse[JSONObject]
	if err := s.command(ctx, "logs", kind, name, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

// ExampleManifest returns a valid example manifest of a kind, to start from.
func (s *ManifestsService) ExampleManifest(ctx context.Context, kind string) (*Manifest, error) {
	var response JournaledResponse[Manifest]
	if err := s.command(ctx, "example-manifest", kind, "", &response); err != nil {
		return nil, err
	}
	if response.Data == nil {
		return nil, &ValueError{Message: "the platform returned no example manifest for " + kind}
	}
	return response.Data, nil
}

// JSONSchema returns the JSON Schema (draft 2020-12) of a kind's manifest, e.g. to validate
// manifests locally or to generate a form.
func (s *ManifestsService) JSONSchema(ctx context.Context, kind string) (JSONObject, error) {
	var response JournaledResponse[JSONObject]
	if err := s.command(ctx, "json-schema", kind, "", &response); err != nil {
		return nil, err
	}
	if response.Data == nil {
		return JSONObject{}, nil
	}
	return *response.Data, nil
}
