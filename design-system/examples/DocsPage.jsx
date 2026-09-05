import React from "react";
import { Input } from "../../components/forms/Input.jsx";
import { Table } from "../../components/display/Table.jsx";
import { SeverityBadge } from "../../components/display/SeverityBadge.jsx";
import { Icon } from "../../components/media/Icon.jsx";
import { CodeBlock } from "../../components/display/CodeBlock.jsx";
import { Kbd } from "../../components/display/Kbd.jsx";
import { Breadcrumb } from "../../components/navigation/Breadcrumb.jsx";
import { Callout } from "../../components/feedback/Callout.jsx";
import { Logo } from "../../components/media/Logo.jsx";
import { VersionSelect } from "../../components/navigation/VersionSelect.jsx";
import { CommandPalette } from "../../components/feedback/CommandPalette.jsx";

/* Bodies use only markdown-expressible blocks; the signals severity table is the exception. */

const P = { t: "p" }, H = { t: "h2" };
const GUIDES = [
  { slug: "using", title: "Using verge-asm", section: "Getting started", desc: "A tour of the operator workflow, from first login to reading drift. Assumes the stack is already up.", blocks: [
    { t: "h2", x: "First-run setup" },
    { t: "p", x: "With no accounts in the database, the console opens a one-time setup window and logs a single-use token. Paste it at `/setup` to mint your admin account; the window closes as soon as the account exists." },
    { t: "code", title: "shell", x: "docker compose logs web | grep /setup\n# web: no accounts yet \u2014 open /setup with this single-use token: <token>" },
    { t: "h2", x: "The four-step checklist" },
    { t: "p", x: "The home page renders your Coverage as a checklist. Each step unlocks a capability; until you complete them, the system says what it cannot yet conclude rather than guessing." },
    { t: "list", ordered: true, items: ["Declare a seed \u2014 a name scope (`acmecorp.io`) or an address scope (`203.0.113.0/24`). A seed is a boundary, not a starting gun.", "Upload a zone file to enable removal detection \u2014 supply is a dated act.", "Add an internet vantage \u2014 provision a prober; exposure needs an outside observer.", "Run the first batch \u2014 scans dispatch on cadence, or trigger one from the worker."] },
    { t: "h2", x: "Reading what it found" },
    { t: "p", x: "Once a batch commits, each page answers a different question \u2014 Coverage before you conclude the estate is empty, Inventory for what you hold now, Exposure for what the internet reaches, Drift for what moved." },
    { t: "callout", x: "Declared is your input and never drifts. Observed is what was measured. Derived is what was concluded \u2014 and two derived values are comparable only within one identical derivation." },
  ] },
  { slug: "first-run", title: "First-run mental model", section: "Getting started", desc: "The three layers, why Exposure needs two legs, and how to tell a scan that ran from one that failed in silence.", blocks: [
    { t: "h2", x: "Declared, Observed, Derived" },
    { t: "p", x: "Everything you assert \u2014 seeds, exclusions, custody \u2014 is Declared and never drifts. Everything a scan measures is Observed. Everything concluded from observations is Derived, and a Break marks where two derivations may not be compared." },
    { t: "h2", x: "Why Exposure needs two legs" },
    { t: "p", x: "A single install can build an honest internal inventory, but probing your own public address from inside is a hairpinning trap. An exposure verdict exists only where the internal leg and the internet leg both hold a value \u2014 one-legged readings get no name." },
    { t: "h2", x: "Confirm a scan actually ran" },
    { t: "p", x: "A scan that ran but resolved nothing commits a `Gap`, not an error \u2014 the period over which we could not say. Read Coverage before concluding the estate is empty; a wrong resolver fails silently." },
    { t: "code", title: "shell", x: "docker compose logs -f worker\n# dispatcher: dns fan-out committed \u00b7 batch 2026-08-22T14:00Z" },
  ] },
  { slug: "zone-files", title: "Zone files", section: "Getting started", desc: "What a zone file is, why removal detection needs one, and how to export one from your provider.", blocks: [
    { t: "h2", x: "Why removal detection needs a zone" },
    { t: "p", x: "Discovery finds what appears; only your zone says what should be there, so a name you deleted can be noticed as gone. The upload instant is the observation instant \u2014 *you stopped telling us* becomes detectable." },
    { t: "h2", x: "Upload and re-supply" },
    { t: "p", x: "Upload under Scope against a name scope; a file whose apex sits outside that scope is refused with the reason. The zone is covered by its own `zone` scan on the re-supply interval you set (shipped monthly)." },
    { t: "callout", tone: "warn", x: "Let a zone age past two re-supply intervals and it ages into a Gap \u2014 the system tells you the source went stale instead of trusting old bytes." },
    { t: "h2", x: "Exporting from your name servers" },
    { t: "code", title: "shell", x: "dig axfr acmecorp.io @ns1.acmecorp.io > acmecorp.io.zone" },
  ] },
  { slug: "reading-the-estate", title: "Reading your attack surface", section: "Getting started", desc: "Where to look for each answer \u2014 coverage, exposure, drift, inventory, the graph, search, and a single scan run.", blocks: [
    { t: "h2", x: "The read surfaces at a glance" },
    { t: "table", cols: [{ key: "q", label: "The question", w: 260 }, { key: "page", label: "Page", w: 110 }, { key: "route", label: "Route", mono: true }], rows: [
      { q: "Is what I am looking at complete?", page: "Coverage", route: "/coverage" },
      { q: "What do I have right now?", page: "Inventory", route: "/inventory" },
      { q: "What is reachable from the internet?", page: "Exposure", route: "/exposure" },
      { q: "What moved since last time?", page: "Drift", route: "/drift" },
      { q: "How does it all connect?", page: "Graph", route: "/graph" },
      { q: "Did a scan run, and what did it touch?", page: "Run detail", route: "/run/<id>" },
    ] },
    { t: "h2", x: "Exporting" },
    { t: "p", x: "Inventory, Signals and Drift export to CSV \u2014 a read of the same facts the page shows, never a mutation. An empty corpus yields a header-only file, never invented rows." },
    { t: "callout", x: "The old top-level Subjects listing now redirects to Inventory \u2014 read *Subjects* as *Inventory plus its drill-downs*." },
  ] },
  { slug: "sources", title: "Discovery sources", section: "Scanning", desc: "Enable or disable discovery sources; a toggle is a dated, audit-trailed act.", blocks: [
    { t: "h2", x: "Consent tiers" },
    { t: "p", x: "Each source names what it reads and where that data goes. Sources that only read public datasets ship enabled; sources that send your seeds to a third party sit behind a terms dialog that gates enabling." },
    { t: "h2", x: "Toggling is a dated act" },
    { t: "p", x: "Enablement is recorded on the batch whose source set it moved, so *why did discovery widen* is always answerable. Viewers read the catalogue; only admins toggle." },
  ] },
  { slug: "prober", title: "Provisioning a prober", section: "Scanning", desc: "Stand up the second Linux host that gives Verge its internet vantage.", blocks: [
    { t: "h2", x: "Why a second host" },
    { t: "p", x: "Exposure requires an outside observer, unconditionally. The prober is a minimal, hardened SSH target; the instance generates the keypair and only the public half ever leaves." },
    { t: "h2", x: "Provision in four values" },
    { t: "p", x: "Supply host, port, and a non-root username \u2014 the instance renders the public key to install in that account's `authorized_keys`. On first connection it reads your egress address and offers it for declaration as an address scope." },
    { t: "code", title: "shell", x: "cd deploy/prober && docker compose up -d\n# paste the rendered public key into authorized_keys" },
    { t: "callout", tone: "warn", x: "The host key is pinned at provisioning. A later change is a hard failure, never a prompt." },
  ] },
  { slug: "signals", title: "Signals reference", section: "Signals & delivery", desc: "The rule set, severity levels, and the annotation dial.", blocks: [
    { t: "h2", x: "Severity levels" },
    { t: "p", x: "Signals use exactly five levels, ordered. The words never change; write them as shown." },
    { t: "sev" },
    { t: "h2", x: "Withdrawn, never resolved" },
    { t: "p", x: "A signal is raised when the world moves your estate into a rule's population, and withdrawn just as quietly when the world moves back. No operator resolves a signal \u2014 the world does." },
    { t: "h2", x: "Annotations" },
    { t: "p", x: "An annotation accepts one rule's firing on one subject as a known risk. It moves no number \u2014 the subject is still measured, still counted \u2014 and changes only the message: an annotated pair's next firing reaches no one." },
  ] },
  { slug: "notification-channels", title: "Notification channels", section: "Signals & delivery", desc: "Where messages go: HTTPS channels, signing secrets, and the delivery record.", blocks: [
    { t: "h2", x: "Declare a channel" },
    { t: "p", x: "A channel is an absolute `https` URL, an optional signing secret, and the subset of classes it carries (drift, coverage, clock). None ships configured \u2014 nothing is routed anywhere until an admin declares one." },
    { t: "h2", x: "What a message carries" },
    { t: "callout", x: "A channel carries the message, never the estate. Set `VERGE_PUBLIC_URL` so bodies link back to your instance; empty leaves the link off rather than fabricating one." },
    { t: "h2", x: "The delivery record" },
    { t: "p", x: "Every outbound POST's outcome is recorded. An undelivered message is legible \u2014 *this could not be delivered*, joined to the message it failed to carry \u2014 and never touches Coverage." },
  ] },
  { slug: "reports", title: "Reports", section: "Signals & delivery", desc: "Recurring reports and the delivered artifact.", blocks: [
    { t: "h2", x: "Recurring reports" },
    { t: "p", x: "Schedule a recurring report to a channel; each delivery renders the period's movement \u2014 signals raised and withdrawn, coverage change, scan activity \u2014 into a single artifact." },
    { t: "h2", x: "The delivered artifact" },
    { t: "p", x: "Open any delivery from the Reports row menu (*View last delivery*). The artifact doubles as the PDF rendering \u2014 a channel receives only a link-only ready-message, so what you see here is the report; the body never leaves the instance." },
  ] },
  { slug: "integrations", title: "Integrations", section: "Signals & delivery", desc: "The tile catalogue: install states and consent.", blocks: [
    { t: "h2", x: "Tiles and consent" },
    { t: "p", x: "Each integration names what it reads and what it sends before you connect it. Install state is real operator data \u2014 *available*, *needs config*, or *installed* \u2014 never fabricated." },
    { t: "h2", x: "Channels are not integrations" },
    { t: "p", x: "A channel is a one-way URL you declare; an integration is a two-sided connection with its own consent. The Settings tabs keep them apart on purpose." },
  ] },
  { slug: "accounts", title: "Accounts, invites & roles", section: "Access", desc: "Two roles, join links, and offboarding.", blocks: [
    { t: "h2", x: "Two roles" },
    { t: "table", cols: [{ key: "r", label: "Role", w: 100 }, { key: "d", label: "What it can do" }], rows: [
      { r: "admin", d: "performs declared acts \u2014 seeds, scans, channels, annotations, team, instance" },
      { r: "viewer", d: "reads everything, changes nothing \u2014 including the sources catalogue" },
    ] },
    { t: "h2", x: "Invites" },
    { t: "p", x: "An invite mints a join link shown once \u2014 Verge keeps only a hash. Hand it over out of band; the role applies on acceptance and the link expires in 7 days." },
    { t: "h2", x: "Removing a member" },
    { t: "p", x: "Removal is a typed-name confirm. Their annotations and audit history stay attributed; personal tokens are revoked. To sign a member out everywhere without removing them, revoke their sessions under Settings \u2192 Sessions." },
  ] },
  { slug: "authentication", title: "Authentication", section: "Access", desc: "Passwords, TOTP enrollment, personal tokens, and sessions.", blocks: [
    { t: "h2", x: "Two-factor enrollment" },
    { t: "p", x: "Enroll TOTP from your profile: scan the secret, confirm one code, store the recovery codes. An admin can require re-enrollment; the current authenticator stops working immediately." },
    { t: "h2", x: "Personal tokens" },
    { t: "p", x: "Mint API tokens from your profile. A token's value is shown once; the list shows *last used*, with an em dash for a token never yet presented \u2014 never a fabricated recency." },
    { t: "h2", x: "Sessions" },
    { t: "p", x: "Your own sessions live on your profile \u2014 end one, or sign out all others. Admins manage every account's sessions under Settings \u2192 Sessions." },
  ] },
  { slug: "sso", title: "Single sign-on (SSO)", section: "Access", desc: "OpenID Connect against an existing account \u2014 never account creation.", blocks: [
    { t: "h2", x: "OIDC only, accounts first" },
    { t: "p", x: "A provider authenticates an existing account by a linked, verified identity \u2014 bound to the provider's stable subject, never a mutable username \u2014 and never creates accounts. Header-trust reverse-proxy auth stays refused." },
    { t: "h2", x: "Linking" },
    { t: "p", x: "Users link a provider from their own profile once an admin enables it; each enabled provider renders a button on the sign-in screen. Passwords and two-factor keep working alongside." },
    { t: "callout", x: "Provider walkthroughs: SSO with Okta, Google Workspace, Microsoft Entra ID, and Keycloak \u2014 each in this section." },
  ] },
  { slug: "sso-okta", title: "SSO with Okta", section: "Access", desc: "Create the Okta app and paste three values.", blocks: [
    { t: "h2", x: "Create the app integration" },
    { t: "list", ordered: true, items: ["Applications \u2192 Create App Integration \u2192 OIDC \u00b7 Web Application.", "Sign-in redirect URI: `https://verge.example.com/login/sso/okta/callback`.", "Assign the people or groups who should sign in."] },
    { t: "h2", x: "Values to paste into Verge" },
    { t: "table", cols: [{ key: "k", label: "Field", w: 130 }, { key: "v", label: "Value", mono: true }], rows: [
      { k: "Issuer URL", v: "https://your-org.okta.com" },
      { k: "Client ID", v: "0oa8\u2026" },
      { k: "Client secret", v: "confidential \u00b7 never shown again" },
    ] },
  ] },
  { slug: "sso-google", title: "SSO with Google", section: "Access", desc: "Google Workspace via OAuth consent + OIDC.", blocks: [
    { t: "h2", x: "Create the OAuth client" },
    { t: "list", ordered: true, items: ["Google Cloud console \u2192 Credentials \u2192 Create OAuth client ID \u00b7 Web application.", "Authorized redirect URI: `https://verge.example.com/login/sso/google/callback`.", "Restrict the consent screen to your Workspace domain."] },
    { t: "h2", x: "Values to paste into Verge" },
    { t: "table", cols: [{ key: "k", label: "Field", w: 130 }, { key: "v", label: "Value", mono: true }], rows: [
      { k: "Issuer URL", v: "https://accounts.google.com" },
      { k: "Client ID", v: "\u2026apps.googleusercontent.com" },
      { k: "Client secret", v: "confidential \u00b7 never shown again" },
    ] },
  ] },
  { slug: "sso-entra-id", title: "SSO with Entra ID", section: "Access", desc: "Microsoft Entra ID app registration.", blocks: [
    { t: "h2", x: "Register the application" },
    { t: "list", ordered: true, items: ["Entra admin center \u2192 App registrations \u2192 New registration.", "Redirect URI (Web): `https://verge.example.com/login/sso/entra/callback`.", "Certificates & secrets \u2192 new client secret; note the value now."] },
    { t: "h2", x: "Values to paste into Verge" },
    { t: "table", cols: [{ key: "k", label: "Field", w: 130 }, { key: "v", label: "Value", mono: true }], rows: [
      { k: "Issuer URL", v: "https://login.microsoftonline.com/<tenant>/v2.0" },
      { k: "Client ID", v: "application (client) ID" },
      { k: "Client secret", v: "the secret value, not its ID" },
    ] },
  ] },
  { slug: "sso-keycloak", title: "SSO with Keycloak", section: "Access", desc: "A self-hosted realm as the identity provider.", blocks: [
    { t: "h2", x: "Create the client" },
    { t: "list", ordered: true, items: ["Your realm \u2192 Clients \u2192 Create client \u00b7 OpenID Connect, confidential.", "Valid redirect URI: `https://verge.example.com/login/sso/keycloak/callback`.", "Credentials tab \u2192 copy the client secret."] },
    { t: "h2", x: "Values to paste into Verge" },
    { t: "table", cols: [{ key: "k", label: "Field", w: 130 }, { key: "v", label: "Value", mono: true }], rows: [
      { k: "Issuer URL", v: "https://id.example.com/realms/acme" },
      { k: "Client ID", v: "verge" },
      { k: "Client secret", v: "confidential \u00b7 never shown again" },
    ] },
  ] },
  { slug: "running", title: "Running verge-asm", section: "Operating", desc: "Deploy, configure, and operate the stack with Docker Compose.", blocks: [
    { t: "h2", x: "First launch" },
    { t: "code", title: "shell", x: "cp .env.example .env\n$EDITOR .env                 # set POSTGRES_PASSWORD\ndocker compose up -d --build" },
    { t: "p", x: "`web` runs migrations on startup \u2014 no separate migrate step. The environment configures the process; the database configures the product: seeds, scans, channels and vantages are rows edited through the UI, because those acts need an author in the audit trail." },
    { t: "h2", x: "On-demand scan triggers" },
    { t: "code", title: "shell", x: "docker compose run --rm worker -trigger dns" },
    { t: "callout", tone: "warn", x: "`hot` and `cold` are active port scans \u2014 real TCP connections across the target ports. `cold` ships disabled and a trigger refuses a disabled scan." },
    { t: "h2", x: "Upgrades" },
    { t: "p", x: "`git pull && docker compose up -d --build`. The schema lands before new code serves traffic \u2014 take a `pgdata` backup first for anything you cannot roll forward through." },
  ] },
  { slug: "backup-and-restore", title: "Backup & restore", section: "Operating", desc: "The three volumes, consistent dumps, and what regenerates when lost.", blocks: [
    { t: "h2", x: "The three volumes" },
    { t: "table", cols: [{ key: "v", label: "Volume", mono: true, w: 120 }, { key: "h", label: "Holds" }, { key: "l", label: "Losing it means" }], rows: [
      { v: "pgdata", h: "the entire estate", l: "total data loss" },
      { v: "web-state", h: "session signing key", l: "all sessions invalidated; regenerated" },
      { v: "worker-state", h: "prober SSH private key", l: "vantages must re-install the new public key" },
    ] },
    { t: "h2", x: "A consistent dump" },
    { t: "code", title: "shell", x: "docker compose exec postgres pg_dump -U verge -Fc verge > verge-$(date +%F).dump" },
    { t: "callout", x: "A backup you have not restored is a hope, not a backup \u2014 test the restore path on a scratch stack." },
  ] },
  { slug: "troubleshooting", title: "Troubleshooting", section: "Operating", desc: "Silent failures and where each symptom is legible.", blocks: [
    { t: "h2", x: "A scan ran but found nothing" },
    { t: "p", x: "A wrong resolver fails silently: the scan commits as completed, resolves nothing, and Coverage shows a `Gap`. Off compose, point the `local` vantage at your own recursive resolver before the first `dns` trigger." },
    { t: "h2", x: "Vantage unreachable" },
    { t: "p", x: "Scans continue from your other vantages; the unreachable one is bannered on the dashboard and its timelines age toward staleness rather than inventing values." },
    { t: "h2", x: "Where to look" },
    { t: "table", cols: [{ key: "s", label: "Symptom" }, { key: "w", label: "Where it is legible", w: 220 }], rows: [
      { s: "Empty estate after a first scan", w: "Coverage \u2192 Gaps" },
      { s: "A message never arrived", w: "Settings \u2192 Delivery record" },
      { s: "A run seems stuck", w: "Settings \u2192 Scans (in flight)" },
    ] },
  ] },
  { slug: "verifying", title: "Verifying verge-asm", section: "Contributing", desc: "Build, test, regenerate code, and reproduce CI locally through Docker.", blocks: [
    { t: "h2", x: "The fast loop" },
    { t: "code", title: "shell", x: "docker run --rm -v \"$PWD\":/src -w /src golang:1.25-bookworm \\\n  sh -c 'go vet ./... && go test ./...'" },
    { t: "h2", x: "The four gates" },
    { t: "list", ordered: true, items: ["`go vet` and `go test` \u2014 clean.", "`go build` for amd64 and arm64 \u2014 both compile.", "`sqlc generate` \u2014 no diff under `internal/db`.", "`docker compose up` \u2014 the stack reaches healthy."] },
    { t: "callout", x: "Windows checkouts: the golden tests compare exact bytes \u2014 run them inside the Linux container so CRLF never touches them." },
  ] },
];

