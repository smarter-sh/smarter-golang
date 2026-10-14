package smarter

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
)

// VectorstoresService holds the operations of Vectorstores: /api/v1/vectorstores/, and the manifest
// api's Vectorstore kind. The platform serves these endpoints only when SMARTER_ENABLE_VECTORSTORE is
// true.
type VectorstoresService struct {
	*KindService[Vectorstore]
}

func vectorstoreDo[T any](ctx context.Context, s *VectorstoresService, method string, id ID, suffix string, body JSONValue) (*T, error) {
	path, err := s.path(id, suffix)
	if err != nil {
		return nil, err
	}
	var result T
	if err := s.c.http.do(ctx, &request{method: method, path: path, body: body}, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Status returns a vectorstore's state, checked now: its status and vector count.
func (s *VectorstoresService) Status(ctx context.Context, id ID) (*VectorstoreStatus, error) {
	return vectorstoreDo[VectorstoreStatus](ctx, s, http.MethodGet, id, "status/", nil)
}

// Deploy deploys a vectorstore: it creates its database, or connects to the provider's. A
// self-hosted vectorstore creates real, possibly billable, Kubernetes resources.
func (s *VectorstoresService) Deploy(ctx context.Context, id ID) (*Vectorstore, error) {
	return vectorstoreDo[Vectorstore](ctx, s, http.MethodPost, id, "deploy/", nil)
}

// Undeploy stops serving a vectorstore. A self-hosted server's data is kept.
func (s *VectorstoresService) Undeploy(ctx context.Context, id ID) (*Vectorstore, error) {
	return vectorstoreDo[Vectorstore](ctx, s, http.MethodPost, id, "undeploy/", nil)
}

// Documents returns a vectorstore's documents, newest first.
func (s *VectorstoresService) Documents(ctx context.Context, id ID) (*VectorstoreDocuments, error) {
	return vectorstoreDo[VectorstoreDocuments](ctx, s, http.MethodGet, id, "documents/", nil)
}

// AddTextParams are the optional arguments of [VectorstoresService.AddText].
type AddTextParams struct {
	// Name is the document's name.
	Name string
	// Metadata is a JSON object, stored with each of the document's chunks.
	Metadata JSONObject
}

// AddText adds a text as a document. It is loaded asynchronously, once the vectorstore is deployed.
func (s *VectorstoresService) AddText(ctx context.Context, id ID, text string, params *AddTextParams) (*VectorstoreDocumentsAdded, error) {
	body := JSONObject{"text": text}
	if params != nil {
		if params.Name != "" {
			body["name"] = params.Name
		}
		if params.Metadata != nil {
			body["metadata"] = params.Metadata
		}
	}
	return vectorstoreDo[VectorstoreDocumentsAdded](ctx, s, http.MethodPost, id, "documents/", body)
}

// AddURLParams are the optional arguments of [VectorstoresService.AddURL].
type AddURLParams struct {
	// Metadata is a JSON object, stored with each of the document's chunks.
	Metadata JSONObject
}

// AddURL adds a public web page or file, by its url, as a document. The platform fetches it.
func (s *VectorstoresService) AddURL(ctx context.Context, id ID, url string, params *AddURLParams) (*VectorstoreDocumentsAdded, error) {
	body := JSONObject{"url": url}
	if params != nil && params.Metadata != nil {
		body["metadata"] = params.Metadata
	}
	return vectorstoreDo[VectorstoreDocumentsAdded](ctx, s, http.MethodPost, id, "documents/", body)
}

// AddFiles uploads files, by their paths, as documents, e.g. PDF, Word, HTML or text files. The
// platform extracts their text.
func (s *VectorstoresService) AddFiles(ctx context.Context, id ID, paths []string) (*VectorstoreDocumentsAdded, error) {
	path, err := s.path(id, "documents/")
	if err != nil {
		return nil, err
	}
	files := make([]formFile, 0, len(paths))
	for _, file := range paths {
		content, err := os.ReadFile(file) //nolint:gosec // G304: the caller names the files to upload
		if err != nil {
			return nil, &ValueError{Message: "the file can't be read: " + err.Error(), Err: err}
		}
		files = append(files, formFile{field: "files", name: filepath.Base(file), content: content})
	}
	var result VectorstoreDocumentsAdded
	if err := s.c.http.do(ctx, &request{method: http.MethodPost, path: path, form: files}, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// DeleteDocument removes a document and its chunks, asynchronously.
func (s *VectorstoresService) DeleteDocument(ctx context.Context, id ID, documentID int) (*VectorstoreDocumentDeleted, error) {
	return vectorstoreDo[VectorstoreDocumentDeleted](ctx, s, http.MethodDelete, id, "documents/"+strconv.Itoa(documentID)+"/", nil)
}

// SearchParams are the optional arguments of [VectorstoresService.Search].
type SearchParams struct {
	// K is how many chunks to return, 4 by default. The platform caps it.
	K int
	// SearchType is similarity (the default), mmr (maximal marginal relevance), or
	// similarity_score_threshold.
	SearchType string
	// ScoreThreshold is the minimum score, for similarity_score_threshold.
	ScoreThreshold *float64
	// FetchK is how many chunks mmr considers.
	FetchK *int
	// LambdaMult is mmr's balance of relevance (1) and diversity (0).
	LambdaMult *float64
	// Filter selects only chunks whose metadata match.
	Filter JSONObject
}

// Search searches a vectorstore, and returns the matching chunks, best first.
func (s *VectorstoresService) Search(ctx context.Context, id ID, text string, params *SearchParams) (*VectorstoreSearchResults, error) {
	if params == nil {
		params = &SearchParams{}
	}
	body := JSONObject{"query": text, "k": 4, "searchType": "similarity"}
	if params.K > 0 {
		body["k"] = params.K
	}
	if params.SearchType != "" {
		body["searchType"] = params.SearchType
	}
	if params.ScoreThreshold != nil {
		body["scoreThreshold"] = *params.ScoreThreshold
	}
	if params.FetchK != nil {
		body["fetchK"] = *params.FetchK
	}
	if params.LambdaMult != nil {
		body["lambdaMult"] = *params.LambdaMult
	}
	if len(params.Filter) > 0 {
		body["filter"] = params.Filter
	}
	return vectorstoreDo[VectorstoreSearchResults](ctx, s, http.MethodPost, id, "search/", body)
}

// Snapshots returns a vectorstore's snapshots.
func (s *VectorstoresService) Snapshots(ctx context.Context, id ID) (*VectorstoreSnapshots, error) {
	return vectorstoreDo[VectorstoreSnapshots](ctx, s, http.MethodGet, id, "snapshots/", nil)
}

// CreateSnapshot takes a snapshot of a vectorstore's data now.
func (s *VectorstoresService) CreateSnapshot(ctx context.Context, id ID) (*VectorstoreSnapshot, error) {
	return vectorstoreDo[VectorstoreSnapshot](ctx, s, http.MethodPost, id, "snapshots/", nil)
}

// Restore replaces a vectorstore's data with one of its snapshots, and returns the restored vector count.
func (s *VectorstoresService) Restore(ctx context.Context, id ID, snapshotID int) (*VectorstoreRestored, error) {
	return vectorstoreDo[VectorstoreRestored](ctx, s, http.MethodPost, id, "snapshots/"+strconv.Itoa(snapshotID)+"/restore/", nil)
}
