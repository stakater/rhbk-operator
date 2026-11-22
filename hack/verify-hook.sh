#!/usr/bin/env bash
# Claude Code PostToolUse hook (wired in .claude/settings.json; story DX.1).
#
# After Claude edits a Go or YAML file (or go.mod, go.sum, the Makefile) in this
# repository, run `make verify` in the tree that holds the file, so a change
# cannot be called done without the gate having run.
#
# Input: the hook payload as JSON on stdin. Success: exit 0, no output.
# Failure: exit 2 with the end of the log on stderr. For PostToolUse, Claude
# Code feeds stderr back to Claude only on exit 2. The full log of the last run
# is bin/verify-hook.log in the verified tree.
set -euo pipefail

payload="$(cat)"
if command -v jq >/dev/null 2>&1; then
  file="$(printf '%s' "$payload" | jq -r '.tool_input.file_path // empty')"
else
  # tool_input comes before tool_response in the payload: take the first match.
  file="$(printf '%s' "$payload" \
    | grep -o '"file_path"[[:space:]]*:[[:space:]]*"[^"]*"' \
    | head -n 1 \
    | sed 's/^"file_path"[[:space:]]*:[[:space:]]*"\(.*\)"$/\1/' || true)"
fi

case "$file" in
  *.go | go.mod | */go.mod | go.sum | */go.sum | *.yaml | *.yml | Makefile | */Makefile) ;;
  *) exit 0 ;; # not a file that make verify checks
esac

# Only files in this repository: the primary checkout or any of its worktrees.
dir="$(dirname "$file")"
[ -d "$dir" ] || exit 0
repo="$(git -C "$dir" rev-parse --show-toplevel 2>/dev/null)" || exit 0
ours="$(git -C "$(dirname "$0")" rev-parse --path-format=absolute --git-common-dir)"
theirs="$(git -C "$repo" rev-parse --path-format=absolute --git-common-dir)"
[ "$ours" = "$theirs" ] || exit 0
# Gitignored files (bin/, bundle/) are not part of what make verify checks.
if git -C "$repo" check-ignore -q -- "$file"; then
  exit 0
fi

# bin/ is gitignored, so the log does not trip the no-diff check.
mkdir -p "$repo/bin"
log="$repo/bin/verify-hook.log"
if make -C "$repo" verify >"$log" 2>&1; then
  exit 0
fi

{
  echo "make verify failed in $repo after the edit to ${file#"$repo"/}."
  echo "Last 100 lines below. Full log: $log"
  echo
  tail -n 100 "$log"
} >&2
exit 2
