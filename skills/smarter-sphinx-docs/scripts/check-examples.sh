#!/usr/bin/env bash
# Compile every Go example of the SDK's pages in the platform's docs against this checkout, so that
# the docs can't describe an api that doesn't exist. Each code block becomes the body of a function
# that returns an error, in a throwaway module that replaces the SDK with this checkout; goimports
# adds the imports. The variables that the pages use without declaring them (client, ctx,
# llmclient, guardrail, mcpclient, store, manifest, chat) are declared.
#
# usage, from the smarter-golang repository root (needs goimports):
#   skills/smarter-sphinx-docs/scripts/check-examples.sh [path of a smarter checkout, default ../smarter]
set -euo pipefail

SMARTER="$(cd "${1:-../smarter}" && pwd)"
SDK="$(pwd)"
WORK=$(mktemp -d)
trap 'rm -rf "$WORK"' EXIT

python3 - "$SMARTER/docs/source/smarter-framework" "$WORK/main.go" <<'PY'
import glob, os, re, sys
docs, output = sys.argv[1], sys.argv[2]
files = [os.path.join(docs, "smarter-golang.rst")] + sorted(glob.glob(os.path.join(docs, "smarter-golang", "*.rst")))
out = [
    "package main", "", "import (", '\t"github.com/smarter-sh/smarter-golang/sam"', '\tsmarter "github.com/smarter-sh/smarter-golang"', ")", "",
    "var (", "\tclient *smarter.Client", "\tctx context.Context", "\tllmclient *smarter.LLMClient", "\tguardrail *smarter.Guardrail",
    "\tmcpclient *smarter.MCPClient", "\tstore *smarter.Vectorstore", "\tmanifest *smarter.Manifest", "\tchat *smarter.ChatSession", ")", "",
    "var _ = sam.Kinds", "", "func main() {}", "",
]
count = 0
for path in files:
    for match in re.finditer(r"\.\. code-block:: go\n\n((?:   .*\n|\n)+)", open(path).read()):
        code = "\n".join(line[3:] for line in match.group(1).split("\n")).rstrip()
        declared = set()
        for line in code.split("\n"):
            found = re.match(r"^((?:\w+\s*,\s*)*\w+)\s*:=", line)
            if found:
                declared.update(name.strip() for name in found.group(1).split(",") if name.strip() != "_")
        uses = "\n".join(f"\t_ = {name}" for name in sorted(declared))
        body = "\n".join("\t" + line for line in code.split("\n"))
        out.append(f"// {os.path.basename(path)}\nfunc example{count}() error {{\n{body}\n{uses}\n\treturn nil\n}}\n")
        count += 1
open(output, "w").write("\n".join(out))
print(f"{count} examples")
PY
cd "$WORK"
go mod init doccheck > /dev/null 2>&1
go mod edit -replace "github.com/smarter-sh/smarter-golang=$SDK" -require github.com/smarter-sh/smarter-golang@v0.0.0
GOFLAGS=-mod=mod go mod tidy > /dev/null 2>&1
goimports -w main.go
go vet . 2>&1 | grep -v "unreachable code" || true
go build ./... && echo "ok"
