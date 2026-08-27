#!/usr/bin/env bash
# Locate claim sources and stub signatures in a repo.
# Usage: find_claim_sources.sh [repo_root] [--stubs-only|--claims-only]
# Read-only; prints paths and grep hits. Safe to run repeatedly.

set -u
ROOT="${1:-.}"
MODE="${2:-all}"

EXCLUDE='--exclude-dir=.git --exclude-dir=node_modules --exclude-dir=vendor --exclude-dir=venv --exclude-dir=.venv --exclude-dir=dist --exclude-dir=build --exclude-dir=__pycache__ --exclude-dir=.next --exclude-dir=target'

section() { printf '\n== %s ==\n' "$1"; }

if [ "$MODE" != "--stubs-only" ]; then
  section "Docs / specs / roadmaps"
  find "$ROOT" -type d \( -name .git -o -name node_modules -o -name vendor -o -name venv -o -name .venv \) -prune -o \
    -type f \( -iname 'README*' -o -iname 'ROADMAP*' -o -iname 'CHANGELOG*' -o -iname 'ARCHITECTURE*' -o -iname 'SPEC*' \
      -o -iname 'DESIGN*' -o -iname 'FEATURES*' -o -iname 'TODO*' -o -path '*/docs/*.md' -o -path '*/spec*/*.md' \
      -o -path '*/rfcs/*' -o -path '*/adr*/*' \) -print 2>/dev/null | sort

  section "API contracts"
  find "$ROOT" -type d \( -name .git -o -name node_modules -o -name vendor \) -prune -o \
    -type f \( -iname 'openapi*' -o -iname 'swagger*' -o -iname '*.graphql' -o -iname '*.gql' -o -iname '*.proto' \
      -o -iname 'schema.*' \) -print 2>/dev/null | sort

  section "Route / command registrations (likely entry points)"
  grep -rIn $EXCLUDE -E \
    '@(app|router|api|bp|blueprint)\.(get|post|put|patch|delete|route)\(|(router|app)\.(get|post|put|patch|delete)\(|add_command\(|@click\.(command|group)|\.command\(|subparsers\.add_parser\(|cobra\.Command\{|\.Handle(Func)?\(|path\(|re_path\(' \
    "$ROOT" 2>/dev/null | head -300

  section "UI navigation / menu / feature-list strings"
  grep -rIn $EXCLUDE -iE \
    '(nav|menu|sidebar|route)[a-z]*\s*[:=]\s*[\[{]|"(label|title|name)"\s*:\s*"|<(NavLink|Link|MenuItem|Tab)\b' \
    "$ROOT" --include='*.tsx' --include='*.jsx' --include='*.vue' --include='*.svelte' --include='*.ts' --include='*.js' --include='*.html' 2>/dev/null | head -200

  section "Config schema / example config keys"
  find "$ROOT" -type d \( -name .git -o -name node_modules -o -name vendor \) -prune -o \
    -type f \( -iname '*.example.*' -o -iname 'config.*' -o -iname '*.sample.*' -o -iname 'settings.*' \) -print 2>/dev/null | sort | head -50
fi

if [ "$MODE" != "--claims-only" ]; then
  section "Stub signatures in non-test code"
  grep -rIn $EXCLUDE --exclude-dir=test --exclude-dir=tests --exclude-dir=__tests__ --exclude-dir=spec \
    -E 'TODO|FIXME|XXX|HACK|NotImplemented|not implemented|not yet implemented|coming soon|unimplemented!|todo!\(|^\s*pass\s*$|^\s*\.\.\.\s*$|return (None|\[\]|\{\}|null|nil)\s*(#.*|//.*)?$|console\.log\(.(stub|todo|not impl)|placeholder|dummy|mock[A-Z_]?|fake[A-Z_]?|sample_?data|fixture' \
    "$ROOT" 2>/dev/null | grep -viE '(_test|\.test\.|\.spec\.|/tests?/|/fixtures?/|/mocks?/)' | head -400

  section "Feature flags (check defaults and whether gated code is complete)"
  grep -rIn $EXCLUDE -iE 'feature_?flag|is_enabled\(|flags?\.[a-z_]+\b\s*(==|===)|ENABLE_[A-Z_]+|FF_[A-Z_]+' "$ROOT" 2>/dev/null | head -100

  section "Empty-body handlers (Python/JS quick heuristic)"
  grep -rIn $EXCLUDE -A2 -E '^\s*(async\s+)?def\s+\w+\(|^\s*(async\s+)?\w+\s*\([^)]*\)\s*(=>)?\s*\{' "$ROOT" --include='*.py' --include='*.js' --include='*.ts' --include='*.tsx' 2>/dev/null \
    | grep -E '^\S+-[0-9]+-\s*(pass|\.\.\.|\}|return;?|return (None|null|\[\]|\{\});?)\s*$' | head -100
fi

printf '\nDone. Treat every hit as a lead, not a verdict — read the surrounding code.\n'
