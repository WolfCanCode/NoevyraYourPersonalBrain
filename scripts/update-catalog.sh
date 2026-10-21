#!/bin/sh
# Update the embedded AAS skills catalog to the latest upstream version.
# Run this before building a release, or let the CI pipeline handle it.
set -eu
url="https://raw.githubusercontent.com/sickn33/agentic-awesome-skills/main/skills_index.json"
dest="internal/intelligence/skills_index.json"
printf 'Fetching %s\n' "$url"
curl -fsSL "$url" -o "$dest"
count=$(python3 -c "import json; print(len(json.load(open('$dest'))))" 2>/dev/null || echo "?")
size=$(wc -c < "$dest")
printf 'Updated: %s skills, %s bytes -> %s\n' "$count" "$size" "$dest"
