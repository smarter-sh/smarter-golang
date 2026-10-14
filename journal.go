package smarter

// The Smarter Journal api contract: the envelope of every journaled response, its error body, and
// a journal entry. These mirror smarter.lib.journal.contract.models in the Smarter platform, whose
// JSON Schema is journal-contract.schema.json, as the Python SDK's smarter.models.journal does. The
// SDK's tests check the platform's schema against them (contracts_test.go).

// SAMJournalAPIMetadata is the metadata of a journaled response: its command and thing, and the key
// of its journal entry.
type SAMJournalAPIMetadata struct {
	// Command is the journal's command, e.g. apply.
	Command string `json:"command"`
	// Thing is the journal's thing, e.g. Guardrail. The string "None" when there is none.
	Thing string `json:"thing"`
	// Key is the key of the response's journal entry, when the journal is enabled.
	Key string `json:"key,omitempty"`
}

// SAMJournalAPIErrorBody is the error of a journaled error response. Every error of the cli api has
// this shape.
type SAMJournalAPIErrorBody struct {
	// ErrorClass is the name of the exception's class, e.g. SAMBrokerErrorNotFound.
	ErrorClass string `json:"errorClass"`
	// Stacktrace is the server's stack trace, when the platform includes it.
	Stacktrace string `json:"stacktrace,omitempty"`
	// Description is a human-readable description of the error.
	Description string `json:"description,omitempty"`
	// Status is the http status.
	Status int `json:"status"`
	// Args are the exception's arguments, or the request's url.
	Args JSONValue `json:"args,omitempty"`
	// Cause is the exception's cause.
	Cause string `json:"cause,omitempty"`
	// Context is the exception's context, or the response's thing and command.
	Context string `json:"context,omitempty"`
	// Thing is the journal's thing, e.g. Guardrail.
	Thing string `json:"thing,omitempty"`
	// Command is the journal's command, e.g. describe.
	Command string `json:"command,omitempty"`
}

// SAMJournalAPIEnvelope holds the fields that every journaled response adds beside its own data.
type SAMJournalAPIEnvelope struct {
	// API is the api version, e.g. "smarter.sh/v1".
	API      string                 `json:"api,omitempty"`
	Metadata *SAMJournalAPIMetadata `json:"metadata,omitempty"`
}

// SAMJournalAPIErrorResponse is the body of a journaled error response.
type SAMJournalAPIErrorResponse struct {
	SAMJournalAPIEnvelope
	Error SAMJournalAPIErrorBody `json:"error"`
}

// JournalMessage is a journaled response that carries only a message, e.g. the response of apply,
// delete or deploy. It is the Python SDK's SmarterJournalMessage.
type JournalMessage struct {
	SAMJournalAPIEnvelope
	// Message is a human-readable message, e.g. "LLMClient calculator applied".
	Message string `json:"message,omitempty"`
}

// JournaledResponse is a successful journaled response whose data is a T. It is the Python SDK's
// SmarterJournaledResponse.
type JournaledResponse[T any] struct {
	SAMJournalAPIEnvelope
	// Data is the response's data, or nil when it has none.
	Data *T `json:"data,omitempty"`
	// Message is a human-readable message.
	Message string `json:"message,omitempty"`
}

// JournalEntry is a Smarter Journal entry, as /api/v1/journal/ returns it.
//
// The list returns entries without Request and Response. Retrieving one entry by its key includes
// them. Only superusers may read the journal, because the request documents hold the callers'
// credentials.
type JournalEntry struct {
	Key       string `json:"key"`
	CreatedAt string `json:"createdAt,omitempty"`
	// User is the username of the caller, or "" for an anonymous request.
	User       string    `json:"user,omitempty"`
	Thing      string    `json:"thing"`
	Command    string    `json:"command"`
	StatusCode int       `json:"statusCode"`
	Request    JSONValue `json:"request,omitempty"`
	Response   JSONValue `json:"response,omitempty"`
}

// JournalPagination describes one page of the journal's entries.
type JournalPagination struct {
	Page       int      `json:"page"`
	PageSize   int      `json:"pageSize"`
	NumPages   int      `json:"numPages"`
	Count      int      `json:"count"`
	Search     string   `json:"search,omitempty"`
	Ordering   string   `json:"ordering,omitempty"`
	SortFields []string `json:"sortFields,omitempty"`
}

// JournalPage is one page of the journal's entries.
type JournalPage struct {
	Objects    []JournalEntry    `json:"objects"`
	Pagination JournalPagination `json:"pagination"`
}
