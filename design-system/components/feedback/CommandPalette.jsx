import React from "react";
import { Icon } from "../media/Icon.jsx";
import { Kbd } from "../display/Kbd.jsx";

/* \u2318K palette. groups: [{label, items: [{id, label, icon?, hint?, onSelect}]}]. Consumer owns the global shortcut. */
export function CommandPalette({ open, onClose, groups = [], placeholder = "Type a command or search\u2026" }) {
  const [q, setQ] = React.useState("");
  const [idx, setIdx] = React.useState(0);
  const listRef = React.useRef(null);
  const panelRef = React.useRef(null);
  React.useEffect(() => {
    if (!open) return;
    const prev = document.activeElement;
    const panel = panelRef.current;
    if (panel && !panel.contains(document.activeElement)) panel.focus();
    const onTab = (e) => {
      if (e.key !== "Tab" || !panelRef.current) return;
      const els = Array.prototype.filter.call(
        panelRef.current.querySelectorAll('button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'),
        (el) => el.offsetParent !== null || el === document.activeElement
      );
      if (!els.length) return;
      const first = els[0], last = els[els.length - 1];
      if (e.shiftKey && document.activeElement === first) { e.preventDefault(); last.focus(); }
      else if (!e.shiftKey && document.activeElement === last) { e.preventDefault(); first.focus(); }
    };
    document.addEventListener("keydown", onTab);
    return () => { document.removeEventListener("keydown", onTab); if (prev && prev.focus) prev.focus(); };
  }, [open]);
  React.useEffect(() => { if (open) { setQ(""); setIdx(0); } }, [open]);
  const filtered = groups.map((g) => ({ label: g.label, items: g.items.filter((it) => !q || (it.label + " " + (it.hint || "")).toLowerCase().includes(q.toLowerCase())) })).filter((g) => g.items.length);
  const flat = [];
  filtered.forEach((g) => g.items.forEach((it) => flat.push(it)));
  const clamp = (n) => (flat.length ? (n + flat.length) % flat.length : 0);
  React.useEffect(() => {
    if (!open) return;
    const onKey = (e) => {
      if (e.key === "Escape") { e.preventDefault(); onClose && onClose(); }
      else if (e.key === "ArrowDown") { e.preventDefault(); setIdx((n) => clamp(n + 1)); }
      else if (e.key === "ArrowUp") { e.preventDefault(); setIdx((n) => clamp(n - 1)); }
      else if (e.key === "Enter") { e.preventDefault(); const it = flat[idx]; if (it) { onClose && onClose(); it.onSelect && it.onSelect(); } }
    };
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, [open, flat.length, idx, onClose]);
  React.useEffect(() => {
    const el = listRef.current && listRef.current.querySelector('[data-active="true"]');
    const box = listRef.current;
    if (el && box) {
      if (el.offsetTop < box.scrollTop) box.scrollTop = el.offsetTop - 6;
      else if (el.offsetTop + el.offsetHeight > box.scrollTop + box.clientHeight) box.scrollTop = el.offsetTop + el.offsetHeight - box.clientHeight + 6;
    }
  }, [idx, q]);
  if (!open) return null;
  let flatIdx = -1;
  return (
    <div onClick={onClose} style={{ position: "fixed", inset: 0, zIndex: 120, background: "rgba(21,18,15,0.4)", animation: "vg-fade-in var(--dur-fast) var(--ease-out)", display: "flex", justifyContent: "center", alignItems: "flex-start", padding: "10vh 16px 16px" }}>
      <div role="dialog" aria-modal="true" aria-label="Command palette" ref={panelRef} tabIndex={-1} onClick={(e) => e.stopPropagation()}
        style={{ position: "relative", width: "min(560px, 100%)", background: "var(--surface)", border: "1px solid var(--border-default)", borderRadius: 16, boxShadow: "var(--shadow-lg)", overflow: "hidden", animation: "vg-pop-in var(--dur-base) var(--ease-out)", fontFamily: "var(--font-ui)" }}>
        <div style={{ display: "flex", alignItems: "center", gap: 10, padding: "0 16px", height: 48, borderBottom: "1px solid var(--row-sep)" }}>
          <span style={{ display: "inline-flex", color: "var(--text-muted)" }}><Icon name="search" size={15} /></span>
          <input autoFocus value={q} placeholder={placeholder}
            onChange={(e) => { setQ(e.target.value); setIdx(0); }}
            style={{ flex: 1, border: "none", outline: "none", background: "transparent", color: "var(--text-ink)", font: "400 13px var(--font-mono)" }} />
          <Kbd keys="esc" />
        </div>
        <div ref={listRef} style={{ maxHeight: 320, overflowY: "auto", overflowX: "hidden", padding: 6 }}>
          {flat.length === 0 && <div style={{ padding: "18px 12px", font: "400 12.5px var(--font-ui)", color: "var(--text-muted)" }}>No matches.</div>}
          {filtered.map((g) => (
            <div key={g.label}>
              <div style={{ padding: "8px 10px 4px", font: "500 10.5px var(--font-mono)", letterSpacing: "0.07em", textTransform: "uppercase", color: "var(--text-muted)" }}>{g.label}</div>
              {g.items.map((it) => {
                flatIdx++;
                const i = flatIdx;
                const active = i === idx;
                return (
                  <button key={it.id || it.label} type="button" data-active={active ? "true" : "false"}
                    onMouseEnter={() => setIdx(i)}
                    onClick={() => { onClose && onClose(); it.onSelect && it.onSelect(); }}
                    style={{ display: "flex", alignItems: "center", gap: 10, width: "100%", padding: "8px 10px", border: "none", borderRadius: 10, textAlign: "left", cursor: "pointer", background: active ? "var(--accent-soft)" : "transparent", color: active ? "var(--link)" : "var(--text-body)", font: "500 13px var(--font-ui)" }}>
                    {it.icon && <span style={{ display: "inline-flex", color: active ? "var(--link)" : "var(--text-secondary)" }}><Icon name={it.icon} size={15} /></span>}
                    <span style={{ flex: 1, whiteSpace: "nowrap", overflow: "hidden", textOverflow: "ellipsis" }}>{it.label}</span>
                    {it.hint && <span style={{ font: "400 11px var(--font-mono)", color: "var(--text-muted)" }}>{it.hint}</span>}
                  </button>
                );
              })}
            </div>
          ))}
        </div>
        <div style={{ display: "flex", alignItems: "center", gap: 14, padding: "8px 16px", borderTop: "1px solid var(--row-sep)", background: "var(--surface-sunken)" }}>
          <span style={{ display: "inline-flex", alignItems: "center", gap: 6, font: "400 11px var(--font-ui)", color: "var(--text-muted)" }}><Kbd keys={["\u2191", "\u2193"]} /> navigate</span>
          <span style={{ display: "inline-flex", alignItems: "center", gap: 6, font: "400 11px var(--font-ui)", color: "var(--text-muted)" }}><Kbd keys={"\u21b5"} /> run</span>
        </div>
      </div>
    </div>
  );
}