const SECTIONS = ["Getting started", "Scanning", "Signals & delivery", "Access", "Operating", "Contributing"];
const slugId = (s) => s.toLowerCase().replace(/[^\w\s-]/g, "").trim().replace(/\s+/g, "-");

function InlineCode({ children }) {
  return <code style={{ font: "400 0.9em var(--font-mono)", background: "var(--surface-sunken)", border: "1px solid var(--border-default)", borderRadius: 6, padding: "1px 5px", color: "var(--text-body)" }}>{children}</code>;
}
function Rich({ x }) {
  const parts = String(x).split("`");
  return <React.Fragment>{parts.map((p, i) => i % 2 ? <InlineCode key={i}>{p}</InlineCode> : <React.Fragment key={i}>{p}</React.Fragment>)}</React.Fragment>;
}
const pStyle = { margin: "12px 0 0", font: "400 15px/1.65 var(--font-ui)", color: "var(--text-body)" };

function Block({ b }) {
  if (b.t === "h2") return <h2 id={slugId(b.x)} style={{ margin: "36px 0 0", font: "600 21px var(--font-ui)", letterSpacing: "var(--heading-tracking)", color: "var(--text-ink)" }}>{b.x}</h2>;
  if (b.t === "p") return <p style={pStyle}><Rich x={b.x} /></p>;
  if (b.t === "code") return <div style={{ marginTop: 14 }}><CodeBlock title={b.title || "shell"}>{b.x}</CodeBlock></div>;
  if (b.t === "callout") return <Callout tone={b.tone} style={{ marginTop: 16 }}><Rich x={b.x} /></Callout>;
  if (b.t === "list") {
    const Tag = b.ordered ? "ol" : "ul";
    return <Tag style={{ margin: "12px 0 0", padding: "0 0 0 22px", display: "flex", flexDirection: "column", gap: 8 }}>
      {b.items.map((it, i) => <li key={i} style={{ font: "400 15px/1.6 var(--font-ui)", color: "var(--text-body)" }}><Rich x={it} /></li>)}
    </Tag>;
  }
  if (b.t === "table") return <div style={{ marginTop: 16 }}>
    <Table columns={b.cols.map((c) => ({ key: c.key, label: c.label, mono: c.mono, width: c.w, clip: false, render: (r) => <span style={{ font: (c.mono ? "400 12.5px var(--font-mono)" : "400 13px/1.5 var(--font-ui)"), color: "var(--text-body)", whiteSpace: "normal", overflowWrap: "anywhere" }}>{r[c.key]}</span> }))} rows={b.rows} rowKey={b.cols[0].key} />
  </div>;
  if (b.t === "sev") return <div style={{ marginTop: 16 }}>
    <Table columns={[
      { key: "level", label: "Level", width: 120, render: (r) => <SeverityBadge level={r.level} size="sm" /> },
      { key: "meaning", label: "Raised when", render: (r) => <span style={{ font: "400 13px/1.5 var(--font-ui)", color: "var(--text-body)", whiteSpace: "normal" }}>{r.meaning}</span> },
    ]} rows={[
      { level: "critical", meaning: "An exposure is exploitable now \u2014 act before the next scan." },
      { level: "high", meaning: "A weakness attackers actively look for." },
      { level: "medium", meaning: "A misconfiguration that widens your surface." },
      { level: "low", meaning: "Hygiene: information leaks and loose ends." },
      { level: "info", meaning: "A change worth knowing about, no action implied." },
    ]} />
  </div>;
  return null;
}

