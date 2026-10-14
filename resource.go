package smarter

import (
	"context"
	"net/http"
)

// named is a REST resource that has a name: every struct that embeds [Resource].
type named interface {
	ResourceName() string
}

// KindService holds the operations of one resource kind, e.g. Guardrail: List, Retrieve and Find,
// from the REST api, and Apply, Describe, Delete, ExampleManifest and JSONSchema, from the manifest
// api (ADR-003). Each resource family's service embeds it.
type KindService[T named] struct {
	c *Client
	// Kind is the manifest kind, e.g. Guardrail.
	Kind string
	// Path is the REST list endpoint, relative to the api's base url, e.g. guardrails/.
	Path string
}

func newKindService[T named](c *Client, kind, path string) *KindService[T] {
	return &KindService[T]{c: c, Kind: kind, Path: path}
}

// path returns a resource's path, e.g. guardrails/17/evaluate/.
func (r *KindService[T]) path(id ID, suffix string) (string, error) {
	segment, err := id.segment()
	if err != nil {
		return "", err
	}
	return r.Path + segment + "/" + suffix, nil
}

// List returns every resource of this kind that the api key's user may read.
func (r *KindService[T]) List(ctx context.Context) ([]T, error) {
	var items []T
	if err := r.c.http.do(ctx, &request{method: http.MethodGet, path: r.Path}, &items); err != nil {
		return nil, err
	}
	return items, nil
}

// Retrieve returns one resource, by its id or hashed id. The error matches [ErrNotFound] if it
// doesn't exist, or the user may not read it.
func (r *KindService[T]) Retrieve(ctx context.Context, id ID) (*T, error) {
	path, err := r.path(id, "")
	if err != nil {
		return nil, err
	}
	var item T
	if err := r.c.http.do(ctx, &request{method: http.MethodGet, path: path}, &item); err != nil {
		return nil, err
	}
	return &item, nil
}

// Find returns the first resource of a name that the user may read. The error matches
// [ErrResourceNotFound] if there is none.
func (r *KindService[T]) Find(ctx context.Context, name string) (*T, error) {
	items, err := r.List(ctx)
	if err != nil {
		return nil, err
	}
	for i := range items {
		if items[i].ResourceName() == name {
			return &items[i], nil
		}
	}
	return nil, &resourceNotFound{kind: r.Kind, name: name}
}

// Apply creates or updates a resource of this kind from its manifest. It returns a [*ValueError] if
// the manifest is of another kind.
func (r *KindService[T]) Apply(ctx context.Context, manifest *Manifest) (*JournalMessage, error) {
	if manifest.Kind != r.Kind {
		return nil, &ValueError{Message: "expected a " + r.Kind + " manifest, not " + manifest.Kind}
	}
	return r.c.Manifests.Apply(ctx, manifest)
}

// Describe returns a resource's manifest, with its read-only status.
func (r *KindService[T]) Describe(ctx context.Context, name string) (*Manifest, error) {
	return r.c.Manifests.Describe(ctx, r.Kind, name)
}

// Delete permanently deletes a resource.
func (r *KindService[T]) Delete(ctx context.Context, name string) (*JournalMessage, error) {
	return r.c.Manifests.Delete(ctx, r.Kind, name)
}

// ExampleManifest returns a valid example manifest of this kind, to start from.
func (r *KindService[T]) ExampleManifest(ctx context.Context) (*Manifest, error) {
	return r.c.Manifests.ExampleManifest(ctx, r.Kind)
}

// JSONSchema returns the JSON Schema of this kind's manifest.
func (r *KindService[T]) JSONSchema(ctx context.Context) (JSONObject, error) {
	return r.c.Manifests.JSONSchema(ctx, r.Kind)
}

// ReadService lists and retrieves the rows of a read-only REST endpoint, e.g. an account's users.
type ReadService[T any] struct {
	c    *Client
	path string
}

// List returns the rows that the api key's user may read.
func (r *ReadService[T]) List(ctx context.Context) ([]T, error) {
	var items []T
	if err := r.c.http.do(ctx, &request{method: http.MethodGet, path: r.path}, &items); err != nil {
		return nil, err
	}
	return items, nil
}

// Retrieve returns a row, by id.
func (r *ReadService[T]) Retrieve(ctx context.Context, id ID) (*T, error) {
	segment, err := id.segment()
	if err != nil {
		return nil, err
	}
	var item T
	if err := r.c.http.do(ctx, &request{method: http.MethodGet, path: r.path + segment + "/"}, &item); err != nil {
		return nil, err
	}
	return &item, nil
}
