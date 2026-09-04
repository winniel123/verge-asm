# Issue tracker: GitHub

Issues and PRDs for this repo live as GitHub issues. Use the `gh` CLI for all operations.

## Conventions

- **Create an issue**: `gh issue create --title "..." --body "..."`. Use a heredoc for a multi-line body.
- **Read an issue**: `gh issue view <number> --json number,title,body,labels,state,comments`. Filter comments with `jq`. The bare `gh issue view <number>` and `--comments` forms fail on this repo. Read "The Projects-classic trap" below.
- **List issues**: `gh issue list --state open --json number,title,body,labels,comments --jq '[.[] | {number, title, body, labels: [.labels[].name], comments: [.comments[].body]}]'` with the `--label` and `--state` filters you need.
- **Comment on an issue**: `gh issue comment <number> --body "..."`
- **Apply / remove labels**: `gh issue edit <number> --add-label "..."` / `--remove-label "..."`
- **Close**: `gh issue close <number> --comment "..."`

`gh` infers the repo from `git remote -v`. It does this automatically when you run it inside a clone.

## Pull requests as a triage surface

**PRs as a request surface: no.** _(Set to `yes` if this repo treats external PRs as feature requests. `/triage` reads this flag.)_

When set to `yes`, PRs run through the same labels and states as issues. Use the `gh pr` equivalents:

- **Read a PR**: `gh pr view <number> --json number,title,body,labels,state,comments` and `gh pr diff <number>` for the diff. The bare and `--comments` forms fail on this repo. Read "The Projects-classic trap" below.
- **List external PRs for triage**: `gh pr list --state open --json number,title,body,labels,author,authorAssociation,comments`. Then keep only `authorAssociation` of `CONTRIBUTOR`, `FIRST_TIME_CONTRIBUTOR`, or `NONE`. Drop `OWNER`/`MEMBER`/`COLLABORATOR`.
- **Comment / label / close**: `gh pr comment`, `gh pr close`. **`gh pr edit --add-label` fails on this repo.** Label a PR with `gh api` instead. Read "The Projects-classic trap" below.

GitHub shares one number space across issues and PRs. So a bare `#42` may be either. Resolve it with `gh pr view 42 --json number`. If that fails, use `gh issue view 42 --json number`.

## When a skill says "publish to the issue tracker"

Create a GitHub issue.

## When a skill says "fetch the relevant ticket"

Run `gh issue view <number> --json number,title,body,labels,state,comments`.

## The Projects-classic trap

**Every `gh` subcommand that renders a default terminal view fails on this repo.** A session measured this on 2026-09-03 with `gh` 2.45.0. Each such command prints:

```
GraphQL: Projects (classic) is being deprecated in favor of the new Projects experience, see:
https://github.blog/changelog/2024-05-23-sunset-notice-projects-classic/. (repository.issue.projectCards)
```

The default view asks GitHub for `projectCards`, and GitHub retired that field. These commands fail: `gh issue view <n>`, `gh issue view <n> --comments`, `gh pr view <n>`, and `gh pr edit --add-label`/`--remove-label`.

Two workarounds answer this, and a session measured both:

- **Read**: add `--json`. `gh issue view <n> --json number,title,body,labels,state,comments` and `gh pr view <n> --json number,labels` both work.
- **Label a PR**: `gh api --method POST repos/<owner>/<repo>/issues/<n>/labels -f "labels[]=<label>"`. The REST issues endpoint reaches a PR by its number.

`gh issue list`, `gh pr list`, `gh issue edit` (including `--add-label` and `--body-file`), `gh issue comment`, `gh pr comment` and every `gh api` call are unaffected.

## The `gh pr update-branch` trap

**`gh pr update-branch` does not exist in `gh` 2.45.0**, the version on the dev machine. `CLAUDE.md`, `docs/spec/comment-policy.md` §7.5 and every stage-D sweep ticket instruct it. The command runs `gh pr --help` and updates nothing, so a session can read it as success.

Update a PR branch through the REST endpoint instead:

```sh
gh api --method PUT repos/winniel123/verge-asm/pulls/<n>/update-branch
```

`main` uses a strict up-to-date policy, so run this on each later branch after an earlier PR merges. The call re-triggers CI.

