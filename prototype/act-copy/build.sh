#!/usr/bin/env bash
# Assembles one self-contained file so the prototype opens on a double-click.
set -euo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
root="$(cd "$here/../.." && pwd)"
ds="$root/design-system"
tmpl="$ds/templates/settings.tmpl"
out="$here/act-copy.html"

# The st- classes live inline in the shipped template, so the prototype reads them from it
# rather than keeping a copy that can drift.
css_start=$(grep -n '^<style>$' "$tmpl" | head -1 | cut -d: -f1)
css_end=$(grep -n '^</style>$' "$tmpl" | head -1 | cut -d: -f1)
settings_css=$(sed -n "$((css_start + 1)),$((css_end - 1))p" "$tmpl")

# A CSS @import must precede every rule, so the font import is hoisted out of typography.css.
font_import=$(grep '^@import' "$ds/tokens/typography.css")

{
  echo '<!doctype html>'
  echo '<html lang="en"><head><meta charset="utf-8">'
  echo '<meta name="viewport" content="width=device-width, initial-scale=1">'
  echo '<title>Prototype — the copy a landed Act corpus makes true (#1792)</title>'
  echo '<style>'
  echo "$font_import"
  for f in colors typography spacing radius elevation motion base; do
    grep -v '^@import' "$ds/tokens/$f.css"
  done
  echo "$settings_css"
  cat "$here/prototype.css"
  echo '</style></head>'
  cat "$here/body.html"
  echo '<script>'
  cat "$here/prototype.js"
  echo '</script></html>'
} > "$out"

echo "wrote $out"
