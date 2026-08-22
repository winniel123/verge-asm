import React from "react";
import { Card } from "../../components/display/Card.jsx";
import { Tag } from "../../components/display/Tag.jsx";
import { MessageList } from "../../components/feedback/MessageList.jsx";
import { EmptyState } from "../../components/feedback/EmptyState.jsx";
import { SegmentedControl } from "../../components/forms/SegmentedControl.jsx";
import { Button } from "../../components/forms/Button.jsx";
import { Icon } from "../../components/media/Icon.jsx";

const BODY = {
  signals: "Vantage eu-west-1 concluded :5900 vnc on edge-gw-03.acmecorp.io is exposed to internet with no transport encryption. Raised as VG-2481, critical. The port appeared in batch 2026-08-22T14:00Z and was reachable on both legs.",
  drift: "staging-5.acmecorp.io appeared in batch 2026-08-22T14:00Z — first seen via certificate transparency, resolving to 203.0.113.88. It inherits the standard profile and joins the next census.",
  coverage: "Zone transfer for internal.acmecorp.io has been silent for 9 days. Discovery narrows while it stays quiet — names only that zone knew about will drop from resolution, not from scope.",
  batches: "Batch 2026-08-22T14:00Z completed across 3 vantages: 214 subjects, 7 transitions, 3 new signals. ap-south-1 missed 2 of 3 checks; its conclusions are marked unverified.",
};
const GOTO = { signals: ["Open signal", "signals"], drift: ["Open drift", "drift"], coverage: ["Open scope", "scope"], batches: ["Open batch detail", "run"] };

/* Every message Verge sent or would send — the mailbox icon's destination. */
const BASE_MSGS = [
    { id: "m1", cls: "signals", text: "VNC exposed to internet · edge-gw-03.acmecorp.io", time: "4m", unread: true },
    { id: "m2", cls: "drift", text: "appeared · staging-5.acmecorp.io", time: "8m", unread: true },
    { id: "m3", cls: "coverage", text: "zone transfer silent for 9d · internal.acmecorp.io", time: "6h" },
    { id: "m4", cls: "batches", text: "batch complete · 3 new signals", time: "6h" },
    { id: "m5", cls: "signals", text: "certificate expires in 23 days · idp-signing-2026", time: "2d" },
];

export function Inbox({ onNavigate, initialId }) {
  const [msgs, setMsgs] = React.useState(() => BASE_MSGS.map((m) => (m.id === initialId ? { ...m, unread: false } : m)));
  const [filter, setFilter] = React.useState("all");
  const [selId, setSelId] = React.useState(initialId || null);
  const shown = filter === "unread" ? msgs.filter((m) => m.unread) : msgs;
  const sel = msgs.filter((m) => m.id === selId)[0] || null;
  const open = (m) => { setSelId(m.id); setMsgs((all) => all.map((x) => (x.id === m.id ? { ...x, unread: false } : x))); };
  const unreadCount = msgs.filter((m) => m.unread).length;
  return (
    <main data-screen-label="Inbox" style={{ maxWidth: 1440, margin: "0 auto", padding: 32, display: "flex", flexDirection: "column", gap: 20 }}>
      <header style={{ display: "flex", alignItems: "center", gap: 16 }}>
        <div style={{ display: "flex", flexDirection: "column", gap: 2 }}>
          <h1 style={{ margin: 0, font: "600 21px var(--font-ui)", letterSpacing: "var(--heading-tracking)", color: "var(--text-ink)" }}>Inbox</h1>
          <span style={{ font: "400 12.5px var(--font-ui)", color: "var(--text-muted)" }}>Everything Verge told you, by class. <span style={{ fontFamily: "var(--font-mono)", fontSize: 12 }}>{unreadCount}</span> unread.</span>
        </div>
        <span style={{ marginLeft: "auto", display: "inline-flex", gap: 8, alignItems: "center" }}>
          <Button variant="ghost" size="sm" disabled={unreadCount === 0} onClick={() => setMsgs((all) => all.map((x) => ({ ...x, unread: false })))}>Mark all read</Button>
          <SegmentedControl label="Filter" value={filter} onChange={setFilter} options={[{ value: "all", label: "All" }, { value: "unread", label: "Unread" }]} />
        </span>
      </header>
      <div style={{ display: "grid", gridTemplateColumns: "380px minmax(0, 1fr)", gap: 24, alignItems: "start" }}>
        <Card pad={10}>
          {shown.length ? <MessageList messages={shown} onOpen={open} /> :
            <EmptyState icon="inbox" message="Nothing unread." detail="New messages land here as batches conclude." style={{ padding: "28px 16px" }} />}
        </Card>
        {sel ? (
          <Card microLabel={sel.cls} title={sel.text} action={<Tag>{sel.cls}</Tag>}>
            <div style={{ display: "flex", flexDirection: "column", gap: 16 }}>
              <span style={{ font: "400 11.5px var(--font-mono)", color: "var(--text-muted)" }}>{sel.time} ago · delivered to ops.acmecorp.io/hook</span>
              <p style={{ margin: 0, font: "400 13.5px/1.65 var(--font-ui)", color: "var(--text-body)", maxWidth: 560 }}>{BODY[sel.cls]}</p>
              <div style={{ display: "flex", gap: 8 }}>
                <Button icon={<Icon name="arrow-right" size={14} />} onClick={() => onNavigate && onNavigate(GOTO[sel.cls][1])}>{GOTO[sel.cls][0]}</Button>
                <Button variant="ghost" onClick={() => { setMsgs((all) => all.map((x) => (x.id === sel.id ? { ...x, unread: true } : x))); setSelId(null); }}>Mark unread</Button>
              </div>
            </div>
          </Card>
        ) : (
          <Card>
            <EmptyState icon="mail-open" message="No message selected." detail="Pick a message on the left to read it here." style={{ padding: "40px 16px" }} />
          </Card>
        )}
      </div>
    </main>
  );
}
