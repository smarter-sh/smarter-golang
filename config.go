package smarter

import (
	"fmt"
	"log/slog"
	"math"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"
)

// Environment is an environment of the smarter cli's configuration file.
type Environment string

// The environments of the smarter cli's configuration file.
const (
	EnvironmentLocal Environment = "local"
	EnvironmentAlpha Environment = "alpha"
	EnvironmentBeta  Environment = "beta"
	EnvironmentNext  Environment = "next"
	EnvironmentProd  Environment = "prod"
)

// Defaults of the configuration.
const (
	// DefaultEnvironment is the environment when none is configured.
	DefaultEnvironment = EnvironmentProd
	// DefaultRootDomain is the platform's root domain when none is configured.
	DefaultRootDomain = "platform.smarter.sh"
	// LocalURL is the url of the Smarter platform's local development server.
	LocalURL = "http://localhost:9357"
	// APIPath is the path of the Smarter api, which every request url is relative to.
	APIPath = "/api/v1/"
	// DefaultTimeout is the timeout of a request. Prompts that call several tools can take tens of seconds.
	DefaultTimeout = 60 * time.Second
	// DefaultMaxRetries is how many times a failed request is retried.
	DefaultMaxRetries = 2
)

func validEnvironment(value string) bool {
	switch Environment(value) {
	case EnvironmentLocal, EnvironmentAlpha, EnvironmentBeta, EnvironmentNext, EnvironmentProd:
		return true
	}
	return false
}

// DefaultConfigFile returns the path of the smarter cli's configuration file, ~/.smarter/config.yaml.
func DefaultConfigFile() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".smarter", "config.yaml")
	}
	return filepath.Join(home, ".smarter", "config.yaml")
}

// settings are the SDK's SMARTER_* environment variables.
type settings struct {
	apiKey, baseURL, rootDomain, configFile, log string
	environment                                  Environment
	timeout                                      time.Duration
	maxRetries                                   int
}

// readSettings reads the SMARTER_* environment variables, with lookup (os.LookupEnv by default).
func readSettings(lookup func(string) (string, bool)) (settings, error) {
	get := func(name string) string {
		value, _ := lookup(name)
		return strings.TrimSpace(value)
	}
	s := settings{
		apiKey:     get("SMARTER_API_KEY"),
		baseURL:    get("SMARTER_BASE_URL"),
		rootDomain: get("SMARTER_ROOT_DOMAIN"),
		configFile: get("SMARTER_CONFIG_FILE"),
		log:        get("SMARTER_LOG"),
		timeout:    DefaultTimeout,
		maxRetries: DefaultMaxRetries,
	}
	if s.configFile == "" {
		s.configFile = DefaultConfigFile()
	}
	if environment := get("SMARTER_ENVIRONMENT"); environment != "" {
		if !validEnvironment(environment) {
			return s, &ConfigurationError{Message: fmt.Sprintf("SMARTER_ENVIRONMENT must be one of local, alpha, beta, next or prod, not %q", environment)}
		}
		s.environment = Environment(environment)
	}
	if s.log != "" && s.log != "debug" && s.log != "info" {
		return s, &ConfigurationError{Message: fmt.Sprintf("SMARTER_LOG must be debug or info, not %q", s.log)}
	}
	if timeout := get("SMARTER_TIMEOUT"); timeout != "" {
		seconds, err := strconv.ParseFloat(timeout, 64)
		if err != nil || seconds <= 0 || math.IsInf(seconds, 0) || math.IsNaN(seconds) {
			return s, &ConfigurationError{Message: fmt.Sprintf("SMARTER_TIMEOUT must be a number of seconds, more than 0, not %q", timeout)}
		}
		s.timeout = time.Duration(seconds * float64(time.Second))
	}
	if retries := get("SMARTER_MAX_RETRIES"); retries != "" {
		n, err := strconv.Atoi(retries)
		if err != nil || n < 0 {
			return s, &ConfigurationError{Message: fmt.Sprintf("SMARTER_MAX_RETRIES must be a whole number, 0 or more, not %q", retries)}
		}
		s.maxRetries = n
	}
	return s, nil
}

// cliConfig is the part of the smarter cli's ~/.smarter/config.yaml that the SDK reads.
type cliConfig struct {
	environment Environment
	rootDomain  string
	apiKey      string            // the top-level api_key, used by any environment
	apiKeys     map[string]string // <environment>.api_key
}

