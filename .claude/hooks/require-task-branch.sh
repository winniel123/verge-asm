#!/usr/bin/env bash
# require-task-branch.sh
#
# PreToolUse hook for Edit, Write, and NotebookEdit.
# It blocks a file change when the target file sits on the trunk branch.
# verge-asm lands every change through a pull request from a task branch.
#
# Exit 0 permits the tool call. Exit 2 blocks it and sends stderr to the agent.
# The hook fails open. Any internal error exits 0.

set -u

command -v jq >/dev/null 2>&1 || exit 0
command -v git >/dev/null 2>&1 || exit 0

raw=$(cat) || exit 0
printf '%s' "$raw" | grep -q '[^[:space:]]' || exit 0

path=$(printf '%s' "$raw" |
    jq -r '.tool_input.file_path // .tool_input.notebook_path // empty' 2>/dev/null) || exit 0
[ -n "$path" ] || exit 0

# Walk up to the first directory that exists. A new file has no parent yet.
dir=$(dirname -- "$path")
while [ ! -d "$dir" ]; do
    parent=$(dirname -- "$dir")
    [ "$parent" != "$dir" ] || exit 0
    dir=$parent
done

branch=$(git -C "$dir" rev-parse --abbrev-ref HEAD 2>/dev/null) || exit 0
[ -n "$branch" ] || exit 0

case "$branch" in
    main | master) ;;
    *) exit 0 ;;
esac

tool=$(printf '%s' "$raw" | jq -r '.tool_name // "The tool"' 2>/dev/null)
[ -n "$tool" ] || tool="The tool"

cat >&2 <<EOF
BLOCKED: $tool targets a file on branch '$branch'.

verge-asm forbids a file change on the trunk. Every change lands through a
pull request from a task branch. Start the work again in this order:

  1. Call the EnterWorktree tool. Use the name '<type>/<kebab-summary>',
     or '<type>/<issue#>-<summary>' when an issue exists.
  2. EnterWorktree renames the branch. Restore the standard name:
     git branch -m <type>/<kebab-summary>
  3. Confirm with 'git status -sb', then retry this edit.

Read the 'Start of work' section in CLAUDE.md.
Target file: $path
EOF

exit 2
