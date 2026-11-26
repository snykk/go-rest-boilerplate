#!/usr/bin/env bash
# Helpers of the release workflow (.github/workflows/release.yml). Both also run locally, from the root of the
# repository, to preview a release:
#
#   bash .github/scripts/release.sh notes vX.Y.Z
#       The notes of the GitHub Release: the version's section of CHANGELOG.md with its wrapped lines joined,
#       links to files of the repository pointed at the tag, then a link to the changes since the previous tag.
#
#   bash .github/scripts/release.sh tag-message vX.Y.Z TITLE
#       The message of the annotated tag: TITLE, a blank line, and the first paragraph of the version's
#       section, which sums up the release, on one line.
set -euo pipefail

repository="${GITHUB_SERVER_URL:-https://github.com}/${GITHUB_REPOSITORY:-snykk/go-rest-boilerplate}"

# The lines of CHANGELOG.md under the "## [vX.Y.Z] - date" heading of a version, up to the next "## " heading
# or the link definitions at the end, without the blank lines before them. awk reads to the end rather than
# exiting early, which would break the pipe for tr under pipefail once the changelog outgrows the pipe's
# buffer; index() matches the heading literally, since the dots of a version would match any character in a
# regular expression.
section() {
  tr -d '\r' < CHANGELOG.md | awk -v tag="$1" '
    /^## / {
      if (inside) done = 1
      else if (index($0, "## [" tag "]") == 1) { inside = 1; next }
    }
    /^\[v[0-9]+\.[0-9]+\.[0-9]+\]: / { if (inside) done = 1 }
    inside && !done { print }
  ' | sed -e '/./,$!d'
}

# GitHub shows a line break inside a paragraph of release notes as a break, so the notes join the lines that
# the changelog wraps: each paragraph and each list item becomes one line, while code blocks, headings,
# tables and quotes keep their lines.
unwrap() {
  awk '
    function flush() { if (buf != "") print buf; buf = "" }
    /^```/ { flush(); print; fence = !fence; next }
    fence { print; next }
    /^[[:space:]]*$/ { flush(); print ""; next }
    /^(#|\||>)/ { flush(); print; next }
    /^[[:space:]]*([-*+]|[0-9]+\.)[[:space:]]/ { flush(); buf = $0; next }
    {
      line = $0
      sub(/^[[:space:]]+/, "", line)
      buf = (buf == "" ? $0 : buf " " line)
    }
    END { flush() }
  '
}

usage() {
  echo "usage: $0 notes vX.Y.Z | tag-message vX.Y.Z TITLE" >&2
  exit 2
}

tag="${2:-}"
[ -n "$tag" ] || usage
case "${1:-}" in
  notes)
    notes="$(section "$tag")"
    if [ -z "$notes" ]; then
      echo "CHANGELOG.md has no section for $tag: its heading is '## [$tag] - YYYY-MM-DD'" >&2
      exit 1
    fi
    previous="$(git describe --tags --abbrev=0 "${tag}^" 2>/dev/null || true)"
    # A link such as (RELEASING.md) would be read relative to the release page; point it at the file as the
    # tag has it.
    printf '%s\n' "$notes" | unwrap | sed -E "s|\]\(([^):#][^):]*)\)|](${repository}/blob/${tag}/\1)|g"
    if [ -n "$previous" ]; then
      printf '\n**Full Changelog**: %s/compare/%s...%s\n' "$repository" "$previous" "$tag"
    fi
    ;;
  tag-message)
    title="${3:-}"
    [ -n "$title" ] || usage
    summary="$(section "$tag" | sed -e '/^$/,$d' | tr '\n' ' ' | sed -e 's/`//g' -e 's/  */ /g' -e 's/ $//')"
    case "$summary" in
      "" | "#"*)
        echo "CHANGELOG.md has no paragraph that sums up $tag" >&2
        exit 1
        ;;
    esac
    printf '%s\n\n%s\n' "$title" "$summary"
    ;;
  *)
    usage
    ;;
esac
