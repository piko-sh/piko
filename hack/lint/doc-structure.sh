#!/usr/bin/env bash
# Copyright 2026 PolitePixels Limited
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0

# Asserts every docs/**/*.md page sits in a folder matching its frontmatter
# nav.sidebar.section value. Keeps docs in their Diataxis quadrant.

set -euo pipefail

cd "$(dirname "$0")/../.."

fail=0

while IFS= read -r -d '' file; do
    rel="${file#docs/}"
    top="${rel%%/*}"
    if [[ "$rel" == "$top" ]]; then
        # File directly under docs/ (e.g. introduction.md, README.md).
        continue
    fi

    section=$(python3 - "$file" <<'EOF'
import sys
import re

path = sys.argv[1]
with open(path, encoding='utf-8') as fh:
    data = fh.read()

match = re.search(r'^---\n(.*?)\n---', data, re.DOTALL)
if not match:
    sys.exit(0)

front = match.group(1)
section_match = re.search(r'^\s*section:\s*(?:"([^"]*)"|\'([^\']*)\'|(\S+))', front, re.MULTILINE)
if not section_match:
    sys.exit(0)

value = section_match.group(1) or section_match.group(2) or section_match.group(3) or ''
print(value.strip())
EOF
)

    if [[ -z "$section" ]]; then
        echo "MISSING nav.sidebar.section: $file" >&2
        fail=1
        continue
    fi

    if [[ "$section" != "$top" ]]; then
        echo "MISMATCH: $file lives under docs/$top/ but declares section: $section" >&2
        fail=1
    fi
done < <(find docs -type f -name '*.md' -not -path 'docs/styles/*' -not -path 'docs/diagrams/*' -print0)

if [[ $fail -eq 0 ]]; then
    echo "OK: all doc pages sit in the correct quadrant folder."
fi

exit $fail