export function DocsPage() {
  const [ver, setVer] = React.useState("v0.9.2");
  const [active, setActive] = React.useState("using");
  const [paletteOpen, setPaletteOpen] = React.useState(false);
  const guide = GUIDES.find((g) => g.slug === active) || GUIDES[0];
  const flat = SECTIONS.flatMap((s) => GUIDES.filter((g) => g.section === s));
  const idx = flat.indexOf(guide);
  const toc = guide.blocks.filter((b) => b.t === "h2").map((b) => b.x);
  const go = (slug, anchor) => {
    setActive(slug);
    setTimeout(() => {
      if (anchor) {
        const el = document.getElementById(slugId(anchor));
        if (el) window.scrollTo({ top: el.getBoundingClientRect().top + window.scrollY - 24, behavior: "smooth" });
      } else window.scrollTo({ top: 0 });
    }, 40);
  };
  React.useEffect(() => {
    const onKey = (e) => {
      if ((e.metaKey || e.ctrlKey) && (e.key === "k" || e.key === "K")) { e.preventDefault(); setPaletteOpen((v) => !v); }
    };
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, []);
  const paletteGroups = [{
    label: "Docs \u00b7 " + ver,
    items: flat.flatMap((g) => [
      { id: g.slug, label: g.title, icon: "file-text", hint: g.slug, onSelect: () => go(g.slug) },
    ].concat(g.blocks.filter((b) => b.t === "h2").map((b) => (
      { id: g.slug + "#" + slugId(b.x), label: g.title + " \u203a " + b.x, icon: "chevron-right", hint: g.slug, onSelect: () => go(g.slug, b.x) }
    )))),
  }];
  return (
    <div data-screen-label={"Docs \u00b7 " + guide.title} style={{ background: "var(--bg-page)", minHeight: "100vh", fontFamily: "var(--font-ui)" }}>
      <nav style={{ display: "flex", alignItems: "center", gap: 16, height: 56, padding: "0 24px", background: "var(--surface)", borderBottom: "1px solid var(--border-default)", position: "sticky", top: 0, zIndex: 30 }}>
        <Logo size={20} wordmarkSize={17} />
        <span style={{ font: "500 13px var(--font-ui)", color: "var(--text-secondary)", paddingLeft: 12, borderLeft: "1px solid var(--border-default)" }}>Docs</span>
        <div onClick={() => setPaletteOpen(true)} style={{ marginLeft: "auto", cursor: "text" }}>
          <Input size="sm" mono readOnly placeholder="Search docs" prefix={<Icon name="search" size={13} />} onFocus={() => setPaletteOpen(true)} style={{ width: 260, cursor: "text" }} />
        </div>
        <Kbd keys={["mod", "K"]} />
        <VersionSelect value={ver} onChange={setVer} versions={[{ value: "v0.9.2", tag: "current" }, { value: "v0.9.1" }, { value: "v0.8.7" }, { value: "main", tag: "dev" }]} />
        <a href="https://github.com/winniel123/verge-asm" target="_blank" rel="noreferrer noopener" style={{ font: "500 12px var(--font-ui)", color: "var(--text-secondary)", textDecoration: "none" }}>GitHub</a>
      </nav>
      <div style={{ maxWidth: 1240, margin: "0 auto", display: "grid", gridTemplateColumns: "220px minmax(0, 1fr) 180px", gap: 48, padding: "40px 32px 80px" }}>
        <aside style={{ display: "flex", flexDirection: "column", gap: 24, position: "sticky", top: 96, alignSelf: "start", maxHeight: "calc(100vh - 120px)", overflowY: "auto", overflowX: "hidden" }}>
          {SECTIONS.map((section) => (
            <div key={section} style={{ display: "flex", flexDirection: "column", gap: 4 }}>
              <span style={{ font: "500 11px var(--font-mono)", letterSpacing: "0.07em", textTransform: "uppercase", color: "var(--text-muted)", marginBottom: 4 }}>{section}</span>
              {GUIDES.filter((g) => g.section === section).map((g) => {
                const on = g.slug === active;
                return <button key={g.slug} type="button" onClick={() => go(g.slug)}
                  style={{ font: (on ? "600" : "400") + " 13px var(--font-ui)", color: on ? "var(--link)" : "var(--text-secondary)", textAlign: "left", border: "none", cursor: "pointer", padding: "5px 10px", borderRadius: 8, background: on ? "var(--accent-soft)" : "transparent" }}>{g.title}</button>;
              })}
            </div>
          ))}
        </aside>
        <article style={{ maxWidth: 760, minWidth: 0 }}>
          <Breadcrumb items={[{ label: "Docs" }, { label: guide.section }, { label: guide.title }]} />
          <h1 style={{ margin: "10px 0 0", font: "600 32px/1.15 var(--font-ui)", letterSpacing: "-0.015em", color: "var(--text-ink)" }}>{guide.title}</h1>
          <p style={{ margin: "16px 0 0", font: "400 15px/1.65 var(--font-ui)", color: "var(--text-body)" }}>{guide.desc}</p>
          {guide.blocks.map((b, i) => <Block key={i} b={b} />)}
          <div style={{ display: "flex", gap: 16, marginTop: 56, paddingTop: 24, borderTop: "1px solid var(--row-sep)" }}>
            {idx > 0 ? (
              <button type="button" onClick={() => go(flat[idx - 1].slug)} style={{ flex: 1, textAlign: "left", background: "var(--surface)", border: "1px solid var(--border-default)", borderRadius: 12, padding: "14px 16px", cursor: "pointer", display: "flex", flexDirection: "column", gap: 4 }}>
                <span style={{ font: "500 11px var(--font-mono)", letterSpacing: "0.07em", textTransform: "uppercase", color: "var(--text-muted)" }}>{"\u2190 Previous"}</span>
                <span style={{ font: "500 13.5px var(--font-ui)", color: "var(--text-ink)" }}>{flat[idx - 1].title}</span>
              </button>
            ) : <span style={{ flex: 1 }} />}
            {idx < flat.length - 1 ? (
              <button type="button" onClick={() => go(flat[idx + 1].slug)} style={{ flex: 1, textAlign: "right", background: "var(--surface)", border: "1px solid var(--border-default)", borderRadius: 12, padding: "14px 16px", cursor: "pointer", display: "flex", flexDirection: "column", gap: 4, alignItems: "flex-end" }}>
                <span style={{ font: "500 11px var(--font-mono)", letterSpacing: "0.07em", textTransform: "uppercase", color: "var(--text-muted)" }}>{"Next \u2192"}</span>
                <span style={{ font: "500 13.5px var(--font-ui)", color: "var(--text-ink)" }}>{flat[idx + 1].title}</span>
              </button>
            ) : <span style={{ flex: 1 }} />}
          </div>
        </article>
        <aside style={{ position: "sticky", top: 96, alignSelf: "start", display: "flex", flexDirection: "column", gap: 8 }}>
          <span style={{ font: "500 11px var(--font-mono)", letterSpacing: "0.07em", textTransform: "uppercase", color: "var(--text-muted)" }}>On this page</span>
          {toc.map((t) => (
            <button key={t} type="button" onClick={() => go(active, t)} style={{ font: "400 12.5px var(--font-ui)", color: "var(--text-secondary)", textAlign: "left", background: "transparent", border: "none", cursor: "pointer", padding: 0 }}>{t}</button>
          ))}
        </aside>
      </div>
      <CommandPalette open={paletteOpen} onClose={() => setPaletteOpen(false)} groups={paletteGroups} placeholder={"Search " + ver + " docs\u2026"} />
    </div>
  );
}
