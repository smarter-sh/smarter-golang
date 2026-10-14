#!/usr/bin/env bash
# Copy the Smarter platform's published api contracts into the SDK's test data.
#
# The platform commits its api contracts, generated from its Pydantic models:
#
# - the chat and journal contracts' JSON Schemas and examples, which the SDK mirrors by hand
#   (chat.go and journal.go). Its tests check that the mirror
#   declares every field of the platform's schemas, and parses the platform's examples.
# - the JSON Schema and an example manifest of every SAM kind, from which
#   internal/cmd/samgen generates sam/. Its tests check that every
#   generated struct decodes its kind's example.
# - the api's OpenAPI document. Its tests check that every request of the SDK to a family that the
#   document covers is in it.
#
# Run this after the platform's contracts change (make contracts), then make sam-models, then
# run the tests.
#
# usage, from the repository root:
#   scripts/sync-contracts.sh [path of a smarter checkout, default ../smarter] [git ref]
#
# With a git ref (e.g. origin/main, or a release tag), the files are read at that ref with git show,
# without changing the checkout. Without one, they are copied from the checkout's working tree.
set -euo pipefail

SMARTER="${1:-../smarter}"
REF="${2:-}"
TARGET="testdata/contracts"
SAM_TARGET="testdata/sam"
API_CONTRACT="smarter/smarter/apps/api/v1/contract"
FILES=(
  "smarter/smarter/apps/prompt/contract/chat-contract.schema.json"
  "smarter/smarter/apps/prompt/contract/chat-contract.example.json"
  "smarter/smarter/lib/journal/contract/journal-contract.schema.json"
  "$API_CONTRACT/openapi.json"
)

if [ ! -d "$SMARTER/smarter/smarter/apps/prompt/contract" ]; then
  echo "No smarter checkout at $SMARTER" >&2
  exit 1
fi

# copy a file of the checkout, at REF if there is one.
copy() {
  if [ -n "$REF" ]; then
    git -C "$SMARTER" show "$REF:$1" > "$2/$(basename "$1")"
  else
    cp "$SMARTER/$1" "$2/"
  fi
}

mkdir -p "$TARGET" "$SAM_TARGET"
for file in "${FILES[@]}"; do
  copy "$file" "$TARGET"
done

# the SAM schemas and examples: the platform's set replaces the SDK's, so a removed kind is removed.
if [ -n "$REF" ]; then
  SAM_FILES=$(git -C "$SMARTER" ls-tree --name-only "$REF" "$API_CONTRACT/sam/")
else
  SAM_FILES=$(cd "$SMARTER" && ls "$API_CONTRACT"/sam/*.json)
fi
if [ -z "$SAM_FILES" ]; then
  echo "No SAM schemas in $SMARTER/$API_CONTRACT/sam/" >&2
  exit 1
fi
rm -f "$SAM_TARGET"/*.schema.json "$SAM_TARGET"/*.example.json
while IFS= read -r file; do
  copy "$file" "$SAM_TARGET"
done <<< "$SAM_FILES"

VERSION=$(git -C "$SMARTER" describe --tags --always "${REF:-HEAD}" 2>/dev/null || echo "$SMARTER")
echo "Copied the contracts of $VERSION into $TARGET and $SAM_TARGET"
