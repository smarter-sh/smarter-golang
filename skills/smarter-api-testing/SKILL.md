---
name: smarter-api-testing
description: Use when you need to exercise the Smarter platform's REST api by hand from the smarter-golang SDK's side, e.g. to learn an endpoint's real request and response before the SDK covers it, to build a sanitized test fixture, or to reproduce an SDK bug against the local platform. Explains the throwaway api key, the endpoints, curl versus the SDK itself, and how to clean up.
---

# Probing the Smarter API

The SDK's structs must match what the platform really returns. Before you add
or change one, look at a real response from the local platform, the one in
`../smarter` (its dev server is `http://localhost:9357`, in the `smarter-app`
container). Start it with `make run` in `../smarter` if `curl -s -o /dev/null -w
'%{http_code}' localhost:9357/` doesn't answer 302. See the platform's own
`smarter-docker-environment` skill.

The Python SDK's `tests/data/` (`../smarter-python/tests/data/`) may already
hold a sanitized copy of the response you need.

## 1. Create a throwaway key

```console
docker exec smarter-app bash -c 'cd /home/smarter_user/smarter && python manage.py shell -c "
from smarter.lib.drf.models import SmarterAuthToken
from smarter.apps.account.utils import smarter_cached_objects as s
from smarter.apps.account.models import UserProfile
u = s.smarter_admin
up = UserProfile.objects.filter(user=u).first()
_, key = SmarterAuthToken.objects.create(user=u, user_profile=up, name=\"sdk_probe\", description=\"DELETE ME\", is_active=True)
print(\"KEY=\" + key)
"' 2>/dev/null | grep '^KEY=' | cut -d= -f2 > "$SCRATCHPAD/key.txt"
```

Keep the key in your scratchpad, never in the repository. The admin user is a
superuser: to see what an ordinary user gets (e.g. a 403 or a shorter list),
create the key for a non-staff user.

## 2. Call the api

With curl, to see the raw response:

```console
KEY=$(cat "$SCRATCHPAD/key.txt")
curl -s -H "Authorization: Token $KEY" http://localhost:9357/api/v1/guardrails/
curl -s -X POST -H "Authorization: Token $KEY" "http://localhost:9357/api/v1/cli/describe/LLMClient/?name=calculator"
curl -s -X POST -H "Authorization: Token $KEY" -H "Content-Type: application/json" \
  -H "X-Smarter-Capabilities: chat-contract-v2" -d '{"messages":[{"role":"user","content":"hi"}]}' \
  http://localhost:9357/api/v1/llm-clients/5/prompt/
```

Or with the SDK itself, which is how a bug report should be reproduced. Write a
throwaway `main` package in your scratchpad, with a `go.mod` that `replace`s the
module with this checkout, and never in the repository:

```console
mkdir -p "$SCRATCHPAD/probe" && cd "$SCRATCHPAD/probe"
cat > main.go <<'GO'
package main

import (
	"context"
	"fmt"

	smarter "github.com/smarter-sh/smarter-golang"
)

func main() {
	client, err := smarter.NewClient()
	if err != nil {
		panic(err)
	}
	me, err := client.WhoAmI(context.Background())
	fmt.Printf("%+v %v\n", me, err)
}
GO
go mod init probe && go mod edit -replace github.com/smarter-sh/smarter-golang=<absolute path of this checkout> && go mod tidy
SMARTER_BASE_URL=http://localhost:9357 SMARTER_API_KEY=$(cat "$SCRATCHPAD/key.txt") SMARTER_LOG=debug go run .
```

| Family              | Endpoints                                                                                           |
| ------------------- | --------------------------------------------------------------------------------------------------- |
| Manifest api (POST) | `cli/apply/`, `cli/validate/`, `cli/{get,describe,delete,deploy,undeploy,logs}/<Kind>/?name=<name>` |
|                     | `cli/example-manifest/<Kind>/`, `cli/json-schema/<Kind>/`, `cli/{whoami,version,status,resources}/` |
| Chat                | `llm-clients/<id>/config/`, `llm-clients/<id>/prompt/` (SSE with `Accept: text/event-stream`)       |
| REST                | `llm-clients/`, `guardrails/`, `mcpclients/`, `vectorstores/`, `llmhosts/`, `secrets/`, `plugins/`, |
|                     | `accounts/`, `prompts/history/`, `journal/`, `providers/`, `proxy/` and their sub-paths             |

Routes: `../smarter/smarter/smarter/apps/api/v1/urls.py` and each app's
`api/v1/urls.py`, and the platform's `openapi.json`
(`../smarter/smarter/smarter/apps/api/v1/contract/openapi.json`).

**Never call `deploy` while probing** unless the user asks: it creates real,
billable infrastructure.

## 3. Turn the response into a fixture

Trim it to one or two items, sanitize it (names, emails, addresses, phone
numbers, account numbers, keys, hostnames: see `smarter-golang-testing`), and
save it in `testdata/`, and in `../smarter-python/tests/data/` if the Python
SDK lacks it. Grep the result before you finish.

## 4. Clean up

```console
docker exec smarter-app bash -c 'cd /home/smarter_user/smarter && python manage.py shell -c "
from smarter.lib.drf.models import SmarterAuthToken
print(SmarterAuthToken.objects.filter(name=\"sdk_probe\", description=\"DELETE ME\").delete())
"' 2>/dev/null
```

Delete any resources you applied while probing.
