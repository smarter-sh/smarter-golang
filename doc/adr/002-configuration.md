# ADR-002: Configuration

## Status

Accepted

Follows the Python SDK's [ADR-002](https://github.com/smarter-sh/smarter-python/blob/main/doc/adr/002-configuration.md), which adapts the platform's [ADR-004: 12-Factor App](https://github.com/smarter-sh/smarter/blob/main/docs/source/adr/004-12-factor.rst) and [ADR-007: Settings](https://github.com/smarter-sh/smarter/blob/main/docs/source/adr/007-settings.rst).

## Context

A client needs an api key and the url of a platform. Programmers configure libraries in code, operators configure them with environment variables (12-factor), and Smarter developers already have both in the smarter cli's `~/.smarter/config.yaml`, which the cli reads with viper.

## Decision

The lookup is exactly the Python SDK's, and its rules are the cli's (`cmd/root.go` and `cmd/api.go` in smarter-cli). One file, `config.go`, is the only code that reads the environment. `NewClient(opts ...Option)` takes each value from the first of:

1. the options (`smarter.WithAPIKey(...)`, `WithBaseURL`, `WithEnvironment`, `WithTimeout`, `WithMaxRetries`),
2. the environment: `SMARTER_API_KEY`, `SMARTER_BASE_URL`, `SMARTER_ENVIRONMENT`, `SMARTER_ROOT_DOMAIN`, `SMARTER_TIMEOUT` (seconds), `SMARTER_MAX_RETRIES`, `SMARTER_LOG` and `SMARTER_CONFIG_FILE`,
3. the cli's configuration file: `config.environment`, `config.root_domain`, the environment's `<environment>.api_key`, then the top-level `api_key`,
4. the defaults: `prod` on `platform.smarter.sh`.

Without a base url, the url is derived from the environment as the cli derives it (`local` is `http://localhost:9357`, `alpha`, `beta` and `next` are `https://<environment>.<root domain>`, `prod` is `https://<root domain>`). Every value is validated by `NewClient`, which returns a `*ConfigurationError` instead of panicking, unlike the cli's `getAPIHost`.

The SDK never reads a `.env` file itself.

## Alternatives Considered

- viper, as the cli does. A large dependency (ADR-011) for reading one YAML file, and global state that a library must not touch.
- Options only.

## Consequences

- **Positive:**
  - Developers who use the cli need no setup, and the cli can pass its own flags as options.
  - Deployments configure the SDK with environment variables alone.
- **Negative:**
  - The lookup order must be documented and tested, because a value can come from four places.

## Related ADRs

- [ADR-000: Introduction](000-intro.md)
- [ADR-001: Scope and Purpose of the SDK](001-scope.md)
- [ADR-016: Security](016-security.md)
- [ADR-020: One Package, Functional Options](020-package-layout.md)