## Wayfinding operations

`/wayfinder` uses these operations. The **map** is a single issue. Its **child** issues are the tickets.

- **Map**: a single issue labelled `wayfinder:map`. It holds the Notes / Decisions-so-far / Fog body. Create it with `gh issue create --label wayfinder:map`.
- **Child ticket**: an issue linked to the map as a GitHub sub-issue. Use `gh api` on the sub-issues endpoint. If sub-issues are not enabled, add the child to a task list in the map body. Then put `Part of #<map>` at the top of the child body. Labels: `wayfinder:<type>` (`research`/`prototype`/`grilling`/`task`). Once you claim the ticket, assign it to the driving dev.
- **Blocking.** Use GitHub native issue dependencies. This is the canonical, UI-visible representation. To add a blocking edge:
  1. Get the blocker's numeric **database id**. Run `gh api repos/<owner>/<repo>/issues/<n> --jq .id`.
  2. Do not use the `#number`. Do not use the `node_id`.
  3. Add the edge. Run `gh api --method POST repos/<owner>/<repo>/issues/<child>/dependencies/blocked_by -F issue_id=<blocker-db-id>`.

  If dependencies are not available, use a `Blocked by: #<n>, #<n>` line at the top of the child body instead. A ticket is unblocked when every blocker is **closed**. The gate is each blocker's `state`, never a count. See the two traps below.
- **Frontier query**: list the map's open children. Drop any with an open blocker or an assignee. First in map order wins. Both list endpoints paginate, so both need `--paginate`:

  ```sh
  # open children of the map
  gh api repos/<owner>/<repo>/issues/<map>/sub_issues --paginate \
    --jq '.[] | select(.state=="open") | .number'

  # per child: open blockers, and the assignee that marks it claimed
  gh api repos/<owner>/<repo>/issues/<n>/dependencies/blocked_by --paginate \
    --jq '[.[] | select(.state=="open") | .number]'
  gh api repos/<owner>/<repo>/issues/<n> --jq '[.assignees[].login]'
  ```

  A child is on the frontier when the open-blocker list **and** the assignee list are both empty.

  **Two traps, both measured on this repo — a session that hits either misreads the frontier:**

  1. **`issue_dependencies_summary.blocked_by` counts every blocker, open or closed.** It is *not* a live gate. After closing #30, its only dependent #29 still reported `blocked_by=1` while being genuinely unblocked. Filter the `blocked_by` list on `state` instead. Never test the summary count.
  2. **These endpoints return only the first 30 entries without `--paginate`.** #12's blocker list read 30 unpaginated and 36 paginated. It silently hid six edges, including one that had just been added. Truncation looks exactly like a missing dependency. So omitting `--paginate` invents frontier tickets that are actually blocked.
- **Claim**: run `gh issue edit <n> --add-assignee @me`. This is the session's first write.
- **Resolve**: run `gh issue comment <n> --body "<answer>"`. Then run `gh issue close <n>`. Then append a context pointer (gist + link) to the map's Decisions-so-far.

## Implementation-map operations

`/to-tickets` and `/implement` use these. An **implementation map** is the parent issue `/to-tickets` cuts from a finished SPEC. It has the same body structure as a wayfinder map, but it is not one: a wayfinder map plans, an implementation map builds.

- **Map**: a single issue labelled `implementation:map`. `/to-tickets` applies the label when it creates the parent. Create it with `gh issue create --label implementation:map`. The label is the only reliable signal that an issue is a map and not a ticket, so never omit it.
- **Child ticket**: same shape as a wayfinder child. Link it as a GitHub sub-issue, record `Blocked by` edges as native issue dependencies, and list it in the map body's Tickets section.
- **Frontier query**: identical to the wayfinder frontier query above, including both traps. Use `--paginate`, and filter blockers on `state`.
- **One ticket per session.** A session that runs `/implement` against an `implementation:map` takes the first frontier ticket, implements it, opens its PR, and stops. It does not continue to the next ticket. This keeps one PR to one ticket, so the review and the blame stay readable.
- **Resolve**: close the ticket when its PR merges. Then append a one-line context pointer (gist + link) to the map's Decisions-so-far. The map closes when every child is closed.
