#!/usr/bin/env bash
set -euo pipefail

GREEN='\033[0;32m' YELLOW='\033[1;33m' CYAN='\033[0;36m' NC='\033[0m'

die() {
    printf '%b\n' "${YELLOW}Error:${NC} $*" >&2
    exit 1
}

git rev-parse --is-inside-work-tree >/dev/null 2>&1 || die "not inside a git worktree"
git diff --quiet || die "working tree has unstaged changes"
git diff --cached --quiet || die "working tree has staged changes"

current=$(git tag --list 'v[0-9]*.[0-9]*.[0-9]*' --sort=-v:refname | sed -n '1p')
current=${current:-v0.0.0}

if [[ ! "$current" =~ ^v([0-9]+)\.([0-9]+)\.([0-9]+)$ ]]; then
    die "latest release tag is not semver-like: $current"
fi

major=${BASH_REMATCH[1]}
minor=${BASH_REMATCH[2]}
patch=${BASH_REMATCH[3]}

printf 'Current: %b%s%b\n' "$GREEN" "$current" "$NC"
printf '1) patch  %bv%s.%s.%s%b\n' "$CYAN" "$major" "$minor" "$((patch + 1))" "$NC"
printf '2) minor  %bv%s.%s.0%b\n' "$CYAN" "$major" "$((minor + 1))" "$NC"
printf '3) major  %bv%s.0.0%b\n' "$CYAN" "$((major + 1))" "$NC"
read -rp "Choice: " choice

case $choice in
    1) new="v${major}.${minor}.$((patch + 1))" ;;
    2) new="v${major}.$((minor + 1)).0" ;;
    3) new="v$((major + 1)).0.0" ;;
    *) die "invalid release type" ;;
esac

if git rev-parse -q --verify "refs/tags/$new" >/dev/null; then
    die "tag already exists: $new"
fi

printf 'Next release: %b%s%b\n' "$CYAN" "$new" "$NC"
read -rp "Description (optional): " desc
read -rp "Create tag $new and push HEAD with tags? [y/N] " confirm

case $confirm in
    y|Y|yes|YES) ;;
    *) die "release cancelled" ;;
esac

git tag -a "$new" -m "${desc:-Release $new}"
git push origin HEAD --follow-tags
printf '%bPushed %s%b\n' "$GREEN" "$new" "$NC"
