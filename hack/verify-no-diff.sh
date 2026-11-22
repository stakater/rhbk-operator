#!/usr/bin/env bash
# Run a command and fail if it changed any file in the working tree.
#
# `make verify` wraps `fmt manifests generate` with it: if formatting or code
# generation rewrites a file, the Go code was unformatted or the generated files
# (CRDs, RBAC, deepcopy) were stale. It compares the tree before and after the
# command, not against HEAD, so it also works on a tree with uncommitted edits
# (the edit-hook runs it mid-change). Untracked files count; gitignored ones
# (bin/, cover.out) do not.
#
# Usage: hack/verify-no-diff.sh <command> [args...]
set -euo pipefail

if [ "$#" -eq 0 ]; then
  echo "usage: $0 <command> [args...]" >&2
  exit 64
fi

cd "$(git rev-parse --show-toplevel)"

# Snapshot into a throwaway index: the real index and the files are not touched.
# (git add writes loose blobs to the object store; git gc prunes them.)
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
snapshot() {
  GIT_INDEX_FILE="$tmp/index" git add --all
  GIT_INDEX_FILE="$tmp/index" git write-tree
}

before="$(snapshot)"
"$@"
after="$(snapshot)"

if [ "$before" != "$after" ]; then
  {
    git --no-pager diff --no-ext-diff "$before" "$after"
    echo
    git --no-pager diff --no-ext-diff --stat "$before" "$after"
    echo
    echo "verify-no-diff: '$*' changed the files above."
    echo "The Go code was unformatted or the generated files were stale. The files"
    echo "are now updated in your working tree: review the change and commit it."
  } >&2
  exit 1
fi
