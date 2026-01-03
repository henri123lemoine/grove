#!/usr/bin/env bash
set -euo pipefail

GREEN='\033[0;32m' YELLOW='\033[1;33m' CYAN='\033[0;36m' NC='\033[0m'

current=$(git tag --sort=-v:refname | head -1)
current=${current:-v0.0.0}
IFS='.' read -r major minor patch <<< "${current#v}"

echo -e "Current: ${GREEN}$current${NC}"
echo -e "1) patch  ${CYAN}v${major}.${minor}.$((patch + 1))${NC}"
echo -e "2) minor  ${CYAN}v${major}.$((minor + 1)).0${NC}"
echo -e "3) major  ${CYAN}v$((major + 1)).0.0${NC}"
read -rp "Choice: " choice

case $choice in
    1) new="v${major}.${minor}.$((patch + 1))" ;;
    2) new="v${major}.$((minor + 1)).0" ;;
    3) new="v$((major + 1)).0.0" ;;
    *) echo "Invalid"; exit 1 ;;
esac

read -rp "Description (optional): " desc
git tag -a "$new" -m "${desc:-Release $new}"
git push origin HEAD --follow-tags
echo -e "${GREEN}Pushed $new${NC}"
