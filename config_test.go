package smarter_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	smarter "github.com/smarter-sh/smarter-golang"
)

const cliConfig = `
config:
  account_number: 1234-5678-9012
  environment: alpha
  root_domain: example.com
local:
  api_key: local-key
alpha:
  api_key: alpha-key
prod:
  api_key: ""
`

func writeCLIConfig(t *testing.T, text string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	check(t, os.WriteFile(path, []byte(text), 0o600))
	return path
}

func newClient(t *testing.T, env map[string]string, opts ...smarter.Option) (*smarter.Client, error) {
	t.Helper()
	return smarter.NewClient(append([]smarter.Option{smarter.WithLookupEnv(testEnv(env))}, opts...)...)
}

func TestEnvironmentURL(t *testing.T) {
	equal(t, smarter.EnvironmentURL(smarter.EnvironmentLocal, "example.com"), "http://localhost:9357")
	equal(t, smarter.EnvironmentURL(smarter.EnvironmentProd, "example.com"), "https://example.com")
	equal(t, smarter.EnvironmentURL(smarter.EnvironmentBeta, "example.com"), "https://beta.example.com")
}

func TestAPIBaseURL(t *testing.T) {
	for _, raw := range []string{"https://example.com", "https://example.com/", "https://example.com/api/v1", " https://example.com/api/v1/ "} {
		got, err := smarter.APIBaseURL(raw)
		check(t, err)
		equal(t, got, "https://example.com/api/v1/")
	}
	got, err := smarter.APIBaseURL("http://host:9357/prefix")
	check(t, err)
	equal(t, got, "http://host:9357/prefix/api/v1/")
	for _, raw := range []string{"ftp://example.com", "example.com", ""} {
		var configuration *smarter.ConfigurationError
		if _, err := smarter.APIBaseURL(raw); !errors.As(err, &configuration) {
			t.Errorf("%q: got %v", raw, err)
		}
	}
}

func TestOptionsWin(t *testing.T) {
	client, err := newClient(t,
		map[string]string{"SMARTER_API_KEY": "env-key", "SMARTER_BASE_URL": "https://env.example.com", "SMARTER_TIMEOUT": "9", "SMARTER_MAX_RETRIES": "1"},
		smarter.WithAPIKey("arg-key"), smarter.WithBaseURL("https://arg.example.com"), smarter.WithEnvironment(smarter.EnvironmentBeta),
		smarter.WithTimeout(5*time.Second), smarter.WithMaxRetries(7), smarter.WithHeader("X-Custom", "1"),
	)
	check(t, err)
	equal(t, client.APIKey(), "arg-key")
	equal(t, client.BaseURL(), "https://arg.example.com/api/v1/")
	equal(t, client.Environment(), smarter.EnvironmentBeta)
}

func TestSettings(t *testing.T) {
	client, err := newClient(t, map[string]string{"SMARTER_API_KEY": "env-key", "SMARTER_ENVIRONMENT": "next", "SMARTER_ROOT_DOMAIN": "x.io", "SMARTER_TIMEOUT": "12.5", "SMARTER_LOG": "info"})
	check(t, err)
	equal(t, client.APIKey(), "env-key")
	equal(t, client.BaseURL(), "https://next.x.io/api/v1/")
	client, err = newClient(t, map[string]string{"SMARTER_LOG": "info"}, smarter.WithAPIKey("k"))
	check(t, err)
	equal(t, client.BaseURL(), "https://platform.smarter.sh/api/v1/")
}

func TestDefaults(t *testing.T) {
	client, err := newClient(t, nil, smarter.WithAPIKey("k"))
	check(t, err)
	equal(t, client.Environment(), smarter.EnvironmentProd)
	equal(t, client.BaseURL(), "https://platform.smarter.sh/api/v1/")
	if smarter.DefaultConfigFile() == "" {
		t.Error("no default configuration file")
	}
}

func TestCLIConfigFile(t *testing.T) {
	path := writeCLIConfig(t, cliConfig)
	env := map[string]string{"SMARTER_CONFIG_FILE": path}
	client, err := newClient(t, env)
	check(t, err)
	equal(t, client.Environment(), smarter.EnvironmentAlpha)
	equal(t, client.APIKey(), "alpha-key")
	equal(t, client.BaseURL(), "https://alpha.example.com/api/v1/")
	local, err := newClient(t, env, smarter.WithEnvironment(smarter.EnvironmentLocal))
	check(t, err)
	equal(t, local.APIKey(), "local-key")
	equal(t, local.BaseURL(), "http://localhost:9357/api/v1/")
}

func TestCLIConfigTopLevelKey(t *testing.T) {
	path := writeCLIConfig(t, cliConfig+"api_key: top-key\n")
	client, err := newClient(t, map[string]string{"SMARTER_CONFIG_FILE": path}, smarter.WithEnvironment(smarter.EnvironmentProd))
	check(t, err)
	equal(t, client.APIKey(), "top-key")
}

func TestNoAPIKey(t *testing.T) {
	_, err := newClient(t, nil)
	var configuration *smarter.ConfigurationError
	if !errors.As(err, &configuration) || !strings.Contains(err.Error(), "SMARTER_API_KEY") || !errors.Is(err, smarter.ErrSmarter) {
		t.Fatalf("got %v", err)
	}
}

func TestInvalidConfiguration(t *testing.T) {
	cases := []struct {
		env  map[string]string
		opts []smarter.Option
	}{
		{map[string]string{"SMARTER_ENVIRONMENT": "mars"}, nil},
		{map[string]string{"SMARTER_TIMEOUT": "soon"}, nil},
		{map[string]string{"SMARTER_TIMEOUT": "-1"}, nil},
		{map[string]string{"SMARTER_MAX_RETRIES": "1.5"}, nil},
		{map[string]string{"SMARTER_LOG": "loud"}, nil},
		{nil, []smarter.Option{smarter.WithEnvironment("mars")}},
		{nil, []smarter.Option{smarter.WithTimeout(0)}},
		{nil, []smarter.Option{smarter.WithMaxRetries(-1)}},
		{nil, []smarter.Option{smarter.WithBaseURL("ftp://example.com")}},
	}
	for _, c := range cases {
		var configuration *smarter.ConfigurationError
		if _, err := newClient(t, c.env, append([]smarter.Option{smarter.WithAPIKey("k")}, c.opts...)...); !errors.As(err, &configuration) {
			t.Errorf("%v: got %v", c.env, err)
		}
	}
}

func TestCLIConfigUnreadable(t *testing.T) {
	// each file has an api key, which would be used if the file were read.
	for _, text := range []string{"api_key: k\n{not: [valid", "- a list\n", "api_key: k\nconfig:\n  environment: mars\n"} {
		_, err := newClient(t, map[string]string{"SMARTER_CONFIG_FILE": writeCLIConfig(t, text)})
		var configuration *smarter.ConfigurationError
		if !errors.As(err, &configuration) {
			t.Errorf("%q: the file was not ignored: %v", text, err)
		}
	}
}