// loadCLIConfig reads the cli's configuration file. A missing, unreadable or invalid file is an empty
// configuration: the file is optional.
func loadCLIConfig(path string, logger *slog.Logger) cliConfig {
	empty := cliConfig{apiKeys: map[string]string{}}
	raw, err := os.ReadFile(path) //nolint:gosec // G304: the path is the caller's configuration
	if err != nil {
		logger.Debug("smarter cli configuration not read", "source", "config", "path", path, "error", err)
		return empty
	}
	var document map[string]any
	if err := yaml.Unmarshal(raw, &document); err != nil {
		logger.Debug("smarter cli configuration not read", "source", "config", "path", path, "error", err)
		return empty
	}
	result := cliConfig{apiKeys: map[string]string{}}
	for name, section := range document {
		if fields, ok := section.(map[string]any); ok {
			if key, ok := fields["api_key"].(string); ok && key != "" {
				result.apiKeys[name] = key
			}
		}
	}
	if key, ok := document["api_key"].(string); ok {
		result.apiKey = key
	}
	if config, ok := document["config"].(map[string]any); ok {
		if environment, ok := config["environment"].(string); ok && environment != "" {
			if !validEnvironment(environment) {
				logger.Warn("smarter cli configuration is invalid and was ignored", "source", "config", "path", path, "environment", environment)
				return empty
			}
			result.environment = Environment(environment)
		}
		if rootDomain, ok := config["root_domain"].(string); ok {
			result.rootDomain = rootDomain
		}
	}
	return result
}

// EnvironmentURL returns the platform url of an environment, as the smarter cli derives it: local is
// http://localhost:9357, alpha, beta and next are https://<environment>.<root domain>, and prod is
// https://<root domain>.
func EnvironmentURL(environment Environment, rootDomain string) string {
	switch environment {
	case EnvironmentLocal:
		return LocalURL
	case EnvironmentProd:
		return "https://" + rootDomain
	default:
		return fmt.Sprintf("https://%s.%s", environment, rootDomain)
	}
}

// APIBaseURL returns the api's base url for a platform url: https://example.com and
// https://example.com/api/v1 both become https://example.com/api/v1/. It returns a
// [*ConfigurationError] if the url is not an http or https url.
func APIBaseURL(raw string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return "", &ConfigurationError{Message: fmt.Sprintf("the Smarter base url must be an http or https url, not %q", raw)}
	}
	path := strings.TrimRight(parsed.Path, "/")
	apiPath := strings.TrimRight(APIPath, "/")
	if !strings.HasSuffix(path, apiPath) {
		path += apiPath
	}
	return parsed.Scheme + "://" + parsed.Host + path + "/", nil
}

// config is a client's resolved configuration.
type config struct {
	apiKey      string
	baseURL     string
	environment Environment
	timeout     time.Duration
	maxRetries  int
}

// resolveConfig takes each value from the first of: the options, the environment variables, the
// cli's configuration file, and the defaults. See ADR-002.
func resolveConfig(o *options, s settings, logger *slog.Logger) (config, error) {
	cli := loadCLIConfig(s.configFile, logger)
	environment := o.environment
	if environment != "" && !validEnvironment(string(environment)) {
		return config{}, &ConfigurationError{Message: fmt.Sprintf("the environment must be one of local, alpha, beta, next or prod, not %q", environment)}
	}
	for _, candidate := range []Environment{environment, s.environment, cli.environment, DefaultEnvironment} {
		if candidate != "" {
			environment = candidate
			break
		}
	}
	apiKey := firstNonEmpty(o.apiKey, s.apiKey, cli.apiKeys[string(environment)], cli.apiKey)
	if apiKey == "" {
		return config{}, &ConfigurationError{Message: "no Smarter api key: use WithAPIKey, set SMARTER_API_KEY, or run 'smarter configure'"}
	}
	rawURL := firstNonEmpty(o.baseURL, s.baseURL)
	if rawURL == "" {
		rawURL = EnvironmentURL(environment, firstNonEmpty(s.rootDomain, cli.rootDomain, DefaultRootDomain))
	}
	baseURL, err := APIBaseURL(rawURL)
	if err != nil {
		return config{}, err
	}
	timeout := s.timeout
	if o.timeout != nil {
		timeout = *o.timeout
	}
	if timeout <= 0 {
		return config{}, &ConfigurationError{Message: fmt.Sprintf("the timeout must be more than 0, not %v", timeout)}
	}
	maxRetries := s.maxRetries
	if o.maxRetries != nil {
		maxRetries = *o.maxRetries
	}
	if maxRetries < 0 {
		return config{}, &ConfigurationError{Message: fmt.Sprintf("max retries must be 0 or more, not %d", maxRetries)}
	}
	return config{apiKey: apiKey, baseURL: baseURL, environment: environment, timeout: timeout, maxRetries: maxRetries}, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}
