#!/bin/sh
set -eu

project_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$project_root"

jq -e '
  .schemaVersion == 2
  and ((.action == "create" or .action == "update") or .action == "update")
  and .owner == "kentomk"
  and (.name | type == "string" and test("^[a-z0-9][a-z0-9-]{1,62}$"))
  and (.description | type == "string" and length >= 20 and length <= 160)
  and (.topics | type == "array" and length >= 1 and length <= 10 and index("kento-oss") != null and all(type == "string"))
  and .candidateId == "20260717T120911Z-5f11"
  and (.targetUsers | type == "string" and length >= 10 and length <= 500)
  and (.jobToBeDone | type == "string" and length >= 10 and length <= 1000)
  and (.distributionPath | type == "string" and length >= 10 and length <= 500)
  and (.successMetric | type == "string" and length >= 10 and length <= 500)
  and (.reviewAfterDays | type == "number" and floor == . and . >= 1 and . <= 30)
  and .opportunityScore == 78
  and (.demandEvidence | type == "array" and length >= 3 and
       all(type == "object" and (.url | type == "string" and startswith("https://")) and
           (.kind | type == "string" and test("^[a-z][a-z0-9-]{2,49}$")) and
           (.independenceKey | type == "string" and (gsub("^\\s+|\\s+$"; "") | length >= 3 and length <= 200))))
  and ((.demandEvidence | map(.independenceKey | gsub("^\\s+|\\s+$"; "") | ascii_downcase) | unique | length) >= 3)
  and ((.demandEvidence | map(.kind) | unique | length) >= 2)
  and (.alternatives | type == "array" and length >= 3 and
       all(type == "object" and (.name | type == "string" and length >= 2 and length <= 200) and
           (.url | type == "string" and startswith("https://")) and .tested == true and
           (.gap | type == "string" and length >= 10 and length <= 1000)))
  and ((.alternatives | map((.name | ascii_downcase) + "\n" + .url) | unique | length) >= 3)
  and .duplicateSearch.completed == true
  and (.duplicateSearch.summary | type == "string" and length >= 20)
  and (.differentiation | type == "string" and length >= 20)
  and .testCommand == "scripts/publisher-gate.sh"
  and .license == "Apache-2.0"
  and (.commitMessage | type == "string" and length >= 10 and length <= 120)
' publish-request.json >/dev/null

jq -e --slurpfile request publish-request.json '
  .schemaVersion == 1
  and .candidateId == $request[0].candidateId
  and (.createdBy | test("Matsuki Kento") and test("@kentomk") and test("AI|automated"; "i"))
' .kento-oss.json >/dev/null

grep -Eq '^## (Installation|Install|Getting Started)\b' README.md
grep -Eq '^## Quick[[:space:]]*start\b' README.md
grep -q 'Matsuki Kento' README.md
grep -q '@kentomk' README.md
grep -Eiq 'AI|automated' README.md

grep -Eq 'uses: actions/checkout@[0-9a-f]{40}([[:space:]]|$)' .github/workflows/ci.yml
grep -Eq 'uses: actions/setup-go@[0-9a-f]{40}([[:space:]]|$)' .github/workflows/ci.yml
if grep -Eq 'uses: actions/(checkout|setup-go)@v[0-9]' .github/workflows/ci.yml; then
  echo 'mutable GitHub Action reference found' >&2
  exit 1
fi

test -x scripts/publisher-gate.sh
sh -n scripts/publisher-gate.sh
