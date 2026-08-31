# require-task-branch.ps1
#
# PreToolUse hook for Edit, Write, and NotebookEdit.
# It blocks a file change when the target file sits on the trunk branch.
# verge-asm lands every change through a pull request from a task branch.
#
# Exit 0 permits the tool call. Exit 2 blocks it and sends stderr to the agent.
# The hook fails open. Any internal error exits 0.

$ErrorActionPreference = 'Continue'

try {
    $raw = [Console]::In.ReadToEnd()
    if ([string]::IsNullOrWhiteSpace($raw)) { exit 0 }

    $payload = $raw | ConvertFrom-Json -ErrorAction Stop
    if ($null -eq $payload) { exit 0 }

    $path = $payload.tool_input.file_path
    if ([string]::IsNullOrWhiteSpace($path)) {
        $path = $payload.tool_input.notebook_path
    }
    if ([string]::IsNullOrWhiteSpace($path)) { exit 0 }

    # Walk up to the first directory that exists. A new file has no parent yet.
    $dir = Split-Path -Parent $path
    while ($dir -and -not (Test-Path -LiteralPath $dir -PathType Container)) {
        $dir = Split-Path -Parent $dir
    }
    if ([string]::IsNullOrWhiteSpace($dir)) { exit 0 }

    $branch = & git -C $dir rev-parse --abbrev-ref HEAD 2>$null
    if ($LASTEXITCODE -ne 0) { exit 0 }   # Not a git repo. Scratchpad files pass.
    if ($null -eq $branch) { exit 0 }
    $branch = "$branch".Trim()

    if ($branch -ne 'main' -and $branch -ne 'master') { exit 0 }

    $tool = $payload.tool_name
    $message = @"
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
"@

    [Console]::Error.WriteLine($message)
    exit 2
}
catch {
    exit 0
}
