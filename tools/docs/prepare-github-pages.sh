#!/bin/sh
# SPDX-License-Identifier: GPL-3.0-or-later

set -eu

if [ "$#" -ne 3 ]; then
  echo "usage: $0 <built-docs-dir> <target-subdir> <pages-dir>" >&2
  exit 64
fi

source_dir=$1
target_dir=$2
pages_dir=$3

case "$target_dir" in
  ""|/*|*..*|*//*|*[^A-Za-z0-9._-]*)
    echo "invalid GitHub Pages target directory: $target_dir" >&2
    exit 64
    ;;
esac

if [ ! -d "$source_dir" ]; then
  echo "built docs directory does not exist: $source_dir" >&2
  exit 66
fi

source_dir=$(cd "$source_dir" && pwd)
branch=${PAGES_BRANCH:-gh-pages}
remote=${PAGES_REMOTE:-origin}

rm -rf "$pages_dir"

if git ls-remote --exit-code "$remote" "refs/heads/$branch" >/dev/null 2>&1; then
  git clone --depth 1 --branch "$branch" "$(git remote get-url "$remote")" "$pages_dir"
else
  mkdir -p "$pages_dir"
  git -C "$pages_dir" init
  git -C "$pages_dir" remote add "$remote" "$(git remote get-url "$remote")"
  git -C "$pages_dir" checkout -b "$branch"
fi

rm -rf "$pages_dir/$target_dir"
mkdir -p "$pages_dir/$target_dir"
cp -R "$source_dir"/. "$pages_dir/$target_dir"/

touch "$pages_dir/.nojekyll"

versions_tmp=$(mktemp)
if [ -d "$pages_dir/stable" ]; then
  printf '%s\n' stable >> "$versions_tmp"
fi
find "$pages_dir" -mindepth 1 -maxdepth 1 -type d -name 'v[0-9]*.[0-9]*' -exec basename {} \; \
  | sed -n 's/^v\([0-9][0-9]*\)\.\([0-9][0-9]*\)$/\1 \2 v\1.\2/p' \
  | sort -k1,1nr -k2,2nr \
  | awk '{print $3}' >> "$versions_tmp"

cat > "$pages_dir/index.html" <<'EOF'
<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>idp-register documentation</title>
    <style>
      :root {
        color-scheme: light dark;
        font-family: system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
      }
      body {
        margin: 0;
        padding: 3rem 1.5rem;
        background: Canvas;
        color: CanvasText;
      }
      main {
        max-width: 48rem;
        margin: 0 auto;
      }
      h1 {
        margin: 0 0 0.75rem;
        font-size: 2rem;
        line-height: 1.2;
      }
      p {
        margin: 0 0 1.5rem;
        color: color-mix(in srgb, CanvasText 72%, Canvas 28%);
      }
      ul {
        list-style: none;
        padding: 0;
        margin: 0;
        border-top: 1px solid color-mix(in srgb, CanvasText 18%, Canvas 82%);
      }
      li {
        border-bottom: 1px solid color-mix(in srgb, CanvasText 18%, Canvas 82%);
      }
      a {
        display: flex;
        justify-content: space-between;
        gap: 1rem;
        padding: 0.9rem 0;
        color: LinkText;
        text-decoration: none;
      }
      a:hover {
        text-decoration: underline;
      }
      .label {
        color: color-mix(in srgb, CanvasText 62%, Canvas 38%);
      }
    </style>
  </head>
  <body>
    <main>
      <h1>idp-register documentation</h1>
      <p>Select a documentation version.</p>
      <ul>
EOF

while IFS= read -r version; do
  [ -n "$version" ] || continue
  label=$version
  if [ "$version" = "stable" ]; then
    label="stable"
  fi
  printf '        <li><a href="%s/"><span>%s</span><span class="label">%s</span></a></li>\n' \
    "$version" \
    "$label" \
    "$version" >> "$pages_dir/index.html"
done < "$versions_tmp"

cat >> "$pages_dir/index.html" <<'EOF'
      </ul>
    </main>
  </body>
</html>
EOF

rm -f "$versions_tmp"

# This script runs as root (the mdBook build step needs apk), so the cloned
# .git and copied files are root-owned. The next pipeline step pushes this tree
# with the appleboy/drone-git-push plugin, whose image runs as a non-root user
# (appuser) and would otherwise fail with "could not lock config file
# .git/config: Permission denied". Hand off a world-writable tree so the
# non-root plugin can configure, commit, and push it.
chmod -R a+rwX "$pages_dir"
