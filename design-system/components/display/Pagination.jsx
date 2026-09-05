import React from "react";

function PageBtn({ children, active, disabled, onClick, label }) {
  const [hov, setHov] = React.useState(false);
  return (
    <button type="button" aria-label={label} disabled={disabled} onClick={onClick}
      onMouseEnter={() => setHov(true)} onMouseLeave={() => setHov(false)}
      style={{ display: "inline-flex", alignItems: "center", justifyContent: "center", minWidth: 28, height: 28, padding: "0 6px", border: "1px solid " + (active ? "var(--accent)" : "transparent"), borderRadius: 8, background: active ? "var(--accent-soft)" : hov && !disabled ? "var(--surface-sunken)" : "transparent", color: active ? "var(--link)" : "var(--text-secondary)", font: (active ? "600" : "500") + " 12px var(--font-mono)", cursor: disabled ? "default" : "pointer", opacity: disabled ? 0.45 : 1, transition: "background var(--dur-fast) var(--ease-out)" }}>
      {children}
    </button>
  );
}

export function Pagination({ page = 1, pageCount = 1, onChange, pageSize, totalItems, style }) {
  const go = (p) => { if (p >= 1 && p <= pageCount && p !== page && onChange) onChange(p); };
  // A fixed slot count is what stops the control resizing as the page moves.
  let pages = [];
  if (pageCount <= 7) { for (let p = 1; p <= pageCount; p++) pages.push(p); }
  else if (page <= 4) pages = [1, 2, 3, 4, 5, "\u2026", pageCount];
  else if (page >= pageCount - 3) pages = [1, "\u2026", pageCount - 4, pageCount - 3, pageCount - 2, pageCount - 1, pageCount];
  else pages = [1, "\u2026", page - 1, page, page + 1, "\u2026", pageCount];
  const from = pageSize ? (page - 1) * pageSize + 1 : null;
  const to = pageSize ? Math.min(page * pageSize, totalItems || page * pageSize) : null;
  return (
    <nav aria-label="Pagination" style={{ display: "flex", alignItems: "center", gap: 4, fontFamily: "var(--font-ui)", ...style }}>
      {pageSize && totalItems != null && (
        <span style={{ font: "400 11.5px var(--font-mono)", color: "var(--text-muted)", marginRight: 10 }}>{from.toLocaleString("en-US")}–{to.toLocaleString("en-US")} of {totalItems.toLocaleString("en-US")}</span>
      )}
      <PageBtn label="Previous page" disabled={page <= 1} onClick={() => go(page - 1)}>
        <svg viewBox="0 0 16 16" width="13" height="13"><path d="M10 3.5L5.5 8 10 12.5" fill="none" stroke="currentColor" strokeWidth="1.75" strokeLinecap="round" strokeLinejoin="round"></path></svg>
      </PageBtn>
      {pages.map((p, i) => p === "\u2026"
        ? <span key={"e" + i} style={{ display: "inline-flex", alignItems: "center", justifyContent: "center", minWidth: 28, height: 28, font: "500 12px var(--font-mono)", color: "var(--text-muted)" }}>{"\u2026"}</span>
        : <PageBtn key={p} active={p === page} label={"Page " + p} onClick={() => go(p)}>{p}</PageBtn>)}
      <PageBtn label="Next page" disabled={page >= pageCount} onClick={() => go(page + 1)}>
        <svg viewBox="0 0 16 16" width="13" height="13"><path d="M6 3.5L10.5 8 6 12.5" fill="none" stroke="currentColor" strokeWidth="1.75" strokeLinecap="round" strokeLinejoin="round"></path></svg>
      </PageBtn>
    </nav>
  );
}
