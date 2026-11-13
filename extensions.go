package smarter

import (
	"context"
	"io"
	"net/http"
	"net/url"
)

// ExtensionsService holds the operations of the platform's Extensions: /api/v1/extensions/. An
// extension is a package of tools, settings and resources that a superuser installs from the
// Extension Registry, or uploads as a .zip archive, and that LLMClients may then use.
//
// Every user may read the extensions: a user who is not a superuser sees only the extensions that
// their account may use, without their settings. Only superusers may install, enable, disable,
// update, roll back, configure, make available or uninstall them: the platform refuses other users
// with a 403, which matches [ErrPermissionDenied]. An unknown slug is a 404, which matches
// [ErrNotFound], and an invalid request a 400, which matches [ErrBadRequest].
//
// An LLMClient's own extensions are read with [LLMClientsService.Extensions].
type ExtensionsService struct {
	c *Client
}

const extensionsPath = "extensions/"

// extensionPath returns an extension's path, e.g. extensions/weather/enable/.
func extensionPath(slug, suffix string) (string, error) {
	segment, err := ID(slug).segment()
	if err != nil {
		return "", err
	}
	return extensionsPath + url.PathEscape(segment) + "/" + suffix, nil
}

// extensionAction sends one of the superuser's actions on an extension, and returns its result.
func (s *ExtensionsService) extensionAction(ctx context.Context, slug, suffix string, body JSONValue) (*ExtensionAction, error) {
	path, err := extensionPath(slug, suffix)
	if err != nil {
		return nil, err
	}
	return s.action(ctx, &request{method: http.MethodPost, path: path, body: body})
}

