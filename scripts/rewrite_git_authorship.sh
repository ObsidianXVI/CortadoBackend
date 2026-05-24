#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Rewrite historical Git authorship across all local refs while preserving
existing author/committer timestamps.

Default behavior:
- Rewrites only commits whose author or committer matches:
    Cortado Dev <861297533844-compute@developer.gserviceaccount.com>
- Uses the current global Git identity as the replacement author/committer.
- Does not make any changes unless --apply is passed.

Usage:
  scripts/rewrite_git_authorship.sh [options]

Options:
  --old-name NAME       Old author/committer name to replace.
  --old-email EMAIL     Old author/committer email to replace.
  --new-name NAME       Replacement author/committer name.
  --new-email EMAIL     Replacement author/committer email.
  --rewrite-all         Rewrite every commit instead of matching a single identity.
  --apply               Perform the history rewrite.
  -h, --help            Show this help text.

Examples:
  scripts/rewrite_git_authorship.sh
  scripts/rewrite_git_authorship.sh --apply
  scripts/rewrite_git_authorship.sh \
    --old-name "Cortado Dev" \
    --old-email "861297533844-compute@developer.gserviceaccount.com" \
    --new-name "ObsidianXVI" \
    --new-email "siddharth.chitikela@gmail.com" \
    --apply

Notes:
- This rewrites commit IDs on every affected branch/tag/ref.
- The script preserves the original GIT_AUTHOR_DATE and GIT_COMMITTER_DATE.
- Run it from a clean clone, review the result, then force-push rewritten refs.
- git-filter-repo is preferable when available, but this script intentionally
  uses git filter-branch for portability.
EOF
}

die() {
  printf 'error: %s\n' "$*" >&2
  exit 1
}

OLD_NAME='Cortado Dev'
OLD_EMAIL='861297533844-compute@developer.gserviceaccount.com'
NEW_NAME=''
NEW_EMAIL=''
REWRITE_ALL=0
APPLY=0

while (($# > 0)); do
  case "$1" in
    --old-name)
      [[ $# -ge 2 ]] || die "--old-name requires a value"
      OLD_NAME="$2"
      shift 2
      ;;
    --old-email)
      [[ $# -ge 2 ]] || die "--old-email requires a value"
      OLD_EMAIL="$2"
      shift 2
      ;;
    --new-name)
      [[ $# -ge 2 ]] || die "--new-name requires a value"
      NEW_NAME="$2"
      shift 2
      ;;
    --new-email)
      [[ $# -ge 2 ]] || die "--new-email requires a value"
      NEW_EMAIL="$2"
      shift 2
      ;;
    --rewrite-all)
      REWRITE_ALL=1
      shift
      ;;
    --apply)
      APPLY=1
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      usage >&2
      die "unknown option: $1"
      ;;
  esac
done

git rev-parse --is-inside-work-tree >/dev/null 2>&1 || die "not inside a Git worktree"
REPO_ROOT="$(git rev-parse --show-toplevel)"
cd "$REPO_ROOT"

NEW_NAME="${NEW_NAME:-$(git config --global user.name || true)}"
NEW_EMAIL="${NEW_EMAIL:-$(git config --global user.email || true)}"

[[ -n "$NEW_NAME" ]] || die "replacement user.name is empty; set git config --global user.name or pass --new-name"
[[ -n "$NEW_EMAIL" ]] || die "replacement user.email is empty; set git config --global user.email or pass --new-email"

git diff --quiet || die "working tree has unstaged tracked changes; commit or stash them first"
git diff --cached --quiet || die "index has staged changes; commit or stash them first"

if [[ "$REWRITE_ALL" -eq 1 ]]; then
  MATCHING_COMMITS="$(git rev-list --all --count)"
else
  MATCHING_COMMITS="$(
    git log --all --format='%H%x09%an%x09%ae%x09%cn%x09%ce' |
      awk -F '\t' \
        -v old_name="$OLD_NAME" \
        -v old_email="$OLD_EMAIL" '
          $2 == old_name || $3 == old_email || $4 == old_name || $5 == old_email {
            count++
          }
          END {
            print count + 0
          }
        '
  )"
fi

printf 'Repository: %s\n' "$REPO_ROOT"
printf 'Replacement identity: %s <%s>\n' "$NEW_NAME" "$NEW_EMAIL"
if [[ "$REWRITE_ALL" -eq 1 ]]; then
  printf 'Rewrite scope: all commits on all local refs\n'
else
  printf 'Rewrite scope: commits matching %s <%s>\n' "$OLD_NAME" "$OLD_EMAIL"
fi
printf 'Matching commits: %s\n' "$MATCHING_COMMITS"
printf '\nCurrent identities in this clone:\n'
git log --all --format='%an <%ae>%n%cn <%ce>' | sort -u | sed 's/^/  /'

if [[ "$MATCHING_COMMITS" -eq 0 ]]; then
  printf '\nNo matching commits found. Nothing to rewrite.\n'
  exit 0
fi

if [[ "$APPLY" -ne 1 ]]; then
  cat <<'EOF'

Dry run only. No refs were rewritten.

To execute the rewrite:
  scripts/rewrite_git_authorship.sh --apply

After verifying the rewritten history, force-push the updated refs manually:
  git push --force origin --all
  git push --force origin --tags
EOF
  exit 0
fi

BACKUP_TAG="backup/pre-authorship-rewrite-$(date -u +%Y%m%d-%H%M%S)"
git tag "$BACKUP_TAG" HEAD

export NEW_NAME NEW_EMAIL OLD_NAME OLD_EMAIL REWRITE_ALL
FILTER_BRANCH_SQUELCH_WARNING=1 \
git filter-branch --force --env-filter '
  rewrite_commit=0

  if [ "$REWRITE_ALL" = "1" ]; then
    rewrite_commit=1
  else
    if [ "$GIT_AUTHOR_NAME" = "$OLD_NAME" ] || [ "$GIT_COMMITTER_NAME" = "$OLD_NAME" ] || \
       [ "$GIT_AUTHOR_EMAIL" = "$OLD_EMAIL" ] || [ "$GIT_COMMITTER_EMAIL" = "$OLD_EMAIL" ]; then
      rewrite_commit=1
    fi
  fi

  if [ "$rewrite_commit" = "1" ]; then
    GIT_AUTHOR_NAME="$NEW_NAME"
    GIT_AUTHOR_EMAIL="$NEW_EMAIL"
    GIT_COMMITTER_NAME="$NEW_NAME"
    GIT_COMMITTER_EMAIL="$NEW_EMAIL"
    export GIT_AUTHOR_NAME GIT_AUTHOR_EMAIL GIT_COMMITTER_NAME GIT_COMMITTER_EMAIL
    export GIT_AUTHOR_DATE GIT_COMMITTER_DATE
  fi
' --tag-name-filter cat -- --all

cat <<EOF

Rewrite complete.
Backup tag created: $BACKUP_TAG

Recommended next steps:
  1. Review:
       git log --all --format='%h %an <%ae> | %cn <%ce>' --decorate --graph
  2. Push rewritten refs:
       git push --force origin --all
       git push --force origin --tags
  3. Once fully satisfied, remove filter-branch backups and collect garbage:
       rm -rf .git/refs/original/
       git reflog expire --expire=now --all
       git gc --prune=now --aggressive
EOF
