package smarter

import (
	"context"
	"net/http"
	"net/url"
)

// AccountsService holds the operations of Accounts: /api/v1/accounts/, and the manifest api's
// Account kind, with the account's users, contacts and user profiles. Only superusers may read
// them: the platform refuses other users with a 403. Users and accounts are created and changed with
// manifests (the User and Account kinds).
//
// Every user may read their own token usage (Usage) and export their own data (DataExports).
type AccountsService struct {
	*KindService[Account]
	// Users are the users of the api key's account: /api/v1/accounts/users/.
	Users *ReadService[User]
	// Contacts are the contacts of the api key's account: /api/v1/accounts/contacts/.
	Contacts *ReadService[AccountContact]
	// UserProfiles are each user's membership of an account: /api/v1/accounts/user-profiles/.
	UserProfiles *ReadService[UserProfile]
	// DataExports are the api key's user's exports of their own data: /api/v1/accounts/data-exports/.
	DataExports *DataExportsService
}

// UsagePeriodicity is the window of time of [AccountsService.Usage].
type UsagePeriodicity string

// The windows of [AccountsService.Usage], and the period of each of their points.
const (
	// UsagePeriodicity12Hours is the last 12 hours, by hour.
	UsagePeriodicity12Hours UsagePeriodicity = "12_hours"
	// UsagePeriodicity24Hours is the last 24 hours, by hour: the default.
	UsagePeriodicity24Hours UsagePeriodicity = "24_hours"
	// UsagePeriodicity7Days is the last 7 days, by day.
	UsagePeriodicity7Days UsagePeriodicity = "7_days"
	// UsagePeriodicity1Month is the last 30 days, by day.
	UsagePeriodicity1Month UsagePeriodicity = "1_month"
	// UsagePeriodicity1Year is the last year, by month.
	UsagePeriodicity1Year UsagePeriodicity = "1_year"
)

// Usage returns the tokens that the api key's user's LLM requests used, and what they cost, over a
// window of time: one point per period, oldest first. An empty periodicity is the platform's
// default, [UsagePeriodicity24Hours]. The platform caches the series for five minutes. The error
// matches [ErrBadRequest] if the periodicity is unknown.
func (s *AccountsService) Usage(ctx context.Context, periodicity UsagePeriodicity) (*TokenUsageSeries, error) {
	q := query{}.str("periodicity", string(periodicity))
	var series TokenUsageSeries
	if err := s.c.http.do(ctx, &request{method: http.MethodGet, path: "accounts/usage/", query: url.Values(q)}, &series); err != nil {
		return nil, err
	}
	return &series, nil
}

// DataExportsService holds the api key's user's exports of their own data, the web console's
// Download My Data: a zip file of the manifests of every resource that the user owns, built in the
// background. A link to it is emailed to the user, and never returned by the api.
type DataExportsService struct {
	c *Client
}

const dataExportsPath = "accounts/data-exports/"

// Overview returns the overview of the user's data exports: the address that the link is emailed to,
// how long the link works, how many exports the user may still start today, and their latest export.
func (s *DataExportsService) Overview(ctx context.Context) (*DataExportOverview, error) {
	var overview DataExportOverview
	if err := s.c.http.do(ctx, &request{method: http.MethodGet, path: dataExportsPath}, &overview); err != nil {
		return nil, err
	}
	return &overview, nil
}

// Start starts an export of the user's data, unless one is already in progress, and returns the
// export: the new one, with Started true, or the one in progress, with Started false. Follow its
// progress with Retrieve. When it succeeds, the platform emails the user a link to its zip file.
//
// It returns an error that matches [ErrRateLimit] when the platform refuses to start one: the user has
// no email address, or has started as many exports today as the platform allows (see
// [DataExportOverview]). The error's message says which. That refusal is the platform's answer, not
// a transient failure, so it is not retried.
func (s *DataExportsService) Start(ctx context.Context) (*DataExport, error) {
	var export DataExport
	var status int
	r := &request{method: http.MethodPost, path: dataExportsPath, final: []int{http.StatusTooManyRequests}, status: &status}
	if err := s.c.http.do(ctx, r, &export); err != nil {
		return nil, err
	}
	export.Started = status == http.StatusAccepted
	return &export, nil
}

// Retrieve returns one of the user's data exports, and its progress, by its id. The error matches
// [ErrNotFound] if the user has no export of that id.
func (s *DataExportsService) Retrieve(ctx context.Context, exportID int) (*DataExport, error) {
	var export DataExport
	path := dataExportsPath + IntID(exportID).String() + "/"
	if err := s.c.http.do(ctx, &request{method: http.MethodGet, path: path}, &export); err != nil {
		return nil, err
	}
	return &export, nil
}
