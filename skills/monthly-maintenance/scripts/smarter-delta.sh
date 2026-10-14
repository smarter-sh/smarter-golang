#!/usr/bin/env bash
# Report what changed in the Smarter platform since smarter-golang's last release, grouped by
# what each change means for the SDK. Read-only: it changes neither repository, apart from
# `git fetch`, which updates only the platform's remote-tracking branches.
#
# usage, from the smarter-golang repository root:
#   skills/monthly-maintenance/scripts/smarter-delta.sh [smarter checkout] [platform ref] [base commit]
#
# - smarter checkout: default ../smarter
# - platform ref: what the SDK syncs with, default origin/main (the platform's released branch)
# - base commit: a platform commit to start from. By default, the platform's commit at the date of
#   smarter-golang's last release tag, or the commit named by the newest "Synced with smarter <sha>"
#   line in smarter-python's history, whichever is later.
set -euo pipefail

SMARTER="${1:-../smarter}"
REF="${2:-origin/main}"
BASE_OVERRIDE="${3:-}"

if [ ! -d "$SMARTER/.git" ]; then
  echo "No smarter checkout at $SMARTER" >&2
  exit 1
fi

git -C "$SMARTER" fetch --quiet origin || echo "warning: git fetch failed; using the local $REF" >&2

TAG=$(git describe --tags --abbrev=0 2>/dev/null || true)
if [ -z "$TAG" ]; then
  echo "smarter-golang has no release tag yet; using its first commit" >&2
  TAG=$(git rev-list --max-parents=0 HEAD | tail -1)
fi
SINCE=$(git log -1 --format=%cI "$TAG")
BASE=$(git -C "$SMARTER" rev-list -1 --before="$SINCE" "$REF")
SOURCE="the platform at smarter-golang's last release, $TAG ($SINCE)"

# a later sync, recorded in a maintenance commit's body
SYNCED=$(git log -1 --format=%B --grep="Synced with smarter " | grep -oE "Synced with smarter [0-9a-f]{7,40}" | awk '{print $4}' || true)
if [ -n "$SYNCED" ] && git -C "$SMARTER" merge-base --is-ancestor "$BASE" "$SYNCED" 2>/dev/null; then
  BASE=$SYNCED
  SOURCE="the last recorded sync (Synced with smarter $SYNCED)"
fi
if [ -n "$BASE_OVERRIDE" ]; then
  BASE=$(git -C "$SMARTER" rev-parse "$BASE_OVERRIDE")
  SOURCE="the given base commit"
fi

echo "baseline: $SOURCE"
echo "platform range: ${BASE:0:12}..$REF ($(git -C "$SMARTER" describe --tags --always "$REF"))"
echo "platform commits in range: $(git -C "$SMARTER" rev-list --count "$BASE..$REF")"

# area | what it means for the SDK | pathspecs, relative to the smarter repository root
AREAS=(
  "Manifest (cli) api|manifests.go, client.go|smarter/smarter/apps/api/v1/"
  "REST urls and views|<family>.go and models.go|:(glob)smarter/smarter/apps/*/api/** :(glob)smarter/smarter/lib/journal/api/**"
  "Serializers (REST response shapes)|models.go|:(glob)smarter/smarter/apps/*/serializers.py :(glob)smarter/smarter/apps/*/serializers/**"
  "SAM manifest models and kinds|make sam-models|:(glob)smarter/smarter/apps/*/manifest/** :(glob)smarter/smarter/lib/manifest/** smarter/smarter/apps/api/v1/manifests/ smarter/smarter/apps/api/v1/contract/sam/"
  "Chat and journal contracts, OpenAPI|make contracts, chat.go, journal.go, contracts_test.go|smarter/smarter/apps/prompt/contract/ smarter/smarter/lib/journal/contract/ smarter/smarter/apps/prompt/progress.py smarter/smarter/apps/api/v1/contract/openapi.json"
  "Django-React contract|copy into react.go|smarter/smarter/lib/django/templatetags/smarter_react_contract.py"
  "Authentication|http.go|smarter/smarter/lib/drf/ smarter/smarter/apps/proxy/authentication.py"
  "Platform settings that gate apis|docs, live tests|smarter/smarter/common/conf/"
  "ADRs|doc/adr/|docs/source/adr/"
  "Skills|skills/ (the adapted copies)|skills/"
  "Lint standard|.pre-commit-config.yaml (generic hooks)|.pre-commit-config.yaml codespell.txt"
  "The SDK's own docs pages|keep the edits; check the examples still compile|docs/source/smarter-framework/smarter-golang.rst docs/source/smarter-framework/smarter-golang/ docs/source/smarter-framework/apis-and-sdks.rst"
)

for area in "${AREAS[@]}"; do
  IFS='|' read -r name action paths <<<"$area"
  # shellcheck disable=SC2086 # the pathspecs are deliberately word-split
  log=$(git -C "$SMARTER" log --no-merges --format='  %h %ad %s' --date=short "$BASE..$REF" -- $paths)
  echo
  echo "== $name  (SDK: $action)"
  if [ -z "$log" ]; then
    echo "  no changes"
  else
    echo "$log"
    # shellcheck disable=SC2086
    git -C "$SMARTER" diff --stat=120 "$BASE..$REF" -- $paths | tail -n 15 | sed 's/^/    /'
  fi
done

# the reference implementation: the Python SDK's own changes since the same date.
PYTHON="${PYTHON_SDK:-../smarter-python}"
echo
echo "== The Python SDK (reference implementation)  (SDK: port each feature and fix)"
if [ -d "$PYTHON/.git" ]; then
  git -C "$PYTHON" fetch --quiet origin || true
  log=$(git -C "$PYTHON" log --no-merges --format='  %h %ad %s' --date=short --since="$SINCE" origin/main -- smarter/ skills/ doc/adr/ 2>/dev/null || true)
  if [ -z "$log" ]; then echo "  no changes"; else echo "$log"; fi
else
  echo "  no smarter-python checkout at $PYTHON"
fi

# the smarter cli, which shares this SDK's configuration file and Go toolchain.
CLI="${SMARTER_CLI:-../smarter-cli}"
echo
echo "== The smarter cli  (SDK: configuration lookup in config.go, Go version, lint hooks)"
if [ -d "$CLI/.git" ]; then
  git -C "$CLI" fetch --quiet origin || true
  log=$(git -C "$CLI" log --no-merges --format='  %h %ad %s' --date=short --since="$SINCE" origin/main -- cmd/root.go cmd/api.go cmd/configure.go .pre-commit-config.yaml go.mod .github/workflows/ 2>/dev/null || true)
  if [ -z "$log" ]; then echo "  no changes"; else echo "$log"; fi
else
  echo "  no smarter-cli checkout at $CLI"
fi
