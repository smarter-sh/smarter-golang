## [1.0.1](https://github.com/smarter-sh/smarter-golang/compare/v1.0.0...v1.0.1) (2026-11-10)


### Bug Fixes

* **contracts:** sync with smarter v0.44.6 ([0903293](https://github.com/smarter-sh/smarter-golang/commit/0903293fefe3f8a1ff4f5e6c95caf60fd4699116))
* **llmclients:** add ProviderRequest for the provider request api ([23389b8](https://github.com/smarter-sh/smarter-golang/commit/23389b88df3770c9d2b6d1d4e46cbad36ce229a4))

# [1.0.0](https://github.com/smarter-sh/smarter-golang/compare/v0.2.2...v1.0.0) (2026-11-01)


### Bug Fixes

* **contracts:** sync with smarter v0.40.4 ([9d11aaf](https://github.com/smarter-sh/smarter-golang/commit/9d11aafc034b6dcd578fee405086d5e850830ba8))
* **contracts:** sync with smarter v0.40.4, annotations become a map ([5fa8872](https://github.com/smarter-sh/smarter-golang/commit/5fa88723f3cdf9a2b5f712feed4d1e71758bafcb))


### BREAKING CHANGES

* **contracts:** annotations are a smarter.Annotations (map[string]string) on Resource, ManifestMetadata and PromptChatAPIPluginMeta, and a map[string]string on every sam struct, instead of a list of one-key objects. Read them as a map, e.g. manifest.Metadata.Annotations["acme.com/crm-id"]. The legacy list is still accepted as input and converted. Manifests.Get takes a fourth argument, a *GetParams, which may be nil.

Synced with smarter v0.40.4 (36fcd2ef4e2ab6e43cf4c42534e0474effe3106e)

Co-authored-by: Lawrence McDaniel <lpm0073@gmail.com>

## [0.2.2](https://github.com/smarter-sh/smarter-golang/compare/v0.2.1...v0.2.2) (2026-10-22)


### Bug Fixes

* **contracts:** sync with smarter v0.36.6 ([4bfad5e](https://github.com/smarter-sh/smarter-golang/commit/4bfad5efc9a7b25e3b74ebe778557d1fa91b26f0))
* **llmclients:** let Config request the verbose chat configuration ([cbd9626](https://github.com/smarter-sh/smarter-golang/commit/cbd96265e263f34171986eeb74ce1bf20c513e8d))

## [0.2.1](https://github.com/smarter-sh/smarter-golang/compare/v0.2.0...v0.2.1) (2026-10-15)


### Bug Fixes

* **contracts:** sync with smarter v0.27.1, functions become extensions ([ff1b166](https://github.com/smarter-sh/smarter-golang/commit/ff1b166af43ad7ac9aae60f063567d023053b387))

# [0.2.0](https://github.com/smarter-sh/smarter-golang/compare/v0.1.1...v0.2.0) (2026-10-15)


### Features

* **llmclients:** add mcpclients, guardrails and tools to llmclient ([294e33a](https://github.com/smarter-sh/smarter-golang/commit/294e33a31b780c612c9c613ea9bd4de4d3d72654))

## [0.1.1](https://github.com/smarter-sh/smarter-golang/compare/v0.1.0...v0.1.1) (2026-10-15)


### Bug Fixes

* **contracts:** sync with smarter v0.25.1 and regenerate sam models ([34c8962](https://github.com/smarter-sh/smarter-golang/commit/34c89626e73f81344f51597c87c998c7089991b1))

# [0.1.0](https://github.com/smarter-sh/smarter-golang/compare/v0.0.0...v0.1.0) (2026-10-14)


### Features

* add the smarter go sdk, modeled on smarter-python ([e09e8d1](https://github.com/smarter-sh/smarter-golang/commit/e09e8d1a5ffe50b9b6c647e170f6df81e450b029))
