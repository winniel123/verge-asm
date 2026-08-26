# Work order — rulings #33–#35 + DF-F4b terminal badges (package v3.17.0)

Use the consume-design-package skill. Templates changed: shell.tmpl, rundetail.tmpl. fixtures.json + states.json updated. Land wholesale.

## Behavior
- **#35 id split**: the Settings active dispatch is now id 1409 (href /runs/1409; job hrefs /runs/1409?job={id}). error·missing-run keeps /runs/1408. Re-enable the parked running-run demo against 1409; the new rundetail·running state (/runs/1409) gets a golden this round.
- **#34 loghead**: rundetail loghead now renders outside {{if .Log}} — the DF-F3b unknown-job edge (chip over "No log to show") renders with zero repo change. Empty-log body is a 300px centered well inside rd-log.
- **DF-F4b**: .Run.Status gains stopped | terminated tokens (class + literal label on the rd-batch badge; stopped = warn, terminated = danger-outline on sunken). Replace the interim failed-mapping: pass the real token. BatchStatus in the component kit carries the same two states.
- **#33 org switcher retired**: shell renders only the static org chip. .Chrome.Orgs is gone from the contract — delete any org-switch plumbing/stubs and the org-open harness state (removed from states.json here). /org/switch never ships. Behavior otherwise: no change.

## Gates
G1: land wholesale. G2: shell goldens regen (org popover markup gone — default renders identically since fixtures now pass orgs:null, but bytes changed); rundetail·default + new rundetail·running; the two scans dialog states re-capture at 1409 (pixel-identical content).