func (s *ExtensionsService) action(ctx context.Context, r *request) (*ExtensionAction, error) {
	var result ExtensionAction
	if err := s.c.http.do(ctx, r, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// List returns the installed extensions that the user may see, whether the Extension Registry is
// configured, and how many extensions have an update in it.
func (s *ExtensionsService) List(ctx context.Context) (*ExtensionList, error) {
	var result ExtensionList
	if err := s.c.http.do(ctx, &request{method: http.MethodGet, path: extensionsPath}, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Registry returns the extensions of the Extension Registry, each with the action that it allows:
// install, update, installed, uploaded or incompatible. The registry's index is cached; refresh reads
// it from the registry again, which only a superuser may do (the platform ignores it for other
// users). When the registry cannot be read, the result's Error says why.
func (s *ExtensionsService) Registry(ctx context.Context, refresh bool) (*ExtensionRegistry, error) {
	q := query{}.flag("refresh", refresh)
	var result ExtensionRegistry
	if err := s.c.http.do(ctx, &request{method: http.MethodGet, path: extensionsPath + "registry/", query: url.Values(q)}, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Retrieve returns an installed extension, by its slug, e.g. weather: its tools, settings, versions,
// resources, switches and availability. The error matches [ErrNotFound] if no extension of that slug
// is installed, or the user may not see it.
func (s *ExtensionsService) Retrieve(ctx context.Context, slug string) (*ExtensionDetail, error) {
	path, err := extensionPath(slug, "")
	if err != nil {
		return nil, err
	}
	var result struct {
		Extension ExtensionDetail `json:"extension"`
	}
	if err := s.c.http.do(ctx, &request{method: http.MethodGet, path: path}, &result); err != nil {
		return nil, err
	}
	return &result.Extension, nil
}

// CheckUpdates reads the Extension Registry's index again, and records the newest compatible version
// of each installed extension. Its message names the updates that are available. Superusers only.
func (s *ExtensionsService) CheckUpdates(ctx context.Context) (*ExtensionAction, error) {
	return s.action(ctx, &request{method: http.MethodPost, path: extensionsPath + "check-updates/"})
}

// Install installs, or updates, an extension from the Extension Registry, by its slug, at a version,
// or at the newest version that is compatible with the platform when version is empty. A newly
// installed extension is disabled: configure its settings, then enable it. Superusers only.
func (s *ExtensionsService) Install(ctx context.Context, slug, version string) (*ExtensionAction, error) {
	body := JSONObject{"slug": slug}
	if version != "" {
		body["version"] = version
	}
	return s.action(ctx, &request{method: http.MethodPost, path: extensionsPath + "install/", body: body})
}

// Upload installs, or updates, an extension from its .zip archive, of at most 10 MiB, read from
// archive and sent as the file filename (extension.zip if empty). Configure its settings, then enable
// it. Superusers only. It returns an error that matches [ErrBadRequest] if the archive is too large
// or is not a valid extension.
func (s *ExtensionsService) Upload(ctx context.Context, archive io.Reader, filename string) (*ExtensionAction, error) {
	content, err := io.ReadAll(archive)
	if err != nil {
		return nil, &ValueError{Message: "the archive can't be read: " + err.Error(), Err: err}
	}
	if filename == "" {
		filename = "extension.zip"
	}
	form := []formFile{{field: "archive", name: filename, content: content}}
	return s.action(ctx, &request{method: http.MethodPost, path: extensionsPath + "upload/", form: form})
}

// Enable enables an installed extension, so that LLMClients may use its tools, after checking that it
// is compatible, that its code loads, and that it has its required settings. The error matches
// [ErrBadRequest] if it does not. Superusers only.
func (s *ExtensionsService) Enable(ctx context.Context, slug string) (*ExtensionAction, error) {
	return s.extensionAction(ctx, slug, "enable/", nil)
}

// Disable disables an installed extension: LLMClients stop offering its tools. Superusers only.
func (s *ExtensionsService) Disable(ctx context.Context, slug string) (*ExtensionAction, error) {
	return s.extensionAction(ctx, slug, "disable/", nil)
}

// Update updates an installed extension to the newest compatible version in the Extension Registry.
// Superusers only.
func (s *ExtensionsService) Update(ctx context.Context, slug string) (*ExtensionAction, error) {
	return s.extensionAction(ctx, slug, "update/", nil)
}

// Rollback makes one of an extension's installed versions the one that runs, by the ID of an
// [ExtensionVersion] of its [ExtensionDetail]. The error matches [ErrBadRequest] if the version is
// not one of the extension's. Superusers only.
func (s *ExtensionsService) Rollback(ctx context.Context, slug string, versionID int) (*ExtensionAction, error) {
	return s.extensionAction(ctx, slug, "rollback/", JSONObject{"versionId": versionID})
}

// ExtensionSecretInput is the value of an extension's secret setting: one of the platform admin's
// Secrets, by its id, or a value, which the platform saves in the Secret
// extension_<slug>_<setting>.
type ExtensionSecretInput struct {
	SecretID *int    `json:"secretId,omitempty"`
	Value    *string `json:"value,omitempty"`
}

// ExtensionConfigureParams are the settings that [ExtensionsService.Configure] changes. Settings that
// are left out don't change.
type ExtensionConfigureParams struct {
	// Values are non-secret settings' values, by name: a string, number or boolean, or nil to reset a
	// setting to its default.
	Values map[string]JSONValue `json:"values,omitempty"`
	// Secrets are secret settings' values, by name.
	Secrets map[string]ExtensionSecretInput `json:"secrets,omitempty"`
}

// Configure changes some of an extension's settings. The error matches [ErrBadRequest] if a setting
// is unknown or its value is invalid. Superusers only.
func (s *ExtensionsService) Configure(ctx context.Context, slug string, params *ExtensionConfigureParams) (*ExtensionAction, error) {
	if params == nil {
		params = &ExtensionConfigureParams{}
	}
	return s.extensionAction(ctx, slug, "configure/", params)
}

// SetAvailability sets which accounts may use an extension: every account when allAccounts is true,
// or else only the accounts of accountIDs. Superusers only.
func (s *ExtensionsService) SetAvailability(ctx context.Context, slug string, allAccounts bool, accountIDs []int) (*ExtensionAction, error) {
	if accountIDs == nil {
		accountIDs = []int{}
	}
	return s.extensionAction(ctx, slug, "availability/", JSONObject{"allAccounts": allAccounts, "accountIds": accountIDs})
}

// Uninstall uninstalls an extension: its resources, switches, settings and versions, and LLMClients'
// links to it. The result's Extension is nil. The error matches [ErrBadRequest] if the extension's
// own pre-uninstall hook refuses. Superusers only.
func (s *ExtensionsService) Uninstall(ctx context.Context, slug string) (*ExtensionAction, error) {
	return s.extensionAction(ctx, slug, "uninstall/", nil)
}
